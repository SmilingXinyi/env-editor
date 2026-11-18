# env-editor

`env-editor` is a terminal editor for `.env` files. It keeps the file's existing layout, writes every confirmed change immediately, and is installed with one command.

Read this in order: what the editor is for, how to install and use it, then how the code is put together when you want to change it.

## Introduction

A `.env` file is a document. Comments, blank lines, `export`, quotes, and notes at the end of a line all matter to the tools that read it later. The editor keeps that document intact:

- A line you do not edit is written back as it was, including its quoting and its line ending.
- Confirming a change saves the file then. Quitting with `q` leaves those saves in place. `u` undoes the last saved change (up to 100).
- When another program has written the file since it was loaded, the save is refused. Reload with `R`, then edit again.
- Text follows the locale. `LC_ALL`, then `LC_MESSAGES`, then `LANG`: a value that starts with `zh` selects Chinese, and any other value selects English.

Open a file, change the lines you care about, and leave. There is no separate save step at the end.

## Usage

### Install or upgrade

This downloads the latest release for the machine you are on, checks it against the release's `SHA256SUMS`, and places the `env-editor` binary in the current user's bin directory. The default directory is `~/.local/bin`. Run the same command again to replace the binary already there.

The archive, the checksum file, and the temporary copy are deleted before the command finishes. Shell startup files are left unchanged. If `~/.local/bin` is not on `PATH`, the command prints the `export` line to add.

```sh
curl -fsSL https://raw.githubusercontent.com/SmilingXinyi/env-editor/main/install.sh | sh
```

To put the binary in another directory, set `ENV_EDITOR_BIN` for that run:

```sh
curl -fsSL https://raw.githubusercontent.com/SmilingXinyi/env-editor/main/install.sh | ENV_EDITOR_BIN="$HOME/bin" sh
```

Published releases currently include `linux/amd64` and `linux/arm64`. Each archive is named `env-editor_<version>_linux_<arch>.tar.gz` and contains the `env-editor` binary.

### Edit a file

```sh
env-editor                  # opens ./.env
env-editor .env.local       # opens a chosen file
env-editor -m               # starts with values hidden
env-editor --help           # prints the key list in the current language
```

A path that does not exist yet stays uncreated until the first change is saved. `-m` / `--mask` hides values on start; `v` shows them again while the editor is open.

### Keys

The bottom of the screen shows the keys for the current mode. The full set:

| Key | Action |
| --- | --- |
| `↑` `↓` `j` `k` | Select a row. The top row is "New variable". |
| `enter` | Edit the selected variable or comment. |
| `a` | Add a variable above the selected row. On the "New variable" row, add it after the last variable. |
| `r` | Rename the selected variable. |
| `n` | Insert a blank line above the selected row. |
| `c` | Insert a comment above the selected row. |
| `#` | Comment out the selected row, or restore it. |
| `y` | Copy the selected row. A variable keeps a commented-out copy above it. |
| `d` | Delete the selected row. Variables take a second press. |
| `f` | Merge consecutive blank lines. |
| `/` | Search keys, values, and comments. `↑` `↓` move between matches, `enter` selects one. |
| `u` | Undo the last saved change. |
| `R` | Reload the file from disk. |
| `v` | Show or hide values. |
| `q` | Quit. Changes are already saved. |

While typing a value: `enter` confirms, `esc` cancels, and the arrow keys move inside the field. On a new variable, `=` starts the value, `ctrl+t` toggles upper-casing the key, and pasting several `KEY=VALUE` lines imports them together.

## Development

Changes follow the same split as the program. File contents live in one package, keystrokes in another, and wording in a third.

| Path | Role |
| --- | --- |
| `main.go` | Reads `-m` / `--mask` and `-h` / `--help`, loads the file, and starts the interface. |
| `internal/envfile` | Parses and writes the document. Untouched lines stay byte for byte. A save replaces the file atomically when the directory allows it, and keeps the previous mode and owner. It returns an error when the file on disk changed after it was loaded. |
| `internal/tui` | Bubble Tea model. Each successful command saves immediately and pushes an undo snapshot. |
| `internal/i18n` | Every string the user sees, in English and Chinese. Add one with `def(english, chinese)`. Tests require both sides to exist and to use the same formatting verbs. |

Go 1.24 is declared in `go.mod`.

```sh
go test ./...          # parser, interface, and translations
go run . .env          # run this checkout against a file
go build -o env-editor .
```

`sh install.sh` downloads the latest GitHub Release, the same way the `curl` command does. A binary from this checkout comes from `go build` above. Copy that file to `ENV_EDITOR_BIN` (or `~/.local/bin`) when the shell should run it.

A published release is a pair of Linux archives, `env-editor_<version>_linux_amd64.tar.gz` and `env-editor_<version>_linux_arm64.tar.gz`, plus `SHA256SUMS`. Inside each archive the binary is `env-editor_<version>_linux_<arch>/env-editor`. The install script resolves `https://github.com/SmilingXinyi/env-editor/releases/latest`, checks the archive for this machine, and replaces `~/.local/bin/env-editor`. The next run of the `curl` command picks up whatever release is latest.
