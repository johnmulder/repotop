package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

func TestREADMEListsEveryShippedOption(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	var stdout, help bytes.Buffer
	if code := run([]string{"--help"}, &stdout, &help); code != 0 {
		t.Fatalf("help exit = %d: %s", code, help.String())
	}
	for _, line := range strings.Split(help.String(), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "-") {
			continue
		}
		option := "-" + fields[0]
		if !bytes.Contains(readme, []byte("`"+option)) {
			t.Errorf("README does not document %s", option)
		}
	}
}

func TestNonASCIISourceIsReviewed(t *testing.T) {
	reviewed := map[string]string{
		"IDEA.md":    "\u2013\u2014\u2191\u2193\u2500\u2713",
		"palette.go": "\u2014\u203a\u2191\u2193\u2500\u2713",
	}
	extensions := map[string]bool{".go": true, ".md": true, ".mod": true}

	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !extensions[filepath.Ext(path)] {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(strings.TrimPrefix(path, "."+string(filepath.Separator)))
		for offset, character := range string(content) {
			if character <= unicode.MaxASCII || strings.ContainsRune(reviewed[name], character) {
				continue
			}
			t.Errorf("%s byte %d contains unreviewed rune U+%04X", name, offset, character)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
