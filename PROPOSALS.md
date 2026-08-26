# Proposals for `repotop`

These proposals turn the intent in `IDEA.md` into an implementation path while preserving the project's main constraint: `repotop` is a small, read-only dashboard, not a repository manager.

Priorities mean:

- **P0:** required for a trustworthy version 0.1
- **P1:** a high-value improvement once the basic dashboard works
- **P2:** defer until actual use demonstrates the need

## 1. Make every non-ASCII glyph deliberate and detectable

**Priority:** P1

### Gap

The example uses an em dash, box-drawing characters, arrows, and a check mark. These improve a capable terminal but render poorly under some locales, fonts, logs, multiplexers, or copy/paste paths. Future contributors also have no way to tell whether a new departure from plain ASCII is intentional.

### Approach

Centralize decorative symbols in two renderer palettes: Unicode and plain ASCII. Support an explicit `--ascii` mode and choose the ASCII palette automatically when the locale does not advertise UTF-8. Keep all status meaning in text so either palette communicates the same facts.

Example mappings:

```text
Unicode       ASCII
✓             ok
↑2            +2
↓3            -3
↑1 ↓2         +1 -2
────────      --------
—             -
```

### Implementation considerations

- Keep Unicode literals out of model and Git-parsing code; only the Unicode palette should contain them.
- Add a small source check that finds non-ASCII bytes and allows only explicitly reviewed files or literals. This makes new deviations visible in review without banning them outright.
- Measure rendered cell width rather than byte or rune count when truncating Unicode text.
- Honor `NO_COLOR`, and ensure ASCII mode remains understandable with color disabled.
- Use ASCII golden snapshots in tests for stable diffs; add a smaller set of Unicode rendering tests for width and glyph selection.

## 2. Test behavior with disposable real Git repositories

**Priority:** P1

### Gap

Parser unit tests alone cannot prove that the program agrees with Git across staged changes, detached heads, worktrees, upstream configuration, divergence, or fetch updates. Hand-maintained repository fixtures are brittle and difficult to review.

### Approach

Build integration tests that create temporary repositories and local bare remotes with the installed `git` executable. Use real commits and refs to exercise end-to-end inspection while keeping network access out of the test suite.

### Implementation considerations

- Set test-local author identity and deterministic branch names rather than relying on global Git configuration.
- Cover clean, staged, modified, untracked, conflicted, ahead, behind, diverged, detached, no upstream, no remote, worktree, and broken-repository states.
- Use local bare remotes to verify fetch and prune behavior without credentials or external services.
- Keep scheduler tests deterministic with an injectable clock or explicit trigger channel; avoid timing assertions based on sleeps.
- Test cancellation and late-result rejection as well as happy paths.
- Skip only tests whose external prerequisites are genuinely unavailable; Git itself is a product requirement.

## 3. Add one-shot output by reusing the same snapshot model

**Priority:** P1

### Gap

The TUI is useful to a person at a terminal, but it is awkward in CI, shell scripts, bug reports, and accessibility tools. Adding a second status implementation later would risk semantic drift from the dashboard.

### Approach

After version 0.1 is stable, add a one-shot mode that performs one local scan and renders the same normalized snapshot as plain text. Add JSON only if a real automation use case appears, using the same model rather than another set of Git queries.

### Implementation considerations

- Do not initialize raw-terminal mode or emit ANSI control sequences when stdout is not a TTY.
- Default one-shot execution to cached local/upstream information so it completes promptly; require an explicit flag if it should fetch first.
- Define exit codes conservatively: repository states such as dirty or behind should normally be data, not command failure.
- Version a future JSON schema before external users depend on it, and represent unavailable values distinctly from numeric zero.
- Keep this read-only and resist turning it into a policy checker or CI gate inside `repotop`.

## 4. Add scan exclusions only when traversal data justifies them

**Priority:** P2

### Gap

Some directory trees contain mounted volumes, generated directories, vendor trees, or permission-heavy areas that can make recursive discovery slow. A large configuration system would solve this speculatively while weakening the project's quiet defaults.

### Approach

Instrument discovery duration and first rely on the built-in pruning rules: do not follow symlinks, skip `.git` internals, and stop at repository roots. If real usage still identifies recurring hotspots, add a repeatable command-line exclusion rather than a configuration file.

### Implementation considerations

- Prefer an explicit root-relative `--exclude PATH` rule before glob languages or ignore-file formats.
- Apply exclusions only to discovery, never silently to repositories already named or selected through another future interface.
- Report invalid exclusions clearly and show the active exclusions in help output.
- Avoid implicitly honoring `.gitignore`; it describes repository contents, not which repositories a dashboard should discover.
- Set a measurable trigger for this work, such as scans that regularly miss the initial-result target on representative 10–200 repository trees.

## Recommended delivery order

Proposals 1 and 2 should land before broad distribution because they protect terminal compatibility and Git correctness. Proposal 3 is the smallest useful extension once the interactive core is stable. Proposal 4 should remain deferred until measured traversal problems appear.
