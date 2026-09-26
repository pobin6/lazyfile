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
- Persist collections, bookmarks, and each bookmark's current browsing path.

## Requirements

- Go 1.27 or later
- A terminal that supports Unicode box-drawing characters and color

## Build and run

From the project root:

```sh
go build -o lazyfile ./cmd/lazyfile
./lazyfile
```

Or run it directly:

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
| `h` / `l` | In Files, go to the parent directory / enter the selected directory |
| `a` | Add a collection, bookmark, file, or directory depending on the focused pane |
| `e` | Edit the selected bookmark |
| `d` | Delete the selected bookmark or Files item; deletion requires confirmation |
| `y` / `x` | Select the current Files item for copy / cut |
| `Ctrl+y` / `Ctrl+x` | Add the current item to the copy / cut selection |
| `Shift+y` / `Shift+x` | Select a range of items for copy / cut |
| `p` | Paste the selected items into the current Files directory |
| `Enter` | Confirm an input or delete confirmation |
| `Esc` | Cancel a dialog, or quit when no dialog is open |
| `Backspace` | Remove the last character in an input |

When creating an item in Files, a name without `/` creates an empty file; a name containing `/` creates a directory path.

## Configuration

lazyfile stores collections, bookmarks, and directory browsing state in:

```text
$XDG_CONFIG_HOME/lazyfile/entries.json
```

If `XDG_CONFIG_HOME` is not set, the operating system's user configuration directory is used (for example, `~/.config/lazyfile/entries.json` on Linux).

## Development

Run tests, static analysis, and build:

```sh
go test ./...
go vet ./...
go build ./...
```

## License

lazyfile is distributed under the Apache License 2.0. See [LICENSE](LICENSE) for the license text.
