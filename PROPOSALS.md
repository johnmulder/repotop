# Proposals for `repotop`

Proposals are ordered by priority. The review that produced this list found the implementation well tested and internally cohesive, so these entries focus on gaps that affect the product contract or make regressions harder to detect.

## P1: Distinguish unusable runs from repository-local errors

### Gap

Recoverable repository failures are correctly represented as row data, but systemic failures can currently look successful. If `git` is unavailable, every discovered repository becomes an error row and the process still exits zero. A root that cannot be traversed can likewise emit a warning followed by `no Git repositories found` and a successful exit. This makes `--once` unreliable for scripts because a complete inability to inspect the requested tree is indistinguishable from a valid result.

### Approach

Define a small run-outcome policy that separates repository-local data errors from failures that prevent any trustworthy scan. Validate the required `git` executable once when at least one repository is found, distinguish a root traversal failure from skipped descendant failures, and return a nonzero execution status when no useful result can be produced. Continue returning zero for dirty, ahead, behind, fetch-failed, or individually broken repositories.

### Implementation considerations

- Preserve typed failure information long enough to classify infrastructure failures without matching diagnostic strings.
- Report one concise systemic diagnostic instead of repeating the same warning for every repository.
- Do not require `git` merely to report that an empty tree contains no repositories.
- Keep descendant permission errors and isolated malformed repositories as partial-result warnings.
- Document and test exit codes for missing Git, unreadable roots, partial scans, all-local-inspection failure, and ordinary repository states.

## P1: Codify the quality gate in one repeatable command

### Gap

The repository has strong unit, integration, race, and source-hygiene tests, but no checked-in command or continuous-integration configuration defines the complete quality gate. Contributors must infer the required combination of formatting, vet, race testing, real-Git integration testing, and building. A locally omitted command can therefore allow a regression even though a suitable check already exists.

### Approach

Add one minimal development entry point, such as `make check`, that verifies formatting, runs `go vet`, executes the uncached race-enabled test suite, and builds the command. Have a small CI workflow call that same entry point on the supported Go version, with at least Linux coverage and a macOS portability check.

### Implementation considerations

- Keep the gate dependency-free beyond Go, Git, and standard shell tooling already required by the project.
- Use Go 1.24 as the compatibility floor rather than silently testing only a newer local toolchain.
- Ensure the real-Git tests receive deterministic author identity and never require network access.
- Avoid a coverage percentage threshold until a specific risk justifies it; exercise behavior, not a number.
- Keep the workflow small and make local and hosted commands identical to prevent CI-only behavior.

## P2: Make terminal width and truncation grapheme-aware

### Gap

The renderer uses a compact rune-range approximation for terminal cell width. It handles common CJK names and combining marks, but it counts joined emoji and emoji-plus-skin-tone sequences as multiple wide glyphs even though terminals commonly display each sequence as one two-cell grapheme. Truncation can also split a joined sequence. The result is avoidable column drift or malformed-looking repository names for valid non-ASCII paths.

### Approach

Add table-driven cases for joined emoji, modifiers, combining sequences, and CJK text, then replace the rune-by-rune width and slicing helpers with a maintained grapheme-width implementation that passes those cases. Route measurement, prefix and suffix truncation, padding, wrapping, and selection indentation through the same width primitive.

### Implementation considerations

- Prefer one small, maintained dependency over expanding custom Unicode tables and grapheme logic.
- Define the expected policy for ambiguous-width characters and verify it on the supported terminals.
- Keep `--ascii` behavior unchanged; it controls application glyphs, not user path names.
- Express Unicode test data with Go escapes or deliberately extend the reviewed-rune allowlist so the source guard remains meaningful.
- Sanitize terminal controls before measuring and never split or retain a dangling zero-width sequence at a truncation boundary.

## P2: Add user and contributor documentation for shipped behavior

### Gap

There is no `README.md`. `IDEA.md` is a product design document and includes candidate flags and interactions that are not all implemented, while `--help` is the only description of the shipped command. Users therefore lack installation instructions, fetch-policy details, output-state definitions, platform support, and a clear distinction between terminal and redirected behavior.

### Approach

Create a concise README that documents building and installing from source, current flags, cached versus fetched remote state, exclusion semantics, terminal and `--once` behavior, supported platforms, exit behavior, and the development quality command. Keep `IDEA.md` as design rationale and label unimplemented behavior there as planned where needed.

### Implementation considerations

- Derive flag examples from the actual CLI and add a test or review step that catches obvious help drift.
- Do not claim downloadable releases or installation channels until they exist.
- Include one plain-ASCII output example so it remains readable everywhere and passes the source guard.
- Update the interaction section when the continuous dashboard proposal lands rather than documenting keys prematurely.
- Keep contributor guidance limited to the few commands needed to build and verify the project.
