# Proposals for `repotop`

Proposals are ordered by priority. They are grounded in the difference between the current implementation and the direction in `IDEA.md`; speculative repository-management features remain out of scope.

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
