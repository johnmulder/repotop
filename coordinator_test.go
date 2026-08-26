package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCoordinatorRejectsStaleUpdatesAndDefersRemoval(t *testing.T) {
	coordinator := newRepositoryCoordinator()
	first := coordinator.beginScan()
	coordinator.apply(repositoryUpdate{Generation: first, Repository: "/a", Status: repoStatus{Path: "a", Branch: "old"}})
	coordinator.apply(repositoryUpdate{Generation: first, Repository: "/b", Status: repoStatus{Path: "b", Branch: "old"}})
	if !coordinator.completeScan(first, []string{"/a", "/b"}) {
		t.Fatal("first scan did not complete")
	}

	second := coordinator.beginScan()
	coordinator.apply(repositoryUpdate{Generation: second, Repository: "/a", Status: repoStatus{Path: "a", Branch: "new"}})
	if coordinator.apply(repositoryUpdate{Generation: first, Repository: "/b", Status: repoStatus{Path: "b", Branch: "stale"}}) {
		t.Fatal("stale update was accepted")
	}
	beforeCompletion := coordinator.snapshot()
	if len(beforeCompletion) != 2 || beforeCompletion[1].Branch != "old" {
		t.Fatalf("repository was removed before completion: %+v", beforeCompletion)
	}
	if coordinator.completeScan(first, []string{"/a"}) {
		t.Fatal("stale completion was accepted")
	}
	if !coordinator.completeScan(second, []string{"/a"}) {
		t.Fatal("active scan did not complete")
	}
	afterCompletion := coordinator.snapshot()
	if len(afterCompletion) != 1 || afterCompletion[0].Path != "a" || afterCompletion[0].Branch != "new" {
		t.Fatalf("unexpected completed snapshot: %+v", afterCompletion)
	}
}

func TestCoordinatorSnapshotIsDetached(t *testing.T) {
	coordinator := newRepositoryCoordinator()
	generation := coordinator.beginScan()
	coordinator.apply(repositoryUpdate{Generation: generation, Repository: "/repo", Status: repoStatus{Path: "repo", Branch: "main"}})
	coordinator.completeScan(generation, []string{"/repo"})

	snapshot := coordinator.snapshot()
	snapshot[0].Branch = "mutated"
	if got := coordinator.snapshot()[0].Branch; got != "main" {
		t.Fatalf("coordinator state mutated through snapshot: %q", got)
	}
}

func TestCoordinatorKeepsFetchMetadataSeparate(t *testing.T) {
	coordinator := newRepositoryCoordinator()
	generation := coordinator.beginScan()
	coordinator.apply(repositoryUpdate{Generation: generation, Repository: "/repo", Status: repoStatus{Path: "repo", Branch: "main", Ahead: 2}})
	coordinator.completeScan(generation, []string{"/repo"})
	coordinator.beginFetch([]string{"/repo"})
	if !coordinator.snapshot()[0].Fetching {
		t.Fatal("fetch activity was not exposed")
	}

	firstAttempt := time.Unix(10, 0)
	firstSuccess := time.Unix(12, 0)
	refreshed := repoStatus{Path: "repo", Branch: "main", Ahead: 1}
	if !coordinator.applyFetch(repositoryFetchUpdate{
		Repository: "/repo", StartedAt: firstAttempt, FinishedAt: firstSuccess, Status: &refreshed,
	}) {
		t.Fatal("successful fetch update was rejected")
	}
	failureAttempt := time.Unix(20, 0)
	if !coordinator.applyFetch(repositoryFetchUpdate{
		Repository: "/repo", StartedAt: failureAttempt, FinishedAt: time.Unix(23, 0), Error: "offline",
	}) {
		t.Fatal("failed fetch update was rejected")
	}

	fetch, ok := coordinator.fetchSnapshot("/repo")
	if !ok || !fetch.LastAttempt.Equal(failureAttempt) || !fetch.LastSuccess.Equal(firstSuccess) || fetch.Duration != 3*time.Second || fetch.Error != "offline" {
		t.Fatalf("unexpected fetch metadata: %+v", fetch)
	}
	status := coordinator.snapshot()[0]
	if status.Ahead != 1 || status.Error != "" {
		t.Fatalf("failed fetch changed local status: %+v", status)
	}
}

func TestRefreshRemotesBoundsDeduplicatesAndIsolatesFailures(t *testing.T) {
	coordinator := newRepositoryCoordinator()
	repositories := make([]string, 12)
	generation := coordinator.beginScan()
	for index := range repositories {
		repository := filepath.Join("/root", fmt.Sprintf("repo-%02d", index))
		repositories[index] = repository
		coordinator.apply(repositoryUpdate{
			Generation: generation,
			Repository: repository,
			Status:     repoStatus{Path: filepath.Base(repository), Branch: "main", Ahead: 2, HasUpstream: true},
		})
	}
	coordinator.completeScan(generation, repositories)
	requests := append(append([]string{}, repositories...), repositories...)

	release := make(chan struct{})
	started := make(chan struct{}, len(repositories))
	var active atomic.Int64
	var maximum atomic.Int64
	counts := make(map[string]int)
	var countsMu sync.Mutex
	fetch := func(ctx context.Context, repository string) error {
		countsMu.Lock()
		counts[repository]++
		countsMu.Unlock()
		current := active.Add(1)
		for {
			previous := maximum.Load()
			if current <= previous || maximum.CompareAndSwap(previous, current) {
				break
			}
		}
		started <- struct{}{}
		select {
		case <-release:
		case <-ctx.Done():
			active.Add(-1)
			return ctx.Err()
		}
		active.Add(-1)
		if repository == repositories[0] {
			return errors.New("remote unavailable")
		}
		return nil
	}
	inspect := func(_ context.Context, _, repository string) repoStatus {
		return repoStatus{Path: filepath.Base(repository), Branch: "main", Ahead: 1, HasUpstream: true}
	}
	done := make(chan error, 1)
	go func() {
		done <- refreshRemotes(context.Background(), coordinator, "/root", requests, fetch, inspect, nil)
	}()
	for range remoteFetchWorkers {
		<-started
	}
	select {
	case <-started:
		t.Fatal("more fetches started than the worker bound")
	default:
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	if got := maximum.Load(); got != remoteFetchWorkers {
		t.Fatalf("maximum concurrent fetches = %d", got)
	}
	countsMu.Lock()
	defer countsMu.Unlock()
	for _, repository := range repositories {
		if counts[repository] != 1 {
			t.Fatalf("fetch count for %s = %d", repository, counts[repository])
		}
	}
	failed, _ := coordinator.fetchSnapshot(repositories[0])
	if failed.Error == "" || coordinator.repositories[repositories[0]].Status.Ahead != 2 {
		t.Fatalf("failed fetch state = %+v, record = %+v", failed, coordinator.repositories[repositories[0]])
	}
	for _, repository := range repositories[1:] {
		fetchState, _ := coordinator.fetchSnapshot(repository)
		if fetchState.Error != "" || fetchState.LastSuccess.IsZero() || coordinator.repositories[repository].Status.Ahead != 1 {
			t.Fatalf("successful fetch state for %s = %+v, record = %+v", repository, fetchState, coordinator.repositories[repository])
		}
	}
}

func TestRefreshRemotesCancellation(t *testing.T) {
	coordinator := newRepositoryCoordinator()
	generation := coordinator.beginScan()
	coordinator.apply(repositoryUpdate{Generation: generation, Repository: "/repo", Status: repoStatus{Path: "repo", HasRemote: true, HasUpstream: true}})
	coordinator.completeScan(generation, []string{"/repo"})

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	fetch := func(ctx context.Context, _ string) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}
	done := make(chan error, 1)
	go func() {
		done <- refreshRemotes(ctx, coordinator, "/", []string{"/repo"}, fetch, inspectRepository, nil)
	}()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("refresh error = %v", err)
	}
}

func TestRefreshRepositoriesBoundsConcurrency(t *testing.T) {
	coordinator := newRepositoryCoordinator()
	repositories := make([]string, 32)
	for index := range repositories {
		repositories[index] = filepath.Join("/root", fmt.Sprintf("repo-%02d", index))
	}
	var active atomic.Int64
	var maximum atomic.Int64
	inspect := func(_ context.Context, root, repository string) repoStatus {
		current := active.Add(1)
		for {
			previous := maximum.Load()
			if current <= previous || maximum.CompareAndSwap(previous, current) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		active.Add(-1)
		relative, _ := filepath.Rel(root, repository)
		return repoStatus{Path: relative, Branch: "main"}
	}

	if err := refreshRepositories(context.Background(), coordinator, "/root", repositories, inspect, nil); err != nil {
		t.Fatal(err)
	}
	if got := maximum.Load(); got < 2 || got > localStatusWorkers {
		t.Fatalf("maximum concurrent inspections = %d", got)
	}
	if got := len(coordinator.snapshot()); got != len(repositories) {
		t.Fatalf("snapshot length = %d, want %d", got, len(repositories))
	}
}

func TestCanceledRefreshPreservesPriorSnapshot(t *testing.T) {
	coordinator := newRepositoryCoordinator()
	first := coordinator.beginScan()
	coordinator.apply(repositoryUpdate{Generation: first, Repository: "/old", Status: repoStatus{Path: "old", Branch: "main"}})
	coordinator.completeScan(first, []string{"/old"})

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	inspect := func(ctx context.Context, _, _ string) repoStatus {
		close(started)
		<-ctx.Done()
		return repoStatus{Path: "new", Error: ctx.Err().Error()}
	}
	done := make(chan error, 1)
	go func() {
		done <- refreshRepositories(ctx, coordinator, "/", []string{"/new"}, inspect, nil)
	}()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("refresh error = %v", err)
	}
	snapshot := coordinator.snapshot()
	foundOld := false
	for _, status := range snapshot {
		foundOld = foundOld || status.Path == "old"
	}
	if !foundOld {
		t.Fatalf("canceled refresh changed prior snapshot: %+v", snapshot)
	}
}
