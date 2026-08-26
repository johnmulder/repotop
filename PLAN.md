# Plan: Make Failure and Freshness States Unambiguous

## Goal

Represent local inspection, remote relationship, fetch activity, and freshness independently so the table never hides known repository state behind an ambiguous fetch result.

## Implementation

1. Add a detached repository snapshot containing local status and fetch metadata, detect `no remote` separately from `no upstream`, and expose queued/running fetch activity without discarding known ahead/behind counts.
2. Derive concise remote and freshness labels from each snapshot using an injected clock and fetch interval, and show selected-repository timestamps and complete sanitized errors in wrapped detail lines.
3. Make summary wording explicitly repository-based, then extend parser, coordinator, fetch, renderer, terminal, and integration tests for every state and narrow width.
4. Run formatting, vet, race-enabled tests with coverage, build, diff checks, and a CLI smoke test before each implementation commit and final cleanup.

## Constraints

- Preserve known local and ahead/behind data when fetches fail.
- Treat freshness as relative to the interval supplied to rendering; use the existing three-minute runtime default without adding an inert CLI setting.
- Sanitize control characters in table cells and detailed diagnostics.
- Add no dependencies and no periodic scheduler in this proposal.

## Completion

- Remove this proposal from `PROPOSALS.md`.
- Delete `PLAN.md`.
- Commit the cleanup, merge the feature branch into `master`, and delete the feature branch.
