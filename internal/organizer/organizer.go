// Package organizer creates the structured movie directory and its metadata
// files. Directory naming rules are kept identical to the Python tool so
// previously organized libraries remain compatible.
package organizer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"go-cine-sort/internal/config"
	"go-cine-sort/internal/logging"
	"go-cine-sort/internal/model"
)

var (
	reInvalidChars  = regexp.MustCompile(`[<>:"/\\|?*]`)
	reMultiDots     = regexp.MustCompile(`\.{2,}`)
	reEnglishInName = regexp.MustCompile(`[a-zA-Z]{2,}`)
)

// SimpleInfo is the minimal input for CreateSimple (mirrors Python's
// create_movie_directory and is used by the naming unit tests).
type SimpleInfo struct {
	Title         string
	OriginalTitle string
	Year          string
	SID           string
	Rating        string
}

type Organizer struct {
	cfg *config.AppConfig
	log *logging.Logger
}

func New(cfg *config.AppConfig, log *logging.Logger) *Organizer {
	return &Organizer{cfg: cfg, log: log}
}

func (o *Organizer) SetConfig(cfg *config.AppConfig) { o.cfg = cfg }

// CleanFilename replaces Windows-invalid characters and spaces with dots.
func CleanFilename(name string) string {
	if name == "" {
		return ""
	}
	cleaned := reInvalidChars.ReplaceAllString(name, ".")
	cleaned = strings.ReplaceAll(cleaned, " ", ".")
	cleaned = strings.Trim(cleaned, " .")
	return reMultiDots.ReplaceAllString(cleaned, ".")
}

// RenderDirectoryName applies the configured template and collapses dots.
func (o *Organizer) RenderDirectoryName(chinese, english, year string) string {
	format := o.cfg.DirectoryFormat
	if format == "" {
		format = config.DefaultDirectoryFormat
	}
	name := strings.NewReplacer(
		"{chinese_title}", chinese,
		"{english_title}", english,
		"{year}", year,
	).Replace(format)
	return strings.TrimRight(reMultiDots.ReplaceAllString(name, "."), ".")
}

func (o *Organizer) baseDir() (string, error) {
	if strings.TrimSpace(o.cfg.MovieFolder) == "" {
		return "", fmt.Errorf("未配置电影目录，请先在「设置」中选择目录")
	}
	return o.cfg.MovieFolder, nil
}

func (o *Organizer) writeFile(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", filepath.Base(path), err)
	}
	return nil
}

// CreateSimple creates a directory from a minimal movie info dict. The SID
// file contains just the SID (Python create_movie_directory behaviour).
func (o *Organizer) CreateSimple(info SimpleInfo) (string, error) {
	if strings.TrimSpace(info.Title) == "" {
		return "", fmt.Errorf("片名不能为空")
	}
	base, err := o.baseDir()
	if err != nil {
		return "", err
	}
	name := o.RenderDirectoryName(
		CleanFilename(strings.TrimSpace(info.Title)),
		CleanFilename(strings.TrimSpace(info.OriginalTitle)),
		info.Year,
	)
	if name == "" {
		return "", fmt.Errorf("生成的目录名为空")
	}
	dir := filepath.Join(base, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}
	if o.cfg.CreateSIDFile && info.SID != "" {
		if err := o.writeFile(filepath.Join(dir, info.SID+".txt"), info.SID); err != nil {
			return "", err
		}
	}
	if o.cfg.CreateRatingFile && info.Rating != "" {
		if err := o.writeFile(filepath.Join(dir, info.Rating+".txt"), ""); err != nil {
			return "", err
		}
	}
	o.log.Info("organizer", "已创建整理目录 %s", dir)
	return dir, nil
}

// CreateByDetail mirrors create_movie_directory_by_detail, including its
// three-way decision about the English title, and writes the full SID text.
// The configured movie folder is used as base directory.
func (o *Organizer) CreateByDetail(d model.MovieDetails) (string, error) {
	base, err := o.baseDir()
	if err != nil {
		return "", err
	}
	return o.CreateByDetailAt(base, d)
}

// CreateByDetailAt is CreateByDetail with an explicit base directory; the
// pipeline uses it when prefer_source_dir prefers the source video's own
// directory over the configured movie folder.
func (o *Organizer) CreateByDetailAt(base string, d model.MovieDetails) (string, error) {
	if strings.TrimSpace(d.Title) == "" {
		return "", fmt.Errorf("片名不能为空")
	}

	chinese := CleanFilename(strings.TrimSpace(d.Title))
	searchTitle := CleanFilename(strings.TrimSpace(d.SearchTitle))
	english := CleanFilename(strings.TrimSpace(d.OriginalTitle))
	year := strings.TrimSpace(d.Year)

	var dirName string
	switch {
	case reEnglishInName.MatchString(d.Title):
		// Title already contains the English name (common on detail pages).
		dirName = o.RenderDirectoryName(chinese, "", year)
	case english == "":
		dirName = o.RenderDirectoryName(chinese, "", year)
	default:
		useChinese := chinese
		if searchTitle != "" {
			useChinese = searchTitle
		}
		dirName = o.RenderDirectoryName(useChinese, english, year)
	}
	if dirName == "" {
		return "", fmt.Errorf("生成的目录名为空")
	}

	dir := filepath.Join(base, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	if o.cfg.CreateRatingFile && d.Rating != "" {
		if err := o.writeFile(filepath.Join(dir, CleanFilename(d.Rating)+".txt"), ""); err != nil {
			return "", err
		}
	}
	if o.cfg.CreateSIDFile {
		fileName := "sid.txt"
		if d.SID != "" {
			fileName = d.SID + ".txt"
		}
		if err := o.writeFile(filepath.Join(dir, fileName), BuildSIDContent(d)); err != nil {
			return "", err
		}
	}
	o.log.Info("organizer", "已创建整理目录 %s", dir)
	return dir, nil
}

// WriteIMDbRating writes the empty "IMDB x.x.txt" marker file.
func (o *Organizer) WriteIMDbRating(dir string, rating float64) (string, error) {
	// Python formats floats as "6.7" / "7.0" / "6.75"; FormatFloat(-1) drops
	// the trailing ".0", so restore it for whole numbers.
	text := strconv.FormatFloat(rating, 'f', -1, 64)
	if !strings.Contains(text, ".") {
		text += ".0"
	}
	// Keep the literal space used by the Python implementation.
	name := "IMDB " + text + ".txt"
	path := filepath.Join(dir, name)
	if err := o.writeFile(path, ""); err != nil {
		return "", err
	}
	return path, nil
}

// BuildSIDContent reproduces the readable metadata text of _build_sid_content.
func BuildSIDContent(d model.MovieDetails) string {
	var b strings.Builder
	add := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }

	add("==================================================\n")
	add("电影基本信息\n")
	add("==================================================\n")
	if d.Title != "" {
		add("中文名称: %s\n", d.Title)
	}
	if d.OriginalTitle != "" {
		add("英文原名: %s\n", d.OriginalTitle)
	}
	if d.Aliases != "" {
		add("其他别名: %s\n", d.Aliases)
	}
	if d.SID != "" {
		add("豆瓣ID: %s\n", d.SID)
	}
	if d.Year != "" {
		add("年份: %s\n", d.Year)
	}
	if d.Rating != "" {
		add("豆瓣评分: %s\n", d.Rating)
	}
	if d.RatingCount != 0 {
		add("评分人数: %d\n", d.RatingCount)
	}
	if d.Top250Rank != 0 {
		add("Top250排名: #%d\n", d.Top250Rank)
	}
	add("\n")
	if len(d.Directors) > 0 {
		add("导演: %s\n", strings.Join(d.Directors, ", "))
	}
	if len(d.Actors) > 0 {
		limit := 10
		if len(d.Actors) < limit {
			limit = len(d.Actors)
		}
		add("主演: %s\n", strings.Join(d.Actors[:limit], ", "))
	}
	add("\n")

	add("--------------------------------------------------\n")
	add("详细信息\n")
	add("--------------------------------------------------\n")
	if len(d.Genres) > 0 {
		add("类型: %s\n", strings.Join(d.Genres, ", "))
	}
	if d.Country != "" {
		add("制片国家/地区: %s\n", d.Country)
	}
	if d.Language != "" {
		add("语言: %s\n", d.Language)
	}
	if d.Runtime != 0 {
		add("片长: %d分钟\n", d.Runtime)
	}
	if d.IMDbID != "" {
		add("IMDb: %s\n", d.IMDbID)
	}
	if len(d.ReleaseDates) > 0 {
		add("上映日期: %s\n", strings.Join(d.ReleaseDates, ", "))
	}
	add("\n")

	add("--------------------------------------------------\n")
	add("剧情简介\n")
	add("--------------------------------------------------\n")
	if d.Description != "" {
		add("%s\n", d.Description)
	} else {
		add("暂无剧情简介\n")
	}
	add("\n")
	if d.ShortCommentsCount != 0 {
		add("短评数量: %d\n", d.ShortCommentsCount)
	}
	if d.ReviewsCount != 0 {
		add("影评数量: %d\n", d.ReviewsCount)
	}
	if len(d.Awards) > 0 {
		add("获奖情况: 共%d条记录\n", len(d.Awards))
		limit := 5
		if len(d.Awards) < limit {
			limit = len(d.Awards)
		}
		for _, a := range d.Awards[:limit] {
			parts := make([]string, 0, 3)
			for _, p := range []string{a.Event, a.Category, a.Recipient} {
				if p != "" {
					parts = append(parts, p)
				}
			}
			if len(parts) > 0 {
				add("  - %s\n", strings.Join(parts, " / "))
			}
		}
	}
	if len(d.SimilarMovies) > 0 {
		add("相似推荐: 共%d部\n", len(d.SimilarMovies))
		limit := 5
		if len(d.SimilarMovies) < limit {
			limit = len(d.SimilarMovies)
		}
		for _, m := range d.SimilarMovies[:limit] {
			add("  - %s (评分: %.1f)\n", m.Title, m.Rating)
		}
	}
	add("\n==================================================\n")
	return strings.TrimSuffix(b.String(), "\n")
}
