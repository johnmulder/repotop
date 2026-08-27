//go:build !darwin && !linux

package main

import (
	"context"
)

func watchTerminalResize(_ context.Context, _ *terminalDashboard) func() {
	return func() {}
}
