package main

import (
	"context"
	"sync"
)

const localStatusWorkers = 8

type repositoryUpdate struct {
	Generation uint64
	Repository string
	Status     repoStatus
}

// repositoryCoordinator is owned by one goroutine; workers communicate only
// through repositoryUpdate values.
type repositoryCoordinator struct {
	generation   uint64
	repositories map[string]repoStatus
}

func newRepositoryCoordinator() *repositoryCoordinator {
	return &repositoryCoordinator{repositories: make(map[string]repoStatus)}
}

func (coordinator *repositoryCoordinator) beginScan() uint64 {
	coordinator.generation++
	return coordinator.generation
}

func (coordinator *repositoryCoordinator) apply(update repositoryUpdate) bool {
	if update.Generation != coordinator.generation {
		return false
	}
	coordinator.repositories[update.Repository] = update.Status
	return true
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
	for _, status := range coordinator.repositories {
		statuses = append(statuses, status)
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
