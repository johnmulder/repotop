# Plan: Reproducible quality gate

## Goal

Define one local command for every required project check and run that exact command in continuous integration on the supported macOS and Linux platforms at the Go 1.24 compatibility floor.

## Steps

- [x] Verify `master` contains every completed plan, has no unmerged feature branches, and has no stale `PLAN.md`.
- [x] Add a minimal `make check` entry point covering format verification, `go vet`, uncached race-enabled tests, and build.
- [x] Add a read-only GitHub Actions workflow that uses Go 1.24 from `go.mod` and calls `make check` on Linux and macOS.
- [x] Verify the local gate, individual targets, workflow structure, repeated tests, and failure behavior for unformatted source.
- [ ] Remove this completed proposal from `PROPOSALS.md`, remove `PLAN.md`, commit the cleanup, merge the feature branch into `master`, and delete the feature branch.

## Acceptance criteria

- `make check` is the only command contributors and CI need for the complete gate.
- The format check reports unformatted files and exits nonzero without modifying them.
- Tests are uncached and race-enabled, and real-Git integration tests remain offline and self-configuring.
- CI runs the same gate on current Linux and macOS runners using the latest Go 1.24 patch release.
- The gate adds no runtime dependency, custom linter, or coverage threshold.
