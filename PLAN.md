# Plan: Bounded Remote Refresh

## Selected proposal

Implement proposal 1, **Make remote refresh bounded, deduplicated, and noninteractive**. It is the first remaining P0 proposal.

## Current state

`repotop` completes a bounded local scan and renders one snapshot. It has no remote-fetch path, fetch metadata, or `--no-fetch` switch. The shared Git runner already provides output limits, typed failures, cancellation, and safe diagnostics.

## Work

1. Add a fetch wrapper around the existing Git runner that executes only `git fetch --prune`, disables terminal prompts, applies an internal network deadline, and inherits caller cancellation.
2. Add a fixed four-worker remote-refresh batch that deduplicates repository roots before queueing, permits at most one queued or active fetch per repository, and treats per-repository failures as results rather than batch failures.
3. Re-inspect local status after each successful fetch so the coordinator receives updated ahead/behind counts.
4. Store last attempt, last success, duration, and concise fetch failure separately from local repository status; a failed fetch must preserve the last successful timestamp and local status.
5. Add `--no-fetch`. By default, start one remote batch only after the initial local scan completes; with `--no-fetch`, do not create or enqueue remote work.
6. Keep the current one-shot renderer final-only. A periodic timer, manual `f` key, and progressive redraw loop do not exist yet and belong to the next rendering/scheduler proposal; do not add inert interval options or a partial terminal event loop.
7. Test fetch arguments and environment, metadata preservation, bounded concurrency without timing sleeps, duplicate coalescing, cancellation, `--no-fetch`, and repository-local failure isolation under the race detector.
8. Document the fetch boundary in `IDEA.md`, run all quality checks, then remove this plan and proposal before merging and deleting the branch.

## Acceptance criteria

- Remote work uses exactly `git fetch --prune` and never changes the worktree or local branch history.
- Background fetches set `GIT_TERMINAL_PROMPT=0` and have a fixed internal timeout.
- No more than four fetches execute concurrently.
- Duplicate repository requests in a refresh batch execute once.
- Cancellation stops queued and active cooperative work promptly.
- One repository's fetch failure does not prevent other repositories from completing.
- Successful fetches refresh ahead/behind status before the final snapshot.
- Fetch timestamps, duration, and failure remain independent from local status.
- `--no-fetch` performs no fetch work.
- `go test -race ./...`, `go vet ./...`, and `go build ./...` pass without new dependencies.

## Out of scope

- Terminal keyboard handling and a manual `f` action
- Periodic timers and `--fetch-interval`
- Progressive redraws or terminal-width behavior
- Detailed freshness/failure presentation in the table

Those require the interactive render loop and state presentation covered by the next two P0 proposals. The batch and metadata added here are their minimal remote-work foundation.
