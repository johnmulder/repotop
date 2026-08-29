# Plan: Versioned release artifacts

## Goal

Add an identifiable development version and a tag-gated GitHub release pipeline that publishes checksummed macOS and Linux archives for amd64 and arm64 without adding a release framework.

## Implementation

- [ ] Add `--version`, default development builds to `devel`, and support build-time version injection with Go linker flags.
- [ ] Test that version output exits without scanning a directory or requiring Git.
- [ ] Add a Make target that builds, smoke-tests, and archives one native release binary using a validated `vMAJOR.MINOR.PATCH` version.
- [ ] Add a tag-triggered GitHub Actions workflow that gates on `make check` and uses native Linux/macOS amd64/arm64 runners.
- [ ] Smoke-test `--help` and `--version` for every matrix artifact before upload.
- [ ] Generate one checksum manifest and publish only the verified existing tag plus its four archives.
- [ ] Document version behavior and the release process without advertising a download URL before a successful hosted release exists.

## Verification

- [ ] Run focused CLI version and release-workflow contract tests.
- [ ] Build and inspect a native test archive through the Make target.
- [ ] Validate workflow YAML and shell syntax with locally available tools.
- [ ] Run `make check` and repeated shuffled tests.
- [ ] Run Git whitespace and merge-tree checks.

## Completion

- [ ] Remove the completed proposal from `PROPOSALS.md`.
- [ ] Remove `PLAN.md`.
- [ ] Commit each major change with a one-line message.
- [ ] Fast-forward the completed feature branch into `master` and delete the feature branch.
