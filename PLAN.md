# Plan: Single-Owner Repository State

## Selected proposal

Implement proposal 1, **Give one coordinator ownership of dashboard state**. It is the first remaining P0 proposal.

## Current state

Discovery returns a complete repository list, then local Git inspection runs sequentially into a temporary slice. There is no shared mutable state yet, but adding refreshes or fetch workers directly to this loop would create overlapping writers and stale-result races.

## Work

1. Add one coordinator that exclusively owns a repository map and issues monotonically increasing scan generations.
2. Apply only updates matching the active generation and identify repositories by absolute root while keeping relative display paths in status values.
3. Preserve repositories from the prior snapshot until the active scan completes, then remove only roots absent from that completed discovery result.
4. Return copied, deterministically sorted snapshots so callers cannot mutate coordinator state.
5. Replace sequential inspection with at most eight local workers that receive repository jobs and send typed generation-tagged updates through a channel.
6. Pass caller cancellation into repository inspection and Git deadlines; an incomplete canceled scan must not perform removal.
7. Keep snapshot publication final-only in the current one-shot CLI, which already bounds publication frequency; progressive publication belongs to the later rendering proposal.
8. Test stale update rejection, deferred removal, snapshot isolation, bounded concurrent inspection, cancellation, and real CLI behavior under the race detector.
9. Document the ownership and generation contract in `IDEA.md`, run all quality checks, then remove this plan and proposal before merging and deleting the branch.

## Acceptance criteria

- Worker goroutines never read or mutate the repository map.
- Every worker result carries a scan generation and repository identity.
- Results from obsolete generations are ignored.
- Repositories disappear only after a matching scan-complete event.
- Snapshots are detached copies in deterministic status order.
- Local inspection uses no more than eight workers and respects caller cancellation.
- A canceled scan returns partial state plus cancellation and does not remove prior repositories.
- Existing CLI output and repository-local failure behavior remain intact.
- `go test -race ./...`, `go vet ./...`, and `go build ./...` pass without new dependencies.

## Out of scope

- Fetch state, fetch activity, and remote freshness fields
- Periodic rescan scheduling and keyboard handling
- Progressive redraw cadence
- Persistence or a repository database

Those states do not exist yet; the generation-tagged single-owner path is the minimum foundation they need.
