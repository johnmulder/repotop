# Plan: Thin Vertical Slice

## Selected proposal

Implement proposal 1, **Build a thin vertical slice first**. It is the first P0 proposal and establishes the executable path that every later proposal depends on.

## Goal

Produce a Go command that accepts zero or one directory, recursively discovers Git repositories, inspects each repository's cached local status with Git porcelain v2, sorts the resulting snapshot, and prints a compact terminal table and summary.

## Work

1. Establish the Go module and a minimal standard-library-only command.
2. Implement repository discovery, stopping descent at each `.git` file or directory and tolerating individual traversal errors.
3. Inspect each repository with one `git status --porcelain=v2 --branch -z` subprocess and normalize branch, dirty-file, untracked-file, upstream, ahead, behind, and error state.
4. Sort repositories by attention level and relative path, then render a deterministic plain-ASCII table and summary.
5. Add focused tests for discovery, porcelain parsing, sorting, rendering, CLI validation, and a real-Git smoke path.
6. Run formatting, unit/integration tests, static analysis, build, whitespace validation, and a manual executable smoke test.
7. Remove this completed plan and proposal 1 from `PROPOSALS.md`, commit the closure, merge the feature branch into `master`, and delete the feature branch.

## Acceptance criteria

- `go build ./...`, `go test ./...`, and `go vet ./...` pass.
- `repotop [directory]` prints discovered repositories without modifying them.
- A missing or invalid directory returns a concise nonzero error.
- A tree with no repositories prints a clear message.
- Repository-local failures appear as row state and do not prevent other rows from rendering.
- Output is stable, plain ASCII, and understandable without color.
- No external Go dependency is introduced.

## Out of scope

- Continuous terminal redraw and keyboard handling
- Background or manual fetch
- Parallel workers and coordinator snapshots
- Unicode/color rendering
- Configuration files or extension points

Those belong to later proposals after the end-to-end local-status path has been exercised.
