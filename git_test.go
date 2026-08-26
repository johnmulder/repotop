package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCappedBuffer(t *testing.T) {
	buffer := cappedBuffer{limit: 4}
	if written, err := buffer.Write([]byte("abcdef")); err != nil || written != 6 {
		t.Fatalf("Write() = %d, %v", written, err)
	}
	if got := buffer.buffer.String(); got != "abcd" || !buffer.truncated {
		t.Fatalf("buffer = %q, truncated = %v", got, buffer.truncated)
	}
	if written, err := buffer.Write([]byte("gh")); err != nil || written != 2 || buffer.buffer.String() != "abcd" {
		t.Fatalf("second Write() = %d, %v, buffer %q", written, err, buffer.buffer.String())
	}
}

func TestFormatGitDiagnostic(t *testing.T) {
	detail := "fatal:\nhttps://user:secret@example.com/repo " + strings.Repeat("x", gitDiagnosticLimit)
	got := formatGitDiagnostic(detail, false)
	if strings.Contains(got, "user:secret") || strings.Contains(got, "\n") {
		t.Fatalf("diagnostic was not sanitized: %q", got)
	}
	if !strings.Contains(got, "https://***@example.com/repo") || !strings.HasSuffix(got, " [truncated]") {
		t.Fatalf("diagnostic was not redacted and truncated: %q", got)
	}
	if len([]rune(got)) > gitDiagnosticLimit {
		t.Fatalf("diagnostic length = %d, want <= %d", len([]rune(got)), gitDiagnosticLimit)
	}
}

func TestRunGitFailureKinds(t *testing.T) {
	installFakeGit(t)
	repository := t.TempDir()

	for _, test := range []struct {
		mode string
		kind gitFailureKind
	}{
		{mode: "permission", kind: gitFailurePermission},
		{mode: "nonrepo", kind: gitFailureNotRepository},
		{mode: "ordinary", kind: gitFailureExit},
		{mode: "output", kind: gitFailureOutputLimit},
	} {
		t.Run(test.mode, func(t *testing.T) {
			t.Setenv("FAKE_GIT_MODE", test.mode)
			_, err := runGit(context.Background(), repository, true, "status")
			failure := requireGitFailure(t, err, test.kind)
			if failure.Operation != "status" {
				t.Fatalf("operation = %q", failure.Operation)
			}
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := runGit(ctx, repository, true, "status")
	requireGitFailure(t, err, gitFailureCanceled)

	ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, err = runGit(ctx, repository, true, "status")
	requireGitFailure(t, err, gitFailureTimeout)
}

func TestRunGitMissingExecutable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := runGit(context.Background(), t.TempDir(), true, "status")
	requireGitFailure(t, err, gitFailureMissing)
}

func TestRunGitReadOnlyEnvironment(t *testing.T) {
	installFakeGit(t)
	t.Setenv("FAKE_GIT_MODE", "environment")
	t.Setenv("GIT_OPTIONAL_LOCKS", "parent")

	output, err := runGit(context.Background(), t.TempDir(), true, "status")
	if err != nil || string(output) != "0" {
		t.Fatalf("read-only output = %q, error = %v", output, err)
	}
	output, err = runGit(context.Background(), t.TempDir(), false, "status")
	if err != nil || string(output) != "parent" {
		t.Fatalf("mutating output = %q, error = %v", output, err)
	}
}

func TestFetchRepositoryIsNoninteractiveAndPrunes(t *testing.T) {
	installFakeGit(t)
	t.Setenv("FAKE_GIT_MODE", "fetch")
	marker := filepath.Join(t.TempDir(), "fetch")
	t.Setenv("FAKE_GIT_MARKER", marker)
	t.Setenv("GIT_TERMINAL_PROMPT", "1")

	if err := fetchRepository(context.Background(), t.TempDir()); err != nil {
		t.Fatal(err)
	}
	output, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(output); got != "0|fetch|--prune" {
		t.Fatalf("fetch invocation = %q", got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	requireGitFailure(t, fetchRepository(ctx, t.TempDir()), gitFailureCanceled)

	t.Setenv("FAKE_GIT_MODE", "timeout")
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	requireGitFailure(t, fetchRepository(ctx, t.TempDir()), gitFailureTimeout)
}

func installFakeGit(t *testing.T) {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, "git")
	script := `#!/bin/sh
case "$FAKE_GIT_MODE" in
  permission) echo "fatal: Permission denied" >&2; exit 1 ;;
  nonrepo) echo "fatal: not a git repository" >&2; exit 1 ;;
  ordinary) echo "fatal: ordinary failure" >&2; exit 1 ;;
  output) exec /bin/dd if=/dev/zero bs=1048577 count=1 2>/dev/null ;;
  environment) printf "%s" "$GIT_OPTIONAL_LOCKS" ;;
  fetch) printf "%s|%s|%s" "$GIT_TERMINAL_PROMPT" "$3" "$4" > "$FAKE_GIT_MARKER" ;;
  timeout) exec /bin/sleep 5 ;;
  run|run-behind|run-dirty)
    if [ "$3" = fetch ]; then
      : > "$FAKE_GIT_MARKER"
    elif [ "$3" = remote ]; then
      printf 'origin\n'
    elif [ "$FAKE_GIT_MODE" = run-behind ]; then
      printf '# branch.head main\0# branch.upstream origin/main\0# branch.ab +0 -1\0'
    elif [ "$FAKE_GIT_MODE" = run-dirty ]; then
      printf '# branch.head main\0# branch.upstream origin/main\0# branch.ab +0 -0\0001 .M N... 100644 100644 100644 abc def tracked.txt\0'
    else
      printf '# branch.head main\0# branch.upstream origin/main\0# branch.ab +0 -0\0'
    fi
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
}

func requireGitFailure(t *testing.T, err error, kind gitFailureKind) *gitFailure {
	t.Helper()
	var failure *gitFailure
	if !errors.As(err, &failure) {
		t.Fatalf("error = %T %v, want *gitFailure", err, err)
	}
	if failure.Kind != kind {
		t.Fatalf("failure kind = %q, want %q (%v)", failure.Kind, kind, failure)
	}
	return failure
}
