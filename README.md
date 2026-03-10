# Context

A lightweight Go CLI that aggregates local project files into an XML-like bundle with readable file bodies and copies the result directly to your clipboard for use with LLMs.

`context` is designed to stay local during normal use. It scans the current working directory, applies extension and ignore filters, builds a project tree plus per-file wrappers, and copies the output to the system clipboard.

## Why

Copy-pasting files one at a time into an AI chat is tedious. `context` collects the files you actually want, preserves the directory map, and formats the result so an LLM can separate structure from content without turning Markdown bodies into hard-to-read XML entities.

## Features

- Fast single-binary workflow written in Go.
- Clipboard-first output for quick paste into ChatGPT, Claude, Gemini, or similar tools.
- Configurable include, ignore, log truncation, and file-size limits.
- Offline-first normal runtime with no network access in the main executable.
- Cross-platform clipboard support for Windows, macOS, and Linux.

## Installation

### Prerequisites

- [Go 1.26.1](https://go.dev/dl/) installed locally.

### Build from source

```bash
git clone https://github.com/mitchellbauer/context.git
cd context
go build -o context.exe .
```

Place `context.exe` in `C:\Tools` or another folder on your `PATH`.
Keep `config.json` in the same directory as the executable so runtime configuration is loaded correctly.

## Usage

Run `context` from the project folder you want to scan.

```bash
context
```

You will see output like:

```text
Scanning project files...
Success! Context copied to clipboard. (Ctrl+V)
```

### Preview mode

Preview which files would be included without touching the clipboard.

```bash
context -p
context --preview
```

### Structure mode

Copy only the directory tree to the clipboard.

```bash
context -s
context --structure
```

## Output format

Normal full-context output keeps XML-like wrappers such as `<project_structure>` and `<file name="...">`, but file bodies are emitted as readable literal text inside CDATA-style sections. This keeps boundaries clear for LLMs without converting every newline, tab, quote, or ampersand into XML entities.

## Security Notes

Normal `context` usage is offline-first:

- The main `context` executable does not perform network I/O during normal scanning or clipboard copy.
- It reads files from the current working directory only.
- On Windows, it writes Unicode text to the native clipboard API directly, preserving emojis and other non-ASCII characters without a shell-based clipboard hop.
- It skips symlinks, ignores a set of sensitive local directories and files by default, truncates configured log types, and skips oversized or non-UTF-8/binary-like content.

Things to keep in mind:

- Clipboard contents are intentionally exposed to your local clipboard manager and any local apps that can read your clipboard.
- Optional maintenance scripts such as [`Update.bat`](./Update.bat) are not part of the trusted offline runtime path; that script reaches out to GitHub by running `git pull`.
- Linux clipboard issues are known to need a follow-up pass and are not part of this Windows-focused security hardening.

## Configuration

`context` reads `config.json` from the executable directory. The file supports JSONC-style comments, so comments are allowed inside the config file.

### Important settings

- `token_limit`: soft warning threshold for estimated tokens.
- `max_log_lines`: truncates files whose extension is listed in `log_extensions`.
- `max_file_bytes`: skips individual files larger than this size.
- `included_extensions`: only these extensions are scanned.
- `ignore_dirs`: directories skipped during the recursive walk.
- `ignore_files`: exact filenames skipped even if their extension is included.

### Example config

```jsonc
{
  "token_limit": 1000000,
  "max_log_lines": 500,
  "max_file_bytes": 2097152,
  "included_extensions": [
    ".md", ".json", ".yaml", ".yml",
    ".py",
    ".cs",
    ".go", ".toml", ".mod",
    ".bat", ".ps1",
    ".js", ".ts", ".tsx", ".css"
  ],
  "log_extensions": [
    ".txt", ".log", ".out", ".err"
  ],
  "ignore_dirs": [
    ".git", ".idea", ".vscode",
    "__pycache__", "venv", "env",
    "vendor", "fyne-cross",
    "bin", "build", "dist",
    "node_modules", ".next",
    "out",
    ".obsidian", ".trash",
    ".stfolder", ".stversions",
    ".ssh", ".aws", ".azure",
    ".kube", ".gnupg"
  ],
  "ignore_files": [
    "go.sum",
    "bundled.go",
    "resource.go",
    "package-lock.json",
    "yarn.lock",
    ".env",
    ".env.local",
    ".env.development",
    ".env.production",
    ".env.test",
    ".npmrc",
    ".pypirc",
    ".netrc",
    "id_rsa",
    "id_ed25519",
    "authorized_keys"
  ]
}
```

## Maintenance scripts

- [`BuildToToolsDir.bat`](./BuildToToolsDir.bat) builds the executable into `C:\tools` and copies `config.json`.
- [`Update.bat`](./Update.bat) is an explicit maintenance helper that runs `git pull` before rebuilding. Use it only when you want to refresh from the remote repository.

## License

MIT
