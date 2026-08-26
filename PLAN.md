# Plan: Add Measurable, Explicit Scan Exclusions

## Goal

Let users skip known discovery hotspots without weakening quiet defaults or introducing a configuration language, while exposing opt-in timing data that makes exclusions evidence-based.

## Implementation

1. Extend discovery with an exact-path exclusion set applied before descending into directories; retain symlink avoidance, `.git` pruning, and repository-root stopping.
2. Add repeatable root-relative `--exclude PATH` validation for existing directories, reject absolute/escaping/root/file paths clearly, and document the active flag in expanded help.
3. Add opt-in `--scan-stats` output with discovery duration, repository count, and exclusion count so users can measure representative 10-200 repository trees.
4. Test repeated exclusions, invalid paths, nested pruning, `.gitignore` independence, quiet defaults, timing output, cancellation, and unchanged rendering/fetch behavior.
5. Run formatting, vet, uncached race-enabled tests with coverage, build, diff checks, and excluded/non-excluded CLI smoke tests before implementation commits and cleanup.

## Constraints

- Match exact root-relative directory paths only; add no globs, ignore files, or config files.
- Apply exclusions only to recursive discovery.
- Keep no-exclusion behavior and output unchanged.
- Add no dependency.

## Completion

- Remove this proposal from `PROPOSALS.md`.
- Delete `PLAN.md`.
- Commit cleanup, fast-forward the feature branch into `master`, and delete the feature branch.
