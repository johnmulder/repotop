# Plan: Add One-Shot Output from the Snapshot Model

## Goal

Make `repotop` predictable in scripts, CI, bug reports, accessibility tools, and forced terminal snapshots without creating a second status or rendering implementation.

## Implementation

1. Add `--once` to suppress progressive terminal initialization and render one final snapshot through the existing renderer.
2. Treat redirected and `--once` execution as one-shot: use cached local/upstream data by default and add explicit `--fetch` for a pre-render remote refresh; retain `--no-fetch` for progressive-terminal control and reject contradictory fetch flags.
3. Extend CLI tests for cached defaults, explicit fetch, flag validation, single-render/no-ANSI output, conservative repository-state exit codes, ASCII/Unicode palettes, and forced one-shot terminal behavior.
4. Run formatting, vet, uncached race-enabled tests with coverage, build, diff checks, and progressive/one-shot CLI smoke tests before implementation commits and cleanup.

## Constraints

- Reuse `repositorySnapshot`, the coordinator, and the existing plain-text renderer.
- Keep dirty, ahead, behind, and diverged states as data with exit code zero.
- Keep all operations read-only except an explicitly requested fetch.
- Do not add JSON until an actual automation consumer requires a versioned schema.
- Add no dependency or second output model.

## Completion

- Remove this proposal from `PROPOSALS.md` and renumber the remainder.
- Delete `PLAN.md`.
- Commit cleanup, fast-forward the feature branch into `master`, and delete the feature branch.
