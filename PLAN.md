# Plan: Accessible terminal status color

## Goal

Add optional semantic ANSI color to interactive repository rows without changing text, width, truncation, ASCII glyph selection, redirected output, or control-character safety.

## Steps

- [x] Preserve the reviewed proposal catalogue, verify `master` contains every completed plan, and confirm there are no stale plans, unmerged completed branches, or extra worktrees.
- [x] Define dependency-free green, yellow, red, and dim row styles with deterministic status priority.
- [x] Apply styles only after plain-text layout and enable them only for a usable interactive terminal when `NO_COLOR` is absent and `TERM` is not `dumb`.
- [x] Add focused tests for style selection, semantic priority, ASCII independence, exact plain-text equivalence after stripping ANSI, and repository-controlled control characters.
- [x] Document interactive color behavior and run focused tests, source hygiene, formatting, vet, uncached race tests, repeated shuffled tests, build, and the complete quality gate.
- [ ] Remove this completed proposal from `PROPOSALS.md`, remove `PLAN.md`, commit the cleanup, merge into `master`, and delete the feature branch.

## Acceptance criteria

- Clean/current rows are green; dirty/ahead/fetching rows are yellow; behind or error rows are red; missing remote/upstream rows are dim.
- Behind/error priority wins over yellow states, and yellow states win over missing-remote dimming.
- Text remains sufficient without color and stripped colored output exactly matches unstyled output.
- `--ascii` may still use color, while `NO_COLOR`, `TERM=dumb`, redirected output, and `--once` emit no status color.
- ANSI bytes never participate in display-width calculations and cannot be supplied by repository paths or Git diagnostics.
