//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalFree       = kernel32.NewProc("GlobalFree")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
)

func copyToClipboard(text string) error {
	return copyToWindowsClipboard(text)
}

func copyToWindowsClipboard(text string) error {
	utf16Text, err := utf16BufferFromString(text)
	if err != nil {
		return err
	}

	size := uintptr(len(utf16Text) * 2)
	hMem, _, allocErr := procGlobalAlloc.Call(gmemMoveable, size)
	if hMem == 0 {
		return fmt.Errorf("global alloc failed: %w", allocErr)
	}

	handOff := false
	defer func() {
		if !handOff {
			procGlobalFree.Call(hMem)
		}
	}()

	ptr, _, lockErr := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return fmt.Errorf("global lock failed: %w", lockErr)
	}

	copy(unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), len(utf16Text)), utf16Text)
	procGlobalUnlock.Call(hMem)

	if err := openClipboardWithRetry(); err != nil {
		return err
	}
	defer procCloseClipboard.Call()

	if r, _, emptyErr := procEmptyClipboard.Call(); r == 0 {
		return fmt.Errorf("empty clipboard failed: %w", emptyErr)
	}
	if r, _, setErr := procSetClipboardData.Call(cfUnicodeText, hMem); r == 0 {
		return fmt.Errorf("set clipboard data failed: %w", setErr)
	}

	handOff = true
	return nil
}

func utf16BufferFromString(text string) ([]uint16, error) {
	utf16Text, err := syscall.UTF16FromString(text)
	if err != nil {
		return nil, fmt.Errorf("utf16 conversion failed: %w", err)
	}
	return utf16Text, nil
}

func openClipboardWithRetry() error {
	var lastErr error
	for range 10 {
		if r, _, err := procOpenClipboard.Call(0); r != 0 {
			return nil
		} else if err != syscall.Errno(0) {
			lastErr = err
		}
		time.Sleep(20 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = os.ErrPermission
	}
	return fmt.Errorf("open clipboard failed: %w", lastErr)
}
