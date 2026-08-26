# Plan: Make Non-ASCII Glyphs Deliberate and Detectable

## Goal

Offer an intentional Unicode terminal presentation with a dependable plain-ASCII fallback, while keeping status meaning readable and terminal alignment correct.

## Implementation

1. Centralize renderer symbols in reviewed ASCII and Unicode palettes; add `--ascii`, select ASCII automatically when the effective locale does not advertise UTF-8, and pass the chosen palette through final and progressive rendering.
2. Replace rune-count truncation, padding, wrapping, and line checks with a small standard-library cell-width implementation covering combining marks, common wide scripts, full-width forms, and emoji.
3. Keep stable golden tests on the ASCII palette; add focused tests for locale selection, Unicode glyphs, display width, CLI override behavior, and color-free readability with `NO_COLOR`.
4. Add a repository test that rejects unreviewed non-ASCII runes outside the palette and explicitly approved design/proposal text.
5. Run formatting, vet, race-enabled tests with coverage, build, diff checks, and ASCII/Unicode CLI smoke tests before implementation commits and cleanup.

## Constraints

- Add no dependency for a handful of width rules; document the approximation ceiling in code.
- Keep Unicode literals out of model, Git, coordination, and renderer logic; only the palette definition may contain runtime glyphs.
- Preserve textual state labels and repository counts in both palettes.
- Emit no ANSI color sequences; the existing color-free renderer therefore honors `NO_COLOR` without a new color subsystem.

## Completion

- Remove this proposal from `PROPOSALS.md` and renumber the remainder.
- Delete `PLAN.md`.
- Commit cleanup, fast-forward the feature branch into `master`, and delete the feature branch.
