# Plan: Bounded Git Command Runner

## Selected proposal

Implement proposal 1, **Put every Git invocation behind one bounded runner**. It is the first remaining P0 proposal.

## Current state

Production has one direct Git subprocess in repository inspection. It already has a five-second context deadline and disables optional locks, but `CombinedOutput` can grow without bound and failures are flattened immediately into unclassified text.

## Work

1. Add one standard-library Git runner that invokes `git` directly, accepts a caller-owned context, repository path, read-only flag, and Git arguments.
2. Capture stdout and stderr independently with fixed memory ceilings while reporting truncation instead of parsing incomplete successful output.
3. Return typed failures containing the Git operation, failure kind, concise diagnostic, and wrapped cause.
4. Distinguish timeout, cancellation, missing executable, permission, non-repository, output-limit, and ordinary exit failures.
5. Collapse diagnostics to one line, redact URL userinfo, and cap displayed diagnostic length without logging command environments or complete remote arguments.
6. Route local status inspection through the runner; keep its five-second deadline and set `GIT_OPTIONAL_LOCKS=0` only for read-only operations.
7. Test capped writes, diagnostic redaction/truncation, typed failure classification, and the existing real-Git inspection path.
8. Run formatting, race-enabled tests, coverage, static analysis, compilation, dependency, whitespace, and executable smoke checks.
9. Remove this completed plan and proposal 1 from `PROPOSALS.md`, commit the closure, merge the feature branch into `master`, and delete the branch.

## Acceptance criteria

- Production has no Git subprocess outside the shared runner.
- The runner never invokes a shell or interpolates a command string.
- Successful output exceeding the ceiling fails instead of reaching the porcelain parser incomplete.
- Failure diagnostics remain bounded, one-line, and credential-redacted.
- Callers can inspect a typed failure kind while the UI receives a concise error string.
- Read-only Git calls disable optional locks; future mutating calls can opt out.
- Local timeout and cancellation are distinguishable.
- `go test -race ./...`, `go vet ./...`, and `go build ./...` pass without new dependencies.

## Out of scope

- Fetch implementation and its network timeout value
- Retry policy or worker-pool scheduling
- Persistent logs or telemetry
- A public/general-purpose process runner

The read-only flag and caller-owned context are the only flexibility needed by the future fetch path.
