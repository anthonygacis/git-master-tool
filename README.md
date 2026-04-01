# gitcompare

A CLI tool to compare two git branches and identify which commits from a source branch have not yet been cherry-picked into a target branch.

## How it works

`gitcompare` finds the common ancestor of both branches, then lists every commit in the source branch since that point. Each commit is matched against the target branch **by commit message** — since cherry-picking preserves the original message, a match means the commit has already been applied.

## Installation

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/anthonygacis/git-compare/main/install.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/anthonygacis/git-compare/main/install.ps1 | iex
```

The installer automatically:

- Detects your OS and architecture
- Downloads the latest release binary from GitHub
- Installs to `/usr/local/bin` (macOS/Linux) or `%LOCALAPPDATA%\Programs\gitcompare` (Windows)
- Adds the install directory to your PATH if needed

On macOS a universal binary is downloaded (runs natively on both Intel and Apple Silicon).

### Build from source

Requires Go 1.21+.

```bash
git clone https://github.com/anthonygacis/git-compare.git
cd git-compare
make install
```

## Usage

```
gitcompare [flags] [source target]
```

All options can be provided as flags, set in `.gitcompare.yml`, or both — **flags always take precedence over the config file**.

### Flags

| Flag            | Description                                  | Default     |
|-----------------|----------------------------------------------|-------------|
| `-source`       | Source branch (commits to cherry-pick from)  |             |
| `-target`       | Target branch (branch to cherry-pick into)   |             |
| `-show`         | `pending` or `all`                           | `all`       |
| `-show-author`  | `true` or `false`                            | `true`      |
| `-format`       | `default`, `table`, `json`, or `csv`         | `default`   |

### Examples

```bash
# positional args (shorthand)
gitcompare develop master

# with flags
gitcompare -show pending -format table develop master

# use config file for source/target, override format
gitcompare -format json

# hide author, show only pending
gitcompare -show-author=false -show pending develop master
```

## Output formats

### default

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

### table

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

### csv

Standard CSV output — useful for importing into spreadsheets or further shell processing.

```csv
status,hash,author,message
pending,20b048a,Jane Doe,feat: add feature C
pending,dacf154,Jane Doe,feat: add feature B
applied,eed2e8d,Jane Doe,feat: add feature A
```

Fields with commas or quotes are automatically escaped per RFC 4180.

### json

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

## Config file

Place a `.gitcompare.yml` file in the root of your repository to set defaults.

```yaml
source: develop
target: master
show: pending        # "pending" or "all" (default: "all")
show_author: true    # show commit author (default: true)
format: default      # "default", "table", or "json" (default: "default")
```

| Field         | Description                                                                                        |
|---------------|----------------------------------------------------------------------------------------------------|
| `source`      | Branch containing the commits you are cherry-picking from                                          |
| `target`      | Branch you are cherry-picking into                                                                 |
| `show`        | `all` shows pending + already applied. `pending` shows only commits that still need cherry-picking |
| `show_author` | `true` displays the commit author next to each entry. `false` hides it. Defaults to `true`         |
| `format`      | Output style: `default` (colored list), `table` (box table), or `json`                             |

## Requirements

- Go 1.21+
- Git installed and available on `$PATH`
- Must be run from inside a git repository
