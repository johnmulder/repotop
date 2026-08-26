package main

import (
	"context"
	"sync"
	"time"
)

const remoteFetchWorkers = 4

func refreshRemotes(
	ctx context.Context,
	coordinator *repositoryCoordinator,
	root string,
	repositories []string,
	fetch func(context.Context, string) error,
	inspect func(context.Context, string, string) repoStatus,
	publish snapshotPublisher,
) error {
	emitter := newSnapshotEmitter(coordinator, publish, time.Now)
	unique := make([]string, 0, len(repositories))
	seen := make(map[string]struct{}, len(repositories))
	for _, repository := range repositories {
		if _, ok := seen[repository]; ok {
			continue
		}
		seen[repository] = struct{}{}
		if !coordinator.canFetch(repository) {
			continue
		}
		unique = append(unique, repository)
	}
	if len(unique) == 0 {
		return ctx.Err()
	}
	coordinator.beginFetch(unique)
	emitter.finished()

	jobs := make(chan string)
	updates := make(chan repositoryFetchUpdate)
	var workers sync.WaitGroup
	workers.Add(min(remoteFetchWorkers, len(unique)))
	for range min(remoteFetchWorkers, len(unique)) {
		go func() {
			defer workers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case repository, ok := <-jobs:
					if !ok {
						return
					}
					startedAt := time.Now()
					fetchErr := fetch(ctx, repository)
					finishedAt := time.Now()
					var status *repoStatus
					if fetchErr == nil {
						refreshed := inspect(ctx, root, repository)
						status = &refreshed
					}
					update := repositoryFetchUpdate{
						Repository: repository,
						StartedAt:  startedAt,
						FinishedAt: finishedAt,
						Status:     status,
					}
					if fetchErr != nil {
						update.Error = formatGitDiagnostic(fetchErr.Error(), false)
					}
					select {
					case updates <- update:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, repository := range unique {
			select {
			case jobs <- repository:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		workers.Wait()
		close(updates)
	}()

	for update := range updates {
		if coordinator.applyFetch(update) {
			emitter.updated()
		}
	}
	coordinator.finishFetch(unique)
	if err := ctx.Err(); err != nil {
		emitter.finished()
		return err
	}
	emitter.finished()
	return nil
}
