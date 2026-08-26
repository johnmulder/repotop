package main

import "strings"

type renderPalette struct {
	rule      string
	selection string
	ahead     string
	behind    string
	current   string
	missing   string
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

func selectPalette(forceASCII bool, getenv func(string) string) renderPalette {
	if forceASCII || !localeSupportsUTF8(getenv) {
		return asciiPalette
	}
	return unicodePalette
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
