# git-master-tool

A CLI toolkit to supercharge your git workflow. Run as `gitmt <command>`.

## Commands

| Command   | Description                                       |
|-----------|---------------------------------------------------|
| `compare` | Compare two branches and surface unmerged commits |

More commands coming soon.

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

```
gitmt compare [flags] [source target]
```

All options can be provided as flags, set in `.gitmt.yml`, or both — **flags always take precedence over the config file**.

### Flags

| Flag                    | Description                                        | Default     |
|-------------------------|----------------------------------------------------|-------------|
| `-source`               | Source branch (commits to cherry-pick from)        |             |
| `-target`               | Target branch (branch to cherry-pick into)         |             |
| `-show`                 | `pending` or `all`                                 | `all`       |
| `-show-author`          | `true` or `false`                                  | `true`      |
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

```
Branch comparison
  source : develop
  target : master
  base   : a1a2198

 Pending — not yet in master:
  1. 20b048a  (Jane Doe)  feat: add feature C
  2. dacf154  (Jane Doe)  feat: add feature B

 Already applied — commit message found in master:
  1. eed2e8d  (Jane Doe)  feat: add feature A

Summary
  pending    2 commit(s) need cherry-picking into master
  applied    1 commit(s) already applied
```

#### table

Box-drawn table with dynamic column widths.

```
┌─────────┬─────────┬──────────┬─────────────────────┐
│ STATUS  │ HASH    │ AUTHOR   │ MESSAGE             │
├─────────┼─────────┼──────────┼─────────────────────┤
│ pending │ 20b048a │ Jane Doe │ feat: add feature C │
│ pending │ dacf154 │ Jane Doe │ feat: add feature B │
│ applied │ eed2e8d │ Jane Doe │ feat: add feature A │
└─────────┴─────────┴──────────┴─────────────────────┘

  2 pending   1 applied
```

#### csv

Standard CSV output — useful for importing into spreadsheets or further shell processing.

```csv
status,hash,author,message
pending,20b048a,Jane Doe,feat: add feature C
pending,dacf154,Jane Doe,feat: add feature B
applied,eed2e8d,Jane Doe,feat: add feature A
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
    { "hash": "20b048a", "author": "Jane Doe", "message": "feat: add feature C", "status": "pending" },
    { "hash": "dacf154", "author": "Jane Doe", "message": "feat: add feature B", "status": "pending" },
    { "hash": "eed2e8d", "author": "Jane Doe", "message": "feat: add feature A", "status": "applied" }
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
format: default            # "default", "table", "json", or "csv" (default: "default")
prefixes:                  # Jira prefixes to scan for ticket summary (default: none)
  - XS
  - XI
show_ticket_authors: true  # show authors in ticket summary (default: true)
```

| Field                 | Description                                                                                        |
|-----------------------|----------------------------------------------------------------------------------------------------|
| `source`              | Branch containing the commits you are cherry-picking from                                          |
| `target`              | Branch you are cherry-picking into                                                                 |
| `show`                | `all` shows pending + already applied. `pending` shows only commits that still need cherry-picking |
| `show_author`         | Show commit author next to each entry. Defaults to `true`                                          |
| `format`              | Output style: `default`, `table`, `json`, or `csv`                                                 |
| `prefixes`            | List of Jira ticket prefixes to scan for and group in the ticket summary                           |
| `show_ticket_authors` | Show authors per ticket in the ticket summary. Defaults to `true`                                  |

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
