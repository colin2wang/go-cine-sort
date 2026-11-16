//go:build windows

package folder

import (
	"net/url"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procIsIconic            = user32.NewProc("IsIconic")
	procShowWindow          = user32.NewProc("ShowWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
)

const (
	swShow              = 5
	swRestore           = 9
	coApartmentThreaded = 0x2
)

// openWindows focuses an existing Explorer window for dir, falling back to
// spawning a new one. The ShellWindows COM enumeration works the same on
// Windows 10 and Windows 11.
func openWindows(dir string) error {
	if focusExistingExplorerWindow(dir) {
		return nil
	}
	return exec.Command("explorer.exe", dir).Start()
}

// focusExistingExplorerWindow walks all open Explorer/Shell windows and
// activates the one whose LocationURL matches dir. Returns false when no
// match is found (or COM is unavailable), so the caller opens a new window.
func focusExistingExplorerWindow(dir string) bool {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ole.CoInitializeEx(0, coApartmentThreaded); err != nil {
		return false
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("Shell.Application")
	if err != nil {
		return false
	}
	defer unknown.Release()

	shell, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return false
	}
	defer shell.Release()

	windowsVal, err := oleutil.CallMethod(shell, "Windows")
	if err != nil {
		return false
	}
	windows := windowsVal.ToIDispatch()
	defer windows.Release()

	countVal, err := oleutil.GetProperty(windows, "Count")
	if err != nil {
		return false
	}
	count := int(countVal.Val)

	target := normalizeDir(dir)
	for i := 0; i < count; i++ {
		itemVal, err := oleutil.CallMethod(windows, "Item", i)
		if err != nil {
			continue
		}
		item := itemVal.ToIDispatch()
		locVal, err := oleutil.GetProperty(item, "LocationURL")
		if err != nil {
			item.Release()
			continue
		}
		windowDir, ok := dirFromLocationURL(locVal.ToString())
		if !ok || windowDir != target {
			item.Release()
			continue
		}
		if hwndVal, err := oleutil.GetProperty(item, "HWND"); err == nil {
			activateWindow(hwndVal.Val)
		}
		item.Release()
		return true
	}
	return false
}

// dirFromLocationURL converts an Explorer LocationURL ("file:///D:/a/b") into
// a normalized directory path. Non-file URLs (e.g. HTTP views) never match.
func dirFromLocationURL(loc string) (string, bool) {
	loc = strings.TrimSpace(loc)
	if loc == "" {
		return "", false
	}
	u, err := url.Parse(loc)
	if err != nil || u.Scheme != "file" {
		return "", false
	}
	p, err := url.PathUnescape(u.Path)
	if err != nil || p == "" {
		return "", false
	}
	// file:///D:/x yields "/D:/x"; strip the leading slash of the host part.
	p = strings.TrimPrefix(p, "/")
	p = filepath.FromSlash(p)
	return normalizeDir(p), true
}

// normalizeDir canonicalizes a directory path for comparison: absolute,
// backslash-separated, lowercased (Windows paths are case-insensitive) and
// without a trailing separator.
func normalizeDir(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	abs = strings.ReplaceAll(abs, "/", "\\")
	abs = strings.TrimRight(abs, "\\")
	return strings.ToLower(abs)
}

// activateWindow restores a minimized window or brings it to the front.
func activateWindow(hwnd int64) {
	h := uintptr(hwnd)
	if iconic, _, _ := procIsIconic.Call(h); iconic != 0 {
		_, _, _ = procShowWindow.Call(h, swRestore)
	} else {
		_, _, _ = procShowWindow.Call(h, swShow)
	}
	_, _, _ = procSetForegroundWindow.Call(h)
}
