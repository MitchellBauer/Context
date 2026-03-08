package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

type Config struct {
	TokenLimit         int      `json:"token_limit"`
	IncludedExtensions []string `json:"included_extensions"`
	LogExtensions      []string `json:"log_extensions"`
	MaxLogLines        int      `json:"max_log_lines"`
	MaxFileBytes       int64    `json:"max_file_bytes"`
	IgnoreDirs         []string `json:"ignore_dirs"`
	IgnoreFiles        []string `json:"ignore_files"`
}

var (
	includedExtensions = make(map[string]bool)
	logExtensions      = make(map[string]bool)
	ignoreDirs         = make(map[string]bool)
	ignoreFiles        = make(map[string]bool)
	maxLogLines        int
	maxFileBytes       int64
	tokenLimit         int
)

func main() {
	loadConfiguration()

	previewMode := flag.Bool("preview", false, "Preview file structure and list without copying")
	pShort := flag.Bool("p", false, "Preview alias")

	structureMode := flag.Bool("structure", false, "Copy only the project structure tree to clipboard")
	sShort := flag.Bool("s", false, "Structure alias")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	isPreview := *previewMode || *pShort
	isStructure := *structureMode || *sShort

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

	if isStructure {
		fmt.Println("🌳 Scanning project structure...")
		tree, count, err := generateProjectTree()
		if err != nil {
			fmt.Printf("❌ Error scanning files: %v\n", err)
			return
		}

		output := "Here is the project structure:\n" +
			"<project_structure>\n" +
			tree +
			"</project_structure>\n"

		estTokens := estimateTokens(output)

		err = copyToClipboard(output)
		if err != nil {
			fmt.Printf("❌ Error copying to clipboard: %v\n", err)
		} else {
			fmt.Println("✅ Success! Project structure copied to clipboard. (Ctrl+V)")
			fmt.Printf("📄 Tree contains %d files/folders.\n", count)
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

	fmt.Println("Scanning project files...")

	fullContext, count, err := generateXMLContext()
	if err != nil {
		fmt.Printf("❌ Error scanning files: %v\n", err)
		return
	}

	estTokens := estimateTokens(fullContext)

	err = copyToClipboard(fullContext)
	if err != nil {
		fmt.Printf("❌ Error copying to clipboard: %v\n", err)
	} else {
		fmt.Println("✅ Success! Context copied to clipboard. (Ctrl+V)")
		fmt.Printf("📄 Aggregated %d documents.\n", count)
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

func loadConfiguration() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("⚠️ Could not determine executable path. Using defaults.")
		applyConfig(defaultConfig())
		return
	}

	loadConfigurationFromPath(filepath.Join(filepath.Dir(exePath), "config.json"))
}

func loadConfigurationFromPath(configPath string) {
	config := defaultConfig()

	fileBytes, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("⚠️ Error reading config.json: %v. Using defaults.\n", err)
		}
		applyConfig(config)
		return
	}

	cleanBytes := stripJSONComments(fileBytes)

	if err := json.Unmarshal(cleanBytes, &config); err != nil {
		fmt.Printf("⚠️ Error parsing config.json: %v. Using defaults.\n", err)
		applyConfig(defaultConfig())
		return
	}

	defaults := defaultConfig()
	if config.MaxLogLines <= 0 {
		config.MaxLogLines = defaults.MaxLogLines
	}
	if config.MaxFileBytes <= 0 {
		config.MaxFileBytes = defaults.MaxFileBytes
	}
	if config.TokenLimit <= 0 {
		config.TokenLimit = defaults.TokenLimit
	}

	fmt.Printf("⚙️ Loaded configuration from %s\n", configPath)
	applyConfig(config)
}

func defaultConfig() Config {
	return Config{
		TokenLimit:         128000,
		IncludedExtensions: []string{".md", ".json", ".yaml", ".yml", ".txt", ".py", ".cs", ".go", ".toml", ".mod", ".bat", ".ps1", ".js", ".ts", ".tsx", ".css"},
		LogExtensions:      []string{".txt", ".log", ".out", ".err"},
		MaxLogLines:        200,
		MaxFileBytes:       2 * 1024 * 1024,
		IgnoreDirs:         []string{".git", ".idea", ".vscode", "__pycache__", "venv", "env", "vendor", "bin", "build", "dist", "node_modules", ".next", "out", ".obsidian", ".trash", ".stfolder", ".stversions", ".ssh", ".aws", ".azure", ".kube", ".gnupg"},
		IgnoreFiles:        []string{"go.sum", "bundled.go", "resource.go", "package-lock.json", "yarn.lock", ".env", ".env.local", ".env.development", ".env.production", ".env.test", ".npmrc", ".pypirc", ".netrc", "id_rsa", "id_ed25519", "authorized_keys"},
	}
}

func applyConfig(config Config) {
	resetLookups()
	populateMap(includedExtensions, config.IncludedExtensions)
	populateMap(logExtensions, config.LogExtensions)
	populateMap(ignoreDirs, config.IgnoreDirs)
	populateMap(ignoreFiles, config.IgnoreFiles)
	maxLogLines = config.MaxLogLines
	maxFileBytes = config.MaxFileBytes
	tokenLimit = config.TokenLimit
}

func resetLookups() {
	clear(includedExtensions)
	clear(logExtensions)
	clear(ignoreDirs)
	clear(ignoreFiles)
}

func populateMap(target map[string]bool, values []string) {
	for _, value := range values {
		target[strings.TrimSpace(value)] = true
	}
}

func stripJSONComments(data []byte) []byte {
	re := regexp.MustCompile(`(?m)\s*//.*$`)
	return re.ReplaceAll(data, nil)
}

func estimateTokens(content string) int {
	return len(content) / 4
}

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
		if shouldSkipDir(d) {
			return filepath.SkipDir
		}
		if shouldSkipEntry(d) {
			return nil
		}

		relPath, relErr := filepath.Rel(rootDir, path)
		if relErr != nil || relPath == "." {
			return relErr
		}

		depth := strings.Count(relPath, string(os.PathSeparator))
		indent := strings.Repeat("    ", depth)

		if d.IsDir() {
			output.WriteString(fmt.Sprintf("%s%s/\n", indent, d.Name()))
			return nil
		}

		if shouldIncludeFile(d.Name()) {
			output.WriteString(fmt.Sprintf("%s%s\n", indent, d.Name()))
			fileCount++
		}
		return nil
	})

	return output.String(), fileCount, err
}

func generateXMLContext() (string, int, error) {
	var output strings.Builder

	treeStructure, _, err := generateProjectTree()
	if err != nil {
		return "", 0, err
	}

	output.WriteString("Here is the file structure and content:\n")
	output.WriteString("<project_structure>\n")
	output.WriteString(treeStructure)
	output.WriteString("</project_structure>\n")

	var fileCount int
	rootDir, err := os.Getwd()
	if err != nil {
		return "", 0, err
	}

	err = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if shouldSkipDir(d) {
			return filepath.SkipDir
		}
		if d.IsDir() || shouldSkipEntry(d) || !shouldIncludeFile(d.Name()) {
			return nil
		}

		content, readErr := readIncludedFile(path, filepath.Ext(d.Name()))
		if readErr != nil {
			fmt.Printf("Skipping %s: %v\n", d.Name(), readErr)
			return nil
		}

		relPath, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			return relErr
		}

		output.WriteString(fmt.Sprintf(
			"<file name=\"%s\">\n%s\n</file>\n",
			xmlEscapeAttribute(filepath.ToSlash(relPath)),
			serializeFileBody(content),
		))
		fileCount++
		return nil
	})

	return output.String(), fileCount, err
}

func shouldSkipDir(d os.DirEntry) bool {
	return d.IsDir() && ignoreDirs[d.Name()]
}

func shouldSkipEntry(d os.DirEntry) bool {
	return d.Type()&os.ModeSymlink != 0
}

func shouldIncludeFile(name string) bool {
	return includedExtensions[filepath.Ext(name)] && !ignoreFiles[name]
}

func readIncludedFile(path, ext string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if maxFileBytes > 0 && info.Size() > maxFileBytes {
		return fmt.Sprintf("[SKIPPED: File exceeded %d bytes]", maxFileBytes), nil
	}

	if logExtensions[ext] {
		return readTruncated(path, maxLogLines)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	if bytes.IndexByte(b, 0) >= 0 || !utf8.Valid(b) {
		return "[SKIPPED: File appears to be binary or non-UTF-8 text]", nil
	}

	return string(b), nil
}

func xmlEscapeAttribute(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(value)
}

func serializeFileBody(value string) string {
	// Split embedded CDATA terminators so file content stays readable.
	parts := strings.Split(value, "]]>")
	if len(parts) == 1 {
		return "<![CDATA[" + value + "]]>"
	}

	var output bytes.Buffer
	for i, part := range parts {
		output.WriteString("<![CDATA[")
		output.WriteString(part)
		output.WriteString("]]>")

		if i < len(parts)-1 {
			output.WriteString("<![CDATA[>]]>")
		}
	}

	return output.String()
}
