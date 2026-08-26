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

	repositories, scanErrors, err := discover(context.Background(), root, nil)
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

func TestInspectAndRefreshRealGitRemoteStates(t *testing.T) {
	root, remote, publisher := newBareRemote(t)
	aheadRepository := cloneRealRepository(t, root, remote, "ahead")
	behindRepository := cloneRealRepository(t, root, remote, "behind")

	clean := inspectRepository(context.Background(), aheadRepository, aheadRepository)
	if clean.Error != "" || !clean.HasRemote || !clean.HasUpstream || clean.Ahead != 0 || clean.Behind != 0 {
		t.Fatalf("clean tracked remote status: %+v", clean)
	}
	commitRealFile(t, aheadRepository, "ahead.txt", "ahead\n", "ahead commit")
	ahead := inspectRepository(context.Background(), aheadRepository, aheadRepository)
	if ahead.Error != "" || ahead.Ahead != 1 || ahead.Behind != 0 {
		t.Fatalf("ahead status: %+v", ahead)
	}

	commitRealFile(t, publisher, "behind.txt", "behind\n", "remote commit")
	runTestGit(t, publisher, "push", "origin", "main")
	coordinator := newRepositoryCoordinator()
	generation := coordinator.beginScan()
	coordinator.apply(repositoryUpdate{
		Generation: generation,
		Repository: behindRepository,
		Status:     inspectRepository(context.Background(), behindRepository, behindRepository),
	})
	coordinator.completeScan(generation, []string{behindRepository})
	if err := refreshRemotes(context.Background(), coordinator, behindRepository, []string{behindRepository}, fetchRepository, inspectRepository, nil); err != nil {
		t.Fatal(err)
	}
	behind := coordinator.snapshot()[0]
	if behind.Error != "" || behind.Ahead != 0 || behind.Behind != 1 || behind.Fetch.Error != "" || behind.Fetch.LastSuccess.IsZero() {
		t.Fatalf("refreshed behind status: %+v", behind)
	}

	if err := fetchRepository(context.Background(), aheadRepository); err != nil {
		t.Fatal(err)
	}
	diverged := inspectRepository(context.Background(), aheadRepository, aheadRepository)
	if diverged.Error != "" || diverged.Ahead != 1 || diverged.Behind != 1 {
		t.Fatalf("diverged status: %+v", diverged)
	}
}

func TestRealGitFetchPrunesDeletedRemoteBranch(t *testing.T) {
	root, remote, publisher := newBareRemote(t)
	repository := cloneRealRepository(t, root, remote, "consumer")
	runTestGit(t, publisher, "checkout", "-b", "obsolete")
	commitRealFile(t, publisher, "obsolete.txt", "obsolete\n", "obsolete branch")
	runTestGit(t, publisher, "push", "-u", "origin", "obsolete")

	if err := fetchRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, repository, "show-ref", "--verify", "refs/remotes/origin/obsolete")
	runTestGit(t, publisher, "checkout", "main")
	runTestGit(t, publisher, "push", "origin", "--delete", "obsolete")
	if err := fetchRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	if _, err := runGit(context.Background(), repository, true, "show-ref", "--verify", "refs/remotes/origin/obsolete"); err == nil {
		t.Fatal("deleted remote branch was not pruned")
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
	configureRealRepository(t, repository)
}

func commitRealFile(t *testing.T, repository, name, content, message string) {
	t.Helper()
	mustWrite(t, filepath.Join(repository, name), content)
	runTestGit(t, repository, "add", name)
	runTestGit(t, repository, "commit", "-m", message)
}

func configureRealRepository(t *testing.T, repository string) {
	t.Helper()
	runTestGit(t, repository, "config", "user.name", "Repotop Test")
	runTestGit(t, repository, "config", "user.email", "repotop@example.invalid")
	runTestGit(t, repository, "config", "commit.gpgSign", "false")
	runTestGit(t, repository, "config", "core.autocrlf", "false")
	runTestGit(t, repository, "config", "core.hooksPath", t.TempDir())
	runTestGit(t, repository, "config", "protocol.file.allow", "always")
}

func newBareRemote(t *testing.T) (root, remote, publisher string) {
	t.Helper()
	requireRealGit(t)
	root = t.TempDir()
	remote = filepath.Join(root, "remote.git")
	runTestGit(t, root, "init", "--bare", "-b", "main", remote)
	publisher = filepath.Join(root, "publisher")
	initRealRepository(t, publisher)
	commitRealFile(t, publisher, "initial.txt", "initial\n", "initial")
	runTestGit(t, publisher, "remote", "add", "origin", remote)
	runTestGit(t, publisher, "push", "-u", "origin", "main")
	return root, remote, publisher
}

func cloneRealRepository(t *testing.T, root, remote, name string) string {
	t.Helper()
	repository := filepath.Join(root, name)
	runTestGit(t, root, "-c", "protocol.file.allow=always", "clone", "--branch", "main", remote, repository)
	configureRealRepository(t, repository)
	return repository
}
