# Plan: Repository Discovery Contract

## Selected proposal

Implement proposal 1, **Specify repository discovery as a testable contract**. It is the first remaining P0 proposal.

## Current state

The vertical slice already uses `filepath.WalkDir`, recognizes `.git` files and directories, stops at repository roots, returns traversal warnings, and derives display paths from one absolute root. The remaining work is to make cancellation and edge-case behavior explicit and verified.

## Work

1. Accept a `context.Context` in discovery and return cancellation separately from nonfatal traversal errors, while retaining deterministically sorted partial results.
2. Keep traversal synchronous and snapshot-based so callers can discard an obsolete scan atomically; generation IDs remain unnecessary until an asynchronous rescan coordinator exists.
3. Verify that directory symlinks—including loops and links to repositories—are not followed.
4. Verify `.git` directory and file markers, root-repository pruning, nested-repository pruning, and deterministic ordering.
5. Verify cancellation, inaccessible-directory warnings, and a repository disappearing between discovery and inspection.
6. Run formatting, race-enabled tests, coverage, static analysis, compilation, whitespace validation, and an executable smoke test.
7. Remove this completed plan and proposal 1 from `PROPOSALS.md`, commit the closure, merge the feature branch into `master`, and delete the branch.

## Acceptance criteria

- Discovery returns repository roots, nonfatal traversal errors, and a distinct fatal/cancellation error.
- A canceled scan stops promptly and can be discarded by its caller.
- Encountered directory symlinks are never traversed.
- One inaccessible or vanished path does not suppress other discovered repositories.
- A vanished repository becomes a repository-local inspection error rather than aborting the result set.
- Repository paths are unique, sorted, and rooted consistently.
- `go test -race ./...`, `go vet ./...`, and `go build ./...` pass without new dependencies.

## Out of scope

- Rescan key handling, asynchronous discovery, and generation ownership
- Repository caching or persistence
- Configurable traversal exclusions
- Following explicitly selected or encountered symlinked directories

These require scheduler or configuration behavior that the current one-shot command does not have.
