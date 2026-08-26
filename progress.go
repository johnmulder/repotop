package main

import (
	"io"
	"os"
	"slices"
	"sync"
	"time"
)

const snapshotInterval = 50 * time.Millisecond

type snapshotPublisher func([]repositorySnapshot)

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
	latest   []repositorySnapshot
	selected string
	last     string
	err      error
	now      func() time.Time
	interval time.Duration
	palette  renderPalette
}

func terminalDashboardFor(output io.Writer, palette renderPalette) *terminalDashboard {
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
	}, palette)
}

func newTerminalDashboard(output io.Writer, width func() int, palette renderPalette) *terminalDashboard {
	return &terminalDashboard{output: output, width: width, now: time.Now, interval: defaultFetchInterval, palette: palette}
}

func (dashboard *terminalDashboard) publish(statuses []repositorySnapshot) {
	dashboard.mu.Lock()
	defer dashboard.mu.Unlock()
	dashboard.latest = slices.Clone(statuses)
	if !containsPath(statuses, dashboard.selected) {
		ordered := slices.Clone(statuses)
		sortSnapshots(ordered)
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
	view := renderSnapshot(dashboard.latest, dashboard.width(), dashboard.selected, dashboard.now(), dashboard.interval, dashboard.palette)
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

func containsPath(statuses []repositorySnapshot, path string) bool {
	if path == "" {
		return false
	}
	return slices.ContainsFunc(statuses, func(status repositorySnapshot) bool { return status.Path == path })
}
