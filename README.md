# repotop

`repotop` is an observational terminal dashboard for the Git repositories beneath a directory tree. It discovers repositories recursively, shows local worktree and upstream state, and refreshes a live display. It never directly edits checked-out files, creates commits, or invokes merge, rebase, pull, push, or checkout. When enabled, `git fetch --prune` updates repository metadata according to each repository's Git configuration, commonly remote-tracking refs and `FETCH_HEAD`.

## Requirements

- macOS or Linux
- Go 1.24 or newer to build from source
- Git available on `PATH`

## Build and install

From a checked-out source tree:

```sh
go build -o repotop .
./repotop --help
```

To install into your configured Go binary directory:

```sh
go install .
```

`go install` writes `repotop` to `GOBIN` when configured, otherwise to the `bin` directory under `GOPATH` (normally `$HOME/go/bin`). Put that directory on `PATH`, then run `repotop --help`.

## Usage

```text
repotop [options] [directory]
```

The directory defaults to the current directory.

| Option | Behavior |
| --- | --- |
| `--ascii` | Use plain ASCII application symbols. Repository names remain Unicode. |
| `--exclude PATH` | Skip an existing root-relative directory and its subtree. Repeat for more exclusions. |
| `--fetch` | Fetch remotes before a finite snapshot. Interactive mode already refreshes remotes by default. |
| `--fetch-interval DURATION` | Set the interactive remote-refresh schedule and freshness threshold. |
| `--json` | Write one machine-readable finite snapshot. |
| `--no-fetch` | Disable automatic and manual remote fetches. |
| `--once` | Render one final snapshot, even when attached to a terminal. |
| `--scan-stats` | Write discovery duration and counts to standard error. |
| `--version` | Print the build version and exit. |

`--fetch` and `--no-fetch` are mutually exclusive. `--fetch-interval` accepts a positive Go duration such as `30s`, `5m`, or `1h`. Exclusions cannot be absolute, missing, ordinary files, or paths outside the scan root.

Examples:

```sh
repotop ~/src
repotop --no-fetch --exclude vendor ~/src
repotop --fetch-interval 30s ~/src
repotop --ascii --once --fetch ~/src
repotop --version
```

Source and other untagged builds report `repotop devel`. Release builds report the version tag injected at build time.

## Output modes and refresh policy

In a supported terminal, `repotop` shows local results as they arrive, refreshes local state every two seconds, and refreshes remotes every three minutes by default. The initial remote refresh starts after local inspection. `--fetch-interval` changes the remote schedule and the threshold after which prior remote data is shown as stale. Available keys are:

| Key | Action |
| --- | --- |
| `q` or `Ctrl-C` | Quit. |
| `r` | Rescan directories and refresh local state. |
| `f` | Fetch remotes now, unless `--no-fetch` is active. |
| Up/Down | Move the selected repository. |
| Page Up/Page Down | Move by one visible page. |

Redirected output and `--once` produce one final snapshot without terminal control sequences. Finite output uses cached remote-tracking refs by default; add `--fetch` to refresh them first. In finite mode, `--fetch-interval` changes only the freshness threshold and never triggers a fetch by itself.

`--json` also selects finite mode and writes exactly one object. The top level contains the absolute `root` and a `repositories` array. Repository `path` values are slash-separated and relative to `root`; `.` identifies a repository at the root itself. Records contain the branch, dirty state and worktree counts, ahead/behind and remote availability, `remote_state`, fetch timestamps and duration, and local or fetch errors. `fetch_duration_ms` is an integer number of milliseconds. Missing timestamps are empty strings; present timestamps are UTC RFC 3339 values. `error_kind` is empty when there is no classified local Git failure and otherwise identifies failures such as `timeout`, `permission`, or `not a repository`. Diagnostics remain on standard error, and `--fetch` retains its normal opt-in behavior.

```json
{"root":"/src","repositories":[{"path":"project","branch":"main","dirty":false,"changed":0,"staged":0,"modified":0,"conflicted":0,"untracked":0,"ahead":0,"behind":0,"has_remote":true,"has_upstream":true,"remote_state":"cached","fetching":false,"fetch_last_attempt":"","fetch_last_success":"","fetch_duration_ms":0,"error":"","error_kind":"","fetch_error":""}]}
```

Interactive row colors follow severity: local or fetch errors and behind state are red; otherwise dirty, ahead, fetching, or stale state is yellow; otherwise missing remote configuration is dim; otherwise current state is green. Clean cached rows use the terminal's default color. Text markers remain authoritative. Color is disabled when `NO_COLOR` is present, when `TERM=dumb`, and for all finite output. Unicode application glyphs are selected when the locale reports UTF-8; otherwise the command falls back to ASCII. `--ascii` forces ASCII glyphs without changing repository names or disabling color, and it has no effect on JSON.

## Reading the display

A plain-ASCII snapshot looks like this:

```text
REPOSITORY                        BRANCH            WORKTREE      REMOTE
--------------------------------  ----------------  ------------  --------------
project-api                       topic             M2 ?1         +1 cached
project-web                       main              clean         current

2 repos (1 clean, 1 dirty, 1 ahead, 0 behind, 0 errors)
```

Worktree fields use `clean`, `M<n>` for changed tracked paths, `C<n>` for conflicts, and `?<n>` for untracked paths. Remote fields use `+<n>` for commits ahead and `-<n>` for commits behind.

Remote freshness states are:

- `cached`: status comes from existing local remote-tracking refs without a successful fetch in this run.
- `current`: the latest fetch succeeded.
- `fetching`: a refresh is running.
- `stale`: prior successful data remains after a failed or overdue refresh.
- `failed`: no fetch has succeeded in this run.
- `no upstream` or `no remote`: the repository cannot be compared with a tracked remote branch.
- `error`: local Git state could not be determined.

Fetch failures and individual repository errors are reported as data and warnings so other repositories remain visible.

## Exit status

- `0`: a valid empty or repository result, including dirty, ahead, behind, fetch-failed, or partially broken collections; also a normal interactive exit.
- `1`: execution failed, such as from an unreadable root, missing Git executable, terminal or render failure, or every repository failing local inspection in finite mode.
- `2`: invalid command-line usage or exclusion.

## Development

Build and run the complete local quality gate with:

```sh
go build ./...
make check
```

`make check` verifies formatting, runs `go vet`, runs uncached race-enabled tests, and builds the command. See [IDEA.md](IDEA.md) for design rationale and possible future directions.

## Releases

Maintainers can build and smoke-test an archive for the current operating system and architecture with:

```sh
make release RELEASE_VERSION=v1.2.3
```

When the repository is hosted on GitHub, a protected `vMAJOR.MINOR.PATCH` tag triggers the release workflow. It runs the normal quality gate, builds and executes native macOS and Linux binaries for amd64 and arm64, publishes four `.tar.gz` archives, and includes a SHA-256 checksum manifest. Configure protection for the `release` environment and version tags before publishing. No download URL is documented until a hosted release succeeds.
