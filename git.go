package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	gitOutputLimit     = 1 << 20
	gitDiagnosticLimit = 512
)

type gitFailureKind string

const (
	gitFailureCanceled      gitFailureKind = "canceled"
	gitFailureExit          gitFailureKind = "exit"
	gitFailureMissing       gitFailureKind = "missing executable"
	gitFailureNotRepository gitFailureKind = "not a repository"
	gitFailureOutputLimit   gitFailureKind = "output limit"
	gitFailurePermission    gitFailureKind = "permission"
	gitFailureTimeout       gitFailureKind = "timeout"
)

type gitFailure struct {
	Operation string
	Kind      gitFailureKind
	Detail    string
	cause     error
}

func (failure *gitFailure) Error() string {
	if failure.Detail == "" {
		return fmt.Sprintf("git %s: %s", failure.Operation, failure.Kind)
	}
	return fmt.Sprintf("git %s: %s", failure.Operation, failure.Detail)
}

func (failure *gitFailure) Unwrap() error {
	return failure.cause
}

type cappedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (buffer *cappedBuffer) Write(data []byte) (int, error) {
	written := len(data)
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining < 0 {
		remaining = 0
	}
	if len(data) > remaining {
		buffer.truncated = true
		data = data[:remaining]
	}
	_, _ = buffer.buffer.Write(data)
	return written, nil
}

func runGit(ctx context.Context, repository string, readOnly bool, operation string, args ...string) ([]byte, error) {
	commandArgs := append([]string{"-C", repository, operation}, args...)
	command := exec.CommandContext(ctx, "git", commandArgs...)
	command.Env = append(os.Environ(), "LC_ALL=C")
	if readOnly {
		command.Env = append(command.Env, "GIT_OPTIONAL_LOCKS=0")
	}
	stdout := cappedBuffer{limit: gitOutputLimit}
	stderr := cappedBuffer{limit: gitOutputLimit}
	command.Stdout = &stdout
	command.Stderr = &stderr

	runErr := command.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, &gitFailure{Operation: operation, Kind: gitFailureTimeout, Detail: "timed out", cause: ctx.Err()}
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return nil, &gitFailure{Operation: operation, Kind: gitFailureCanceled, Detail: "canceled", cause: ctx.Err()}
	}
	if runErr != nil {
		detail := gitDiagnostic(&stdout, &stderr)
		if detail == "" {
			detail = oneLine(runErr.Error())
		}
		return nil, &gitFailure{Operation: operation, Kind: classifyGitFailure(runErr, detail), Detail: detail, cause: runErr}
	}
	if stdout.truncated || stderr.truncated {
		return nil, &gitFailure{
			Operation: operation,
			Kind:      gitFailureOutputLimit,
			Detail:    fmt.Sprintf("output exceeded %d bytes", gitOutputLimit),
		}
	}
	return bytes.Clone(stdout.buffer.Bytes()), nil
}

func classifyGitFailure(runErr error, detail string) gitFailureKind {
	switch {
	case errors.Is(runErr, exec.ErrNotFound):
		return gitFailureMissing
	case errors.Is(runErr, fs.ErrPermission):
		return gitFailurePermission
	}
	lower := strings.ToLower(detail)
	switch {
	case strings.Contains(lower, "permission denied"), strings.Contains(lower, "operation not permitted"):
		return gitFailurePermission
	case strings.Contains(lower, "not a git repository"), strings.Contains(lower, "not a git directory"):
		return gitFailureNotRepository
	default:
		return gitFailureExit
	}
}

func gitDiagnostic(stdout, stderr *cappedBuffer) string {
	selected := stderr
	if selected.buffer.Len() == 0 {
		selected = stdout
	}
	return formatGitDiagnostic(selected.buffer.String(), selected.truncated)
}

var urlUserinfo = regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.-]*://)[^/@\s]+@`)

func formatGitDiagnostic(detail string, truncated bool) string {
	detail = safeCell(oneLine(detail))
	detail = urlUserinfo.ReplaceAllString(detail, "$1***@")
	runes := []rune(detail)
	if len(runes) > gitDiagnosticLimit {
		truncated = true
	}
	if !truncated {
		return detail
	}
	suffix := []rune(" [truncated]")
	limit := gitDiagnosticLimit - len(suffix)
	if len(runes) > limit {
		runes = runes[:limit]
	}
	return string(runes) + string(suffix)
}
