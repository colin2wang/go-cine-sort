// Package media implements video file tag (title/comment) editing and the
// naming-convention renamer ported from the org-tools/media Python module.
package media

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VideoExtensions lists the video file extensions handled by the renamer.
var VideoExtensions = map[string]bool{
	".mp4": true, ".mkv": true, ".avi": true, ".mov": true, ".flv": true,
	".wmv": true, ".mpeg": true, ".mpg": true, ".m4v": true, ".webm": true,
	".ts": true,
}

// videoTags are technical tokens that terminate the title part of a filename.
var videoTags = map[string]bool{
	"xxx": true, "720p": true, "1080p": true, "hevc": true, "x265": true,
	"prt": true, "mp4": true, "mkv": true,
}

// RenamePlan describes one file rename following the naming convention.
type RenamePlan struct {
	Filename string `json:"filename"` // current file name (with extension)
	NewName  string `json:"newName"`  // planned name (with extension); equals Filename when unmatched
	Dir      string `json:"dir"`
	Matched  bool   `json:"matched"`
}

// ParseConventionName parses a filename following the
// "Word.Words.YY.MM.DX..." convention and returns "Title Words (YY.MM.DD)".
// It returns an empty string when the name does not match the convention.
func ParseConventionName(filename string) string {
	base := filename
	if ext := filepath.Ext(filename); ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	parts := strings.Split(base, ".")
	if len(parts) < 5 {
		return ""
	}

	datePart := strings.Join(parts[1:4], ".")
	titleParts := make([]string, 0, len(parts))
	for _, part := range parts[4:] {
		if videoTags[strings.ToLower(part)] {
			break
		}
		titleParts = append(titleParts, part)
	}
	if len(titleParts) == 0 {
		return ""
	}

	words := make([]string, len(titleParts))
	for i, w := range titleParts {
		words[i] = capitalize(w)
	}
	return fmt.Sprintf("%s (%s)", strings.Join(words, " "), datePart)
}

func capitalize(w string) string {
	if w == "" {
		return w
	}
	return strings.ToUpper(w[:1]) + w[1:]
}

// RenameFile renames one file in dir according to the naming convention.
// It returns the resulting name (unchanged when unmatched) and whether the
// rename actually happened.
func RenameFile(dir, filename string) (result string, renamed bool, err error) {
	newBase := ParseConventionName(filename)
	if newBase == "" {
		return filename, false, nil
	}
	ext := filepath.Ext(filename)
	newName := newBase + ext
	if newName == filename {
		return filename, false, nil
	}
	if _, e := os.Stat(filepath.Join(dir, newName)); e == nil {
		return filename, false, fmt.Errorf("目标文件已存在：%s", newName)
	}
	if e := os.Rename(filepath.Join(dir, filename), filepath.Join(dir, newName)); e != nil {
		return filename, false, e
	}
	return newName, true, nil
}
