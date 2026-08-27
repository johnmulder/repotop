package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestReadSessionRequest(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		requests []sessionRequest
	}{
		{name: "commands", input: "xrfq", requests: []sessionRequest{requestRescan, requestFetch, requestQuit}},
		{name: "control c", input: "\x03", requests: []sessionRequest{requestQuit}},
		{name: "arrows", input: "\x1b[A\x1b[Bq", requests: []sessionRequest{requestUp, requestDown, requestQuit}},
		{name: "pages", input: "\x1b[5~\x1b[6~q", requests: []sessionRequest{requestPageUp, requestPageDown, requestQuit}},
		{name: "unknown escape", input: "\x1b[Zq", requests: []sessionRequest{requestQuit}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(test.input))
			for index, want := range test.requests {
				got, err := readSessionRequest(reader)
				if err != nil {
					t.Fatalf("request %d: %v", index, err)
				}
				if got != want {
					t.Fatalf("request %d = %d, want %d", index, got, want)
				}
			}
		})
	}
}

func TestInteractiveSessionSerializesAndCoalescesRefreshes(t *testing.T) {
	repository := filepath.Join("/root", "a")
	coordinator := newRepositoryCoordinator()
	var output, stderr bytes.Buffer
	dashboard := newSizedTerminalDashboard(&output, func() int { return 80 }, func() int { return 24 }, asciiPalette)
	session := newInteractiveSession("/root", nil, []string{repository}, coordinator, dashboard, &stderr, true)

	var discoveries, fetches atomic.Int32
	fetchStarted := make(chan struct{}, 1)
	releaseFetch := make(chan struct{})
	session.inspect = func(_ context.Context, _, path string) repoStatus {
		return repoStatus{Path: filepath.Base(path), Branch: "main", HasRemote: true, HasUpstream: true}
	}
	session.fetch = func(ctx context.Context, _ string) error {
		fetches.Add(1)
		fetchStarted <- struct{}{}
		select {
		case <-releaseFetch:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	session.discover = func(context.Context, string, map[string]struct{}) ([]string, []error, error) {
		discoveries.Add(1)
		return []string{repository, filepath.Join("/root", "b")}, []error{fmt.Errorf("skipped subtree")}, nil
	}

	requests := make(chan sessionRequest, 8)
	localTicks := make(chan time.Time, 8)
	remoteTicks := make(chan time.Time, 8)
	result := make(chan error, 1)
	go func() {
		result <- session.runWithTicks(context.Background(), requests, localTicks, remoteTicks)
	}()

	select {
	case <-fetchStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("initial fetch did not start")
	}
	for range 3 {
		localTicks <- time.Now()
		requests <- requestRescan
		requests <- requestFetch
		remoteTicks <- time.Now()
	}
	waitForSessionInputs(t, requests, localTicks, remoteTicks)
	close(releaseFetch)
	waitForDashboardSize(t, dashboard, 2)
	requests <- requestQuit
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("session did not quit")
	}
	if discoveries.Load() != 1 {
		t.Fatalf("discoveries = %d, want 1", discoveries.Load())
	}
	if fetches.Load() != 1 {
		t.Fatalf("fetches = %d, want 1", fetches.Load())
	}
	if !strings.Contains(stderr.String(), "warning: skipped subtree") {
		t.Fatalf("scan warning = %q", stderr.String())
	}
}

func waitForSessionInputs(t *testing.T, requests chan sessionRequest, localTicks, remoteTicks chan time.Time) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(requests) == 0 && len(localTicks) == 0 && len(remoteTicks) == 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("session did not consume queued inputs")
}

func TestInteractiveSessionNoFetchPolicy(t *testing.T) {
	coordinator := newRepositoryCoordinator()
	dashboard := newTerminalDashboard(&bytes.Buffer{}, func() int { return 80 }, asciiPalette)
	session := newInteractiveSession("/root", nil, []string{"/root/a"}, coordinator, dashboard, &bytes.Buffer{}, false)
	session.inspect = func(context.Context, string, string) repoStatus {
		return repoStatus{Path: "a", Branch: "main", HasRemote: true, HasUpstream: true}
	}
	session.fetch = func(context.Context, string) error {
		t.Fatal("fetch called with fetch disabled")
		return nil
	}
	requests := make(chan sessionRequest, 2)
	remoteTicks := make(chan time.Time, 1)
	result := make(chan error, 1)
	go func() {
		result <- session.runWithTicks(context.Background(), requests, nil, remoteTicks)
	}()
	waitForDashboardSize(t, dashboard, 1)
	requests <- requestFetch
	remoteTicks <- time.Now()
	requests <- requestQuit
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

func waitForDashboardSize(t *testing.T, dashboard *terminalDashboard, size int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		dashboard.mu.Lock()
		got := len(dashboard.latest)
		dashboard.mu.Unlock()
		if got == size {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("dashboard did not reach %d repositories", size)
}
