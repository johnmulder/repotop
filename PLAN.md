# Plan: Improve documentation accuracy

## Goal

Align the getting-started and design documentation with the current command, installation behavior, Git side effects, output semantics, and implemented architecture.

## Implementation

- [x] Clarify that fetch can update Git metadata while leaving checked-out files and local history untouched.
- [x] Explain where `go install .` writes the binary and that the install directory must be on `PATH`.
- [x] Tighten the descriptions of JSON paths, error fields, glyph selection, and color precedence.
- [x] Make exit-status wording cover both finite and interactive failures.
- [x] Remove stale claims in `IDEA.md` that shipped options, finite output, and the Go implementation are only future possibilities.
- [x] Preserve the source-build-only guidance because the repository still has no release host or downloadable artifacts.

## Verification

- [x] Compare every documented option with `repotop --help` and run the README option-drift test.
- [x] Run `make check` and repeated shuffled tests.
- [ ] Run Git whitespace and merge-tree checks.

## Completion

- [ ] Remove `PLAN.md` after the documentation corrections are complete.
- [ ] Commit each major change with a one-line message.
- [ ] Fast-forward the completed feature branch into `master` and delete the feature branch.
