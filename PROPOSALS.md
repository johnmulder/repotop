# Proposals for `repotop`

Proposals are ordered by priority. They are grounded in the difference between the current implementation and the direction in `IDEA.md`; speculative repository-management features remain out of scope.

## P1: Reset the cursor column between interactive rows

### Gap

Interactive mode switches the terminal to raw mode, which disables output post-processing on supported Unix terminals. The dashboard emits bare line feeds, so terminals that honor the raw setting move the cursor down without returning it to column zero. Rows then appear progressively offset even though the renderer's cell-width calculations are correct. Buffer-based renderer tests do not exercise this terminal behavior.

### Approach

Translate the renderer's line feeds to carriage-return/line-feed pairs only at the interactive dashboard write boundary. Keep the pure renderer and every finite output mode on ordinary line feeds.

### Implementation considerations

- Perform the translation after rendering so width, truncation, and snapshot deduplication remain unchanged.
- Do not alter repository text or replace line endings in finite text or JSON output.
- Add a dashboard-level regression test that rejects bare line feeds and confirms rows begin with CRLF while raw-terminal control sequences remain intact.
- Preserve the existing terminal restoration and resize behavior.

## P3: Produce versioned macOS and Linux release artifacts

### Gap

The deployment goal in `IDEA.md` is to download one binary, put it on `PATH`, and run it, but the repository has no configured remote, release workflow, downloadable artifacts, checksums, or `--version` output. The README can therefore offer only source builds.

### Approach

After a release host is chosen, add a tag-triggered workflow that runs `make check`, injects the tag into a small `--version` value, builds macOS and Linux archives for amd64 and arm64, and publishes checksums with the artifacts. Use the Go toolchain and simple archive commands before considering a release framework.

### Implementation considerations

- Do not add or advertise a download URL until the repository has an authoritative remote and a successful release.
- Keep untagged development builds identifiable, for example as `devel`, without requiring Git at runtime.
- Build only the currently supported macOS and Linux targets and smoke-test `--help` and `--version` for each artifact.
- Publish only from protected version tags after the normal CI gate passes; ordinary pushes must remain read-only checks.
- Defer signing, package managers, and additional architectures until real distribution demand justifies them.
