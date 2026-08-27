package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"
)

const localRefreshInterval = 2 * time.Second

type sessionRequest byte

const (
	requestQuit sessionRequest = iota + 1
	requestLocalRefresh
	requestRescan
	requestFetch
	requestUp
	requestDown
	requestPageUp
	requestPageDown
)

type sessionTask byte

const (
	taskLocal sessionTask = iota + 1
	taskRescan
	taskFetch
)

type sessionTaskResult struct {
	task         sessionTask
	repositories []string
	scanErrors   []error
	err          error
}

type interactiveSession struct {
	root         string
	exclusions   map[string]struct{}
	repositories []string
	coordinator  *repositoryCoordinator
	dashboard    *terminalDashboard
	stderr       io.Writer
	allowFetch   bool
	discover     func(context.Context, string, map[string]struct{}) ([]string, []error, error)
	inspect      func(context.Context, string, string) repoStatus
	fetch        func(context.Context, string) error
}

func newInteractiveSession(
	root string,
	exclusions map[string]struct{},
	repositories []string,
	coordinator *repositoryCoordinator,
	dashboard *terminalDashboard,
	stderr io.Writer,
	allowFetch bool,
) *interactiveSession {
	return &interactiveSession{
		root:         root,
		exclusions:   exclusions,
		repositories: slices.Clone(repositories),
		coordinator:  coordinator,
		dashboard:    dashboard,
		stderr:       stderr,
		allowFetch:   allowFetch,
		discover:     discover,
		inspect:      inspectRepository,
		fetch:        fetchRepository,
	}
}

func (session *interactiveSession) run(ctx context.Context, requests <-chan sessionRequest) error {
	localTicks := time.NewTicker(localRefreshInterval)
	defer localTicks.Stop()
	var remoteTicks *time.Ticker
	var remoteTick <-chan time.Time
	if session.allowFetch {
		remoteTicks = time.NewTicker(defaultFetchInterval)
		remoteTick = remoteTicks.C
		defer remoteTicks.Stop()
	}
	return session.runWithTicks(ctx, requests, localTicks.C, remoteTick)
}

func (session *interactiveSession) runWithTicks(
	parent context.Context,
	requests <-chan sessionRequest,
	localTicks, remoteTicks <-chan time.Time,
) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	done := make(chan sessionTaskResult, 1)
	active := sessionTask(0)
	pendingLocal := true
	pendingFetch := session.allowFetch
	pendingRescan := false

	startNext := func() {
		if active != 0 {
			return
		}
		switch {
		case pendingRescan:
			active, pendingRescan, pendingLocal = taskRescan, false, false
		case pendingLocal:
			active, pendingLocal = taskLocal, false
		case pendingFetch:
			active, pendingFetch = taskFetch, false
		default:
			return
		}
		repositories := slices.Clone(session.repositories)
		go func(task sessionTask) {
			done <- session.runTask(ctx, task, repositories)
		}(active)
	}

	queue := func(task sessionTask) {
		switch task {
		case taskLocal:
			if active != taskLocal && active != taskRescan && !pendingRescan {
				pendingLocal = true
			}
		case taskRescan:
			if active != taskRescan {
				pendingRescan = true
			}
			pendingLocal = false
		case taskFetch:
			if session.allowFetch && active != taskFetch {
				pendingFetch = true
			}
		}
	}
	finish := func(result sessionTaskResult) error {
		active = 0
		if result.task == taskRescan && result.err == nil {
			session.repositories = result.repositories
		}
		for _, scanErr := range result.scanErrors {
			fmt.Fprintf(session.stderr, "warning: %s\n", oneLine(scanErr.Error()))
		}
		if result.err != nil && !errors.Is(result.err, context.Canceled) {
			return result.err
		}
		return nil
	}

	for {
		startNext()
		select {
		case <-parent.Done():
			cancel()
			if active != 0 {
				if err := finish(<-done); err != nil {
					return err
				}
			}
			return nil
		case request, ok := <-requests:
			if !ok {
				requests = nil
				continue
			}
			switch request {
			case requestQuit:
				cancel()
				if active != 0 {
					if err := finish(<-done); err != nil {
						return err
					}
				}
				return nil
			case requestLocalRefresh:
				queue(taskLocal)
			case requestRescan:
				queue(taskRescan)
			case requestFetch:
				queue(taskFetch)
			case requestUp:
				session.dashboard.moveSelection(-1)
			case requestDown:
				session.dashboard.moveSelection(1)
			case requestPageUp:
				session.dashboard.moveSelection(-session.dashboard.pageSize())
			case requestPageDown:
				session.dashboard.moveSelection(session.dashboard.pageSize())
			}
		case <-localTicks:
			queue(taskLocal)
		case <-remoteTicks:
			queue(taskFetch)
		case result := <-done:
			if err := finish(result); err != nil {
				return err
			}
		}
		if err := session.dashboard.writeError(); err != nil {
			cancel()
			if active != 0 {
				_ = finish(<-done)
			}
			return err
		}
	}
}

func (session *interactiveSession) runTask(ctx context.Context, task sessionTask, repositories []string) sessionTaskResult {
	result := sessionTaskResult{task: task, repositories: repositories}
	switch task {
	case taskRescan:
		result.repositories, result.scanErrors, result.err = session.discover(ctx, session.root, session.exclusions)
		if result.err == nil {
			result.err = refreshRepositories(ctx, session.coordinator, session.root, result.repositories, session.inspect, session.dashboard.publish)
		}
	case taskLocal:
		result.err = refreshRepositories(ctx, session.coordinator, session.root, repositories, session.inspect, session.dashboard.publish)
	case taskFetch:
		result.err = refreshRemotes(ctx, session.coordinator, session.root, repositories, session.fetch, session.inspect, session.dashboard.publish)
	}
	return result
}
