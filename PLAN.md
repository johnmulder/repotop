# Plan: Review every project text format

## Goal

Make the reviewed-character test cover project text files regardless of extension without relying on Git metadata or treating generated binary artifacts as source text.

## Implementation

- [x] Replace the extension allowlist with a source-tree walk over regular files.
- [x] Skip `.git` directories, invalid UTF-8 files, and files containing NUL bytes.
- [x] Preserve the explicit per-path reviewed-rune allowlist and path, byte-offset, and code-point diagnostics.
- [x] Add synthetic YAML and extensionless text fixtures that prove non-ASCII characters are detected.
- [x] Add binary and `.git` fixtures that prove non-project content is ignored.

## Verification

- [x] Run focused reviewed-character tests.
- [x] Run `make check` and repeated shuffled tests.
- [ ] Run Git whitespace and merge-tree checks.

## Completion

- [ ] Remove the completed proposal from `PROPOSALS.md`.
- [ ] Remove `PLAN.md`.
- [ ] Commit each major change with a one-line message.
- [ ] Fast-forward the completed feature branch into `master` and delete the feature branch.
