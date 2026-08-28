# Proposals for `repotop`

Proposals are ordered by priority. The review that produced this list found the implementation well tested and internally cohesive, so these entries focus on gaps that affect the product contract or make regressions harder to detect.

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
