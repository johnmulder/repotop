//go:build !darwin && !linux

package main

import (
	"context"
	"os"
)

func terminalWidth(_ *os.File) (int, bool) {
	return 0, false
}

func watchTerminalResize(_ context.Context, _ *terminalDashboard) func() {
	return func() {}
}
