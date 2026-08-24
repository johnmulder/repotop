# Proposals for `repotop`

These proposals turn the intent in `IDEA.md` into an implementation path while preserving the project's main constraint: `repotop` is a small, read-only dashboard, not a repository manager.

Priorities mean:

- **P0:** required for a trustworthy version 0.1
- **P1:** a high-value improvement once the basic dashboard works
- **P2:** defer until actual use demonstrates the need

## 1. Build a thin vertical slice first

**Priority:** P0

### Gap

The repository currently contains a product idea but no executable, implementation language, or build and release path. Designing every package before any repository status is visible would delay feedback on the parts most likely to shape the program: Git parsing, refresh behavior, and terminal rendering.

### Approach

Use Go for the initial implementation and build the smallest end-to-end path first:

```text
directory -> repository discovery -> local Git status -> sorted snapshot -> terminal output
```

Add continuous redraw and background fetches only after this path works against real repositories. Keep the internal seams named in `IDEA.md`, but create packages only where they isolate a real boundary, such as Git subprocesses or scheduler state.

### Implementation considerations

- Go supports a single portable binary and has adequate standard-library support for traversal, subprocesses, cancellation, and concurrency.
- Limit the terminal layer to one focused dependency, or a very small ANSI/raw-terminal adapter; do not adopt a broad application framework solely for future flexibility.
- Make `go build ./...` and `go test ./...` the initial quality gates.
- Establish macOS and Linux release builds early enough to expose terminal portability issues.
- Avoid configuration, plugin, and persistence abstractions until a concrete requirement needs them.

## 2. Specify repository discovery as a testable contract

**Priority:** P0

### Gap

`IDEA.md` describes discovery well at a product level, but edge cases remain capable of producing duplicate repositories, runaway traversal, or surprising omissions. These include a root that is itself a repository, `.git` files used by worktrees, nested repositories, symlinks, permission failures, and repositories disappearing during a scan.

### Approach

Implement discovery as a pure, independently testable operation that returns repository roots plus nonfatal discovery errors. A directory is a repository when it contains a `.git` file or directory. Once found, emit it and stop descending into that directory. Do not follow directory symlinks in version 0.1.

### Implementation considerations

- Use `filepath.WalkDir` so directories can be pruned without reading all of their contents.
- Normalize the selected root once and derive display paths relative to it.
- Define root behavior explicitly: if the selected root is a repository, report it and do not search inside it for arbitrary nested repositories.
- Treat permission and race errors as scan results, not process-fatal errors.
- Cancel an obsolete scan when `r` starts a newer one; tag results with a scan generation so late results cannot reintroduce removed repositories.
- Cover `.git` files, symlink loops, unreadable directories, and repository removal with filesystem tests.

## 3. Define exact status-count semantics around porcelain v2

**Priority:** P0

### Gap

The model lists `modified`, `staged`, and `untracked`, but does not define whether a file changed in both the index and worktree counts once or twice, how conflicts are represented, or how rename records and unusual filenames are parsed. Ambiguous counts will make the table disagree with users' normal Git tools.

### Approach

Use `git status --porcelain=v2 --branch -z` as the single local-status query and document a normalized interpretation of its headers and record types. Track staged files, worktree-modified files, conflicted files, and untracked files separately; derive any compact display count from those fields rather than discarding detail during parsing.

### Implementation considerations

- Parse NUL-delimited output so spaces, tabs, newlines, and non-ASCII bytes in filenames cannot corrupt records.
- Handle ordinary, renamed/copied, unmerged, untracked, and ignored record forms deliberately; ignored files can be omitted from the model.
- Read branch name, detached state, upstream, and ahead/behind counts from porcelain headers rather than extra subprocesses where possible.
- Decide whether the compact `M` count means unique tracked paths or the sum of staged and unstaged changes; unique paths are less surprising.
- Surface conflicts distinctly because they require more attention than an ordinary dirty worktree.
- Keep parser fixtures for every supported record form, including filenames that are hostile to line-oriented parsing.

## 4. Put every Git invocation behind one bounded runner

**Priority:** P0

### Gap

A malformed repository, locked filesystem, credential prompt, hook, or slow remote can otherwise block a worker indefinitely. Scattered subprocess calls also make timeout, environment, output-size, and error behavior inconsistent.

### Approach

Create one small Git command runner used by both local inspection and fetch work. It should invoke Git directly without a shell, accept a context deadline, capture bounded output, and return structured failure information containing the operation and a concise diagnostic.

### Implementation considerations

- Use `exec.CommandContext` with separate local-status and network-fetch deadlines.
- Pass repository paths as arguments rather than interpolating command strings.
- Set `GIT_OPTIONAL_LOCKS=0` for read-only status operations to avoid unnecessary lock contention.
- Bound captured stdout and stderr so a broken command cannot consume unbounded memory.
- Distinguish timeout, missing executable, permission, non-repository, and ordinary Git exit failures.
- Truncate secrets and excessively long diagnostics before they reach the UI; never log the complete inherited environment or remote URLs containing credentials.

## 5. Give one coordinator ownership of dashboard state

**Priority:** P0

### Gap

Local scans and remote fetches update overlapping repository records at different speeds. If workers mutate shared records directly, late or stale results can overwrite newer branch, worktree, or fetch information and introduce data races.

### Approach

Use a single coordinator to own the repository map. Workers produce typed results, and the coordinator merges them into immutable snapshots consumed by the renderer. Include repository identity and scan/fetch generations in results so obsolete work can be discarded.

### Implementation considerations

- Keep local status, remote freshness, and transient activity as separate fields so one update cannot erase another.
- Preserve the last successful remote comparison while a new fetch is running, but label it stale or fetching.
- Remove repositories only after a completed rescan confirms they are absent.
- Publish snapshots at a controlled rate to avoid redrawing once for every worker result during a large scan.
- Prefer channels and single ownership over locks spread across the model.
- Run tests with Go's race detector once concurrency exists.

## 6. Make remote refresh bounded, deduplicated, and noninteractive

**Priority:** P0

### Gap

The idea calls for asynchronous `git fetch`, but a simple timer that enqueues every repository can create duplicate work, exhaust network or credential-helper capacity, and allow a prompt or slow remote to occupy a worker forever.

### Approach

Use a small fixed fetch worker pool and at most one queued or active fetch per repository. Schedule initial fetches only after local results are visible, coalesce timer and manual requests, and update ahead/behind status as each fetch completes.

### Implementation considerations

- Start with four fetch workers and keep the value internal until measurement justifies an option.
- Use `git fetch --prune` as the only intentional repository mutation and document that it updates remote-tracking refs.
- Set `GIT_TERMINAL_PROMPT=0` for background work so authentication needs become visible failures instead of invisible hangs.
- Apply a network timeout and ensure quitting cancels queued and active work promptly.
- Respect `--no-fetch` completely, including startup and the periodic timer; define whether pressing `f` remains disabled or explicitly overrides it.
- Track last attempt, last success, duration, and concise failure independently for each repository.

## 7. Render progressive results deterministically

**Priority:** P0

### Gap

The desired interface depends on partial results and continuously changing state. Without stable ordering and explicit width behavior, rows will jump during refresh, long paths will destroy columns, and narrow terminals may hide the most useful information.

### Approach

Make rendering a pure transformation from a snapshot plus terminal dimensions to rows. Display each repository as soon as its local inspection completes, but use deterministic severity and path keys so the order settles predictably. Define a compact layout for narrow terminals rather than relying on clipping by the terminal.

### Implementation considerations

- Assign every status exactly one primary sort bucket, then sort by relative path; explicitly decide where diverged and conflicted repositories rank.
- Reserve space for status columns and truncate repository paths in the middle so the distinguishing suffix remains visible.
- Handle terminal resize events and preserve the selected repository by identity rather than row index.
- Coalesce rapid updates to prevent flicker and excessive CPU use.
- Make color a presentation layer over authoritative text.
- Golden-test the renderer at several widths using the ASCII palette proposed below.

## 8. Make failure and freshness states unambiguous

**Priority:** P0

### Gap

Labels such as `fetch failed`, `no upstream`, and `fetched 34s ago` can conceal materially different situations. A single header timestamp is especially misleading when repositories were fetched at different times or some failed while others succeeded.

### Approach

Model local inspection, remote relationship, fetch activity, and data freshness as separate states. Keep the table concise, but show the selected repository's full error and timestamps in a footer or status line. Derive any aggregate header text from repository-level data.

### Implementation considerations

- Distinguish no remote, no upstream, fetching, current, stale, failed-with-prior-data, and failed-without-data.
- Never replace known ahead/behind counts with zero after a fetch failure.
- Label counts in the summary as repository counts, not commit totals.
- Define stale relative to the configured fetch interval rather than a hard-coded age.
- Sanitize control characters in Git diagnostics before displaying them in a terminal.
- Keep detailed failures available without turning the main table into a log viewer.

## 9. Make every non-ASCII glyph deliberate and detectable

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

## 10. Test behavior with disposable real Git repositories

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

## 11. Add one-shot output by reusing the same snapshot model

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

## 12. Add scan exclusions only when traversal data justifies them

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

Implement proposals 1–4 as the first local-status milestone, then 5–8 to complete the responsive version 0.1 dashboard. Proposals 9 and 10 should land before broad distribution because they protect terminal compatibility and Git correctness. Proposal 11 is the smallest useful extension once the interactive core is stable. Proposal 12 should remain deferred until measured traversal problems appear.
