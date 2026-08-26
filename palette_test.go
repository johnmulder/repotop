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
