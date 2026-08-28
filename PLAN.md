# Plan: Grapheme-aware terminal layout

## Goal

Measure and slice terminal text by Unicode grapheme cluster so joined emoji and modifiers occupy the expected cells and are never split, while preserving existing ASCII behavior and control sanitization.

## Steps

- [x] Verify `master` contains every completed plan, has no unmerged completed feature branch, and has no stale `PLAN.md`.
- [x] Add focused width, truncation, wrapping, ambiguous-width, and sanitization cases for combining text, CJK text, emoji modifiers, and joined emoji.
- [x] Replace the custom rune-width approximation with `rivo/uniseg` and route measurement, slicing, wrapping, padding, and selection indentation through grapheme-aware helpers.
- [x] Run formatting, vet, uncached race tests, repeated shuffled tests, build, and the complete quality gate.
- [ ] Remove this completed proposal from `PROPOSALS.md`, remove `PLAN.md`, commit the cleanup, merge into `master`, and delete the feature branch.

## Acceptance criteria

- Joined emoji and emoji-plus-modifier clusters measure as two cells and are never split by prefix, suffix, middle, end, or wrapping boundaries.
- Combining sequences and CJK text retain their existing expected widths.
- East Asian ambiguous characters use the package default of one cell.
- `--ascii` continues to affect only application glyphs, and terminal controls remain sanitized before layout.
- The implementation adds only the maintained grapheme-width package and removes the custom Unicode range table.
