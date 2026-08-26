package main

import (
	"context"
	"sync"
	"time"
)

const localStatusWorkers = 8

type repositoryUpdate struct {
	Generation uint64
	Repository string
	Status     repoStatus
}

type fetchStatus struct {
	LastAttempt time.Time
	LastSuccess time.Time
	Duration    time.Duration
	Error       string
}

type repositoryFetchUpdate struct {
	Repository string
	StartedAt  time.Time
	FinishedAt time.Time
	Status     *repoStatus
	Error      string
}

type repositoryRecord struct {
	Status repoStatus
	Fetch  fetchStatus
}

// repositoryCoordinator is owned by one goroutine; workers communicate only
// through typed update values.
type repositoryCoordinator struct {
	generation   uint64
	repositories map[string]repositoryRecord
}

func newRepositoryCoordinator() *repositoryCoordinator {
	return &repositoryCoordinator{repositories: make(map[string]repositoryRecord)}
}

func (coordinator *repositoryCoordinator) beginScan() uint64 {
	coordinator.generation++
	return coordinator.generation
}

func (coordinator *repositoryCoordinator) apply(update repositoryUpdate) bool {
	if update.Generation != coordinator.generation {
		return false
	}
	record := coordinator.repositories[update.Repository]
	record.Status = update.Status
	coordinator.repositories[update.Repository] = record
	return true
}

func (coordinator *repositoryCoordinator) applyFetch(update repositoryFetchUpdate) bool {
	record, ok := coordinator.repositories[update.Repository]
	if !ok {
		return false
	}
	record.Fetch.LastAttempt = update.StartedAt
	record.Fetch.Duration = update.FinishedAt.Sub(update.StartedAt)
	record.Fetch.Error = update.Error
	if update.Error == "" {
		record.Fetch.LastSuccess = update.FinishedAt
		if update.Status != nil {
			record.Status = *update.Status
		}
	}
	coordinator.repositories[update.Repository] = record
	return true
}

func (coordinator *repositoryCoordinator) fetchSnapshot(repository string) (fetchStatus, bool) {
	record, ok := coordinator.repositories[repository]
	return record.Fetch, ok
}

func (coordinator *repositoryCoordinator) completeScan(generation uint64, repositories []string) bool {
	if generation != coordinator.generation {
		return false
	}
	present := make(map[string]struct{}, len(repositories))
	for _, repository := range repositories {
		present[repository] = struct{}{}
	}
	for repository := range coordinator.repositories {
		if _, ok := present[repository]; !ok {
			delete(coordinator.repositories, repository)
		}
	}
	return true
}

func (coordinator *repositoryCoordinator) snapshot() []repoStatus {
	statuses := make([]repoStatus, 0, len(coordinator.repositories))
	for _, record := range coordinator.repositories {
		statuses = append(statuses, record.Status)
	}
	sortStatuses(statuses)
	return statuses
}

func refreshRepositories(
	ctx context.Context,
	coordinator *repositoryCoordinator,
	root string,
	repositories []string,
	inspect func(context.Context, string, string) repoStatus,
) error {
	generation := coordinator.beginScan()
	if len(repositories) == 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		coordinator.completeScan(generation, repositories)
		return nil
	}

	jobs := make(chan string)
	updates := make(chan repositoryUpdate)
	var workers sync.WaitGroup
	workers.Add(min(localStatusWorkers, len(repositories)))
	for range min(localStatusWorkers, len(repositories)) {
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
					update := repositoryUpdate{
						Generation: generation,
						Repository: repository,
						Status:     inspect(ctx, root, repository),
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
		for _, repository := range repositories {
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
		coordinator.apply(update)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	coordinator.completeScan(generation, repositories)
	return nil
}
