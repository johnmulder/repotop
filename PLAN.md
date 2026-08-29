# Plan: Align interactive terminal rows

## Goal

Ensure every interactive dashboard row returns to column zero when terminal output post-processing is disabled by raw mode, without changing finite output.

## Implementation

- [ ] Convert renderer line feeds to CRLF only when the terminal dashboard writes a frame.
- [ ] Keep renderer output, cached frame comparison, finite text, and JSON line endings unchanged.
- [ ] Add a dashboard regression test that detects any bare line feed in an interactive frame.
- [ ] Preserve ANSI styling, repaint suppression, resizing, and terminal restoration.

## Verification

- [ ] Run focused renderer, dashboard, and PTY integration tests.
- [ ] Run `make check` and repeated shuffled tests.
- [ ] Run Git whitespace and merge-tree checks.

## Completion

- [ ] Remove the completed proposal from `PROPOSALS.md`.
- [ ] Remove `PLAN.md`.
- [ ] Commit each major change with a one-line message.
- [ ] Fast-forward the completed feature branch into `master` and delete the feature branch.
