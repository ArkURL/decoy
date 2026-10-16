# decoy

[![Go Version](https://img.shields.io/github/go-mod/go-version/ArkURL/decoy?logo=go&logoColor=white)](go.mod)
[![License](https://img.shields.io/github/license/ArkURL/decoy)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)](#install)
[![Built with Bubble Tea](https://img.shields.io/badge/built%20with-Bubble%20Tea-ff69b4)](https://github.com/charmbracelet/bubbletea)
[![Stars](https://img.shields.io/github/stars/ArkURL/decoy?logo=github)](https://github.com/ArkURL/decoy/stargazers)
[![Last Commit](https://img.shields.io/github/last-commit/ArkURL/decoy)](https://github.com/ArkURL/decoy/commits)

[中文](README.md) · **English**

> A terminal EPUB reader with a **boss key** that disguises the screen as a Claude Code session — read at work, look busy.

`decoy` is a terminal EPUB reader written in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea). It does one thing: let you read quietly in your terminal, and when someone walks up, press `b` to turn the whole screen into a convincing Claude Code session.

## Features

- **EPUB 2 / 3 parser** — lenient parsing of metadata, table of contents (NCX and EPUB 3 nav) and chapter text. **No DRM.**
- **Correct CJK typography** — wrapping is based on East-Asian character width, so CJK/English mixed text and emoji never misalign.
- **Reading** — line scrolling, page turns, chapter navigation, table of contents, jump to top/bottom, and a current/total page + percentage indicator.
- **Progress memory** — saved on quit, every 10 seconds, and on `kill`; reopening resumes where you left off, even at a different window width.
- **Boss key** — `b` toggles a fake Claude Code screen (English text, a new line every 3 seconds, a working input box with fake replies); the **terminal window title is disguised too**, and it auto-triggers after 30 seconds idle.
- **Configurable** — `idleSeconds`, the fake title, and key bindings can all be overridden.

## Demo

![decoy demo](demo/demo.gif)

> Demo book: 西遊記 (public domain, from Chinese Wikisource). Re-record with
> `brew install vhs ttyd ffmpeg` → `./demo/fetch.sh` → `go build -o decoy .` →
> `vhs demo/demo.tape` (script: [`demo/demo.tape`](demo/demo.tape)).

## Install

Requires Go 1.27+.

```bash
go install github.com/arkurl/decoy@latest
```

From source:

```bash
git clone https://github.com/ArkURL/decoy.git
cd decoy
go build -o decoy .
```

> Put the binary on your `$PATH`. It works on macOS, Linux and Windows (terminal capabilities vary on Windows).

## Usage

```bash
decoy /path/to/book.epub
```

A single file is opened directly (there is no bookshelf yet).

## Key bindings

Reading view:

| Key | Action |
| --- | --- |
| `j` / `k`, `↓` / `↑` | Scroll one line (**within the current chapter only**) |
| `n`, `space`, `pgdn`, `f` | Next page (crosses chapters) |
| `p`, `pgup` | Previous page (crosses chapters) |
| `d` / `u`, `ctrl+d` / `ctrl+u` | Half page down / up |
| `J` / `]`, `K` / `[` | Next / previous chapter |
| `g` / `home`, `G` / `end` | Top / bottom of chapter |
| `t` | Table of contents |
| `b` | Boss key |
| `?` | Help |
| `q`, `ctrl+c` | Quit |

> Pressing `j` at the end of a chapter does **not** jump to the next chapter (that would break your reading flow); the status bar shows "end of chapter · n for next". Use `n` / `p` to change chapters.

Boss screen:

| Key | Action |
| --- | --- |
| `b` | Back to reading (browse mode) |
| `ctrl+b` | Back to reading from **any** state (escape hatch) |
| `i` / `enter` | Focus the input line |
| `enter` | Send, then append a fake reply |
| `esc` | Leave the input / go back to reading |
| `j` / `k`, `pgup` / `pgdn` | Scroll the conversation |

## Configuration

The config file is optional and lives under `os.UserConfigDir()`:

- Linux: `~/.config/decoy/config.json`
- macOS: `~/Library/Application Support/decoy/config.json`
- Windows: `%AppData%\decoy\config.json`

```json
{
  "idleSeconds": 30,
  "fakeTitle": "~/dev/api — vim",
  "keys": {
    "boss": ["b"],
    "pageup": ["p", "pgup"]
  }
}
```

- `idleSeconds`: seconds of inactivity before the boss screen appears (`0` disables it).
- `fakeTitle`: window title used while disguised.
- `keys`: override bindings by **action name**; each value is a list of keys. Available actions: `down` `up` `pagedown` `pageup` `halfdown` `halfup` `nextchapter` `prevchapter` `top` `bottom` `toc` `boss` `bossback` `bossfocus` `help` `quit`.

Reading progress is stored as `progress.json` in the same directory.

## Limitations

- EPUB only (no mobi/azw3) and **no DRM** support.
- Images and tables are flattened to plain text; no full-text search, bookmarks or highlights yet.
- Footnotes / internal links in the text are not navigable yet.
- Changing the window width re-lays out the whole book to compute page numbers (tens of milliseconds on large books).

## Development

```bash
go build ./...      # build
go test ./...       # tests
go vet ./...        # vet
gofmt -l .          # format check
```

Layout:

```
main.go              # entry point: args, config, start Bubble Tea
internal/epub/       # lenient EPUB 2/3 parsing (container / OPF / spine / NCX / nav / text)
internal/render/     # East-Asian-width wrapping + layout and reading-position math
internal/reader/     # reading view: scroll / page / chapter / TOC / page numbers
internal/boss/       # the fake Claude Code boss screen
internal/storage/    # reading-progress JSON persistence
internal/config/     # config and key map
internal/app/        # root model: mode switching, title disguise, idle timer
```

## License

[MIT](LICENSE) © 2026 ArkURL
