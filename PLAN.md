# Plan: Document shipped behavior

## Goal

Give users and contributors one concise, accurate README for the command that exists today, while keeping `IDEA.md` clearly identified as design rationale and automatically detecting undocumented CLI flags.

## Steps

- [x] Verify `master` contains every completed plan, has no unmerged completed feature branch, and has no stale `PLAN.md`.
- [x] Document source installation, current options, discovery and exclusion rules, fetch policy, terminal versus finite output, state meanings, platform support, exit codes, and the development quality command.
- [x] Mark aspirational CLI examples in `IDEA.md` as design direction rather than shipped documentation.
- [x] Add a small test that derives the option list from actual help output and requires every shipped option to appear in the README.
- [x] Run source hygiene, focused documentation tests, formatting, vet, uncached race tests, repeated shuffled tests, build, and the complete quality gate.
- [ ] Remove this completed proposal from `PROPOSALS.md`, remove `PLAN.md`, commit the cleanup, merge into `master`, and delete the feature branch.

## Acceptance criteria

- The README contains no installation or release channel the repository does not provide.
- Every current flag and important runtime mode is documented from implemented behavior.
- The output example is plain ASCII and source hygiene continues to pass.
- New CLI flags fail a test until the README mentions them.
- Contributor instructions remain limited to building, testing, and running `make check`.
