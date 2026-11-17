// Package youtube implements the YouTube archive sorter: files named
// "<uploader> - <title> (...).<ext>" inside a source directory are grouped by
// uploader and moved into <targetBase>/<uploader>/ subdirectories.
package youtube

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// uploaderPattern matches the "uploader - " prefix of a yt-dlp style filename.
// Example: SomeChannel - Video Title (20240101) [1080p].mp4
var uploaderPattern = regexp.MustCompile(`^(?P<uploader>.+?) - `)

// MovePlan describes one file that would be moved.
type MovePlan struct {
	Filename  string `json:"filename"`
	SourceDir string `json:"sourceDir"`
	Uploader  string `json:"uploader"`
	DestDir   string `json:"destDir"`
	DestPath  string `json:"destPath"`
}

// PlanResult is the outcome of a preview scan.
type PlanResult struct {
	Moves     []MovePlan `json:"moves"`
	Conflicts []string   `json:"conflicts"`
	Unmatched []string   `json:"unmatched"`
}

// ConflictAction decides what happens when the destination file already exists.
type ConflictAction string

const (
	ConflictSkip      ConflictAction = "skip"
	ConflictOverwrite ConflictAction = "overwrite"
)

// ParseUploader extracts the uploader segment from a filename. It returns an
// empty string when the name does not follow the "<uploader> - " convention.
func ParseUploader(filename string) string {
	m := uploaderPattern.FindStringSubmatch(filename)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// Plan scans sourceDir and classifies every regular file into moves,
// destination conflicts and names that do not match the convention.
func Plan(sourceDir, targetBase string) (*PlanResult, error) {
	sourceDir = strings.TrimSpace(sourceDir)
	targetBase = strings.TrimSpace(targetBase)
	if sourceDir == "" || targetBase == "" {
		return nil, fmt.Errorf("源目录和目标目录不能为空")
	}
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("读取源目录失败: %w", err)
	}

	result := &PlanResult{Moves: []MovePlan{}, Conflicts: []string{}, Unmatched: []string{}}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		uploader := ParseUploader(name)
		if uploader == "" {
			result.Unmatched = append(result.Unmatched, name)
			continue
		}
		plan := MovePlan{
			Filename:  name,
			SourceDir: sourceDir,
			Uploader:  uploader,
			DestDir:   filepath.Join(targetBase, uploader),
			DestPath:  filepath.Join(targetBase, uploader, name),
		}
		result.Moves = append(result.Moves, plan)
		if _, err := os.Stat(plan.DestPath); err == nil {
			result.Conflicts = append(result.Conflicts, name)
		}
	}
	return result, nil
}

// ProgressFunc receives each finished file with its error ("" on success).
type ProgressFunc func(index, total int, plan MovePlan, moveErr string)

// Sort moves the planned files. Conflicting destinations are resolved with the
// given action: skipped or overwritten. Directories are created on demand.
func Sort(moves []MovePlan, conflicts []string, action ConflictAction, onProgress ProgressFunc) (success, skipped, failed int, err error) {
	conflictSet := make(map[string]bool, len(conflicts))
	for _, c := range conflicts {
		conflictSet[c] = true
	}

	total := len(moves)
	for i, plan := range moves {
		moveErr := ""
		if conflictSet[plan.Filename] && action == ConflictSkip {
			skipped++
			moveErr = "目标已存在，已跳过"
		} else {
			moveErr = move(plan)
			if moveErr == "" {
				success++
			} else {
				failed++
			}
		}
		if onProgress != nil {
			onProgress(i+1, total, plan, moveErr)
		}
	}
	return success, skipped, failed, nil
}

func move(plan MovePlan) string {
	if err := os.MkdirAll(plan.DestDir, 0o755); err != nil {
		return fmt.Sprintf("创建目录失败: %v", err)
	}
	// os.Rename fails across volumes; fall back to copy+delete.
	err := os.Rename(plan.SourceDir+string(os.PathSeparator)+plan.Filename, plan.DestPath)
	if err == nil {
		return ""
	}
	if linkErr := copyFile(plan.SourceDir+string(os.PathSeparator)+plan.Filename, plan.DestPath); linkErr != nil {
		return fmt.Sprintf("移动失败: %v", linkErr)
	}
	if err := os.Remove(plan.SourceDir + string(os.PathSeparator) + plan.Filename); err != nil {
		return fmt.Sprintf("复制成功但删除源文件失败: %v", err)
	}
	return ""
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".cinesort-tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
