package main

import (
	"encoding/json"
	"io"
	"time"
)

type jsonDocument struct {
	Root         string           `json:"root"`
	Repositories []jsonRepository `json:"repositories"`
}

type jsonRepository struct {
	Path             string         `json:"path"`
	Branch           string         `json:"branch"`
	Dirty            bool           `json:"dirty"`
	Changed          int            `json:"changed"`
	Staged           int            `json:"staged"`
	Modified         int            `json:"modified"`
	Conflicted       int            `json:"conflicted"`
	Untracked        int            `json:"untracked"`
	Ahead            int            `json:"ahead"`
	Behind           int            `json:"behind"`
	HasRemote        bool           `json:"has_remote"`
	HasUpstream      bool           `json:"has_upstream"`
	RemoteState      string         `json:"remote_state"`
	Fetching         bool           `json:"fetching"`
	FetchLastAttempt string         `json:"fetch_last_attempt"`
	FetchLastSuccess string         `json:"fetch_last_success"`
	FetchDurationMS  int64          `json:"fetch_duration_ms"`
	Error            string         `json:"error"`
	ErrorKind        gitFailureKind `json:"error_kind"`
	FetchError       string         `json:"fetch_error"`
}

func renderJSON(output io.Writer, root string, statuses []repositorySnapshot, now time.Time, fetchInterval time.Duration) error {
	ordered := append([]repositorySnapshot(nil), statuses...)
	sortSnapshots(ordered)
	repositories := make([]jsonRepository, len(ordered))
	for index, status := range ordered {
		repositories[index] = jsonRepository{
			Path:             status.Path,
			Branch:           status.Branch,
			Dirty:            status.dirty(),
			Changed:          status.Changed,
			Staged:           status.Staged,
			Modified:         status.Modified,
			Conflicted:       status.Conflicted,
			Untracked:        status.Untracked,
			Ahead:            status.Ahead,
			Behind:           status.Behind,
			HasRemote:        status.HasRemote,
			HasUpstream:      status.HasUpstream,
			RemoteState:      freshnessState(status, now, fetchInterval),
			Fetching:         status.Fetching,
			FetchLastAttempt: jsonTime(status.Fetch.LastAttempt),
			FetchLastSuccess: jsonTime(status.Fetch.LastSuccess),
			FetchDurationMS:  status.Fetch.Duration.Milliseconds(),
			Error:            status.Error,
			ErrorKind:        status.FailureKind,
			FetchError:       status.Fetch.Error,
		}
	}
	return json.NewEncoder(output).Encode(jsonDocument{Root: root, Repositories: repositories})
}

func jsonTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
