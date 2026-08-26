package main

import (
	"bytes"
	"context"
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
	emitter := newSnapshotEmitter(coordinator, func(statuses []repoStatus) {
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
	var localSnapshots [][]repoStatus
	inspect := func(_ context.Context, _, repository string) repoStatus {
		return repoStatus{Path: filepath.Base(repository), Branch: "main", Ahead: 2, HasUpstream: true}
	}
	if err := refreshRepositories(context.Background(), coordinator, "/root", repositories, inspect, func(statuses []repoStatus) {
		localSnapshots = append(localSnapshots, append([]repoStatus(nil), statuses...))
	}); err != nil {
		t.Fatal(err)
	}
	if len(localSnapshots) < 2 || len(localSnapshots[0]) != 1 || len(localSnapshots[len(localSnapshots)-1]) != 2 {
		t.Fatalf("local snapshot sizes = %v", snapshotSizes(localSnapshots))
	}

	var remoteSnapshots [][]repoStatus
	refreshedInspect := func(_ context.Context, _, repository string) repoStatus {
		return repoStatus{Path: filepath.Base(repository), Branch: "main", Ahead: 1, HasUpstream: true}
	}
	if err := refreshRemotes(context.Background(), coordinator, "/root", repositories,
		func(context.Context, string) error { return nil }, refreshedInspect,
		func(statuses []repoStatus) {
			remoteSnapshots = append(remoteSnapshots, append([]repoStatus(nil), statuses...))
		}); err != nil {
		t.Fatal(err)
	}
	if len(remoteSnapshots) < 2 || countAhead(remoteSnapshots[0], 1) != 1 || countAhead(remoteSnapshots[len(remoteSnapshots)-1], 1) != 2 {
		t.Fatalf("remote snapshots = %+v", remoteSnapshots)
	}
}

func TestTerminalDashboardRepaintsOnChangeAndResizeByIdentity(t *testing.T) {
	var output bytes.Buffer
	width := 80
	dashboard := newTerminalDashboard(&output, func() int { return width })
	dashboard.publish([]repoStatus{{Path: "z-selected", Branch: "main", HasUpstream: true}})
	dashboard.publish([]repoStatus{
		{Path: "a-error", Error: "broken"},
		{Path: "z-selected", Branch: "main", Changed: 1, HasUpstream: true},
	})
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

func snapshotSizes(snapshots [][]repoStatus) []int {
	sizes := make([]int, len(snapshots))
	for index := range snapshots {
		sizes[index] = len(snapshots[index])
	}
	return sizes
}

func countAhead(statuses []repoStatus, ahead int) int {
	count := 0
	for _, status := range statuses {
		if status.Ahead == ahead {
			count++
		}
	}
	return count
}
