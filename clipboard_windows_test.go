//go:build windows

package main

import (
	"testing"
	"unicode/utf16"
)

func TestUTF16BufferFromStringPreservesUnicode(t *testing.T) {
	text := "emoji 😄 smart ’ dash — accent café"
	buffer, err := utf16BufferFromString(text)
	if err != nil {
		t.Fatalf("utf16BufferFromString: %v", err)
	}
	if len(buffer) == 0 || buffer[len(buffer)-1] != 0 {
		t.Fatal("expected a null-terminated UTF-16 buffer")
	}

	decoded := string(utf16.Decode(buffer[:len(buffer)-1]))
	if decoded != text {
		t.Fatalf("unicode round-trip mismatch\nwant: %q\n got: %q", text, decoded)
	}
}
