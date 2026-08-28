# Proposals for `repotop`

Proposals are ordered by priority. The review that produced this list found the implementation well tested and internally cohesive, so these entries focus on gaps that affect the product contract or make regressions harder to detect.

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
