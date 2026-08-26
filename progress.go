package main

import (
	"io"
	"os"
	"slices"
	"sync"
	"time"
)

const snapshotInterval = 50 * time.Millisecond

type snapshotPublisher func([]repoStatus)

type snapshotEmitter struct {
	coordinator *repositoryCoordinator
	publish     snapshotPublisher
	now         func() time.Time
	last        time.Time
	published   bool
}

func newSnapshotEmitter(coordinator *repositoryCoordinator, publish snapshotPublisher, now func() time.Time) *snapshotEmitter {
	return &snapshotEmitter{coordinator: coordinator, publish: publish, now: now}
}

func (emitter *snapshotEmitter) updated() {
	if emitter.publish == nil {
		return
	}
	now := emitter.now()
	if !emitter.published || now.Sub(emitter.last) >= snapshotInterval {
		emitter.publish(emitter.coordinator.snapshot())
		emitter.last = now
		emitter.published = true
	}
}

func (emitter *snapshotEmitter) finished() {
	if emitter.publish != nil {
		emitter.publish(emitter.coordinator.snapshot())
	}
}

type terminalDashboard struct {
	mu       sync.Mutex
	output   io.Writer
	width    func() int
	latest   []repoStatus
	selected string
	last     string
	err      error
}

func terminalDashboardFor(output io.Writer) *terminalDashboard {
	file, ok := output.(*os.File)
	if !ok {
		return nil
	}
	if _, ok := terminalWidth(file); !ok {
		return nil
	}
	return newTerminalDashboard(output, func() int {
		width, ok := terminalWidth(file)
		if !ok {
			return defaultRenderWidth
		}
		return width
	})
}

func newTerminalDashboard(output io.Writer, width func() int) *terminalDashboard {
	return &terminalDashboard{output: output, width: width}
}

func (dashboard *terminalDashboard) publish(statuses []repoStatus) {
	dashboard.mu.Lock()
	defer dashboard.mu.Unlock()
	dashboard.latest = slices.Clone(statuses)
	if !containsPath(statuses, dashboard.selected) {
		ordered := slices.Clone(statuses)
		sortStatuses(ordered)
		if len(ordered) == 0 {
			dashboard.selected = ""
		} else {
			dashboard.selected = ordered[0].Path
		}
	}
	dashboard.drawLocked()
}

func (dashboard *terminalDashboard) redraw() {
	dashboard.mu.Lock()
	defer dashboard.mu.Unlock()
	dashboard.drawLocked()
}

func (dashboard *terminalDashboard) drawLocked() {
	if dashboard.err != nil {
		return
	}
	view := renderSnapshot(dashboard.latest, dashboard.width(), dashboard.selected)
	if view == dashboard.last {
		return
	}
	_, dashboard.err = io.WriteString(dashboard.output, "\x1b[H\x1b[2J"+view)
	if dashboard.err == nil {
		dashboard.last = view
	}
}

func (dashboard *terminalDashboard) writeError() error {
	dashboard.mu.Lock()
	defer dashboard.mu.Unlock()
	return dashboard.err
}

func containsPath(statuses []repoStatus, path string) bool {
	if path == "" {
		return false
	}
	return slices.ContainsFunc(statuses, func(status repoStatus) bool { return status.Path == path })
}
