package main

import (
	"io"
	"os"
	"slices"
	"sync"
	"time"

	"golang.org/x/term"
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
	height   func() int
	latest   []repositorySnapshot
	selected string
	last     string
	err      error
	now      func() time.Time
	interval time.Duration
	palette  renderPalette
	style    terminalStyle
}

func terminalDashboardFor(input io.Reader, output io.Writer, fetchInterval time.Duration, palette renderPalette) *terminalDashboard {
	inputFile, inputOK := input.(*os.File)
	outputFile, outputOK := output.(*os.File)
	if !inputOK || !outputOK || !term.IsTerminal(int(inputFile.Fd())) || !term.IsTerminal(int(outputFile.Fd())) {
		return nil
	}
	size := func() (int, int) {
		width, height, err := term.GetSize(int(outputFile.Fd()))
		if err != nil || width <= 0 {
			width = defaultRenderWidth
		}
		return width, height
	}
	dashboard := newSizedTerminalDashboard(output, func() int {
		width, _ := size()
		return width
	}, func() int {
		_, height := size()
		return height
	}, palette)
	dashboard.interval = fetchInterval
	dashboard.style = selectTerminalStyle(os.LookupEnv)
	return dashboard
}

func makeTerminalRaw(file *os.File) (func() error, error) {
	state, err := term.MakeRaw(int(file.Fd()))
	if err != nil {
		return nil, err
	}
	return func() error { return term.Restore(int(file.Fd()), state) }, nil
}

func newTerminalDashboard(output io.Writer, width func() int, palette renderPalette) *terminalDashboard {
	return newSizedTerminalDashboard(output, width, func() int { return 0 }, palette)

}

func newSizedTerminalDashboard(output io.Writer, width, height func() int, palette renderPalette) *terminalDashboard {
	return &terminalDashboard{output: output, width: width, height: height, now: time.Now, interval: defaultFetchInterval, palette: palette}
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

func (dashboard *terminalDashboard) moveSelection(delta int) {
	dashboard.mu.Lock()
	defer dashboard.mu.Unlock()
	ordered := slices.Clone(dashboard.latest)
	sortSnapshots(ordered)
	if len(ordered) == 0 {
		return
	}
	index := 0
	for candidate := range ordered {
		if ordered[candidate].Path == dashboard.selected {
			index = candidate
			break
		}
	}
	index = max(0, min(len(ordered)-1, index+delta))
	dashboard.selected = ordered[index].Path
	dashboard.drawLocked()
}

func (dashboard *terminalDashboard) pageSize() int {
	dashboard.mu.Lock()
	defer dashboard.mu.Unlock()
	if height := dashboard.height(); height > 8 {
		return height - 8
	}
	return 10
}

func (dashboard *terminalDashboard) drawLocked() {
	if dashboard.err != nil {
		return
	}
	view := renderStyledSnapshotSized(dashboard.latest, dashboard.width(), dashboard.height(), dashboard.selected, dashboard.now(), dashboard.interval, dashboard.palette, dashboard.style)
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
