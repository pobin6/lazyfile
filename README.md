# lazyfile

[中文说明](README_zh.md)

lazyfile is a terminal-based file manager written in Go. It provides a three-column browser for managing directory collections, navigating files, previewing content, and performing common file operations.

## Features

- Organize saved directory bookmarks into collections.
- Browse files and directories in three panes: collections/bookmarks, Files, and Preview.
- Navigate directories with `h` and `l` while keeping navigation within the selected bookmark.
- Preview text files (up to 1 KiB) and directory contents.
- Create files and directories, delete items with confirmation, and copy or move files and directories.
- Select multiple items for copy or cut operations.
- View file metadata, recent operation commands, and the current clipboard selection in the bottom panes.
- See shortcuts for the focused pane in the unbordered bottom row; long hints are truncated to fit the terminal.
- Persist collections, bookmarks, and each bookmark's current browsing path.

## Requirements

- Go 1.27 or later
- Linux or Windows 10/11
- A terminal that supports Unicode box-drawing characters and color; Windows Terminal is recommended on Windows

## Build and run

On Linux:

```sh
go build -o lazyfile ./cmd/lazyfile
./lazyfile
```

On Windows PowerShell:

```powershell
go build -o lazyfile.exe ./cmd/lazyfile
.\lazyfile.exe
```

Or run it directly on either platform:

```sh
go run ./cmd/lazyfile
```

The application starts with the saved collection and browsing state. To start with an empty configuration, move or remove the configuration file described below.

## Keyboard shortcuts

| Key | Action |
| --- | --- |
| `q` | Quit |
| `0` | Focus the path bar |
| `1` | Focus the Collections/Directories pane |
| `2` | Focus the Files pane |
| `3` | Focus the Preview pane |
| `[` / `]` | Switch between collection and bookmark pages |
| `j` / `k` | Move down/up in the focused pane |
| `/` | Search in the focused Collections/Directories or Files pane; matches update as you type |
| `n` / `N` | Move to the next/previous search match |
| `Enter` / `Esc` | Finish search on the first match / clear the search |
| `h` | In Files, go to the parent directory |
| `Enter` / `l` | In Files, enter a selected directory or open a selected file with the system default application |
| `o` | Open the current directory in the system file manager and reveal the selected item when supported |
| `r` | Rename the selected file or directory |
| `a` | Add a collection, bookmark, file, or directory depending on the focused pane |
| `e` | Edit the selected bookmark |
| `d` | Delete the selected bookmark or Files item; deletion requires confirmation |
| `y` / `x` | Select the current Files item for copy / cut |
| `Ctrl+y` / `Ctrl+x` | Add the current item to the copy / cut selection |
| `Shift+y` / `Shift+x` | Select a range of items for copy / cut |
| `p` | Paste the selected items into the current Files directory |
| `Enter` | In Files, enter a directory or open a file; otherwise confirm an input, deletion, or search |
| `Esc` | Cancel a dialog, or clear the Files selection when no dialog is open |
| `Ctrl+c` | Quit |
| `Backspace` | Remove the last character in an input |

When creating an item in Files, a name without a path separator creates an empty file; a name containing `/` creates a directory path, and Windows also accepts `\`.

On Linux, opening files requires `xdg-open` (usually provided by `xdg-utils`).

## Configuration

lazyfile stores collections, bookmarks, and directory browsing state in:

```text
%AppData%\lazyfile\entries.json
```

on Windows, and in:

```text
$XDG_CONFIG_HOME/lazyfile/entries.json
```

If `XDG_CONFIG_HOME` is not set on Linux, the operating system's user configuration directory is used (typically `~/.config/lazyfile/entries.json`). The application uses Go's `os.UserConfigDir`, so it follows the platform's standard user configuration location.

## Development

Run tests, static analysis, and build:

```sh
go test ./...
go vet ./...
go build ./...
```

The CI workflow runs these checks on Linux and Windows. For an interactive Windows smoke test, launch the application in Windows Terminal and verify keyboard input, resizing/redraw, file operations, and clean terminal restoration on exit.

## License

lazyfile is distributed under the Apache License 2.0. See [LICENSE](LICENSE) for the license text.
