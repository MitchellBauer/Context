# Context

A lightweight CLI tool written in Go designed to streamline the workflow of using Large Language Models (LLMs) like Gemini, ChatGPT, or Claude with your codebase.

It recursively scans your current directory, filters for relevant file types (code, docs, config), ignores junk folders (git, bin, vendor), and copies a structured XML representation of your project directly to your clipboard.

## 🚀 Why?

Copy-pasting files one by one into an AI chat is tedious. `context` does it all in one command, formatting the output so the AI understands both your **file structure** and **file contents**.

## ✨ Features

* **Blazing Fast:** Built with Go for instant execution.
* **Smart Filtering:** Customizable filtering for file extensions and ignored directories.
* **LLM-Ready Format:** Wraps content in XML tags (`<file name="...">`) which helps models distinguish between instruction and code.
* **Visual Map:** Generates a tree view of your project structure at the top of the output.
* **Clipboard Integration:** Output goes straight to your clipboard—just run and paste (Ctrl+V).
* **Cross-Platform:** Works on Windows, macOS, and Linux.

## 🛠️ Installation

### 1. Prerequisites

* [Go](https://go.dev/dl/) installed on your machine.

### 2. Build from Source

1. Clone this repository:
```bash
git clone https://github.com/mitchellbauer/context.git

```


2. Build the executable:
```bash
cd context
go build -o context.exe context.go

```


3. Create a folder named `Tools` directly on your C: drive (`C:\Tools`).
4. Move the generated `context.exe` into `C:\Tools`.
5. **(Optional)** Create a `config.json` file in `C:\Tools` to customize behavior (see Configuration below).

### 3. Setting up the System Path (Windows 11)

To run `context` from any terminal window without typing the full path, you must add `C:\Tools` to your System Variables.

1. Press the **Windows Key**, type **"env"**, and select **Edit the system environment variables**.
2. In the window that appears, click the **Environment Variables** button (bottom right).
3. In the bottom section labeled **System variables**, scroll down and select the variable named **Path**.
4. Click the **Edit...** button.
5. Click **New** on the right side and type exactly: `C:\Tools`
6. Click **OK** on all three open windows to save the changes.
7. **Restart your terminal** (PowerShell or CMD) for the changes to take effect.

## 💻 Usage

Navigate to any project folder in your terminal and run:

```bash
context

```

You will see:

```text
Scanning project files...
✅ Success! Context copied to clipboard.
Ready to paste into Gemini! (Ctrl+V)

```

### Advanced Modes

**🔎 Preview Mode**
Use this to check which files will be included without copying anything to the clipboard. Useful for verifying your `.gitignore` or `config.json` rules.

```bash
context -p
# OR
context --preview

```

**🌳 Structure Mode**
Use this to copy **only** the file and folder hierarchy to your clipboard (no file contents). This gives the AI a "map" of your directory without the heavy text.

* **Codebase Architecture:** Ask high-level questions about how to structure your project.
* **Folder Organization:** Perfect for asking advice on reorganizing personal files, such as cleaning up an **Obsidian Vault** or sorting a document library.

```bash
context -s
# OR
context --structure

```

## ⚙️ Configuration

You can customize behavior by editing the `config.json` file located in your installation directory (e.g., `C:\Tools\config.json`).

### Adjusting Limits

* **Log File Limit:** Large log files can eat up context windows.
* **How to change:** Update `"max_log_lines"` in `config.json`.
* **Behavior:** Files matching extensions in `"log_extensions"` will be truncated to this number of lines.


* **Token Limit:** The tool estimates tokens to help you avoid context overflow.
* **How to change:** Update `"token_limit"` in `config.json`.
* **Behavior:** This is a *soft limit*. The tool will print a warning ⚠️ if the estimated tokens exceed this number, but it will still copy the content to your clipboard.



### Configuration Reference

The tool supports **JSONC** (JSON with Comments).

**Example `config.json`:**

```jsonc
{
  "token_limit": 1000000,              // Warn if estimated tokens exceed this
  "max_log_lines": 500,                // Truncate logs larger than this
  
  "included_extensions": [
    ".md", ".json", ".yaml", ".txt",   // Documentation
    ".py",                             // Python
    ".cs",                             // C#
    ".go", ".toml", ".mod",            // Go
    ".bat", ".ps1",                    // Scripts
    ".js", ".ts", ".tsx", ".css"       // Web Development
  ],
  "log_extensions": [
    ".txt", ".log", ".out", ".err"     // Extensions to treat as logs
  ],
  "ignore_dirs": [
    ".git", ".idea", ".vscode",        // IDE & Git
    "__pycache__", "venv", "env",      // Python Virtual Envs
    "vendor", "fyne-cross",            // Go Build Artifacts
    "bin", "build", "dist",            // Compiled Binaries
    "node_modules", ".next"            // Node JS
  ],
  "ignore_files": [
    "go.sum",
    "bundled.go",
    "package-lock.json",
    "yarn.lock"
  ]
}

```

*Note: If `config.json` is missing, the tool will default to a standard set of extensions and ignore rules.*

## 📄 License

MIT