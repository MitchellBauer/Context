package main

import (
	"bufio"
	"encoding/json"
	"flag"
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

type Config struct {
	TokenLimit         int      `json:"token_limit"`
	IncludedExtensions []string `json:"included_extensions"`
	LogExtensions      []string `json:"log_extensions"`
	MaxLogLines        int      `json:"max_log_lines"`
	IgnoreDirs         []string `json:"ignore_dirs"`
	IgnoreFiles        []string `json:"ignore_files"`
}

// Global maps for O(1) lookups
var (
	includedExtensions = make(map[string]bool)
	logExtensions      = make(map[string]bool)
	ignoreDirs         = make(map[string]bool)
	ignoreFiles        = make(map[string]bool)
	maxLogLines        int
	tokenLimit         int
)

func main() {
	// 1. Load Configuration first so filters are ready
	loadConfiguration()

	// 2. Parse Flags
	previewMode := flag.Bool("preview", false, "Preview file structure and list without copying")
	pShort := flag.Bool("p", false, "Preview alias")

	structureMode := flag.Bool("structure", false, "Copy only the project structure tree to clipboard")
	sShort := flag.Bool("s", false, "Structure alias")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// Combine long and short flags
	isPreview := *previewMode || *pShort
	isStructure := *structureMode || *sShort

	// 3. Handle Preview Mode (Visual only, no clipboard)
	if isPreview {
		fmt.Println("🔎 Scanning project for preview...")
		tree, count, err := generateProjectTree()
		if err != nil {
			fmt.Printf("❌ Error scanning files: %v\n", err)
			return
		}

		fmt.Println("\n" + tree)
		fmt.Printf("📄 Found %d files.\n", count)
		return
	}

	// 4. Handle Structure Mode (Tree only -> Clipboard)
	if isStructure {
		fmt.Println("🌳 Scanning project structure...")
		tree, count, err := generateProjectTree()
		if err != nil {
			fmt.Printf("❌ Error scanning files: %v\n", err)
			return
		}

		// Wrap in tags so the LLM knows what it is looking at
		output := "Here is the project structure:\n" +
			"<project_structure>\n" +
			tree +
			"</project_structure>\n"

		// Calculate tokens for the structure
		estTokens := estimateTokens(output)

		err = copyToClipboard(output)
		if err != nil {
			fmt.Printf("❌ Error copying to clipboard: %v\n", err)
		} else {
			fmt.Println("✅ Success! Project structure copied to clipboard. (Ctrl+V)")
			fmt.Printf("📄 Tree contains %d files/folders.\n", count)

			// Print token stats
			fmt.Printf("📊 Estimated Tokens: %d", estTokens)
			if tokenLimit > 0 {
				fmt.Printf(" / %d", tokenLimit)
				if estTokens > tokenLimit {
					fmt.Printf("\n⚠️  WARNING: Output exceeds configured limit by ~%d tokens!", estTokens-tokenLimit)
				}
			}
			fmt.Println()
		}

		fmt.Print("Press Enter to exit...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		return
	}

	// 5. Normal Mode (Full Context -> Clipboard)
	fmt.Println("Scanning project files...")

	fullContext, count, err := generateXMLContext()
	if err != nil {
		fmt.Printf("❌ Error scanning files: %v\n", err)
		return
	}

	// Calculate and check tokens
	estTokens := estimateTokens(fullContext)

	err = copyToClipboard(fullContext)
	if err != nil {
		fmt.Printf("❌ Error copying to clipboard: %v\n", err)
	} else {
		fmt.Println("✅ Success! Context copied to clipboard. (Ctrl+V)")
		fmt.Printf("📄 Aggregated %d documents.\n", count)

		// Print token stats
		fmt.Printf("📊 Estimated Tokens: %d", estTokens)
		if tokenLimit > 0 {
			fmt.Printf(" / %d", tokenLimit)
			if estTokens > tokenLimit {
				fmt.Printf("\n⚠️  WARNING: Output exceeds configured limit by ~%d tokens!", estTokens-tokenLimit)
			}
		}
		fmt.Println()
	}

	fmt.Print("Press Enter to exit...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// ... (Rest of the file: loadConfiguration, generateProjectTree, generateXMLContext, etc. remain unchanged)
// ... Be sure to keep the helper functions from your original file.

func loadConfiguration() {
	// ... (Your existing code)
	// Defaults
	defaultExts := []string{".md", ".json", ".yaml", ".txt", ".py", ".cs", ".go", ".toml", ".mod"}
	defaultLogExts := []string{".txt", ".log", ".out", ".err"}
	defaultDirs := []string{".git", ".idea", ".vscode", "__pycache__", "venv", "env", "vendor", "bin", "build", "node_modules"}
	defaultFiles := []string{"go.sum", "bundled.go", "resource.go"}
	defaultMaxLines := 200
	defaultTokenLimit := 128000

	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("⚠️ Could not determine executable path. Using defaults.")
		populateMaps(defaultExts, defaultLogExts, defaultDirs, defaultFiles, defaultMaxLines, defaultTokenLimit)
		return
	}
	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, "config.json")

	fileBytes, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("⚠️ Error reading config.json: %v. Using defaults.\n", err)
		}
		populateMaps(defaultExts, defaultLogExts, defaultDirs, defaultFiles, defaultMaxLines, defaultTokenLimit)
		return
	}

	cleanBytes := stripJSONComments(fileBytes)

	var config Config
	if err := json.Unmarshal(cleanBytes, &config); err != nil {
		fmt.Printf("⚠️ Error parsing config.json: %v. Using defaults.\n", err)
		populateMaps(defaultExts, defaultLogExts, defaultDirs, defaultFiles, defaultMaxLines, defaultTokenLimit)
		return
	}

	fmt.Printf("⚙️ Loaded configuration from %s\n", configPath)

	if config.MaxLogLines == 0 {
		config.MaxLogLines = defaultMaxLines
	}

	populateMaps(config.IncludedExtensions, config.LogExtensions, config.IgnoreDirs, config.IgnoreFiles, config.MaxLogLines, config.TokenLimit)
}

func stripJSONComments(data []byte) []byte {
	re := regexp.MustCompile(`(?m)\s*//.*$`)
	return re.ReplaceAll(data, nil)
}

func populateMaps(exts, logs, dirs, files []string, maxLines, tLimit int) {
	for _, e := range exts {
		includedExtensions[e] = true
	}
	for _, l := range logs {
		logExtensions[l] = true
	}
	for _, d := range dirs {
		ignoreDirs[d] = true
	}
	for _, f := range files {
		ignoreFiles[f] = true
	}
	maxLogLines = maxLines
	tokenLimit = tLimit
}

func estimateTokens(content string) int {
	return len(content) / 4
}

// --- FILE READING & GENERATION ---

func readTruncated(path string, limit int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var b strings.Builder
	scanner := bufio.NewScanner(f)

	lineCount := 0
	truncated := false

	for scanner.Scan() {
		if lineCount >= limit {
			truncated = true
			break
		}
		b.Write(scanner.Bytes())
		b.WriteByte('\n')
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if truncated {
		b.WriteString(fmt.Sprintf("\n... [TRUNCATED: File exceeded %d lines] ...\n", limit))
	}

	return b.String(), nil
}

// generateProjectTree handles the recursive directory walk to build the visual tree
// and count files. It is used by both the Preview mode and the XML generation.
func generateProjectTree() (string, int, error) {
	var output strings.Builder
	var fileCount int

	rootDir, err := os.Getwd()
	if err != nil {
		return "", 0, err
	}

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
			ext := filepath.Ext(d.Name())
			if includedExtensions[ext] && !ignoreFiles[d.Name()] {
				output.WriteString(fmt.Sprintf("%s%s\n", indent, d.Name()))
				fileCount++ // Count valid files
			}
		}
		return nil
	})

	return output.String(), fileCount, err
}

func generateXMLContext() (string, int, error) {
	var output strings.Builder

	// --- PASS 1: Tree Structure ---
	// Reuse the exact same tree logic for consistency
	treeStructure, _, err := generateProjectTree()
	if err != nil {
		return "", 0, err
	}

	output.WriteString("Here is the file structure and content:\n")
	output.WriteString("<project_structure>\n")
	output.WriteString(treeStructure)
	output.WriteString("</project_structure>\n")

	// --- PASS 2: File Contents ---
	// We count again here to ensure the number of file blocks matches exactly what was read
	var fileCount int
	rootDir, _ := os.Getwd()

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
		if !includedExtensions[ext] || ignoreFiles[d.Name()] {
			return nil
		}

		var content string

		if logExtensions[ext] {
			content, err = readTruncated(path, maxLogLines)
		} else {
			var b []byte
			b, err = os.ReadFile(path)
			content = string(b)
		}

		if err != nil {
			fmt.Printf("Skipping %s: %v\n", d.Name(), err)
			return nil
		}

		relPath, _ := filepath.Rel(rootDir, path)
		relPath = filepath.ToSlash(relPath)

		output.WriteString(fmt.Sprintf("<file name=\"%s\">\n%s\n</file>\n", relPath, content))
		fileCount++
		return nil
	})

	return output.String(), fileCount, err
}

func copyToClipboard(text string) error {
	switch runtime.GOOS {
	case "windows":
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
