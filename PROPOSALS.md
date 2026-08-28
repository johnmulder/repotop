# Proposals for `repotop`

Proposals are ordered by priority. They are grounded in the difference between the current implementation and the direction in `IDEA.md`; speculative repository-management features remain out of scope.

## P1: Add accessible status color without changing layout

### Gap

Color-coded status is part of the core product description and initial success criteria, but the renderer currently emits only ASCII or Unicode glyph palettes and never emits color. The existing `NO_COLOR` test passes because there is no color to disable. Users therefore cannot distinguish clean, attention-worthy, behind, and unavailable states as quickly as the design intends.

### Approach

Add a small semantic style layer for green clean/current state, yellow dirty/ahead state, red behind/error state, and dim missing-remote state. Enable ANSI styling only for an interactive terminal when `NO_COLOR` is unset and the terminal is usable. Apply styling after cell measurement and truncation so escape sequences never affect grapheme width or column alignment.

### Implementation considerations

- Keep every textual marker authoritative; color must never be the only status signal.
- Treat `--ascii` and color as independent choices so ASCII glyphs may still be colored.
- Honor `NO_COLOR`, redirected output, and a dumb terminal without adding a color dependency.
- Never pass repository-controlled text through an ANSI-generating path, and retain the existing control-character sanitization.
- Test that stripping ANSI from colored output exactly matches the existing snapshot and that all no-color modes remain byte-for-byte stable.

## P2: Add a machine-readable finite snapshot

### Gap

Redirected output is deterministic but still a width-dependent human table. Scripts cannot reliably consume repository, worktree, upstream, freshness, and error state without parsing presentation text. `IDEA.md` identifies JSON or line-oriented output as a possible future feature, and the existing normalized snapshot model already contains the necessary data.

### Approach

Add a `--json` option that selects finite mode and writes one JSON document using the standard library. Serialize an explicit public response shape rather than the renderer's internal structs, while reusing the existing discovery, optional fetch, warning, and exit-status pipeline.

### Implementation considerations

- Emit exactly one JSON value on standard output and keep diagnostics on standard error.
- Preserve the meanings of `--fetch`, `--no-fetch`, exclusions, partial repository failures, and exit statuses.
- Use stable field names, RFC 3339 timestamps, and explicit empty or unavailable states.
- Escape repository-controlled strings through `encoding/json`; never include ANSI or display glyphs.
- Start with one document; defer NDJSON, streaming, and schema negotiation until a concrete consumer needs them.

## P2: Make the remote refresh interval configurable

### Gap

Interactive local refresh is fixed at two seconds and remote refresh at three minutes. The local default is inexpensive, but one remote schedule cannot fit both large hosted-repository trees and fast local or filesystem remotes. The only current alternative is `--no-fetch`, which disables remote truth entirely, while `IDEA.md` already names a fetch-interval option as a likely small extension.

### Approach

Add one `--fetch-interval DURATION` flag parsed with `time.ParseDuration`, defaulting to the current three minutes. Pass it to the interactive ticker and freshness renderer. Keep the local interval and worker counts internal until evidence shows they also need configuration.

### Implementation considerations

- Require a positive duration; continue using `--no-fetch` as the explicit disabled state.
- Document that the option controls interactive scheduling and the stale threshold, while finite mode still fetches only with `--fetch`.
- Preserve the initial local-first render followed by asynchronous remote refresh.
- Inject or reuse test tick channels so interval tests do not sleep or depend on wall-clock timing.
- Let the existing README option-drift test require documentation when the flag is added.

## P2: Review non-ASCII text in every project text format

### Gap

The reviewed-character test scans `.go`, `.md`, and `.mod` files only. The project now also contains `Makefile`, `.github/workflows/ci.yml`, and `go.sum`, so a new non-ASCII character in build or CI configuration can bypass the guard. This weakens the stated policy precisely as new file types are added.

### Approach

Extend the source walk to classify and scan project text files regardless of extension, while excluding `.git`, generated binaries, and clearly binary content. Keep the per-path reviewed-rune allowlist so intentional Unicode remains explicit and searchable.

### Implementation considerations

- Do not depend on current-checkout Git metadata; tests should also work from a source archive.
- Use a small deterministic binary/text rule, such as valid UTF-8 without NUL bytes, plus explicit exclusions for generated artifacts.
- Report the path, byte offset, and code point as the current test does.
- Add a fixture covering a non-ASCII YAML or extensionless file so the broader guard cannot silently regress.
- Avoid banning user data or test repositories created outside the source tree; this check is only for checked-in project text.

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
