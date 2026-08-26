# Proposals for `repotop`

These proposals turn the intent in `IDEA.md` into an implementation path while preserving the project's main constraint: `repotop` is a small, read-only dashboard, not a repository manager.

Priorities mean:

- **P0:** required for a trustworthy version 0.1
- **P1:** a high-value improvement once the basic dashboard works
- **P2:** defer until actual use demonstrates the need

## 1. Add scan exclusions only when traversal data justifies them

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

Proposal 1 should remain deferred until measured traversal problems appear.
