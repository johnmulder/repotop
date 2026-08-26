package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

const localStatusTimeout = 5 * time.Second

type repoStatus struct {
	Path        string
	Branch      string
	Changed     int
	Staged      int
	Modified    int
	Conflicted  int
	Untracked   int
	Ahead       int
	Behind      int
	HasUpstream bool
	Error       string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("repotop", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { fmt.Fprintln(stderr, "usage: repotop [--no-fetch] [directory]") }
	noFetch := flags.Bool("no-fetch", false, "skip remote refresh")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() > 1 {
		fmt.Fprintln(stderr, "repotop: expected at most one directory")
		flags.Usage()
		return 2
	}

	root := "."
	if flags.NArg() == 1 {
		root = flags.Arg(0)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "repotop: %v\n", err)
		return 1
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		fmt.Fprintf(stderr, "repotop: %s: %v\n", safeCell(root), err)
		return 1
	}
	if !info.IsDir() {
		fmt.Fprintf(stderr, "repotop: %s: not a directory\n", safeCell(root))
		return 1
	}

	repositories, scanErrors, err := discover(context.Background(), absRoot)
	if err != nil {
		fmt.Fprintf(stderr, "repotop: scan: %v\n", err)
		return 1
	}
	for _, scanErr := range scanErrors {
		fmt.Fprintf(stderr, "warning: %s\n", oneLine(scanErr.Error()))
	}
	if len(repositories) == 0 {
		fmt.Fprintf(stdout, "no Git repositories found beneath %s\n", safeCell(absRoot))
		return 0
	}

	coordinator := newRepositoryCoordinator()
	if err := refreshRepositories(context.Background(), coordinator, absRoot, repositories, inspectRepository); err != nil {
		fmt.Fprintf(stderr, "repotop: inspect: %v\n", err)
		return 1
	}
	if !*noFetch {
		if err := refreshRemotes(context.Background(), coordinator, absRoot, repositories, fetchRepository, inspectRepository); err != nil {
			fmt.Fprintf(stderr, "repotop: fetch: %v\n", err)
			return 1
		}
	}
	statuses := coordinator.snapshot()
	for _, status := range statuses {
		if status.Error != "" {
			fmt.Fprintf(stderr, "warning: %s: %s\n", safeCell(status.Path), oneLine(status.Error))
		}
	}
	for _, repository := range repositories {
		fetch, ok := coordinator.fetchSnapshot(repository)
		if !ok || fetch.Error == "" {
			continue
		}
		relative, err := filepath.Rel(absRoot, repository)
		if err != nil {
			relative = repository
		}
		fmt.Fprintf(stderr, "warning: %s: fetch: %s\n", safeCell(filepath.ToSlash(relative)), fetch.Error)
	}
	if err := render(stdout, statuses); err != nil {
		fmt.Fprintf(stderr, "repotop: render: %v\n", err)
		return 1
	}
	return 0
}

func discover(ctx context.Context, root string) ([]string, []error, error) {
	var repositories []string
	var scanErrors []error

	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			scanErrors = append(scanErrors, fmt.Errorf("%s: %w", path, walkErr))
			if entry != nil && entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			return nil
		}

		gitMarker := filepath.Join(path, ".git")
		markerInfo, err := os.Lstat(gitMarker)
		switch {
		case err == nil && (markerInfo.IsDir() || markerInfo.Mode().IsRegular()):
			repositories = append(repositories, path)
			return fs.SkipDir
		case err != nil && !errors.Is(err, fs.ErrNotExist):
			scanErrors = append(scanErrors, fmt.Errorf("%s: %w", gitMarker, err))
			return fs.SkipDir
		default:
			return nil
		}
	})
	sort.Strings(repositories)
	if err := ctx.Err(); err != nil {
		return repositories, scanErrors, err
	}
	if walkErr != nil {
		scanErrors = append(scanErrors, walkErr)
	}
	return repositories, scanErrors, nil
}

func inspectRepository(parent context.Context, root, repository string) repoStatus {
	relative, err := filepath.Rel(root, repository)
	if err != nil {
		relative = repository
	}
	status := repoStatus{Path: filepath.ToSlash(relative), Branch: "unknown"}

	ctx, cancel := context.WithTimeout(parent, localStatusTimeout)
	defer cancel()
	output, err := runGit(ctx, repository, true, "status", "--porcelain=v2", "--branch", "-z")
	if err != nil {
		status.Error = err.Error()
		return status
	}

	parsed, err := parsePorcelain(output)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	parsed.Path = status.Path
	return parsed
}

func parsePorcelain(output []byte) (repoStatus, error) {
	status := repoStatus{Branch: "unknown"}
	records := bytes.Split(output, []byte{0})
	for index := 0; index < len(records); index++ {
		if len(records[index]) == 0 {
			continue
		}
		record := string(records[index])
		switch {
		case strings.HasPrefix(record, "# branch.head "):
			status.Branch = strings.TrimPrefix(record, "# branch.head ")
			if status.Branch == "(detached)" {
				status.Branch = "detached"
			}
		case strings.HasPrefix(record, "# branch.upstream "):
			status.HasUpstream = true
		case strings.HasPrefix(record, "# branch.ab "):
			if _, err := fmt.Sscanf(strings.TrimPrefix(record, "# branch.ab "), "+%d -%d", &status.Ahead, &status.Behind); err != nil {
				return repoStatus{}, fmt.Errorf("parse branch distance: %w", err)
			}
		case strings.HasPrefix(record, "# "):
			// Porcelain v2 permits future headers; unknown headers are ignored.
		case record[0] == '1' || record[0] == '2':
			indexStatus, worktreeStatus, err := changedStates(record)
			if err != nil {
				return repoStatus{}, err
			}
			status.Changed++
			if indexStatus != '.' {
				status.Staged++
			}
			if worktreeStatus != '.' {
				status.Modified++
			}
			if record[0] == '2' {
				if index+1 >= len(records) || len(records[index+1]) == 0 {
					return repoStatus{}, errors.New("parse renamed path: missing original path")
				}
				index++
			}
		case record[0] == 'u':
			if _, _, err := changedStates(record); err != nil {
				return repoStatus{}, err
			}
			status.Conflicted++
		case record[0] == '?':
			status.Untracked++
		case record[0] == '!':
		default:
			return repoStatus{}, fmt.Errorf("unknown porcelain record type %q", record[0])
		}
	}
	return status, nil
}

func changedStates(record string) (byte, byte, error) {
	fields := strings.Fields(record)
	if len(fields) < 2 || len(fields[1]) != 2 {
		return 0, 0, fmt.Errorf("malformed porcelain record %q", safeCell(record))
	}
	return fields[1][0], fields[1][1], nil
}

func sortStatuses(statuses []repoStatus) {
	sort.Slice(statuses, func(i, j int) bool {
		left, right := severity(statuses[i]), severity(statuses[j])
		if left != right {
			return left < right
		}
		return statuses[i].Path < statuses[j].Path
	})
}

func severity(status repoStatus) int {
	switch {
	case status.Error != "":
		return 0
	case status.Behind > 0 && status.Ahead == 0:
		return 1
	case status.Behind > 0 && status.Ahead > 0:
		return 2
	case status.Conflicted > 0:
		return 3
	case status.dirty():
		return 4
	case status.Ahead > 0:
		return 5
	case !status.HasUpstream:
		return 6
	default:
		return 7
	}
}

func render(output io.Writer, statuses []repoStatus) error {
	_, err := io.WriteString(output, renderSnapshot(statuses, defaultRenderWidth, ""))
	return err
}

func plural(count int, singular string) string {
	if count == 1 {
		return singular
	}
	return singular + "s"
}

func (status repoStatus) dirty() bool {
	return status.Changed > 0 || status.Conflicted > 0 || status.Untracked > 0
}

func worktreeText(status repoStatus) string {
	if status.Error != "" {
		return "error"
	}
	var parts []string
	if status.Conflicted > 0 {
		parts = append(parts, fmt.Sprintf("C%d", status.Conflicted))
	}
	if status.Changed > 0 {
		parts = append(parts, fmt.Sprintf("M%d", status.Changed))
	}
	if status.Untracked > 0 {
		parts = append(parts, fmt.Sprintf("?%d", status.Untracked))
	}
	if len(parts) == 0 {
		return "clean"
	}
	return strings.Join(parts, " ")
}

func remoteText(status repoStatus) string {
	if status.Error != "" {
		return "error"
	}
	if !status.HasUpstream {
		return "no upstream"
	}
	var parts []string
	if status.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("+%d", status.Ahead))
	}
	if status.Behind > 0 {
		parts = append(parts, fmt.Sprintf("-%d", status.Behind))
	}
	if len(parts) == 0 {
		return "ok"
	}
	return strings.Join(parts, " ")
}

func safeCell(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '?'
		}
		return r
	}, value)
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
