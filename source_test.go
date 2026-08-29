package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
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

func TestReleaseWorkflowContract(t *testing.T) {
	workflow, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"make check",
		"ubuntu-24.04\n",
		"ubuntu-24.04-arm\n",
		"macos-15-intel\n",
		"macos-15\n",
		"make release RELEASE_VERSION=\"$GITHUB_REF_NAME\"",
		"actions/upload-artifact@v7",
		"archive: false",
		"actions/download-artifact@v8",
		"skip-decompress: true",
		"sha256sum *.tar.gz > checksums.txt",
		"--verify-tag --generate-notes",
	} {
		if !bytes.Contains(workflow, []byte(required)) {
			t.Errorf("release workflow is missing %q", required)
		}
	}
}

func TestNonASCIISourceIsReviewed(t *testing.T) {
	reviewed := map[string]string{
		"IDEA.md":    "\u2013\u2014\u2191\u2193\u2500\u2713",
		"palette.go": "\u2014\u203a\u2191\u2193\u2500\u2713",
	}
	findings, err := findUnreviewedRunes(".", reviewed)
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		t.Error(finding)
	}
}

func TestFindUnreviewedRunesCoversProjectTextFormats(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"Makefile":    []byte("target \u2602\n"),
		"config.yml":  []byte("name: \u2603\n"),
		"binary.dat":  {0, 0xe2, 0x98, 0x83},
		"invalid.bin": {0xff, 0xfe},
		".git/config": []byte("ignored \u2605\n"),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	findings, err := findUnreviewedRunes(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("findings = %v, want YAML and extensionless text", findings)
	}
	joined := strings.Join(findings, "\n")
	for _, want := range []string{
		"Makefile byte 7 contains unreviewed rune U+2602",
		"config.yml byte 6 contains unreviewed rune U+2603",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("findings = %v, missing %q", findings, want)
		}
	}
}

func findUnreviewedRunes(root string, reviewed map[string]string) ([]string, error) {
	var findings []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.IndexByte(content, 0) >= 0 || !utf8.Valid(content) {
			return nil
		}
		name, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name = filepath.ToSlash(name)
		for offset, character := range string(content) {
			if character <= unicode.MaxASCII || strings.ContainsRune(reviewed[name], character) {
				continue
			}
			findings = append(findings, fmt.Sprintf("%s byte %d contains unreviewed rune U+%04X", name, offset, character))
		}
		return nil
	})
	return findings, err
}
