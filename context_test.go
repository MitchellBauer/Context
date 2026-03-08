package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigurationFromPathUsesDefaultsWhenMissing(t *testing.T) {
	loadConfigurationFromPath(filepath.Join(t.TempDir(), "missing-config.json"))

	if !includedExtensions[".go"] {
		t.Fatal("expected default .go extension to be included")
	}
	if !ignoreDirs[".ssh"] {
		t.Fatal("expected sensitive directories to be ignored by default")
	}
	if !ignoreFiles[".env"] {
		t.Fatal("expected .env to be ignored by default")
	}
	if maxLogLines != 200 {
		t.Fatalf("expected default maxLogLines 200, got %d", maxLogLines)
	}
	if maxFileBytes != 2*1024*1024 {
		t.Fatalf("expected default maxFileBytes, got %d", maxFileBytes)
	}
}

func TestLoadConfigurationFromPathResetsMapsAndReadsJSONC(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	config := `{
  "token_limit": 42,
  "included_extensions": [".txt"],
  "log_extensions": [".txt"],
  "max_log_lines": 3,
  "max_file_bytes": 64,
  "ignore_dirs": ["cache"],
  "ignore_files": ["secret.txt"] // comment
}`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	loadConfigurationFromPath(configPath)

	if includedExtensions[".go"] {
		t.Fatal("expected config reload to clear previous extension values")
	}
	if !includedExtensions[".txt"] {
		t.Fatal("expected configured extension to be included")
	}
	if !ignoreDirs["cache"] || !ignoreFiles["secret.txt"] {
		t.Fatal("expected configured ignore rules to apply")
	}
	if maxLogLines != 3 || maxFileBytes != 64 || tokenLimit != 42 {
		t.Fatalf("unexpected config values: maxLogLines=%d maxFileBytes=%d tokenLimit=%d", maxLogLines, maxFileBytes, tokenLimit)
	}
}

func TestGenerateXMLContextUsesReadableCDATAAndSkips(t *testing.T) {
	applyConfig(defaultConfig())
	root := t.TempDir()

	readableContent := strings.Join([]string{
		"# Title 😄",
		"",
		"\t- tabbed bullet with <tag> & \"quotes\" and 'apostrophe'",
		"Literal CDATA marker ]]> stays whole.",
		"smart ’ dash — accent café",
	}, "\n") + "\n"

	mustWriteFile(t, filepath.Join(root, "safe.md"), readableContent)
	mustWriteFile(t, filepath.Join(root, "package-lock.json"), "SECRET=1\n")
	mustWriteFile(t, filepath.Join(root, "big.go"), strings.Repeat("a", int(maxFileBytes)+1))
	mustWriteFile(t, filepath.Join(root, "binary.go"), string([]byte{0x00, 0x01, 0x02}))
	mustWriteFile(t, filepath.Join(root, "a&b.go"), "package main\n")
	mustWriteFile(t, filepath.Join(root, ".ssh", "config.txt"), "Host *\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer os.Chdir(oldWD)
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	tree, treeCount, err := generateProjectTree()
	if err != nil {
		t.Fatalf("generateProjectTree: %v", err)
	}
	if treeCount != 4 {
		t.Fatalf("expected 4 included files in tree, got %d", treeCount)
	}
	if strings.Contains(tree, ".ssh") || strings.Contains(tree, "package-lock.json") {
		t.Fatal("expected ignored files and directories to be absent from the tree")
	}

	xmlContext, fileCount, err := generateXMLContext()
	if err != nil {
		t.Fatalf("generateXMLContext: %v", err)
	}
	if fileCount != 4 {
		t.Fatalf("expected 4 included documents, got %d", fileCount)
	}
	if !strings.Contains(xmlContext, `name="a&amp;b.go"`) {
		t.Fatal("expected file names to be XML-escaped")
	}
	if strings.Contains(xmlContext, "&#xA;") || strings.Contains(xmlContext, "&#x9;") {
		t.Fatal("expected literal newlines and tabs in file bodies")
	}
	if !strings.Contains(xmlContext, "<![CDATA[# Title 😄") {
		t.Fatal("expected file bodies to be wrapped in CDATA")
	}
	if !strings.Contains(xmlContext, "\t- tabbed bullet with <tag> & \"quotes\" and 'apostrophe'") {
		t.Fatal("expected file contents to remain readable and unescaped")
	}
	if !strings.Contains(xmlContext, "smart ’ dash — accent café") {
		t.Fatal("expected unicode punctuation and emoji to remain readable")
	}
	if !strings.Contains(xmlContext, "Literal CDATA marker ]]><![CDATA[>]]><![CDATA[ stays whole.") {
		t.Fatal("expected embedded CDATA terminators to be split safely")
	}
	if !strings.Contains(xmlContext, "[SKIPPED: File exceeded") {
		t.Fatal("expected oversized files to be represented by a skip marker")
	}
	if !strings.Contains(xmlContext, "[SKIPPED: File appears to be binary or non-UTF-8 text]") {
		t.Fatal("expected binary-like files to be represented by a skip marker")
	}
	if strings.Contains(xmlContext, ".ssh/config.txt") || strings.Contains(xmlContext, "package-lock.json") {
		t.Fatal("expected ignored local-sensitive content to stay out of XML output")
	}
}

func TestSerializeFileBodySplitsCDATAEndMarkers(t *testing.T) {
	got := serializeFileBody("alpha ]]> beta")
	want := "<![CDATA[alpha ]]><![CDATA[>]]><![CDATA[ beta]]>"
	if got != want {
		t.Fatalf("unexpected CDATA split\nwant: %q\n got: %q", want, got)
	}
}

func TestReadTruncated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	mustWriteFile(t, path, "one\ntwo\nthree\n")

	content, err := readTruncated(path, 2)
	if err != nil {
		t.Fatalf("readTruncated: %v", err)
	}
	if !strings.Contains(content, "one\ntwo\n") || !strings.Contains(content, "TRUNCATED") {
		t.Fatal("expected truncated output to contain the first lines and a truncation marker")
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
