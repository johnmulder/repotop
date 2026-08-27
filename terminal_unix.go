//go:build darwin || linux

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func watchTerminalResize(ctx context.Context, dashboard *terminalDashboard) func() {
	resizes := make(chan os.Signal, 1)
	signal.Notify(resizes, syscall.SIGWINCH)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-resizes:
				dashboard.redraw()
			}
		}
	}()
	return func() { signal.Stop(resizes) }
}
