# Plan: Machine-readable finite snapshot

## Goal

Add one stable JSON document for finite repository inspection while preserving the existing discovery, fetch, warning, error, ordering, and exit-status behavior.

## Steps

- [x] Verify `master` contains every completed plan, has no stale plan or unmerged completed branch, and has one clean worktree.
- [x] Define a small explicit JSON contract for the absolute root and normalized repository records, including worktree counts, remote state, fetch metadata, and errors.
- [x] Add `--json` as a finite output mode that reuses `--fetch`, `--no-fetch`, exclusions, scan statistics, warnings, and existing exit statuses.
- [x] Cover deterministic serialization, control-character escaping, empty roots, optional fetch behavior, partial failures, total failures, and output-write errors.
- [x] Document the option and schema, remove JSON from the future-feature list, and run focused tests, source hygiene, formatting, vet, uncached race tests, repeated shuffled tests, build, and the complete gate.
- [ ] Remove this completed proposal from `PROPOSALS.md`, remove `PLAN.md`, commit the cleanup, merge into `master`, and delete the feature branch.

## Acceptance criteria

- Standard output contains exactly one JSON value followed by a newline, with no ANSI, display glyphs, or human table text.
- Repository-controlled strings are preserved through JSON escaping and diagnostics remain on standard error.
- Missing timestamps are explicit empty strings; present timestamps are UTC RFC 3339; duration is an integer number of milliseconds.
- `--json` implies finite mode, fetches only with `--fetch`, and returns the existing 0, 1, or 2 outcome for the same underlying result.
- The implementation uses only `encoding/json`, exposes no internal renderer structs, and defers NDJSON and schema negotiation.
