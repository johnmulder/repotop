# Plan: Test Behavior with Disposable Real Git Repositories

## Goal

Prove that repository discovery, inspection, and refresh behavior agrees with an installed Git executable across representative local and remote states without network access or persistent fixtures.

## Implementation

1. Reuse the existing Git test runner and add only small helpers for initialized repositories, local author identity, commits, clones, and expected Git failures.
2. Add real-repository tests for clean, staged, modified, untracked, conflicted, detached, no-remote, no-upstream, linked-worktree, and broken-repository states.
3. Add local bare-remote tests for ahead, behind, diverged, successful coordinator refresh metadata, fetch updates, and pruning deleted remote branches.
4. Keep timing deterministic: use no sleeps or external services, and retain the existing injected-clock, cancellation, and stale-generation tests for scheduler and late-result behavior.
5. Run formatting, vet, uncached race-enabled tests with coverage, build, diff checks, and the ASCII/Unicode CLI smoke tests before implementation commits and cleanup.

## Constraints

- Use only temporary directories and local filesystem remotes.
- Set repository-local author identity and explicit `main`/topic branch names.
- Skip real-Git tests only when the `git` executable is unavailable.
- Add no production abstraction or dependency solely for tests.

## Completion

- Remove this proposal from `PROPOSALS.md` and renumber the remainder.
- Delete `PLAN.md`.
- Commit cleanup, fast-forward the feature branch into `master`, and delete the feature branch.
