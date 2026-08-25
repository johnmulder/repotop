# Plan: Exact Porcelain v2 Status Semantics

## Selected proposal

Implement proposal 1, **Define exact status-count semantics around porcelain v2**. It is the first remaining P0 proposal.

## Current state

`repotop` already reads one NUL-delimited `git status --porcelain=v2 --branch -z` result, handles branch headers and every record prefix, and counts each tracked record once. It currently discards the record's index/worktree distinction and treats unmerged paths as ordinary modifications.

## Work

1. Define durable model semantics for staged, worktree-modified, changed, conflicted, and untracked path counts.
2. Count each non-conflicted tracked record once as `Changed`; also count its index (`X`) and worktree (`Y`) state independently as `Staged` and `Modified`, so one path may contribute to both detailed counts without inflating `Changed`.
3. Count unmerged (`u`) records separately as `Conflicted` and ignored (`!`) records as no status.
4. Render compact worktree state as `C<n> M<n> ?<n>`, where conflicts and non-conflicted tracked changes do not overlap.
5. Update dirty-state and severity logic to include every local-change category.
6. Add parser fixtures for ordinary, renamed, copied, unmerged, untracked, ignored, and hostile NUL-delimited filenames, plus real-Git staged/worktree overlap coverage.
7. Document the normalized semantics in `IDEA.md` and run formatting, race-enabled tests, coverage, static analysis, compilation, whitespace validation, and an executable smoke test.
8. Remove this completed plan and proposal 1 from `PROPOSALS.md`, commit the closure, merge the feature branch into `master`, and delete the branch.

## Acceptance criteria

- `Staged` counts non-conflicted records whose index status is not `.`.
- `Modified` counts non-conflicted records whose worktree status is not `.`.
- `Changed` counts unique non-conflicted tracked paths regardless of whether one or both sides changed.
- `Conflicted` counts unmerged paths separately.
- `Untracked` counts `?` records; ignored records do not affect the model.
- Rename/copy source paths and hostile filenames cannot be misread as status records.
- Compact output reports conflicts distinctly and never double-counts one tracked path in `M`.
- `go test -race ./...`, `go vet ./...`, and `go build ./...` pass without new dependencies.

## Out of scope

- Per-file status display or retaining filenames in the model
- Ignored-file reporting
- Fetching or recomputing remote refs
- A details pane exposing the staged/worktree breakdown

The normalized fields preserve that future detail without expanding the current table.
