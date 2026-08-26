//go:build darwin || linux

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

type windowSize struct {
	Rows    uint16
	Columns uint16
	Width   uint16
	Height  uint16
}

func terminalWidth(file *os.File) (int, bool) {
	var size windowSize
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&size)),
	)
	return int(size.Columns), errno == 0 && size.Columns > 0
}

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
