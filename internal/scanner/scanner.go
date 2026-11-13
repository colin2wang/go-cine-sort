// Package scanner discovers video files and extracts a clean movie name and
// release year from messy filenames. The pipeline mirrors Python's
// utils/movie_file_util.py, with Unicode-aware adaptations for Go's RE2.
package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"go-cine-sort/internal/config"
	"go-cine-sort/internal/logging"
	"go-cine-sort/internal/model"
)

var techIndicators = []string{"1080", "720", "4K", "HD", "BD", "DVD", "X264", "H264", "AAC"}
var noiseSegments = map[string]bool{"电影": true, "最新": true, "下载": true}

type Scanner struct {
	cfg   *config.AppConfig
	rules *config.CompiledRules
	log   *logging.Logger
}

func New(cfg *config.AppConfig, log *logging.Logger) (*Scanner, error) {
	rules, err := config.CompileRules(cfg)
	if err != nil {
		return nil, err
	}
	return &Scanner{cfg: cfg, rules: rules, log: log}, nil
}

// Refresh recompiles rules after the configuration changes.
func (s *Scanner) Refresh(cfg *config.AppConfig) error {
	rules, err := config.CompileRules(cfg)
	if err != nil {
		return err
	}
	s.cfg = cfg
	s.rules = rules
	return nil
}

func (s *Scanner) accepts(ext string) bool {
	return s.rules.Extensions[strings.ToLower(ext)]
}

// isOrganizedDir detects directories previously produced by this app by
// looking for the numeric ".txt" metadata files (rating / SID / IMDb rating).
func isOrganizedDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".txt") {
			continue
		}
		stem := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		stem = strings.TrimPrefix(stem, "IMDB ")
		stem = strings.TrimSpace(stem)
		if stem == "sid" {
			return true
		}
		if _, err := strconv.ParseFloat(stem, 64); err == nil {
			return true
		}
		if len(stem) >= 6 && onlyDigits(stem) {
			return true
		}
	}
	return false
}

func onlyDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ScanDirectory walks dir for video files. Directories already organized by
// the app are skipped so repeated runs do not re-fetch the same movies.
func (s *Scanner) ScanDirectory(ctx context.Context, dir string, recursive bool) ([]model.MovieFileInfo, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("目录不可用: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("路径不是目录: %s", dir)
	}

	var paths []string
	if !recursive {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("读取目录: %w", err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				paths = append(paths, filepath.Join(dir, e.Name()))
			}
		}
	} else {
		err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				s.log.Warn("scanner", "跳过无法访问的 %s: %v", path, walkErr)
				return nil
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if d.IsDir() {
				if path != dir && isOrganizedDir(path) {
					s.log.Debug("scanner", "跳过已整理目录 %s", path)
					return filepath.SkipDir
				}
				return nil
			}
			paths = append(paths, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	results := s.InfoForPaths(paths)
	s.log.Info("scanner", "扫描完成，共发现 %d 个视频文件", len(results))
	return results, nil
}

// InfoForPaths parses an explicit list of files (used when the user selects
// individual rows in the workbench).
func (s *Scanner) InfoForPaths(paths []string) []model.MovieFileInfo {
	out := make([]model.MovieFileInfo, 0, len(paths))
	for _, path := range paths {
		info, err := s.ExtractMovieInfo(path)
		if err != nil {
			s.log.Warn("scanner", "解析失败 %s: %v", filepath.Base(path), err)
			continue
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RawFilename < out[j].RawFilename })
	return out
}

func (s *Scanner) ExtractMovieInfo(path string) (model.MovieFileInfo, error) {
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)

	year := s.ExtractYear(stem)
	movieName := s.CleanMovieName(stem)
	if movieName == "" {
		return model.MovieFileInfo{}, fmt.Errorf("无法从文件名提取片名")
	}
	return model.MovieFileInfo{
		FilePath:    path,
		MovieName:   movieName,
		Year:        year,
		Extension:   strings.ToLower(ext),
		RawFilename: name,
	}, nil
}

func isUnicodeWord(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}

// removeStandaloneYears reproduces Python's Unicode-aware \b semantics.
// Go's RE2 \b is ASCII-only, so "我的1919" would otherwise lose the "1919".
func removeStandaloneYears(text string, re *regexp.Regexp) string {
	matches := re.FindAllStringIndex(text, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		start, end := matches[i][0], matches[i][1]
		before, _ := utf8.DecodeLastRuneInString(text[:start])
		after, _ := utf8.DecodeRuneInString(text[end:])
		if isUnicodeWord(before) || isUnicodeWord(after) {
			continue
		}
		text = text[:start] + " " + text[end:]
	}
	return text
}

// CleanMovieName implements the Python 4-step pipeline, plus a Unicode-aware
// pre-pass required because Go's RE2 \b treats Chinese characters as non-word.
func (s *Scanner) CleanMovieName(filename string) string {
	processed := filename

	// Step 0a: site markers together with their domain suffix.
	if s.rules.SiteMarkers != nil {
		processed = s.rules.SiteMarkers.ReplaceAllString(processed, " ")
	}
	// Step 0b: remove whole Chinese-tagged segments before ASCII keywords can
	// split them (e.g. "HD国语中字无水印" must be removed as one unit).
	if s.rules.TagSegments != nil {
		processed = s.rules.TagSegments.ReplaceAllString(processed, " ")
	}
	// Step 1: single-pass removal of all remaining cleanup patterns.
	if s.rules.Cleanup != nil {
		processed = s.rules.Cleanup.ReplaceAllString(processed, " ")
	}
	// Step 2: normalize separators and whitespace.
	processed = s.rules.Separators.ReplaceAllString(processed, " ")
	processed = strings.TrimSpace(s.rules.Spaces.ReplaceAllString(processed, " "))
	// Step 3: remove isolated residual years.
	processed = removeStandaloneYears(processed, s.rules.Year)
	processed = strings.TrimSpace(s.rules.Spaces.ReplaceAllString(processed, " "))

	// Step 4: drop isolated single characters but keep single Chinese/letter
	// titles (2026-06-07 Python fix, e.g. 「腿」).
	words := strings.Fields(processed)
	kept := make([]string, 0, len(words))
	for _, w := range words {
		runes := []rune(w)
		if len(runes) > 1 || (len(runes) == 1 && unicode.IsLetter(runes[0])) {
			kept = append(kept, w)
		}
	}
	result := strings.Join(kept, " ")

	if result == "" {
		segments := s.rules.ChineseSegments.FindAllString(filename, -1)
		fallback := make([]string, 0, len(segments))
		for _, seg := range segments {
			if !noiseSegments[seg] {
				fallback = append(fallback, seg)
			}
		}
		if len(fallback) > 0 {
			result = strings.Join(fallback, " ")
		} else {
			result = s.rules.NonWordOrSpace.ReplaceAllString(filename, " ")
			result = strings.TrimSpace(s.rules.Spaces.ReplaceAllString(result, " "))
		}
	}
	return result
}

type yearCandidate struct {
	year  string
	score float64
	start int
}

// ExtractYear uses context scoring identical to the Python implementation.
// Rune indices are used so Chinese filenames score the same way as Python's
// character-based string indices.
func (s *Scanner) ExtractYear(filename string) string {
	matches := s.rules.Year.FindAllStringIndex(filename, -1)
	if len(matches) == 0 {
		return ""
	}
	runes := []rune(filename)
	total := len(runes)
	candidates := make([]yearCandidate, 0, len(matches))

	for _, m := range matches {
		year := filename[m[0]:m[1]]
		n, err := strconv.Atoi(year)
		if err != nil || n < 1900 || n > 2030 {
			continue
		}
		startRune := utf8.RuneCountInString(filename[:m[0]])
		endRune := utf8.RuneCountInString(filename[:m[1]])
		lo := startRune - 10
		if lo < 0 {
			lo = 0
		}
		hi := endRune + 10
		if hi > total {
			hi = total
		}
		context := strings.ToUpper(string(runes[lo:startRune]) + string(runes[endRune:hi]))

		score := 0.0
		for _, ind := range techIndicators {
			if strings.Contains(context, ind) {
				score += 2
				break
			}
		}
		score += (float64(startRune) / float64(total)) * 0.5
		candidates = append(candidates, yearCandidate{year: year, score: score, start: startRune})
	}
	if len(candidates) == 0 {
		return ""
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].start > candidates[j].start
	})
	return candidates[0].year
}
