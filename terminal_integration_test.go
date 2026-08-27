//go:build darwin || linux

package main

import (
	"bytes"
	"os"
	"os/exec"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func TestInteractiveTerminalRestoresState(t *testing.T) {
	stops := []struct {
		name string
		stop func(*exec.Cmd, *os.File) error
	}{
		{name: "quit key", stop: func(_ *exec.Cmd, terminal *os.File) error {
			_, err := terminal.Write([]byte("q"))
			return err
		}},
		{name: "interrupt", stop: func(command *exec.Cmd, _ *os.File) error {
			return command.Process.Signal(os.Interrupt)
		}},
		{name: "terminate", stop: func(command *exec.Cmd, _ *os.File) error {
			return command.Process.Signal(syscall.SIGTERM)
		}},
	}
	for _, test := range stops {
		t.Run(test.name, func(t *testing.T) {
			assertInteractiveTerminalRestores(t, test.stop)
		})
	}
}

func assertInteractiveTerminalRestores(t *testing.T, stop func(*exec.Cmd, *os.File) error) {
	t.Helper()
	requireRealGit(t)
	repository := newRealRepository(t)
	primary, secondary, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer primary.Close()
	defer secondary.Close()
	if err := pty.Setsize(primary, &pty.Winsize{Cols: 80, Rows: 24}); err != nil {
		t.Fatal(err)
	}
	before, err := term.GetState(int(secondary.Fd()))
	if err != nil {
		t.Fatal(err)
	}

	command := exec.Command(os.Args[0], "-test.run=^TestInteractiveTerminalHelper$")
	command.Env = append(os.Environ(), "REPOTOP_PTY_HELPER=1", "REPOTOP_PTY_ROOT="+repository)
	command.Stdin, command.Stdout, command.Stderr = secondary, secondary, secondary
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	seenDashboard := make(chan struct{}, 1)
	go func() {
		buffer := make([]byte, 4096)
		var output bytes.Buffer
		for {
			count, err := primary.Read(buffer)
			output.Write(buffer[:count])
			if bytes.Contains(output.Bytes(), []byte("REPOSITORY")) {
				select {
				case seenDashboard <- struct{}{}:
				default:
				}
				output.Reset()
			}
			if err != nil {
				return
			}
		}
	}()
	select {
	case <-seenDashboard:
	case <-time.After(5 * time.Second):
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatal("interactive dashboard did not render")
	}
	if err := stop(command, primary); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		_ = command.Process.Kill()
		<-done
		t.Fatal("interactive dashboard did not quit on q")
	}
	after, err := term.GetState(int(secondary.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatal("terminal state was not restored")
	}
}

func TestInteractiveTerminalHelper(t *testing.T) {
	if os.Getenv("REPOTOP_PTY_HELPER") != "1" {
		return
	}
	if code := run([]string{"--ascii", "--no-fetch", os.Getenv("REPOTOP_PTY_ROOT")}, os.Stdout, os.Stderr); code != 0 {
		t.Fatalf("interactive run exit = %d", code)
	}
}
