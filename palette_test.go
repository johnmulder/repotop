package main

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPaletteSelectionUsesEffectiveLocaleAndOverride(t *testing.T) {
	tests := []struct {
		name        string
		forceASCII  bool
		environment map[string]string
		want        renderPalette
	}{
		{name: "explicit ASCII", forceASCII: true, environment: map[string]string{"LC_ALL": "en_US.UTF-8"}, want: asciiPalette},
		{name: "LC_ALL wins", environment: map[string]string{"LC_ALL": "C", "LANG": "en_US.UTF-8"}, want: asciiPalette},
		{name: "LC_CTYPE UTF8", environment: map[string]string{"LC_CTYPE": "C.UTF8", "LANG": "C"}, want: unicodePalette},
		{name: "LANG UTF-8", environment: map[string]string{"LANG": "en_US.UTF-8"}, want: unicodePalette},
		{name: "unadvertised", environment: map[string]string{}, want: asciiPalette},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			getenv := func(name string) string { return test.environment[name] }
			if got := selectPalette(test.forceASCII, getenv); got != test.want {
				t.Fatalf("palette = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestTerminalStyleSelection(t *testing.T) {
	tests := []struct {
		name        string
		environment map[string]string
		want        terminalStyle
	}{
		{name: "terminal default", environment: map[string]string{}, want: ansiTerminalStyle},
		{name: "NO_COLOR present", environment: map[string]string{"NO_COLOR": ""}, want: terminalStyle{}},
		{name: "dumb terminal", environment: map[string]string{"TERM": " DUMB "}, want: terminalStyle{}},
		{name: "usable terminal", environment: map[string]string{"TERM": "xterm-256color"}, want: ansiTerminalStyle},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lookupEnv := func(name string) (string, bool) {
				value, ok := test.environment[name]
				return value, ok
			}
			if got := selectTerminalStyle(lookupEnv); got != test.want {
				t.Fatalf("style = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestTerminalStyleIsSemanticAndLayoutNeutral(t *testing.T) {
	now := time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC)
	comparable := repoStatus{Path: "repo", Branch: "main", HasRemote: true, HasUpstream: true}
	tests := []struct {
		name     string
		snapshot repositorySnapshot
		code     string
	}{
		{name: "local error", snapshot: repositorySnapshot{repoStatus: repoStatus{Error: "broken"}}, code: ansiTerminalStyle.danger},
		{name: "fetch error", snapshot: repositorySnapshot{repoStatus: comparable, Fetch: fetchStatus{Error: "offline"}}, code: ansiTerminalStyle.danger},
		{name: "behind wins over dirty", snapshot: repositorySnapshot{repoStatus: repoStatus{Behind: 1, Changed: 1, HasRemote: true, HasUpstream: true}}, code: ansiTerminalStyle.danger},
		{name: "dirty wins over missing", snapshot: repositorySnapshot{repoStatus: repoStatus{Changed: 1}}, code: ansiTerminalStyle.attention},
		{name: "ahead", snapshot: repositorySnapshot{repoStatus: repoStatus{Ahead: 1, HasRemote: true, HasUpstream: true}}, code: ansiTerminalStyle.attention},
		{name: "fetching", snapshot: repositorySnapshot{repoStatus: comparable, Fetching: true}, code: ansiTerminalStyle.attention},
		{name: "stale", snapshot: repositorySnapshot{repoStatus: comparable, Fetch: fetchStatus{LastSuccess: now.Add(-4 * time.Minute)}}, code: ansiTerminalStyle.attention},
		{name: "missing remote", snapshot: repositorySnapshot{repoStatus: repoStatus{}}, code: ansiTerminalStyle.muted},
		{name: "missing upstream", snapshot: repositorySnapshot{repoStatus: repoStatus{HasRemote: true}}, code: ansiTerminalStyle.muted},
		{name: "current", snapshot: repositorySnapshot{repoStatus: comparable, Fetch: fetchStatus{LastSuccess: now}}, code: ansiTerminalStyle.clean},
		{name: "cached", snapshot: repositorySnapshot{repoStatus: comparable}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			want := "row"
			if test.code != "" {
				want = test.code + want + ansiTerminalStyle.reset
			}
			if got := ansiTerminalStyle.apply(test.snapshot, now, defaultFetchInterval, "row"); got != want {
				t.Fatalf("styled row = %q, want %q", got, want)
			}
		})
	}

	status := repositorySnapshot{repoStatus: repoStatus{
		Path:        "repo\x1b[31m",
		Branch:      "main",
		Ahead:       1,
		HasRemote:   true,
		HasUpstream: true,
	}}
	plain := renderSnapshotSized([]repositorySnapshot{status}, 80, 0, "", now, defaultFetchInterval, asciiPalette)
	colored := renderStyledSnapshotSized([]repositorySnapshot{status}, 80, 0, "", now, defaultFetchInterval, asciiPalette, ansiTerminalStyle)
	stripped := strings.NewReplacer(
		ansiTerminalStyle.clean, "",
		ansiTerminalStyle.attention, "",
		ansiTerminalStyle.danger, "",
		ansiTerminalStyle.muted, "",
		ansiTerminalStyle.reset, "",
	).Replace(colored)
	if stripped != plain {
		t.Fatalf("stripped colored output changed layout:\n%q\nwant:\n%q", stripped, plain)
	}
	if strings.Count(colored, "\x1b[") != 2 || !strings.Contains(colored, ansiTerminalStyle.attention+"repo?[31m") {
		t.Fatalf("unexpected ANSI or unsanitized path: %q", colored)
	}
}

func TestUnicodePaletteUsesReviewedGlyphsWithoutLosingText(t *testing.T) {
	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	statuses := []repositorySnapshot{
		{
			repoStatus: repoStatus{Path: "diverged", Branch: "main", Ahead: 2, Behind: 1, HasRemote: true, HasUpstream: true},
			Fetch:      fetchStatus{LastSuccess: now},
		},
		{
			repoStatus: repoStatus{Path: "current", Branch: "main", HasRemote: true, HasUpstream: true},
			Fetch:      fetchStatus{LastSuccess: now},
		},
		{repoStatus: repoStatus{Path: "broken", Error: "unavailable"}},
	}
	output := renderSnapshot(statuses, 80, "diverged", now, defaultFetchInterval, unicodePalette)
	for _, want := range []string{
		strings.Repeat(unicodePalette.rule, 8),
		unicodePalette.selection + "diverged",
		unicodePalette.ahead + "2 " + unicodePalette.behind + "1 current",
		unicodePalette.current + "current",
		unicodePalette.missing,
		"3 repos",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("Unicode rendering missing %q:\n%s", want, output)
		}
	}
	for _, line := range strings.Split(strings.TrimSuffix(output, "\n"), "\n") {
		if displayWidth(line) > 80 {
			t.Fatalf("Unicode line uses %d cells: %q", displayWidth(line), line)
		}
	}
}

func TestTerminalCellWidthTruncationPaddingAndWrapping(t *testing.T) {
	joinedEmoji := "\U0001f469\u200d\U0001f4bb"
	modifiedEmoji := "\U0001f44d\U0001f3fd"
	tests := []struct {
		value string
		want  int
	}{
		{value: "ascii", want: 5},
		{value: "e\u0301", want: 1},
		{value: "\u4ed3\u5e93", want: 4},
		{value: "\U0001f680", want: 2},
		{value: "\u00b7", want: 1},
		{value: modifiedEmoji, want: 2},
		{value: joinedEmoji, want: 2},
	}
	for _, test := range tests {
		if got := displayWidth(test.value); got != test.want {
			t.Fatalf("displayWidth(%q) = %d, want %d", test.value, got, test.want)
		}
	}

	if got := truncateEnd("ab\u4ed3\u5e93cd", 7); got != "ab\u4ed3..." || displayWidth(got) != 7 {
		t.Fatalf("end truncation = %q (%d cells)", got, displayWidth(got))
	}
	if got := middleTruncate("ab\u4ed3\u5e93cd", 7); got != "a...cd" || displayWidth(got) > 7 {
		t.Fatalf("middle truncation = %q (%d cells)", got, displayWidth(got))
	}
	if got := padRight("\u4ed3", 4); displayWidth(got) != 4 {
		t.Fatalf("padding = %q (%d cells)", got, displayWidth(got))
	}

	value := "a\u4ed3e\u0301b"
	lines := wrappedLine(value, 3)
	if strings.Join(lines, "") != value {
		t.Fatalf("wrapping changed content: %q", lines)
	}
	for _, line := range lines {
		if displayWidth(line) > 3 {
			t.Fatalf("wrapped line uses %d cells: %q", displayWidth(line), line)
		}
	}
	if got := takeSuffixCells("ae\u0301", 1); got != "e\u0301" {
		t.Fatalf("suffix split combining sequence: %q", got)
	}

	graphemeValue := "a" + joinedEmoji + "b"
	if got := takePrefixCells(graphemeValue, 3); got != "a"+joinedEmoji {
		t.Fatalf("prefix split joined emoji: %q", got)
	}
	if got := takePrefixCells(graphemeValue, 2); got != "a" {
		t.Fatalf("prefix retained partial joined emoji: %q", got)
	}
	if got := takeSuffixCells(graphemeValue, 3); got != joinedEmoji+"b" {
		t.Fatalf("suffix split joined emoji: %q", got)
	}
	if got := takeSuffixCells(graphemeValue, 2); got != "b" {
		t.Fatalf("suffix retained partial joined emoji: %q", got)
	}
	if got := truncateEnd(graphemeValue+"c", 4); got != "a..." {
		t.Fatalf("end truncation split joined emoji: %q", got)
	}
	if got := middleTruncate("abcdef"+joinedEmoji+"b", 7); got != "a..."+joinedEmoji+"b" {
		t.Fatalf("middle truncation split joined emoji: %q", got)
	}
	if got := wrappedLine(graphemeValue, 2); !reflect.DeepEqual(got, []string{"a", joinedEmoji, "b"}) {
		t.Fatalf("wrapping split joined emoji: %q", got)
	}
	if got := wrappedLine(joinedEmoji, 1); !reflect.DeepEqual(got, []string{"?"}) {
		t.Fatalf("narrow wrapping split joined emoji: %q", got)
	}
	if got := selectionPrefix("other", "selected", renderPalette{selection: joinedEmoji}); got != "  " {
		t.Fatalf("selection indentation = %q, want two cells", got)
	}
	if got := safeCell(joinedEmoji + "\x1b"); got != joinedEmoji+"?" || displayWidth(got) != 3 {
		t.Fatalf("sanitized grapheme = %q (%d cells)", got, displayWidth(got))
	}
}

func TestRunASCIIOverrideAndNoColor(t *testing.T) {
	installFakeGit(t)
	t.Setenv("FAKE_GIT_MODE", "run")
	t.Setenv("FAKE_GIT_MARKER", filepath.Join(t.TempDir(), "fetch"))
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LANG", "en_US.UTF-8")
	t.Setenv("NO_COLOR", "1")
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".git"))

	var unicodeOutput, stderr bytes.Buffer
	if code := run([]string{"--no-fetch", root}, &unicodeOutput, &stderr); code != 0 {
		t.Fatalf("Unicode run exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(unicodeOutput.String(), unicodePalette.rule) || strings.Contains(unicodeOutput.String(), "\x1b") {
		t.Fatalf("unexpected color-free Unicode output: %q", unicodeOutput.String())
	}

	var asciiOutput bytes.Buffer
	stderr.Reset()
	if code := run([]string{"--ascii", "--no-fetch", root}, &asciiOutput, &stderr); code != 0 {
		t.Fatalf("ASCII run exit = %d, stderr = %q", code, stderr.String())
	}
	if strings.Contains(asciiOutput.String(), unicodePalette.rule) || !strings.Contains(asciiOutput.String(), "cached") || strings.Contains(asciiOutput.String(), "\x1b") {
		t.Fatalf("unexpected ASCII output: %q", asciiOutput.String())
	}
}
