# repotop

This file records product design and future direction. See `README.md` for the behavior, options, and controls implemented by the current command; candidate behavior below is not a shipped contract.

## Idea

`repotop` is an `htop`-like terminal dashboard for Git repositories beneath a directory tree.

Run it against a directory and it recursively discovers Git repositories, then continuously presents their state in a compact, color-coded terminal UI.

The goal is not to manage repositories. The goal is to answer, at a glance:

- What repositories are here?
- What branch is each repository on?
- Which repositories have uncommitted work?
- Which repositories are ahead of their upstream?
- Which repositories are behind their upstream?
- Which repositories have no upstream or remote?
- Which directories that were expected to be repositories are not Git repositories?

`repotop` should feel lightweight, immediate, and unsurprising: a read-only status monitor for a tree of repositories.

## Example

```text
$ repotop ~/src

 repotop — ~/src                                      fetched 34s ago

 REPOSITORY                 BRANCH             WORKTREE   REMOTE
 ────────────────────────────────────────────────────────────────
 proseprobe                 main               clean      ↑2
 last                       main               M1         ✓
 py-doc-extract             feature/json       clean      ↓3
 makegraph                  main               M2 ?1      ↑1 ↓1
 experiments/widget         main               clean      no upstream
 archive/old-project        main               clean      no remote

 6 repos   3 clean   3 dirty   2 ahead   2 behind

 q quit    r rescan    f fetch
```

Colors should communicate status without being required to understand it:

- green: clean and synchronized
- yellow: local changes, ahead, or other attention-worthy local state
- red: behind remote or otherwise likely to require action
- dim/neutral: no remote, no upstream, or unavailable information

The textual symbols remain authoritative so the display is usable without color.

## Scope

### In scope

- Recursively discover Git repositories beneath a directory.
- Show repository path relative to the selected root.
- Show current branch.
- Distinguish detached `HEAD`.
- Show dirty/clean working-tree state.
- Show useful counts for modified and untracked files.
- Show commits ahead of upstream.
- Show commits behind upstream.
- Detect missing remotes or missing upstream tracking branches.
- Periodically refresh inexpensive local state.
- Periodically refresh remote-tracking information with `git fetch`.
- Allow an immediate manual remote refresh.
- Color-code status.
- Sort repositories so interesting/problematic states are easy to see.
- Continue operating when individual repositories are inaccessible, broken, or slow.
- Work well on macOS and Linux.

### Explicitly out of scope

`repotop` is not a Git client or repository manager.

It should not:

- clone repositories
- create or delete repositories
- pull
- push
- merge
- rebase
- commit
- switch branches
- edit files
- manage GitHub/GitLab issues or pull requests
- maintain a repository database
- require GitHub/GitLab APIs
- require accounts, tokens, or hosted services

If an operation changes repository history or working-tree contents, it probably does not belong in `repotop`.

## Interface

This section captures design direction. The complete shipped syntax and option list are maintained in `README.md`; candidate options below may not exist yet.

The primary interface should remain:

```text
repotop [directory]
```

If `directory` is omitted, use the current directory.

Options should be kept deliberately small. Reasonable candidates are:

```text
repotop [directory]
repotop --no-fetch [directory]
repotop --fetch-interval SECONDS [directory]
repotop --help
repotop --version
```

Avoid configuration files unless a compelling real-world requirement appears.

Defaults should be good enough that most users simply run:

```text
repotop ~/src
```

## Interaction

Keep keyboard interaction minimal:

```text
q       quit
r       rescan repositories and refresh local state
f       fetch remotes immediately
↑/↓     move selection or scroll
PgUp/
PgDn    scroll larger repository sets
```

Additional controls should only be added when they solve an obvious recurring problem.

A help overlay via `?` may be useful if the key set grows, but the preferred outcome is that the interface remains simple enough not to need one.

## Refresh Model

Local and remote state should refresh independently.

### Local refresh

Local state is inexpensive and can be refreshed frequently, perhaps every 1–2 seconds:

- branch
- detached `HEAD`
- working-tree changes
- untracked files
- ahead/behind counts against the currently cached upstream ref

### Remote refresh

Remote truth requires network I/O and should refresh much less frequently, perhaps every 2–5 minutes.

Remote refresh should run asynchronously so a slow or unreachable remote never freezes the UI.

Conceptually:

```text
local scan      every 2 seconds
remote fetch    every 180 seconds
manual fetch    f
```

After a fetch completes, ahead/behind state should update immediately.

The UI should display how stale the remote information is:

```text
fetched 34s ago
```

or, for a repository-specific failure:

```text
fetch failed
```

## Repository Discovery

Walk the requested directory recursively.

A directory containing `.git` is a repository root. `.git` may be either a directory or a file, since Git worktrees and submodules can use a `.git` file.

Once a repository root is discovered, do not recursively inspect its contents for arbitrary nested repositories unless there is a clear reason to support that behavior. This avoids traversing `.git`, vendored trees, and repository internals unnecessarily.

Discovery should:

- ignore `.git` internals
- tolerate permission errors
- avoid following symlink loops
- behave predictably around Git worktrees and submodules
- avoid requiring an index or persistent cache

A rescan should be cheap enough to perform manually with `r`.

## Git Status Model

For each repository, collect a small normalized record:

```text
path
branch
detached
changed
staged
modified
conflicted
untracked
upstream
ahead
behind
remote_state
fetch_state
last_fetch
error
```

Prefer Git's stable plumbing or machine-readable interfaces rather than parsing human-oriented output.

Likely commands include:

```bash
git status --porcelain=v2 --branch -z
git rev-parse --show-toplevel
git fetch --prune
```

Where useful, combine queries to minimize subprocess creation.

Git itself should remain the source of truth.

Local counts have exact path-based semantics derived from porcelain v2 records:

- `changed`: non-conflicted tracked paths represented by ordinary (`1`) or rename/copy (`2`) records; each path counts once even when both index and worktree state changed
- `staged`: non-conflicted records whose index (`X`) status is not `.`
- `modified`: non-conflicted records whose worktree (`Y`) status is not `.`
- `conflicted`: unmerged (`u`) records, kept separate from other tracked changes
- `untracked`: untracked (`?`) records

A path may count as both staged and modified, but it still counts only once as changed. Ignored (`!`) records do not affect the model. The compact `M` display uses `changed`, while conflicts use a separate `C` display.

## Sorting

The default ordering should make repositories needing attention visually prominent.

A reasonable priority is:

1. errors
2. behind remote
3. diverged (ahead and behind)
4. dirty working tree
5. ahead remote
6. missing upstream/remote
7. clean and synchronized

Within a category, sort alphabetically by relative path.

If this feels too surprising in practice, simple alphabetical sorting is preferable to adding sorting configuration.

## Status Semantics

The display should distinguish at least these cases:

```text
✓            clean and synchronized
↑2           two commits ahead
↓3           three commits behind
↑1 ↓2        diverged
M2           two non-conflicted tracked paths changed
C1           one conflicted path
?3           three untracked files
no upstream  branch does not track an upstream
no remote    repository has no usable remote
detached     HEAD is detached
error        Git state could not be determined
```

Avoid ambiguous icon-only status. Symbols should augment concise text rather than replace it.

## "Not a Git Repo"

Recursive discovery naturally finds repositories rather than non-repositories, so displaying "not a git repo" needs a precise interpretation.

The simplest behavior is:

- if the root passed to `repotop` contains repositories, display discovered repositories only
- if the root itself is not a repository and contains no repositories, clearly report:

```text
no Git repositories found beneath ~/some/path
```

Do not attempt to display every ordinary directory as "not a git repo"; that would make recursive scans mostly noise.

A future explicit-path mode could report individual requested paths as `not a git repo`, but it is unnecessary for the initial version.

## Architecture

Keep the architecture small.

A likely decomposition:

```text
scanner
    discovers repository roots

git
    queries local Git state
    performs fetches

model
    normalized repository status

scheduler
    coordinates local scans and slower remote refreshes

ui
    renders terminal state and handles keys
```

The core repository inspection logic should not depend on the TUI. This makes it straightforward to test and leaves open the possibility of a future noninteractive output mode without designing around it now.

The scheduler's coordinator is the sole owner of the repository map. Inspection workers emit repository identity, scan generation, and normalized status through a channel; they never mutate shared state. The coordinator ignores obsolete generations, removes missing repositories only after a matching scan completes, and publishes copied snapshots so consumers cannot mutate its state.

Remote refresh uses a fixed four-worker batch after local inspection. Repository roots are deduplicated before queueing, fetch failures are stored independently from local status, and successful fetches trigger local reinspection for current ahead/behind counts. Fetch commands are limited to `git fetch --prune`, disable terminal prompts, and inherit cancellation plus an internal deadline. `--no-fetch` skips the batch completely.

Rendering is a pure transformation of a sorted snapshot, terminal width, and selected repository path. Wide, compact, and narrow ASCII layouts reserve status space and middle-truncate repository paths. Real terminals repaint the first result immediately, coalesce rapid updates, redraw the latest snapshot after resize, and keep selection attached to repository identity; redirected output remains a single final snapshot without terminal controls.

## Dependencies

Minimize external dependencies.

The only required external executable should be:

```text
git
```

Implementation-language libraries should be chosen conservatively.

A compiled implementation in Go or Rust would provide a convenient single executable. A Python implementation could be attractive for development speed but would either depend on the user's Python environment or require packaging into a standalone executable.

For this tool, deployment simplicity matters:

```text
download binary
put it on PATH
run repotop
```

That should influence the implementation choice.

## Concurrency

Repository inspection and remote fetching should be concurrent, but bounded.

A directory tree may contain dozens or hundreds of repositories. Launching an unconstrained `git fetch` for every repository could overwhelm:

- network connections
- credential helpers
- remote services
- the local machine

Use a small worker pool for network operations.

For example:

```text
local status workers: 8
fetch workers:        4
```

Exact values should be implementation details rather than user configuration unless experience proves otherwise.

The UI must never wait synchronously for all repositories to finish fetching.

## Failure Behavior

Failures are expected and should be represented as state rather than treated as fatal errors.

Examples:

- remote host unavailable
- authentication required
- deleted remote repository
- permission error
- malformed Git repository
- repository disappears during a scan
- Git command times out
- fetch takes unusually long

One broken repository should not affect monitoring of the others.

Errors should be concise in the main table, with fuller detail available through selection or a status line if necessary.

### Process exit status

Exit status describes whether the command produced a trustworthy result, not whether repositories need attention:

- `0`: a valid empty or repository result, including dirty, ahead, behind, fetch-failed, or partially broken collections; normal interactive exit also returns zero so transient repository failures can recover while monitoring
- `1`: execution could not produce a trustworthy finite result, such as an unreadable scan root, unavailable Git executable, render failure, or every discovered repository failing local inspection
- `2`: invalid command-line usage

When every repository fails local inspection, finite output still renders the error rows before returning status 1. A repeated systemic failure, such as a missing Git executable, should produce one concise diagnostic rather than one warning per repository.

## Performance

Optimize for repository collections typical of a developer workstation:

```text
10–200 repositories
```

Initial local results should appear progressively rather than waiting for every repository.

Network fetches should happen after repositories become visible.

Avoid doing expensive work merely to make the dashboard exact to the second. `repotop` is an awareness tool, not a synchronization protocol.

## Design Principles

### Read-only

The safest and clearest boundary is that `repotop` observes repositories but does not modify their contents or history.

`git fetch` is the intentional exception: it updates remote-tracking refs so remote status can be determined, without changing the working tree or local branch history.

### Git-native

Do not reproduce Git behavior in application code when Git can answer the question directly.

### No service dependency

The tool should work equally well with GitHub, GitLab, Bitbucket, self-hosted Git, SSH remotes, filesystem remotes, and other normal Git configurations.

### Progressive results

Local information should appear quickly. Remote information can arrive later.

### Quiet defaults

A user should not need to configure the tool before it becomes useful.

### Information over action

When tempted to add `pull`, `push`, `checkout`, repository management, or hosted-service integration, resist it.

The value of `repotop` is that it remains a dashboard.

## Possible Future Features

Only consider these after the basic tool has proven useful:

- filtering/search
- alternate sort order
- JSON or line-oriented noninteractive output
- fetch only selected repository
- optional recognition of bare repositories
- shell command to print the selected repository path

These should not be part of the initial design unless implementation experience makes them obviously necessary.

## Initial Success Criteria

Version 0.1 is successful if:

```bash
repotop ~/src
```

can recursively discover repositories and present a responsive terminal dashboard that reliably answers:

```text
Where are my repositories?
What branch is each on?
Which have local changes?
Which are ahead?
Which are behind?
Which cannot currently be compared to a remote?
```

with sensible color coding and asynchronous remote refresh.

Everything beyond that is secondary.

## One-Sentence Definition

> `repotop` is a lightweight, read-only, `htop`-style terminal dashboard for the local and remote status of every Git repository beneath a directory.
