package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type failingJSONWriter struct {
	err error
}

func (writer failingJSONWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

func TestRenderJSONContractAndEscaping(t *testing.T) {
	now := time.Date(2026, time.August, 28, 12, 0, 0, 0, time.UTC)
	status := repositorySnapshot{
		repoStatus: repoStatus{
			Path:        "repo\x1b[31m",
			Branch:      "topic",
			Changed:     2,
			Staged:      1,
			Modified:    1,
			Conflicted:  1,
			Untracked:   3,
			Ahead:       4,
			Behind:      5,
			HasRemote:   true,
			HasUpstream: true,
			Error:       "local\nerror",
			FailureKind: gitFailureExit,
		},
		Fetch: fetchStatus{
			LastAttempt: now.Add(-2 * time.Second),
			LastSuccess: now.Add(-time.Minute),
			Duration:    1500 * time.Millisecond,
			Error:       "fetch\tfailed",
		},
		Fetching: true,
	}
	var output bytes.Buffer
	if err := renderJSON(&output, "/root", []repositorySnapshot{status}, now, defaultFetchInterval); err != nil {
		t.Fatal(err)
	}
	want := `{"root":"/root","repositories":[{"path":"repo\u001b[31m","branch":"topic","dirty":true,"changed":2,"staged":1,"modified":1,"conflicted":1,"untracked":3,"ahead":4,"behind":5,"has_remote":true,"has_upstream":true,"remote_state":"error","fetching":true,"fetch_last_attempt":"2026-08-28T11:59:58Z","fetch_last_success":"2026-08-28T11:59:00Z","fetch_duration_ms":1500,"error":"local\nerror","error_kind":"exit","fetch_error":"fetch\tfailed"}]}` + "\n"
	if output.String() != want {
		t.Fatalf("JSON output:\n%s\nwant:\n%s", output.String(), want)
	}

	output.Reset()
	if err := renderJSON(&output, "/empty", nil, now, defaultFetchInterval); err != nil {
		t.Fatal(err)
	}
	if output.String() != "{\"root\":\"/empty\",\"repositories\":[]}\n" {
		t.Fatalf("empty JSON output = %q", output.String())
	}

	output.Reset()
	status.Error = ""
	status.Fetching = false
	status.Fetch.Error = ""
	if err := renderJSON(&output, "/root", []repositorySnapshot{status}, now, 30*time.Second); err != nil {
		t.Fatal(err)
	}
	if got := decodeJSONDocument(t, output.Bytes()).Repositories[0].RemoteState; got != "stale" {
		t.Fatalf("configured freshness state = %q, want stale", got)
	}

	wantErr := errors.New("write failed")
	if err := renderJSON(failingJSONWriter{err: wantErr}, "/root", nil, now, defaultFetchInterval); !errors.Is(err, wantErr) {
		t.Fatalf("render error = %v, want %v", err, wantErr)
	}
}

func TestRunJSONSnapshotAndFetchPolicy(t *testing.T) {
	installFakeGit(t)
	t.Setenv("FAKE_GIT_MODE", "run-behind")
	marker := filepath.Join(t.TempDir(), "fetch")
	t.Setenv("FAKE_GIT_MARKER", marker)
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".git"))

	var stdout, stderr bytes.Buffer
	if code := run([]string{"--ascii", "--json", "--fetch-interval", "15s", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("cached JSON exit = %d, stderr = %q", code, stderr.String())
	}
	document := decodeJSONDocument(t, stdout.Bytes())
	if document.Root != root || len(document.Repositories) != 1 {
		t.Fatalf("JSON document = %+v", document)
	}
	repository := document.Repositories[0]
	if repository.Path != "." || repository.Behind != 1 || repository.RemoteState != "cached" || repository.FetchLastAttempt != "" || repository.Fetching {
		t.Fatalf("cached repository = %+v", repository)
	}
	if strings.Contains(stdout.String(), "REPOSITORY") || strings.Contains(stdout.String(), "\x1b") || stderr.Len() != 0 {
		t.Fatalf("JSON leaked presentation output: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("default JSON fetched: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"--json", "--fetch", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("fetched JSON exit = %d, stderr = %q", code, stderr.String())
	}
	document = decodeJSONDocument(t, stdout.Bytes())
	repository = document.Repositories[0]
	if repository.RemoteState != "current" || repository.FetchLastAttempt == "" || repository.FetchLastSuccess == "" || repository.FetchError != "" {
		t.Fatalf("fetched repository = %+v", repository)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("explicit JSON fetch marker: %v", err)
	}
}

func TestRunJSONEmptyAndFailureOutcomes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--json", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("empty JSON exit = %d, stderr = %q", code, stderr.String())
	}
	if document := decodeJSONDocument(t, stdout.Bytes()); document.Root != root || len(document.Repositories) != 0 {
		t.Fatalf("empty JSON document = %+v", document)
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"--json", root}, failingJSONWriter{err: errors.New("closed")}, &stderr); code != 1 {
		t.Fatalf("JSON write failure exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "repotop: render: closed") {
		t.Fatalf("JSON write failure diagnostic = %q", stderr.String())
	}

	installFakeGit(t)
	t.Setenv("FAKE_GIT_MODE", "run-partial")
	root = t.TempDir()
	for _, name := range []string{"broken", "good"} {
		mustMkdir(t, filepath.Join(root, name, ".git"))
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"--json", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("partial JSON exit = %d, stderr = %q", code, stderr.String())
	}
	document := decodeJSONDocument(t, stdout.Bytes())
	if len(document.Repositories) != 2 || document.Repositories[0].Path != "broken" || document.Repositories[0].RemoteState != "error" || document.Repositories[0].Error == "" || !strings.Contains(stderr.String(), "warning: broken:") {
		t.Fatalf("partial JSON: document=%+v stderr=%q", document, stderr.String())
	}

	t.Setenv("FAKE_GIT_MODE", "ordinary")
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"--json", root}, &stdout, &stderr); code != 1 {
		t.Fatalf("failed JSON exit = %d, stderr = %q", code, stderr.String())
	}
	document = decodeJSONDocument(t, stdout.Bytes())
	if len(document.Repositories) != 2 || document.Repositories[0].Error == "" || document.Repositories[1].Error == "" || strings.Contains(stderr.String(), "warning:") || strings.Count(stderr.String(), "unable to inspect any repository") != 1 {
		t.Fatalf("failed JSON: document=%+v stderr=%q", document, stderr.String())
	}
}

func decodeJSONDocument(t *testing.T, data []byte) jsonDocument {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	var document jsonDocument
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, data)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		t.Fatalf("JSON contained more than one value: %v", err)
	}
	return document
}
