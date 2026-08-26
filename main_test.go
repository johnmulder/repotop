package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDiscover(t *testing.T) {
	root := t.TempDir()
	repository := filepath.Join(root, "repo-a")
	worktree := filepath.Join(root, "group", "worktree")
	mustMkdir(t, filepath.Join(repository, ".git"))
	mustMkdir(t, filepath.Join(repository, "nested", ".git"))
	mustMkdir(t, worktree)
	mustWrite(t, filepath.Join(worktree, ".git"), "gitdir: elsewhere\n")
	mustMkdir(t, filepath.Join(root, "ordinary"))

	repositories, scanErrors, err := discover(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(scanErrors) != 0 {
		t.Fatalf("discover errors: %v", scanErrors)
	}
	want := []string{worktree, repository}
	if !reflect.DeepEqual(repositories, want) {
		t.Fatalf("discover() = %v, want %v", repositories, want)
	}
}

func TestDiscoverRootRepository(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".git"))
	mustMkdir(t, filepath.Join(root, "nested", ".git"))

	repositories, scanErrors, err := discover(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(scanErrors) != 0 {
		t.Fatalf("discover errors: %v", scanErrors)
	}
	if !reflect.DeepEqual(repositories, []string{root}) {
		t.Fatalf("discover() = %v, want root only", repositories)
	}
}

func TestDiscoverDoesNotFollowSymlinks(t *testing.T) {
	root := t.TempDir()
	repository := filepath.Join(root, "repo")
	mustMkdir(t, filepath.Join(repository, ".git"))

	external := t.TempDir()
	mustMkdir(t, filepath.Join(external, ".git"))
	if err := os.Symlink(external, filepath.Join(root, "linked-repo")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(root, filepath.Join(root, "loop")); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(root, "fake")
	mustMkdir(t, fake)
	if err := os.Symlink(filepath.Join(external, ".git"), filepath.Join(fake, ".git")); err != nil {
		t.Fatal(err)
	}

	repositories, scanErrors, err := discover(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(scanErrors) != 0 {
		t.Fatalf("discover errors: %v", scanErrors)
	}
	if !reflect.DeepEqual(repositories, []string{repository}) {
		t.Fatalf("discover() = %v, want only %s", repositories, repository)
	}

	repositories, scanErrors, err = discover(context.Background(), filepath.Join(root, "linked-repo"))
	if err != nil || len(scanErrors) != 0 || len(repositories) != 0 {
		t.Fatalf("symlink root: repositories=%v errors=%v fatal=%v", repositories, scanErrors, err)
	}
}

func TestDiscoverCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repositories, scanErrors, err := discover(ctx, t.TempDir())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("discover error = %v, want context canceled", err)
	}
	if len(repositories) != 0 || len(scanErrors) != 0 {
		t.Fatalf("canceled discover returned repositories=%v errors=%v", repositories, scanErrors)
	}
}

func TestDiscoverContinuesAfterInaccessibleDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can traverse permissionless directories")
	}
	root := t.TempDir()
	blocked := filepath.Join(root, "blocked")
	repository := filepath.Join(root, "repo")
	mustMkdir(t, blocked)
	mustMkdir(t, filepath.Join(repository, ".git"))
	if err := os.Chmod(blocked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })

	repositories, scanErrors, err := discover(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(repositories, []string{repository}) {
		t.Fatalf("discover() = %v, want %s", repositories, repository)
	}
	if len(scanErrors) == 0 {
		t.Fatal("expected inaccessible-directory warning")
	}
}

func TestParsePorcelain(t *testing.T) {
	records := []string{
		"# branch.oid 0123456789",
		"# branch.head feature/test",
		"# branch.upstream origin/feature/test",
		"# branch.ab +2 -3",
		"1 M. N... 100644 100644 100644 abc def staged.txt",
		"1 .M N... 100644 100644 100644 abc def strange\tline\ncaf\u00e9.txt",
		"1 MM N... 100644 100644 100644 abc def both.txt",
		"2 R. N... 100644 100644 100644 abc def R100 renamed.txt",
		"# branch.ab +99 -99",
		"2 C. N... 100644 100644 100644 abc def C100 copied.txt",
		"copy source.txt",
		"u UU N... 100644 100644 100644 100644 abc def ghi conflicted.txt",
		"? untracked\nfile.txt",
		"! ignored.txt",
	}
	status, err := parsePorcelain([]byte(strings.Join(records, "\x00") + "\x00"))
	if err != nil {
		t.Fatal(err)
	}
	if status.Branch != "feature/test" || !status.HasUpstream || status.Ahead != 2 || status.Behind != 3 ||
		status.Changed != 5 || status.Staged != 4 || status.Modified != 2 || status.Conflicted != 1 || status.Untracked != 1 {
		t.Fatalf("unexpected status: %+v", status)
	}
}

func TestParseDetachedHeadAndMalformedDistance(t *testing.T) {
	status, err := parsePorcelain([]byte("# branch.head (detached)\x00"))
	if err != nil {
		t.Fatal(err)
	}
	if status.Branch != "detached" {
		t.Fatalf("branch = %q, want detached", status.Branch)
	}
	if _, err := parsePorcelain([]byte("# branch.ab invalid\x00")); err == nil {
		t.Fatal("expected malformed branch distance error")
	}
	if _, err := parsePorcelain([]byte("1 malformed\x00")); err == nil {
		t.Fatal("expected malformed changed record error")
	}
	if _, err := parsePorcelain([]byte("2 R. N... new.txt\x00")); err == nil {
		t.Fatal("expected missing rename source error")
	}
}

func TestSortStatuses(t *testing.T) {
	statuses := []repoStatus{
		{Path: "clean", HasUpstream: true},
		{Path: "missing"},
		{Path: "ahead", Ahead: 1, HasUpstream: true},
		{Path: "diverged", Ahead: 1, Behind: 1, HasUpstream: true},
		{Path: "dirty-b", Changed: 1, HasUpstream: true},
		{Path: "dirty-a", Untracked: 1, HasUpstream: true},
		{Path: "conflict", Conflicted: 1, HasUpstream: true},
		{Path: "behind", Behind: 1, HasUpstream: true},
		{Path: "error", Error: "broken"},
	}
	sortStatuses(statuses)
	var got []string
	for _, status := range statuses {
		got = append(got, status.Path)
	}
	want := []string{"error", "behind", "diverged", "conflict", "dirty-a", "dirty-b", "ahead", "missing", "clean"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sort = %v, want %v", got, want)
	}
}

func TestRender(t *testing.T) {
	statuses := []repoStatus{
		{Path: "dirty", Branch: "topic", Changed: 1, Staged: 1, Modified: 1, Conflicted: 1, Untracked: 2, Ahead: 2, Behind: 1, HasUpstream: true},
		{Path: "clean", Branch: "main", HasUpstream: true},
	}
	var output bytes.Buffer
	if err := render(&output, snapshotsOf(statuses...)); err != nil {
		t.Fatal(err)
	}
	want := "REPOSITORY                        BRANCH            WORKTREE      REMOTE\n" +
		"--------------------------------  ----------------  ------------  --------------\n" +
		"dirty                             topic             C1 M1 ?2      +2 -1 cached\n" +
		"clean                             main              clean         cached\n" +
		"\n2 repos (1 clean, 1 dirty, 1 ahead, 1 behind, 0 errors)\n"
	if output.String() != want {
		t.Fatalf("render output:\n%q\nwant:\n%q", output.String(), want)
	}
}

func TestRenderCompactAndNarrowGolden(t *testing.T) {
	statuses := []repoStatus{
		{Path: "dirty", Branch: "topic", Changed: 1, Conflicted: 1, Untracked: 2, Ahead: 2, Behind: 1, HasUpstream: true},
		{Path: "clean", Branch: "main", HasUpstream: true},
	}
	tests := []struct {
		name  string
		width int
		want  string
	}{
		{
			name:  "compact",
			width: 50,
			want: "REPOSITORY            WORKTREE      REMOTE\n" +
				"--------------------  ------------  --------------\n" +
				"dirty                 C1 M1 ?2      +2 -1 cached\n" +
				"clean                 clean         cached\n" +
				"\n2 repos (1 clean, 1 dirty, 0 errors)\n",
		},
		{
			name:  "narrow",
			width: 30,
			want: "REPOS...  STATE\n" +
				"--------  --------------------\n" +
				"dirty     C1 M1 ?2 +2 -1 ca...\n" +
				"clean     clean cached\n" +
				"\n2 repos (1 dirty, 0 errors)\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := renderSnapshot(snapshotsOf(statuses...), test.width, "", time.Unix(100, 0), defaultFetchInterval); got != test.want {
				t.Fatalf("render output:\n%q\nwant:\n%q", got, test.want)
			}
		})
	}
}

func TestRenderTruncatesWithinWidthAndKeepsSelectionIdentity(t *testing.T) {
	statuses := []repoStatus{
		{Path: "alpha/beta/repository-name", Branch: "feature/an-extremely-long-branch", HasUpstream: true},
		{Path: "target", Branch: "main", Changed: 1, HasUpstream: true},
	}
	if got := middleTruncate(statuses[0].Path, 14); got != "alp...ory-name" {
		t.Fatalf("middle truncation = %q", got)
	}
	for _, width := range []int{80, 50, 30, 10} {
		output := renderSnapshot(snapshotsOf(statuses...), width, "target", time.Unix(100, 0), defaultFetchInterval)
		for _, line := range strings.Split(strings.TrimSuffix(output, "\n"), "\n") {
			if len([]rune(line)) > width {
				t.Fatalf("width %d line has %d characters: %q", width, len([]rune(line)), line)
			}
		}
		if !strings.Contains(output, "> ") {
			t.Fatalf("width %d lost selection marker:\n%s", width, output)
		}
		if width >= 30 && !strings.Contains(output, "> target") {
			t.Fatalf("width %d lost selected identity:\n%s", width, output)
		}
	}
	if statuses[0].Path != "alpha/beta/repository-name" {
		t.Fatalf("renderer mutated input order: %+v", statuses)
	}
}

func TestRenderSingularSummary(t *testing.T) {
	var output bytes.Buffer
	if err := render(&output, snapshotsOf(repoStatus{Path: ".", Error: "broken"})); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "1 repo (0 clean, 0 dirty, 0 ahead, 0 behind, 1 error)") {
		t.Fatalf("unexpected singular summary: %q", output.String())
	}
}

func TestFreshnessStatesPreserveKnownDistance(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	interval := 3 * time.Minute
	comparable := repoStatus{Path: "repo", HasRemote: true, HasUpstream: true, Ahead: 2, Behind: 1}
	tests := []struct {
		name     string
		snapshot repositorySnapshot
		want     string
	}{
		{name: "local error", snapshot: repositorySnapshot{repoStatus: repoStatus{Error: "broken"}}, want: "error"},
		{name: "no remote", snapshot: repositorySnapshot{repoStatus: repoStatus{}}, want: "no remote"},
		{name: "no upstream", snapshot: repositorySnapshot{repoStatus: repoStatus{HasRemote: true}}, want: "no upstream"},
		{name: "fetching", snapshot: repositorySnapshot{repoStatus: comparable, Fetching: true}, want: "+2 -1 fetching"},
		{name: "current at boundary", snapshot: repositorySnapshot{repoStatus: comparable, Fetch: fetchStatus{LastSuccess: now.Add(-interval)}}, want: "+2 -1 current"},
		{name: "stale", snapshot: repositorySnapshot{repoStatus: comparable, Fetch: fetchStatus{LastSuccess: now.Add(-interval - time.Nanosecond)}}, want: "+2 -1 stale"},
		{name: "failed without prior data", snapshot: repositorySnapshot{repoStatus: comparable, Fetch: fetchStatus{Error: "offline"}}, want: "+2 -1 failed"},
		{name: "failed with prior data", snapshot: repositorySnapshot{repoStatus: comparable, Fetch: fetchStatus{LastSuccess: now.Add(-time.Minute), Error: "offline"}}, want: "+2 -1 stale"},
		{name: "cached", snapshot: repositorySnapshot{repoStatus: comparable}, want: "+2 -1 cached"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := remoteText(test.snapshot, now, interval); got != test.want {
				t.Fatalf("remote text = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSelectedDetailsShowSanitizedErrorsAndTimestamps(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	snapshot := repositorySnapshot{
		repoStatus: repoStatus{
			Path:        "repo",
			Branch:      "topic\x1b",
			Ahead:       2,
			HasRemote:   true,
			HasUpstream: true,
		},
		Fetch: fetchStatus{
			LastAttempt: now.Add(-2 * time.Minute),
			LastSuccess: now.Add(-10 * time.Minute),
			Duration:    2 * time.Second,
			Error:       "credential\x1b[31m\nrejected",
		},
	}
	output := renderSnapshot([]repositorySnapshot{snapshot}, 80, "repo", now, defaultFetchInterval)
	for _, want := range []string{
		"remote: +2 stale; freshness interval 3m0s",
		"fetch: failed; attempted 2026-08-25T11:58:00Z; duration 2s",
		"last success: 2026-08-25T11:50:00Z",
		"fetch error: credential?[31m rejected",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("selected detail missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "\x1b") {
		t.Fatalf("selected detail contains a control character: %q", output)
	}
	for _, line := range strings.Split(strings.TrimSuffix(output, "\n"), "\n") {
		if len([]rune(line)) > 80 {
			t.Fatalf("detail line exceeded width: %q", line)
		}
	}
}

func TestRunEmptyAndInvalidDirectories(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := run([]string{root}, &stdout, &stderr); code != 0 {
		t.Fatalf("empty directory exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no Git repositories found beneath") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{filepath.Join(root, "missing")}, &stdout, &stderr); code != 1 {
		t.Fatalf("missing directory exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "repotop:") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunHelpAndExtraArgument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "usage: repotop [--no-fetch] [directory]") {
		t.Fatalf("unexpected help: %q", stderr.String())
	}

	stderr.Reset()
	if code := run([]string{"one", "two"}, &stdout, &stderr); code != 2 {
		t.Fatalf("extra argument exit = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunNoFetch(t *testing.T) {
	installFakeGit(t)
	t.Setenv("FAKE_GIT_MODE", "run")
	marker := filepath.Join(t.TempDir(), "fetch")
	t.Setenv("FAKE_GIT_MARKER", marker)
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".git"))

	var stdout, stderr bytes.Buffer
	if code := run([]string{"--no-fetch", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("no-fetch exit = %d, stderr = %q", code, stderr.String())
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("fetch marker with --no-fetch: %v", err)
	}
	if strings.Contains(stdout.String(), "\x1b[") || strings.Count(stdout.String(), "REPOSITORY") != 1 {
		t.Fatalf("non-terminal no-fetch output was progressive: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{root}, &stdout, &stderr); code != 0 {
		t.Fatalf("default fetch exit = %d, stderr = %q", code, stderr.String())
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("default fetch marker: %v", err)
	}
	if strings.Contains(stdout.String(), "\x1b[") || strings.Count(stdout.String(), "REPOSITORY") != 1 {
		t.Fatalf("non-terminal fetch output was progressive: %q", stdout.String())
	}
}

func TestInspectRepositoryWithRealGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required")
	}
	repository := t.TempDir()
	runTestGit(t, repository, "init", "-b", "main")
	runTestGit(t, repository, "config", "user.name", "Repotop Test")
	runTestGit(t, repository, "config", "user.email", "repotop@example.invalid")
	mustWrite(t, filepath.Join(repository, "tracked.txt"), "initial\n")
	runTestGit(t, repository, "add", "tracked.txt")
	runTestGit(t, repository, "commit", "-m", "initial")
	mustWrite(t, filepath.Join(repository, "tracked.txt"), "staged\n")
	runTestGit(t, repository, "add", "tracked.txt")
	mustWrite(t, filepath.Join(repository, "tracked.txt"), "modified again\n")
	mustWrite(t, filepath.Join(repository, "untracked.txt"), "new\n")

	status := inspectRepository(context.Background(), repository, repository)
	if status.Error != "" {
		t.Fatalf("inspect error: %s", status.Error)
	}
	if status.Path != "." || status.Branch != "main" || status.Changed != 1 || status.Staged != 1 || status.Modified != 1 ||
		status.Conflicted != 0 || status.Untracked != 1 || status.HasRemote || status.HasUpstream {
		t.Fatalf("unexpected status: %+v", status)
	}

	runTestGit(t, repository, "remote", "add", "origin", "https://example.invalid/repo.git")
	status = inspectRepository(context.Background(), repository, repository)
	if status.Error != "" || !status.HasRemote || status.HasUpstream {
		t.Fatalf("remote without upstream was not distinguished: %+v", status)
	}
}

func TestInspectDiscoveredRepositoryThatDisappears(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required")
	}
	root := t.TempDir()
	repository := filepath.Join(root, "repo")
	mustMkdir(t, repository)
	runTestGit(t, repository, "init", "-b", "main")

	repositories, scanErrors, err := discover(context.Background(), root)
	if err != nil || len(scanErrors) != 0 || !reflect.DeepEqual(repositories, []string{repository}) {
		t.Fatalf("discover: repositories=%v errors=%v fatal=%v", repositories, scanErrors, err)
	}
	if err := os.RemoveAll(repository); err != nil {
		t.Fatal(err)
	}
	if status := inspectRepository(context.Background(), root, repository); status.Error == "" {
		t.Fatalf("inspect missing repository = %+v, want repository-local error", status)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runTestGit(t *testing.T, directory, operation string, args ...string) {
	t.Helper()
	output, err := runGit(context.Background(), directory, false, operation, args...)
	if err != nil {
		t.Fatalf("git %s %s: %v\n%s", operation, strings.Join(args, " "), err, output)
	}
}

func snapshotsOf(statuses ...repoStatus) []repositorySnapshot {
	snapshots := make([]repositorySnapshot, len(statuses))
	for index, status := range statuses {
		snapshots[index] = repositorySnapshot{repoStatus: status}
	}
	return snapshots
}
