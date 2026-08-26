package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
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

	if err := refreshRepositories(context.Background(), coordinator, "/root", repositories, inspect); err != nil {
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
		done <- refreshRepositories(ctx, coordinator, "/", []string{"/new"}, inspect)
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
