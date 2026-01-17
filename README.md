Here is the updated `README.md`. I have revised the **Installation** and **Configuration** sections to reflect the new `config.json` workflow and the cross-platform clipboard updates.

```markdown
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
   git clone [https://github.com/mitchelljbauer/context.git](https://github.com/mitchelljbauer/context.git)

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

## ⚙️ Configuration

You can customize which files are included or ignored by creating a `config.json` file in the same directory as the executable (`C:\Tools\config.json`).

The tool supports **JSONC** (JSON with Comments), allowing you to annotate your configuration.

**Example `config.json`:**

```jsonc
{
  "included_extensions": [
    ".md", ".json", ".yaml", ".txt",  // Documentation
    ".py",                            // Python
    ".cs",                            // C#
    ".go", ".toml", ".mod",           // Go
    ".bat", ".ps1",                   // Scripts
    ".js", ".ts", ".tsx", ".css"      // Web Development
  ],
  "ignore_dirs": [
    ".git", ".idea", ".vscode",       // IDE & Git
    "__pycache__", "venv", "env",     // Python Virtual Envs
    "vendor", "fyne-cross",           // Go Build Artifacts
    "bin", "build", "dist",           // Compiled Binaries
    "node_modules", ".next"           // Node JS
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

```

```