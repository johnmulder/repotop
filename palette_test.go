package main

import (
	"bytes"
	"path/filepath"
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
	tests := []struct {
		value string
		want  int
	}{
		{value: "ascii", want: 5},
		{value: "e\u0301", want: 1},
		{value: "\u4ed3\u5e93", want: 4},
		{value: "\U0001f680", want: 2},
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
