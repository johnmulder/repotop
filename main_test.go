package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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

	repositories, scanErrors := discover(root)
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

	repositories, scanErrors := discover(root)
	if len(scanErrors) != 0 {
		t.Fatalf("discover errors: %v", scanErrors)
	}
	if !reflect.DeepEqual(repositories, []string{root}) {
		t.Fatalf("discover() = %v, want root only", repositories)
	}
}

func TestParsePorcelain(t *testing.T) {
	records := []string{
		"# branch.oid 0123456789",
		"# branch.head feature/test",
		"# branch.upstream origin/feature/test",
		"# branch.ab +2 -3",
		"1 M. N... 100644 100644 100644 abc def staged.txt",
		"1 .M N... 100644 100644 100644 abc def modified.txt",
		"2 R. N... 100644 100644 100644 abc def R100 renamed.txt",
		"old.txt",
		"? untracked file.txt",
	}
	status, err := parsePorcelain([]byte(strings.Join(records, "\x00") + "\x00"))
	if err != nil {
		t.Fatal(err)
	}
	if status.Branch != "feature/test" || !status.HasUpstream || status.Ahead != 2 || status.Behind != 3 || status.Modified != 3 || status.Untracked != 1 {
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
}

func TestSortStatuses(t *testing.T) {
	statuses := []repoStatus{
		{Path: "clean", HasUpstream: true},
		{Path: "missing"},
		{Path: "ahead", Ahead: 1, HasUpstream: true},
		{Path: "dirty-b", Modified: 1, HasUpstream: true},
		{Path: "dirty-a", Untracked: 1, HasUpstream: true},
		{Path: "behind", Behind: 1, HasUpstream: true},
		{Path: "error", Error: "broken"},
	}
	sortStatuses(statuses)
	var got []string
	for _, status := range statuses {
		got = append(got, status.Path)
	}
	want := []string{"error", "behind", "dirty-a", "dirty-b", "ahead", "missing", "clean"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sort = %v, want %v", got, want)
	}
}

func TestRender(t *testing.T) {
	statuses := []repoStatus{
		{Path: "dirty", Branch: "topic", Modified: 1, Untracked: 2, Ahead: 2, Behind: 1, HasUpstream: true},
		{Path: "clean", Branch: "main", HasUpstream: true},
	}
	var output bytes.Buffer
	if err := render(&output, statuses); err != nil {
		t.Fatal(err)
	}
	want := "REPOSITORY  BRANCH  WORKTREE  REMOTE\n" +
		"----------  ------  --------  ------\n" +
		"dirty       topic   M1 ?2     +2 -1\n" +
		"clean       main    clean     ok\n" +
		"\n2 repos  1 clean  1 dirty  1 ahead  1 behind  0 errors\n"
	if output.String() != want {
		t.Fatalf("render output:\n%q\nwant:\n%q", output.String(), want)
	}
}

func TestRenderSingularSummary(t *testing.T) {
	var output bytes.Buffer
	if err := render(&output, []repoStatus{{Path: ".", Error: "broken"}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "1 repo  0 clean  0 dirty  0 ahead  0 behind  1 error") {
		t.Fatalf("unexpected singular summary: %q", output.String())
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
	if !strings.Contains(stderr.String(), "usage: repotop [directory]") {
		t.Fatalf("unexpected help: %q", stderr.String())
	}

	stderr.Reset()
	if code := run([]string{"one", "two"}, &stdout, &stderr); code != 2 {
		t.Fatalf("extra argument exit = %d, stderr = %q", code, stderr.String())
	}
}

func TestInspectRepositoryWithRealGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required")
	}
	repository := t.TempDir()
	runGit(t, repository, "init", "-b", "main")
	mustWrite(t, filepath.Join(repository, "tracked.txt"), "initial\n")
	runGit(t, repository, "add", "tracked.txt")
	runGit(t, repository, "-c", "user.name=Repotop Test", "-c", "user.email=repotop@example.invalid", "commit", "-m", "initial")
	mustWrite(t, filepath.Join(repository, "tracked.txt"), "changed\n")
	mustWrite(t, filepath.Join(repository, "untracked.txt"), "new\n")

	status := inspectRepository(repository, repository)
	if status.Error != "" {
		t.Fatalf("inspect error: %s", status.Error)
	}
	if status.Path != "." || status.Branch != "main" || status.Modified != 1 || status.Untracked != 1 || status.HasUpstream {
		t.Fatalf("unexpected status: %+v", status)
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

func runGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	commandArgs := append([]string{"-C", directory}, args...)
	output, err := exec.Command("git", commandArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}
