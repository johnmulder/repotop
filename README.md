# repotop

`repotop` is a read-only terminal dashboard for the Git repositories beneath a directory tree. It discovers repositories recursively, shows local worktree and upstream state, and refreshes a live display without modifying branches or working files. The optional `git fetch --prune` refresh changes remote-tracking refs only.

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
| `--json` | Write one machine-readable finite snapshot. |
| `--no-fetch` | Disable automatic and manual remote fetches. |
| `--once` | Render one final snapshot, even when attached to a terminal. |
| `--scan-stats` | Write discovery duration and counts to standard error. |

`--fetch` and `--no-fetch` are mutually exclusive. Exclusions cannot be absolute, missing, ordinary files, or paths outside the scan root.

Examples:

```sh
repotop ~/src
repotop --no-fetch --exclude vendor ~/src
repotop --ascii --once --fetch ~/src
```

## Output modes and refresh policy

In a supported terminal, `repotop` shows local results as they arrive, refreshes local state every two seconds, and refreshes remotes every three minutes. The initial remote refresh starts after local inspection. Available keys are:

| Key | Action |
| --- | --- |
| `q` or `Ctrl-C` | Quit. |
| `r` | Rescan directories and refresh local state. |
| `f` | Fetch remotes now, unless `--no-fetch` is active. |
| Up/Down | Move the selected repository. |
| Page Up/Page Down | Move by one visible page. |

Redirected output and `--once` produce one final snapshot without terminal control sequences. Finite output uses cached remote-tracking refs by default; add `--fetch` to refresh them first.

`--json` also selects finite mode and writes exactly one object. The top level contains the absolute `root` and a `repositories` array. Repository records contain the path and branch, dirty state and worktree counts, ahead/behind and remote availability, `remote_state`, fetch timestamps and duration, and local or fetch errors. Missing timestamps are empty strings; present timestamps are UTC RFC 3339 values. Diagnostics remain on standard error, and `--fetch` retains its normal opt-in behavior.

```json
{"root":"/src","repositories":[{"path":"project","branch":"main","dirty":false,"changed":0,"staged":0,"modified":0,"conflicted":0,"untracked":0,"ahead":0,"behind":0,"has_remote":true,"has_upstream":true,"remote_state":"cached","fetching":false,"fetch_last_attempt":"","fetch_last_success":"","fetch_duration_ms":0,"error":"","error_kind":"","fetch_error":""}]}
```

Interactive repository rows use green for clean/current state, yellow for attention states such as dirty, ahead, fetching, or stale, red for behind or error state, and dim text for missing remote configuration. Text markers remain authoritative. Color is disabled when `NO_COLOR` is present, when `TERM=dumb`, and for all finite output. `--ascii` changes glyphs independently and does not disable color; it has no effect on JSON.

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
- `1`: no trustworthy finite result could be produced, such as an unreadable root, missing Git executable, render failure, or every repository failing local inspection.
- `2`: invalid command-line usage or exclusion.

## Development

Build and run the complete local quality gate with:

```sh
go build ./...
make check
```

`make check` verifies formatting, runs `go vet`, runs uncached race-enabled tests, and builds the command. See [IDEA.md](IDEA.md) for design rationale and possible future directions.
