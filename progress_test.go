package main

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSnapshotEmitterPublishesFirstCoalescesAndFinishes(t *testing.T) {
	coordinator := newRepositoryCoordinator()
	generation := coordinator.beginScan()
	now := time.Unix(100, 0)
	var sizes []int
	emitter := newSnapshotEmitter(coordinator, func(statuses []repositorySnapshot) {
		sizes = append(sizes, len(statuses))
	}, func() time.Time { return now })

	coordinator.apply(repositoryUpdate{Generation: generation, Repository: "/a", Status: repoStatus{Path: "a"}})
	emitter.updated()
	now = now.Add(10 * time.Millisecond)
	coordinator.apply(repositoryUpdate{Generation: generation, Repository: "/b", Status: repoStatus{Path: "b"}})
	emitter.updated()
	now = now.Add(40 * time.Millisecond)
	coordinator.apply(repositoryUpdate{Generation: generation, Repository: "/c", Status: repoStatus{Path: "c"}})
	emitter.updated()
	now = now.Add(time.Millisecond)
	coordinator.apply(repositoryUpdate{Generation: generation, Repository: "/d", Status: repoStatus{Path: "d"}})
	emitter.updated()
	emitter.finished()

	want := []int{1, 3, 4}
	if len(sizes) != len(want) {
		t.Fatalf("published sizes = %v", sizes)
	}
	for index := range want {
		if sizes[index] != want[index] {
			t.Fatalf("published sizes = %v, want %v", sizes, want)
		}
	}
}

func TestRefreshPipelinesPublishProgressAndCompletion(t *testing.T) {
	repositories := []string{filepath.Join("/root", "a"), filepath.Join("/root", "b")}
	coordinator := newRepositoryCoordinator()
	var localSnapshots [][]repositorySnapshot
	inspect := func(_ context.Context, _, repository string) repoStatus {
		return repoStatus{Path: filepath.Base(repository), Branch: "main", Ahead: 2, HasUpstream: true}
	}
	if err := refreshRepositories(context.Background(), coordinator, "/root", repositories, inspect, func(statuses []repositorySnapshot) {
		localSnapshots = append(localSnapshots, append([]repositorySnapshot(nil), statuses...))
	}); err != nil {
		t.Fatal(err)
	}
	if len(localSnapshots) < 2 || len(localSnapshots[0]) != 1 || len(localSnapshots[len(localSnapshots)-1]) != 2 {
		t.Fatalf("local snapshot sizes = %v", snapshotSizes(localSnapshots))
	}

	var remoteSnapshots [][]repositorySnapshot
	refreshedInspect := func(_ context.Context, _, repository string) repoStatus {
		return repoStatus{Path: filepath.Base(repository), Branch: "main", Ahead: 1, HasUpstream: true}
	}
	if err := refreshRemotes(context.Background(), coordinator, "/root", repositories,
		func(context.Context, string) error { return nil }, refreshedInspect,
		func(statuses []repositorySnapshot) {
			remoteSnapshots = append(remoteSnapshots, append([]repositorySnapshot(nil), statuses...))
		}); err != nil {
		t.Fatal(err)
	}
	if len(remoteSnapshots) < 3 || countFetching(remoteSnapshots[0]) != 2 || countAhead(remoteSnapshots[len(remoteSnapshots)-1], 1) != 2 || countFetching(remoteSnapshots[len(remoteSnapshots)-1]) != 0 {
		t.Fatalf("remote snapshots = %+v", remoteSnapshots)
	}
}

func TestTerminalDashboardRepaintsOnChangeAndResizeByIdentity(t *testing.T) {
	var output bytes.Buffer
	width := 80
	dashboard := newTerminalDashboard(&output, func() int { return width }, asciiPalette)
	dashboard.publish(snapshotsOf(repoStatus{Path: "z-selected", Branch: "main", HasUpstream: true}))
	dashboard.publish(snapshotsOf(
		repoStatus{Path: "a-error", Error: "broken"},
		repoStatus{Path: "z-selected", Branch: "main", Changed: 1, HasUpstream: true},
	))
	if dashboard.selected != "z-selected" || !strings.Contains(dashboard.last, "> z-selected") {
		t.Fatalf("selection moved after reorder: selected=%q\n%s", dashboard.selected, dashboard.last)
	}
	beforeDuplicate := output.Len()
	dashboard.publish(dashboard.latest)
	if output.Len() != beforeDuplicate {
		t.Fatal("unchanged snapshot was repainted")
	}

	width = 30
	dashboard.redraw()
	if dashboard.selected != "z-selected" || !strings.Contains(dashboard.last, "> z...cted") {
		t.Fatalf("selection moved after resize: selected=%q\n%s", dashboard.selected, dashboard.last)
	}
	dashboard.publish(nil)
	if dashboard.selected != "" || !strings.Contains(dashboard.last, "0 repos") {
		t.Fatalf("empty snapshot was not rendered: selected=%q\n%s", dashboard.selected, dashboard.last)
	}
	if got := strings.Count(output.String(), "\x1b[H\x1b[2J"); got != 4 {
		t.Fatalf("repaint count = %d", got)
	}
	if err := dashboard.writeError(); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalDashboardMovesSelectionThroughViewport(t *testing.T) {
	var output bytes.Buffer
	dashboard := newSizedTerminalDashboard(&output, func() int { return 80 }, func() int { return 12 }, asciiPalette)
	statuses := make([]repositorySnapshot, 20)
	for index := range statuses {
		statuses[index] = repositorySnapshot{repoStatus: repoStatus{
			Path:        fmt.Sprintf("repo-%02d", index),
			Branch:      "main",
			HasUpstream: true,
		}}
	}
	dashboard.publish(statuses)
	dashboard.moveSelection(dashboard.pageSize())
	if dashboard.selected != "repo-04" || !strings.Contains(dashboard.last, "> repo-04") {
		t.Fatalf("page selection = %q\n%s", dashboard.selected, dashboard.last)
	}
	if lines := strings.Count(dashboard.last, "\n"); lines > 12 {
		t.Fatalf("viewport uses %d lines:\n%s", lines, dashboard.last)
	}
	dashboard.moveSelection(100)
	if dashboard.selected != "repo-19" || !strings.Contains(dashboard.last, "> repo-19") {
		t.Fatalf("last selection = %q\n%s", dashboard.selected, dashboard.last)
	}
	dashboard.moveSelection(-100)
	if dashboard.selected != "repo-00" {
		t.Fatalf("first selection = %q", dashboard.selected)
	}
}

func snapshotSizes(snapshots [][]repositorySnapshot) []int {
	sizes := make([]int, len(snapshots))
	for index := range snapshots {
		sizes[index] = len(snapshots[index])
	}
	return sizes
}

func countAhead(statuses []repositorySnapshot, ahead int) int {
	count := 0
	for _, status := range statuses {
		if status.Ahead == ahead {
			count++
		}
	}
	return count
}

func countFetching(statuses []repositorySnapshot) int {
	count := 0
	for _, status := range statuses {
		if status.Fetching {
			count++
		}
	}
	return count
}
