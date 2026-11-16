// Package folder opens directories in the OS file browser. On Windows it
// reuses an existing Explorer window that already shows the directory instead
// of spawning a new one on every call.
package folder

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Open reveals the directory in the platform's file browser. It returns at
// once after handing the request off to the OS.
func Open(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return errors.New("路径为空")
	}
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("路径不存在: %w", err)
	}
	switch runtime.GOOS {
	case "windows":
		return openWindows(dir)
	case "darwin":
		return exec.Command("open", dir).Start()
	default:
		return exec.Command("xdg-open", dir).Start()
	}
}
