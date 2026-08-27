# Plan: Continuous interactive dashboard

## Goal

Turn the existing progressive terminal render into the continuous, keyboard-driven dashboard described by `IDEA.md`, while preserving finite redirected and `--once` output.

## Steps

- [x] Verify `master` contains every completed plan, has no unmerged feature branches, and has no stale `PLAN.md`.
- [ ] Add a serialized interactive refresh loop that keeps input responsive, refreshes local state every two seconds, refreshes remotes every three minutes, handles manual rescan/fetch requests, and cancels cleanly.
- [ ] Add terminal-only raw input, signal-safe restoration, key decoding for quit/rescan/fetch and selection movement, plus height-aware rendering for larger repository sets.
- [ ] Add deterministic unit tests and a focused pseudo-terminal integration test covering scheduling, key handling, selection identity, viewport behavior, and terminal restoration.
- [ ] Run formatting, vet, uncached race tests, coverage, build, and interactive/redirected smoke checks.
- [ ] Remove this completed proposal from `PROPOSALS.md`, remove `PLAN.md`, commit the cleanup, merge the feature branch into `master`, and delete the feature branch.

## Acceptance criteria

- Interactive terminals remain open until `q` or a termination signal.
- `r` rediscovers repositories, `f` requests an immediate fetch, arrows move one row, and Page Up/Page Down move by a visible page.
- Local and remote work never overlap coordinator mutation, and repeated timer or key events are coalesced.
- Terminal state is restored after normal exit, signals, cancellation, and errors.
- Redirected output and `--once` still render exactly one finite snapshot, with the existing fetch policy.
- All quality gates pass without network-dependent tests.
