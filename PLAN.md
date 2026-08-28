# Plan: Configurable remote refresh interval

## Goal

Add a positive `--fetch-interval DURATION` option that changes the interactive remote-refresh schedule and the remote freshness threshold while preserving the existing three-minute default and finite-mode fetch policy.

## Implementation

- [ ] Parse the option with Go's duration flag support and reject zero or negative values as command-line usage errors.
- [ ] Pass the selected interval to the interactive session ticker, terminal dashboard, finite text renderer, and JSON freshness renderer.
- [ ] Preserve `--no-fetch` as the only way to disable remote fetching and preserve `--fetch` as finite mode's explicit fetch opt-in.
- [ ] Add focused tests for parsing, validation, propagation, freshness behavior, and unchanged finite fetch behavior without wall-clock sleeps.
- [ ] Document the option, its accepted duration syntax, its interactive scheduling behavior, and its finite-mode freshness-only effect.

## Verification

- [ ] Run focused option, session, renderer, JSON, and documentation tests.
- [ ] Run `make check` and repeat/shuffle tests.
- [ ] Run Git whitespace and merge-tree checks.

## Completion

- [ ] Remove this completed proposal from `PROPOSALS.md`.
- [ ] Remove `PLAN.md`.
- [ ] Commit each major change with a one-line message.
- [ ] Fast-forward the completed feature branch into `master` and delete the feature branch.
