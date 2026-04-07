# git-master-tool

A CLI toolkit to supercharge your git workflow. Run as `gitmt <command>`.

## Commands

| Command   | Description                                                   |
|-----------|---------------------------------------------------------------|
| `compare` | Compare two branches and surface unmerged commits             |
| `scan`    | Group pending commits into safe cherry-pick batches           |
| `pick`    | Interactively cherry-pick a batch or individual commit        |
| `search`  | Check if a commit message already exists to ensure uniqueness |
| `tagdiff` | List commits between two tags                                 |
| `upgrade` | Upgrade gitmt to the latest release                           |

---

## compare

Identify which commits from a source branch have not yet been cherry-picked into a target branch.

### How it works

`gitmt compare` finds the common ancestor of both branches, then lists every commit in the source branch since that point. Each commit is matched against the target branch **by commit message** — since cherry-picking preserves the original message, a match means the commit has already been applied.

### Installation

#### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/anthonygacis/git-master-tool/main/scripts/install.sh | bash
```

#### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/anthonygacis/git-master-tool/main/scripts/install.ps1 | iex
```

The installer automatically:

- Detects your OS and architecture
- Downloads the latest release binary from GitHub
- Installs to `/usr/local/bin` (macOS/Linux) or `%LOCALAPPDATA%\Programs\gitmt` (Windows)
- Adds the install directory to your PATH if needed

On macOS a universal binary is downloaded (runs natively on both Intel and Apple Silicon).

#### Build from source

Requires Go 1.21+.

```bash
git clone https://github.com/anthonygacis/git-master-tool.git
cd git-master-tool
make install
```

`make install` builds a universal macOS binary and copies it to `/usr/local/bin`. See [Makefile](Makefile) for all available targets (`release`, `build-linux-amd64`, `build-windows`, etc.).

### Usage

```text
gitmt compare [flags] [source target]
```

All options can be provided as flags, set in `.gitmt.yml`, or both — **flags always take precedence over the config file**.

### Flags

| Flag                    | Description                                        | Default     |
|-------------------------|----------------------------------------------------|-------------|
| `-source`               | Source branch (commits to cherry-pick from)        |             |
| `-target`               | Target branch (branch to cherry-pick into)         |             |
| `-show`                 | `pending` or `all`                                 | `all`       |
| `-show-author`          | Show commit author: `true` or `false`              | `true`      |
| `-show-date`            | Show commit date and time: `true` or `false`       | `true`      |
| `-format`               | `default`, `table`, `json`, or `csv`               | `default`   |
| `-prefixes`             | Comma-separated Jira prefixes (e.g. `XS,XI`)       |             |
| `-show-ticket-authors`  | Show authors in ticket summary: `true` or `false`  | `true`      |

### Examples

```bash
# positional args (shorthand)
gitmt compare develop master

# with flags
gitmt compare -show pending -format table develop master

# use config file for source/target, override format
gitmt compare -format json

# hide author, show only pending
gitmt compare -show-author=false -show pending develop master

# ticket summary grouped by Jira prefix
gitmt compare -prefixes XS,XI develop master
```

### Output formats

#### default

Colored list grouped by status.

```text
Branch comparison
  source : develop
  target : master
  base   : a1a2198

 Pending — not yet in master:
  1. 20b048a  2026-03-01 14:22  (Jane Doe)  feat: add feature C
  2. dacf154  2026-02-28 09:05  (Jane Doe)  feat: add feature B

 Already applied — commit message found in master:
  1. eed2e8d  2026-02-20 11:30  (Jane Doe)  feat: add feature A

Summary
  pending    2 commit(s) need cherry-picking into master
  applied    1 commit(s) already applied
```

#### table

Box-drawn table with dynamic column widths.

```text
┌─────────┬─────────┬──────────────────┬──────────┬─────────────────────┐
│ STATUS  │ HASH    │ DATE             │ AUTHOR   │ MESSAGE             │
├─────────┼─────────┼──────────────────┼──────────┼─────────────────────┤
│ pending │ 20b048a │ 2026-03-01 14:22 │ Jane Doe │ feat: add feature C │
│ pending │ dacf154 │ 2026-02-28 09:05 │ Jane Doe │ feat: add feature B │
│ applied │ eed2e8d │ 2026-02-20 11:30 │ Jane Doe │ feat: add feature A │
└─────────┴─────────┴──────────────────┴──────────┴─────────────────────┘

  2 pending   1 applied
```

#### csv

Standard CSV output — useful for importing into spreadsheets or further shell processing.

```csv
status,hash,date,author,message
pending,20b048a,2026-03-01 14:22,Jane Doe,feat: add feature C
pending,dacf154,2026-02-28 09:05,Jane Doe,feat: add feature B
applied,eed2e8d,2026-02-20 11:30,Jane Doe,feat: add feature A
```

Fields with commas or quotes are automatically escaped per RFC 4180.

#### json

Machine-readable output, useful for piping into other tools.

```json
{
  "source": "develop",
  "target": "master",
  "base": "a1a2198",
  "commits": [
    { "hash": "20b048a", "date": "2026-03-01 14:22", "author": "Jane Doe", "message": "feat: add feature C", "status": "pending" },
    { "hash": "dacf154", "date": "2026-02-28 09:05", "author": "Jane Doe", "message": "feat: add feature B", "status": "pending" },
    { "hash": "eed2e8d", "date": "2026-02-20 11:30", "author": "Jane Doe", "message": "feat: add feature A", "status": "applied" }
  ],
  "summary": { "pending": 2, "applied": 1 }
}
```

### Ticket summary

When `-prefixes` is set, `gitmt compare` scans each commit message for Jira ticket IDs and groups them by prefix. A single commit can match multiple tickets (e.g. `XI-002 XS-001 update auth flow` counts toward both). The summary respects the `-show` filter — only commits visible in the output are counted.

```text
Ticket Summary

XS
  XS-001        2 commits   Jane Doe
  total         2 commits

XI
  XI-002        2 commits   Jane Doe
  total         2 commits
```

In JSON output the ticket summary appears as a `ticket_summary` array with a `total` per prefix group.

### Config file

Place a `.gitmt.yml` file in the root of your repository to set defaults for `gitmt compare`.

```yaml
source: develop
target: master
show: all                  # "pending" or "all" (default: "all")
show_author: true          # show commit author (default: true)
show_date: true            # show commit date and time (default: true)
format: default            # "default", "table", "json", or "csv" (default: "default")
prefixes:                  # Jira prefixes to scan for ticket summary (default: none)
  - XS
  - XI
show_ticket_authors: true  # show authors in ticket summary (default: true)
```

The `show_author`, `show_date` fields are shared across `compare`, `scan`, and `pick`.

| Field                 | Description                                                                                        |
|-----------------------|----------------------------------------------------------------------------------------------------|
| `source`              | Branch containing the commits you are cherry-picking from                                          |
| `target`              | Branch you are cherry-picking into                                                                 |
| `show`                | `all` shows pending + already applied. `pending` shows only commits that still need cherry-picking |
| `show_author`         | Show commit author next to each entry. Defaults to `true`                                          |
| `show_date`           | Show commit date and time next to each entry. Defaults to `true`                                   |
| `format`              | Output style: `default`, `table`, `json`, or `csv`                                                 |
| `prefixes`            | List of Jira ticket prefixes to scan for and group in the ticket summary                           |
| `show_ticket_authors` | Show authors per ticket in the ticket summary. Defaults to `true`                                  |

---

## scan

Analyzes pending commits and groups them into batches that must be cherry-picked together. Two commits end up in the same batch when they touch at least one common file — cherry-picking them separately would risk a conflict.

`gitmt scan` runs the same branch comparison as `compare` to find pending commits, then fetches the file list for each one (`git diff-tree`). It uses a union-find algorithm to cluster commits that share files into a single batch. Commits with no file overlap with any other pending commit are marked independent and can be cherry-picked on their own.

```text
gitmt scan [flags] [source target]
```

### scan flags

| Flag            | Description                              | Default |
|-----------------|------------------------------------------|---------|
| `-source`       | Source branch                            |         |
| `-target`       | Target branch                            |         |
| `-show-author`  | Show commit author: `true\|false`        | `true`  |
| `-show-date`    | Show commit date and time: `true\|false` | `true`  |

Reads `-source`, `-target`, `show_author`, and `show_date` from `.gitmt.yml` when present — flags take precedence.

### scan examples

```bash
# positional args
gitmt scan develop master

# hide author
gitmt scan -show-author=false develop master

# with flags
gitmt scan -source develop -target master
```

### scan output

Each batch is listed in order from newest to oldest commit. Independent commits are green; batches that must go together are yellow.

```text
Scan — cherry-pick batches
  source  : develop
  target  : master
  pending : 5 commit(s)
  batches : 3

● Batch 1  2 commits — cherry-pick together
  a1b2c3d  2026-03-01 14:22  (Jane Doe)  feat: update user auth service  (3 file(s))
  e4f5g6h  2026-03-01 09:11  (Jane Doe)  fix: auth token validation edge case  (2 file(s))
  shared:
    src/auth/service.go
    src/auth/token.go

● Batch 2  1 commit — independent
  i7j8k9l  2026-02-28 17:45  (John Smith)  feat: add dashboard summary widget  (1 file(s))

● Batch 3  2 commits — cherry-pick together
  m1n2o3p  2026-02-27 10:30  (Jane Doe)  refactor: extract payment helpers  (4 file(s))
  q4r5s6t  2026-02-26 16:02  (Jane Doe)  fix: payment rounding error  (2 file(s))
  shared:
    src/payment/helpers.go
```

When all pending commits are independent the output shows one green batch per commit. When all commits are applied the scan exits with a message instead.

```text
All commits already applied — nothing to scan.
```

---

## pick

Interactively cherry-picks pending commits onto the current branch. Only unapplied commits are shown. Navigate with arrow keys, toggle selections with space, and confirm with enter. Supports multi-select for both modes.

```text
gitmt pick [flags] [source target]
```

### pick flags

| Flag            | Description                              | Default |
|-----------------|------------------------------------------|---------|
| `-source`       | Source branch                            |         |
| `-target`       | Target branch                            |         |
| `-show-author`  | Show commit author: `true\|false`        | `true`  |
| `-show-date`    | Show commit date and time: `true\|false` | `true`  |

### Controls

| Key         | Action                          |
|-------------|---------------------------------|
| `↑` / `↓`   | Move cursor                     |
| `space`     | Toggle selection (multi-select) |
| `enter`     | Confirm / select                |
| `q` / `esc` | Quit                            |

### Batch mode

Shows commits grouped by shared files (same logic as `scan`). Multiple batches can be selected. All selected commits are applied oldest-first to avoid conflicts.

### Individual mode

Lists every pending commit. Commits belonging to a multi-commit batch are flagged with `⚠ batch N` so the user knows cherry-picking them alone may cause a conflict. Multiple commits can be selected.

### pick output

**Step 1 — mode selection:**

```text
Cherry-pick   develop → master
  5 pending commit(s)   3 batch(es)

 ▶ Batch      cherry-pick related commits together
   Individual  pick any commits regardless of grouping

  ↑↓ move   [enter] select   [q] quit
```

**Step 2 — batch multi-select:**

```text
Select batches to cherry-pick
  3 batch(es)

 ▶ [✓] Batch 1  2 commits — cherry-pick together
        a1b2c3d  2026-03-01 14:22  (Jane Doe)  feat: update user auth service
        e4f5g6h  2026-03-01 09:11  (Jane Doe)  fix: auth token validation edge case
        shared: src/auth/service.go, src/auth/token.go
   [ ] Batch 2  1 commit — independent
        i7j8k9l  2026-02-28 17:45  (John Smith)  feat: add dashboard summary widget
   [✓] Batch 3  2 commits — cherry-pick together
        m1n2o3p  2026-02-27 10:30  (Jane Doe)  refactor: extract payment helpers
        q4r5s6t  2026-02-26 16:02  (Jane Doe)  fix: payment rounding error
        shared: src/payment/helpers.go

  ↑↓ move   [space] toggle   [enter] confirm   [q] quit
```

**Step 2 — individual multi-select:**

```text
Select commits to cherry-pick
  5 pending commit(s)

 ▶ [✓] a1b2c3d  2026-03-01 14:22  (Jane Doe)  feat: update user auth service  ⚠ batch 1
   [ ] e4f5g6h  2026-03-01 09:11  (Jane Doe)  fix: auth token validation  ⚠ batch 1
   [✓] i7j8k9l  2026-02-28 17:45  (John Smith)  feat: add dashboard summary widget
   [ ] m1n2o3p  2026-02-27 10:30  (Jane Doe)  refactor: extract payment helpers  ⚠ batch 3
   [ ] q4r5s6t  2026-02-26 16:02  (Jane Doe)  fix: payment rounding error  ⚠ batch 3

  ↑↓ move   [space] toggle   [enter] confirm   [q] quit
```

**Step 3 — confirmation:**

```text
About to cherry-pick 3 commits:
  git cherry-pick e4f5g6h a1b2c3d i7j8k9l

Proceed? [y]es / [n]o: y

→ git cherry-pick e4f5b6h a1b2c3d i7j8k9l

✓ Cherry-pick complete
```

Commits are always applied oldest-first regardless of selection order.

If cherry-pick fails due to a conflict, git is left in the usual conflict state:

```text
error: cherry-pick failed
       resolve conflicts then run: git cherry-pick --continue
```

---

## search

`compare` identifies applied commits by matching commit messages **exactly** between branches. This means every commit message must be unique — if two different commits share the same message, `compare` will incorrectly treat both as applied.

Use `search` to check whether an exact commit message (e.g. a PR title) already exists in the branch history before merging or cherry-picking. It performs a `compare` internally and looks for an exact match in the combined result (pending + applied).

```text
gitmt search [flags] <message> [source target]
```

### search flags

| Flag           | Description                              | Default     |
|----------------|------------------------------------------|-------------|
| `-source`      | Source branch to search in               |             |
| `-target`      | Target branch to check against           |             |
| `-show-author` | Show commit author: `true\|false`        | `true`      |
| `-show-date`   | Show commit date and time: `true\|false` | `true`      |
| `-format`      | Output format: `default` or `json`       | `default`   |

Reads `source`, `target`, `show_author`, and `show_date` from `.gitmt.yml` when present — flags take precedence.

### search examples

```bash
# check if an exact commit message already exists
gitmt search "fix: pagination off-by-one" develop master

# json output — returns {"match": true/false}
gitmt search -format json "fix: pagination off-by-one" develop master

# using flags
gitmt search -source develop -target master "fix: pagination off-by-one"

# source only (target from config)
gitmt search "feat: TICKET-123 add export endpoint" develop
```

### search output

```text
Search
  message : "fix: pagination off-by-one"
  source  : develop
  target  : master
  base    : abc1234
  found   : 1 match(es)  (0 pending, 1 applied)

  applied  e4f5g6h  2026-03-09 09:11  (Jane Doe)  fix: pagination off-by-one  ✓ already in master
```

- `applied` — a commit with this exact message already exists in the target branch. `compare` would treat any new commit with the same message as already applied. **The commit message is not unique — rename it.**
- `pending` — the message exists only in source and has not been applied to target yet.

When no match is found:

```text
No commits with message "fix: pagination off-by-one" found in develop since merge base.
```

#### search json output

```bash
gitmt search -format json "fix: pagination off-by-one" develop master
```

```json
{ "match": true }
```

```json
{ "match": false }
```

---

## tagdiff

Lists commits between two tags. If only one tag is provided, the previous tag is detected automatically.

```text
gitmt tagdiff [from] <to>
```

### tagdiff flags

| Flag            | Description                              | Default     |
|-----------------|------------------------------------------|-------------|
| `-show-author`  | Show commit author: `true\|false`        | `true`      |
| `-show-date`    | Show commit date and time: `true\|false` | `true`      |
| `-format`       | Output format: `default` or `json`       | `default`   |

Reads `show_author`, `show_date`, and `format` from `.gitmt.yml` when present — flags take precedence.

### tagdiff examples

```bash
# explicit range
gitmt tagdiff v0.0.1 v0.0.2

# auto-detect previous tag
gitmt tagdiff v0.0.2

# diff previous tag vs the latest tag
gitmt tagdiff latest

# hide author and date
gitmt tagdiff -show-author=false -show-date=false v0.0.1 v0.0.2

# JSON output
gitmt tagdiff -format json v0.0.1 v0.0.2
```

### tagdiff output

```text
Tag diff
  from  : v0.0.1
  to    : v0.0.2
  total : 3 commit(s)

  1. a1b2c3d  2026-03-10 14:22  (Jane Doe)  feat: add export endpoint
  2. e4f5g6h  2026-03-09 09:11  (Jane Doe)  fix: pagination off-by-one
  3. i7j8k9l  2026-03-08 17:45  (John Smith)  chore: bump dependencies
```

#### json output

```json
{
  "from": "v0.0.1",
  "to": "v0.0.2",
  "total": 3,
  "commits": [
    { "hash": "a1b2c3d", "date": "2026-03-10 14:22", "author": "Jane Doe", "message": "feat: add export endpoint" },
    { "hash": "e4f5g6h", "date": "2026-03-09 09:11", "author": "Jane Doe", "message": "fix: pagination off-by-one" },
    { "hash": "i7j8k9l", "date": "2026-03-08 17:45", "author": "John Smith", "message": "chore: bump dependencies" }
  ]
}
```

When only one tag is given, the `from` is resolved automatically:

```bash
$ gitmt tagdiff v0.0.2
# equivalent to: gitmt tagdiff v0.0.1 v0.0.2
```

If there are no commits between the two tags:

```text
No commits between v0.0.1 and v0.0.2.
```

---

## upgrade

Downloads and replaces the current binary with the latest release from GitHub. The platform and architecture are detected automatically.

```bash
gitmt upgrade
```

If gitmt is already on the latest version, the command exits early:

```text
Upgrade gitmt

→ Checking latest release...
✓ Already on the latest version (v0.1.0)
```

Otherwise it downloads the new binary and installs it in-place:

```text
Upgrade gitmt

→ Checking latest release...
  current : v0.0.3
  latest  : v0.1.0

→ Downloading gitmt-darwin-universal...
→ Installing to /usr/local/bin/gitmt...
✓ Updated to v0.1.0
```

If the install path requires elevated permissions (e.g. `/usr/local/bin` on macOS), run with `sudo`:

```bash
sudo gitmt upgrade
```

---

## Project structure

```text
git-master-tool/
├── Makefile
├── README.md
├── src/               Go source code
│   ├── main.go
│   ├── git.go
│   ├── output.go
│   ├── tickets.go
│   ├── go.mod
│   └── go.sum
├── scripts/           Installer scripts
│   ├── install.sh     macOS / Linux
│   └── install.ps1    Windows
└── dist/              Build output (generated by make)
```

## Requirements

- Git installed and available on `$PATH`
- Must be run from inside a git repository
- Go 1.21+ only required when building from source
