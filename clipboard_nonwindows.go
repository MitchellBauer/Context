//go:build !windows

package main

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
)

func copyToClipboard(text string) error {
	switch runtime.GOOS {
	case "darwin":
		return copyToCommandClipboard(text, exec.Command("pbcopy"))
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			return copyToCommandClipboard(text, exec.Command("xclip", "-selection", "clipboard"))
		}
		if _, err := exec.LookPath("xsel"); err == nil {
			return copyToCommandClipboard(text, exec.Command("xsel", "--clipboard", "--input"))
		}
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return copyToCommandClipboard(text, exec.Command("wl-copy"))
		}
		return fmt.Errorf("no clipboard tool found (install xclip, xsel, or wl-copy)")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func copyToCommandClipboard(text string, cmd *exec.Cmd) error {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if _, err := io.WriteString(stdin, text); err != nil {
		stdin.Close()
		cmd.Wait()
		return err
	}
	if err := stdin.Close(); err != nil {
		cmd.Wait()
		return err
	}
	return cmd.Wait()
}
