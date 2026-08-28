package main

import (
	"strings"
	"time"
)

type renderPalette struct {
	rule      string
	selection string
	ahead     string
	behind    string
	current   string
	missing   string
}

type terminalStyle struct {
	clean     string
	attention string
	danger    string
	muted     string
	reset     string
}

var asciiPalette = renderPalette{
	rule:      "-",
	selection: "> ",
	ahead:     "+",
	behind:    "-",
	missing:   "-",
}

var unicodePalette = renderPalette{
	rule:      "─",
	selection: "› ",
	ahead:     "↑",
	behind:    "↓",
	current:   "✓ ",
	missing:   "—",
}

var ansiTerminalStyle = terminalStyle{
	clean:     "\x1b[32m",
	attention: "\x1b[33m",
	danger:    "\x1b[31m",
	muted:     "\x1b[2m",
	reset:     "\x1b[0m",
}

func selectPalette(forceASCII bool, getenv func(string) string) renderPalette {
	if forceASCII || !localeSupportsUTF8(getenv) {
		return asciiPalette
	}
	return unicodePalette
}

func selectTerminalStyle(lookupEnv func(string) (string, bool)) terminalStyle {
	if _, disabled := lookupEnv("NO_COLOR"); disabled {
		return terminalStyle{}
	}
	if termName, _ := lookupEnv("TERM"); strings.EqualFold(strings.TrimSpace(termName), "dumb") {
		return terminalStyle{}
	}
	return ansiTerminalStyle
}

func (style terminalStyle) apply(snapshot repositorySnapshot, now time.Time, fetchInterval time.Duration, value string) string {
	code := ""
	state := freshnessState(snapshot, now, fetchInterval)
	switch {
	case snapshot.Error != "", snapshot.Fetch.Error != "", snapshot.Behind > 0:
		code = style.danger
	case snapshot.dirty(), snapshot.Ahead > 0, snapshot.Fetching, state == "stale":
		code = style.attention
	case !snapshot.HasRemote, !snapshot.HasUpstream:
		code = style.muted
	case state == "current":
		code = style.clean
	}
	if code == "" {
		return value
	}
	return code + value + style.reset
}

func localeSupportsUTF8(getenv func(string) string) bool {
	for _, name := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		value := getenv(name)
		if value == "" {
			continue
		}
		value = strings.ReplaceAll(strings.ToLower(value), "-", "")
		return strings.Contains(value, "utf8")
	}
	return false
}
