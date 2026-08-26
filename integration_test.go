package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInspectRealGitConflict(t *testing.T) {
	repository := newRealRepository(t)
	commitRealFile(t, repository, "conflict.txt", "base\n", "base")
	runTestGit(t, repository, "checkout", "-b", "topic")
	commitRealFile(t, repository, "conflict.txt", "topic\n", "topic change")
	runTestGit(t, repository, "checkout", "main")
	commitRealFile(t, repository, "conflict.txt", "main\n", "main change")

	if _, err := runGit(context.Background(), repository, false, "merge", "--no-edit", "topic"); err == nil {
		t.Fatal("conflicting merge succeeded")
	}
	status := inspectRepository(context.Background(), repository, repository)
	if status.Error != "" || status.Branch != "main" || status.Conflicted != 1 {
		t.Fatalf("conflicted repository status: %+v", status)
	}
}

func TestDiscoverAndInspectRealGitWorktree(t *testing.T) {
	requireRealGit(t)
	root := t.TempDir()
	repository := filepath.Join(root, "repository")
	worktree := filepath.Join(root, "worktree")
	initRealRepository(t, repository)
	commitRealFile(t, repository, "tracked.txt", "initial\n", "initial")
	runTestGit(t, repository, "worktree", "add", "-b", "topic", worktree)

	repositories, scanErrors, err := discover(context.Background(), root)
	if err != nil || len(scanErrors) != 0 || !reflect.DeepEqual(repositories, []string{repository, worktree}) {
		t.Fatalf("worktree discovery: repositories=%v errors=%v fatal=%v", repositories, scanErrors, err)
	}
	status := inspectRepository(context.Background(), root, worktree)
	if status.Error != "" || status.Path != "worktree" || status.Branch != "topic" || status.dirty() {
		t.Fatalf("linked worktree status: %+v", status)
	}
}

func TestInspectBrokenRealGitRepository(t *testing.T) {
	repository := newRealRepository(t)
	if err := os.Remove(filepath.Join(repository, ".git", "HEAD")); err != nil {
		t.Fatal(err)
	}
	if status := inspectRepository(context.Background(), repository, repository); status.Error == "" {
		t.Fatalf("broken repository status: %+v", status)
	}
}

func requireRealGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is unavailable")
	}
}

func newRealRepository(t *testing.T) string {
	t.Helper()
	repository := t.TempDir()
	initRealRepository(t, repository)
	return repository
}

func initRealRepository(t *testing.T, repository string) {
	t.Helper()
	requireRealGit(t)
	mustMkdir(t, repository)
	runTestGit(t, repository, "init", "-b", "main")
	runTestGit(t, repository, "config", "user.name", "Repotop Test")
	runTestGit(t, repository, "config", "user.email", "repotop@example.invalid")
}

func commitRealFile(t *testing.T, repository, name, content, message string) {
	t.Helper()
	mustWrite(t, filepath.Join(repository, name), content)
	runTestGit(t, repository, "add", name)
	runTestGit(t, repository, "commit", "-m", message)
}
