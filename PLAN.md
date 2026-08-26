# Plan: Deterministic Progressive Rendering

## Selected proposal

Implement proposal 1, **Render progressive results deterministically**. It is the first remaining P0 proposal.

## Current state

Local and remote workers update a single-owner coordinator, but callers receive only the final snapshots. Rendering uses an unconstrained tab writer, so it has no terminal-width input, path truncation, compact layout, repaint behavior, resize handling, or stable selection identity.

## Work

1. Define one explicit primary severity bucket for every repository: error, behind, diverged, conflicted, dirty, ahead, no upstream, then synchronized; use relative path as the deterministic secondary key.
2. Replace tab-dependent layout with a pure ASCII transformation from snapshot, width, and selected repository identity to rendered text.
3. Add wide, compact, and narrow layouts that reserve status space, truncate repository paths in the middle, truncate other cells safely, and keep every rendered line within the requested width.
4. Keep plain status text authoritative. Do not add color or Unicode policy before their dedicated proposal; the renderer's plain cells remain suitable for a later presentation layer.
5. Add a terminal publisher that selects repositories by path identity, repaints changed snapshots with ANSI controls, queries width using the native terminal ioctl on macOS/Linux, and redraws on `SIGWINCH`.
6. Publish the first local result immediately, coalesce rapid later updates to at most 20 redraws per second, and always publish the completed local and remote snapshots.
7. Keep non-terminal writers final-only and free of ANSI control sequences.
8. Golden-test rendering at representative wide, compact, and narrow widths; test line bounds, middle truncation, selection stability after reorder, coalescing, progressive local/remote publication, and non-terminal behavior under the race detector.
9. Document the render and publication contract in `IDEA.md`, run all quality checks, then remove this plan and proposal before merging and deleting the branch.

## Acceptance criteria

- Sorting is deterministic and every status occupies exactly one documented primary bucket.
- Diverged repositories sort after behind-only repositories; conflicts sort after divergence and before other dirty states.
- Rendering is pure for the same snapshot, width, and selected identity.
- Repository suffixes remain visible when paths are truncated.
- Every output line fits the requested positive width.
- Wide, compact, and narrow layouts retain authoritative worktree and remote text.
- Terminal snapshots appear during local inspection and remote refresh rather than only at completion.
- Rapid updates are coalesced while the first and final snapshots are never lost.
- Resize redraws use the latest snapshot, and selection follows repository identity across reordering.
- Redirected output remains one final, ANSI-free snapshot.
- `go test -race ./...`, `go vet ./...`, and `go build ./...` pass without new dependencies.

## Out of scope

- Raw keyboard mode, selection movement, scrolling, and a long-running refresh loop
- Color policy and `NO_COLOR`
- Unicode palettes and display-cell-width handling
- Detailed fetch freshness and failure presentation

Those behaviors have dedicated later proposals. This plan supplies the deterministic renderer and progressive publication boundary they need without adding a terminal framework.
