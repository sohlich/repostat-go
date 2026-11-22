# repostat

Prints a status table for git repositories inside a folder. Go port of `repo_status_check.py`.

## Requirements

- Go 1.24 or newer (uses `strings.SplitSeq`; the go.mod directive is set by your installed toolchain)
- `git` on your `PATH` (repostat shells out to it)

There are no third-party Go modules. Only the standard library is used, so there is nothing to download.

## Build

```sh
go build -o repostat .
# or install to $GOPATH/bin
go install .
```

## Usage

```sh
repostat [--fetch] [-r] [folder]
```

| Option | Description |
|--------|-------------|
| `folder` | Directory to scan (default: current directory) |
| `--fetch` | Run `git fetch` in each repo first so ahead/behind is up to date |
| `-r`, `--recursive` | Scan recursively; stops descending at a repo root and skips hidden directories |

Without `-r`, only direct children of `folder` that contain `.git` are checked.

## Output

One row per repo:

| Column | Meaning |
|--------|---------|
| Repo | Path relative to the scanned folder |
| Branch | Current branch, or `(detached <sha>)` |
| State | `clean` or `dirty` |
| Chg | Tracked files with changes |
| Untr | Untracked files |
| Ahead/Behind | `+ahead/-behind` versus upstream, or `no upstream` |
| Stash | Number of stash entries |
| Last commit | Date of the latest commit |
| Subject | Latest commit subject, truncated to 40 characters |

A summary line (`N repos, M dirty`) follows the table. Repos are inspected concurrently, 8 at a time.
