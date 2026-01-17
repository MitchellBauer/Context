package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// --- CONFIGURATION STRUCTS ---

// Config matches the structure of config.json
type Config struct {
	IncludedExtensions []string `json:"included_extensions"`
	IgnoreDirs         []string `json:"ignore_dirs"`
	IgnoreFiles        []string `json:"ignore_files"`
}

// Global maps for O(1) lookups
var (
	includedExtensions = make(map[string]bool)
	ignoreDirs         = make(map[string]bool)
	ignoreFiles        = make(map[string]bool)
)

func main() {
	// 1. Load Configuration (from JSON or Defaults)
	loadConfiguration()

	fmt.Println("Scanning project files...")

	// 2. Generate the context string
	fullContext, count, err := generateXMLContext()
	if err != nil {
		fmt.Printf("❌ Error scanning files: %v\n", err)
		return
	}

	// 3. Copy to clipboard
	err = copyToClipboard(fullContext)
	if err != nil {
		fmt.Printf("❌ Error copying to clipboard: %v\n", err)
	} else {
		fmt.Println("✅ Success! Context copied to clipboard. (Ctrl+V)")
		fmt.Printf("📄 Aggregated %d documents.\n", count)
	}

	// 4. Keep window open
	fmt.Print("Press Enter to exit...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func loadConfiguration() {
	// Default values (Fallback if config.json is missing)
	defaultExts := []string{".md", ".json", ".yaml", ".txt", ".py", ".cs", ".go", ".toml", ".mod"}
	defaultDirs := []string{".git", ".idea", ".vscode", "__pycache__", "venv", "env", "vendor", "bin", "build", "node_modules"}
	defaultFiles := []string{"go.sum", "bundled.go", "resource.go"}

	// Determine where the executable is running from
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("⚠️ Could not determine executable path. Using defaults.")
		populateMaps(defaultExts, defaultDirs, defaultFiles)
		return
	}
	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, "config.json")

	// Try to read config.json
	fileBytes, err := os.ReadFile(configPath)
	if err != nil {
		// It's okay if the file doesn't exist; just use defaults
		if !os.IsNotExist(err) {
			fmt.Printf("⚠️ Error reading config.json: %v. Using defaults.\n", err)
		}
		populateMaps(defaultExts, defaultDirs, defaultFiles)
		return
	}

	// CLEAN COMMENTS BEFORE PARSING
	// This enables "JSONC" (JSON with comments) support
	cleanBytes := stripJSONComments(fileBytes)

	// Parse JSON
	var config Config
	if err := json.Unmarshal(cleanBytes, &config); err != nil {
		fmt.Printf("⚠️ Error parsing config.json: %v. Using defaults.\n", err)
		populateMaps(defaultExts, defaultDirs, defaultFiles)
		return
	}

	// Use loaded config
	fmt.Printf("⚙️ Loaded configuration from %s\n", configPath)
	populateMaps(config.IncludedExtensions, config.IgnoreDirs, config.IgnoreFiles)
}

// stripJSONComments removes lines starting with // or whitespace+//
func stripJSONComments(data []byte) []byte {
	// Regex matches optional whitespace, then //, then anything until end of line
	re := regexp.MustCompile(`(?m)\s*//.*$`)
	return re.ReplaceAll(data, nil)
}

// Helper to convert slices to maps
func populateMaps(exts, dirs, files []string) {
	for _, e := range exts {
		includedExtensions[e] = true
	}
	for _, d := range dirs {
		ignoreDirs[d] = true
	}
	for _, f := range files {
		ignoreFiles[f] = true
	}
}

func generateXMLContext() (string, int, error) {
	var output strings.Builder
	var fileCount int

	rootDir, err := os.Getwd()
	if err != nil {
		return "", 0, err
	}

	output.WriteString("Here is the file structure and content:\n")

	// --- PASS 1: Generate Tree Structure ---
	output.WriteString("<project_structure>\n")

	err = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && ignoreDirs[d.Name()] {
			return filepath.SkipDir
		}

		relPath, _ := filepath.Rel(rootDir, path)
		if relPath == "." {
			return nil
		}

		depth := strings.Count(relPath, string(os.PathSeparator))
		indent := strings.Repeat("    ", depth)

		if d.IsDir() {
			output.WriteString(fmt.Sprintf("%s%s/\n", indent, d.Name()))
		} else {
			// Tree View Logic: Only show if it matches extension AND is not ignored
			ext := filepath.Ext(d.Name())
			if includedExtensions[ext] && !ignoreFiles[d.Name()] {
				output.WriteString(fmt.Sprintf("%s%s\n", indent, d.Name()))
			}
		}
		return nil
	})
	if err != nil {
		return "", 0, err
	}
	output.WriteString("</project_structure>\n")

	// --- PASS 2: Add File Contents ---
	err = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if ignoreDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(d.Name())
		// Content Logic: Included ext AND NOT ignored file? -> Read it.
		if !includedExtensions[ext] || ignoreFiles[d.Name()] {
			return nil
		}

		contentBytes, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Skipping %s: %v\n", d.Name(), err)
			return nil
		}

		relPath, _ := filepath.Rel(rootDir, path)
		relPath = filepath.ToSlash(relPath)

		output.WriteString(fmt.Sprintf("<file name=\"%s\">\n%s\n</file>\n", relPath, string(contentBytes)))

		// Increment count
		fileCount++

		return nil
	})

	return output.String(), fileCount, err
}

func copyToClipboard(text string) error {
	switch runtime.GOOS {
	case "windows":
		// Windows: Use PowerShell to handle UTF-8/Emojis correctly
		tmpFile, err := os.CreateTemp("", "context-*.txt")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(text); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to write to temp file: %w", err)
		}
		tmpFile.Close()

		cmd := exec.Command("powershell", "-NoProfile", "-Command",
			fmt.Sprintf("Get-Content -Path '%s' -Raw -Encoding UTF8 | Set-Clipboard", tmpFile.Name()))
		return cmd.Run()

	case "darwin":
		// macOS: Use 'pbcopy'
		cmd := exec.Command("pbcopy")
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		if err := cmd.Start(); err != nil {
			return err
		}
		io.WriteString(stdin, text)
		stdin.Close()
		return cmd.Wait()

	case "linux":
		// Linux: Try 'xclip', then 'xsel', then 'wl-copy'
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd := exec.Command("xclip", "-selection", "clipboard")
			stdin, err := cmd.StdinPipe()
			if err != nil {
				return err
			}
			if err := cmd.Start(); err != nil {
				return err
			}
			io.WriteString(stdin, text)
			stdin.Close()
			return cmd.Wait()
		}
		if _, err := exec.LookPath("xsel"); err == nil {
			cmd := exec.Command("xsel", "--clipboard", "--input")
			stdin, err := cmd.StdinPipe()
			if err != nil {
				return err
			}
			if err := cmd.Start(); err != nil {
				return err
			}
			io.WriteString(stdin, text)
			stdin.Close()
			return cmd.Wait()
		}
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd := exec.Command("wl-copy")
			stdin, err := cmd.StdinPipe()
			if err != nil {
				return err
			}
			if err := cmd.Start(); err != nil {
				return err
			}
			io.WriteString(stdin, text)
			stdin.Close()
			return cmd.Wait()
		}
		return fmt.Errorf("no clipboard tool found (install xclip, xsel, or wl-copy)")

	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
