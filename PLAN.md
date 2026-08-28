# Plan: Reliable run outcomes

## Goal

Make successful exit status mean that `repotop` produced a trustworthy result, while preserving repository-local and fetch failures as visible, nonfatal data whenever useful repository status remains available.

## Steps

- [x] Verify `master` contains every completed plan, has no unmerged feature branches, and has no stale `PLAN.md`.
- [x] Treat failures at the discovery root as fatal while retaining descendant traversal failures as partial-result warnings.
- [x] Validate the Git executable once after discovering repositories, without requiring Git for a valid empty-tree result.
- [x] Preserve typed local Git failure kinds and return a concise nonzero outcome when finite output cannot inspect any discovered repository.
- [x] Test missing Git, unreadable roots, partial scans, isolated repository failures, all-local-inspection failure, and ordinary repository states.
- [ ] Run formatting, vet, uncached race tests, coverage, build, and representative CLI smoke checks.
- [ ] Remove this completed proposal from `PROPOSALS.md`, remove `PLAN.md`, commit the cleanup, merge the feature branch into `master`, and delete the feature branch.

## Acceptance criteria

- A nonempty discovery with no usable `git` executable exits one with one concise diagnostic.
- An empty tree still exits zero without requiring Git.
- Failure to traverse the requested root exits one; unreadable descendants remain warnings when other results are available.
- Redirected and `--once` output exit one after rendering when every local inspection fails.
- Dirty, ahead, behind, fetch-failed, and isolated broken repository states still exit zero.
- Interactive monitoring does not terminate merely because its current snapshot contains only repository-local failures.
