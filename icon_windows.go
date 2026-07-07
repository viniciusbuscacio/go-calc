//go:build windows

package main

import (
	"os"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	shell32          = syscall.NewLazyDLL("shell32.dll")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procFindWindowW  = user32.NewProc("FindWindowW")
	procSendMessageW = user32.NewProc("SendMessageW")
	procExtractIconW = shell32.NewProc("ExtractIconW")
	procGetModuleH   = kernel32.NewProc("GetModuleHandleW")
)

const (
	wmSetIcon = 0x0080
	iconSmall = 0
	iconBig   = 1
)

// fixTaskbarIcon sets the window's *large* icon so the Windows taskbar shows
// the app icon. Wails only sets the small icon on frameless windows, which
// leaves ICON_BIG empty and the taskbar button blank on some Windows builds.
// It polls for the window (created shortly after startup), extracts the icon
// embedded in our own exe, and applies it.
func fixTaskbarIcon(title string) {
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exePtr, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return
	}
	hInst, _, _ := procGetModuleH.Call(0)

	for i := 0; i < 50; i++ { // up to ~5s
		hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
		if hwnd != 0 {
			hIcon, _, _ := procExtractIconW.Call(hInst, uintptr(unsafe.Pointer(exePtr)), 0)
			// ExtractIcon returns 0 (no icon) or 1 (bad index) on failure.
			if hIcon != 0 && hIcon != 1 {
				procSendMessageW.Call(hwnd, wmSetIcon, iconBig, hIcon)
				procSendMessageW.Call(hwnd, wmSetIcon, iconSmall, hIcon)
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
