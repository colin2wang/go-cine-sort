// Package config loads and persists the application configuration as YAML in
// the executable's own directory, with embedded defaults as a safety net.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	MovieFolder      string   `yaml:"movie_folder" json:"movieFolder"`
	DirectoryFormat  string   `yaml:"directory_format" json:"directoryFormat"`
	Recursive        bool     `yaml:"recursive" json:"recursive"`
	ArchiveMove      bool     `yaml:"archive_move" json:"archiveMove"`
	ConflictStrategy string   `yaml:"conflict_strategy" json:"conflictStrategy"`
	CreateRatingFile bool     `yaml:"create_rating_file" json:"createRatingFile"`
	CreateSIDFile    bool     `yaml:"create_sid_file" json:"createSidFile"`
	DownloadPosters  bool     `yaml:"download_posters" json:"downloadPosters"`
	FetchIMDb        bool     `yaml:"fetch_imdb" json:"fetchImdb"`
	Extensions       []string `yaml:"extensions" json:"extensions"`
	CleanupPatterns  []string `yaml:"cleanup_patterns" json:"cleanupPatterns"`
	TechPatterns     []string `yaml:"tech_patterns" json:"techPatterns"`
	LanguagePatterns []string `yaml:"language_patterns" json:"languagePatterns"`

	// Resolved paths are runtime-only and never persisted.
	ConfigPath string `yaml:"-" json:"configPath"`
	DataDir    string `yaml:"-" json:"dataDir"`
}

// Dir returns the application data directory: the directory containing the
// running executable, so all generated files stay next to the program (portable)
// instead of under the user's AppData\Roaming.
func Dir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return filepath.Dir(exe), nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yml"), nil
}

// Load reads config.yml. Missing keys keep their embedded default values; if
// the file does not exist it is created from the defaults on first run.
func Load() (*AppConfig, error) {
	cfg, err := Defaults()
	if err != nil {
		return nil, err
	}

	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "config.yml")
	cfg.ConfigPath = path
	cfg.DataDir = dir

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config: %w", err)
		}
		if err := Save(&cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	// Resolved fields are never taken from user input.
	cfg.ConfigPath = path
	cfg.DataDir = dir
	cfg.normalize()
	return &cfg, nil
}

// Save persists the configuration. It creates the directory when needed.
func Save(cfg *AppConfig) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	cfg.normalize()

	type persisted AppConfig
	out, err := yaml.Marshal((*persisted)(cfg))
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	header := "# Cine Sort 配置 — 由「设置」页生成，也可手工编辑\n" +
		"# 目录名模板可用变量：{chinese_title} / {english_title} / {year}\n\n"
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, append([]byte(header), out...), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	cfg.ConfigPath = path
	cfg.DataDir = dir
	return nil
}

func (c *AppConfig) normalize() {
	if strings.TrimSpace(c.DirectoryFormat) == "" {
		c.DirectoryFormat = DefaultDirectoryFormat
	}
	switch c.ConflictStrategy {
	case "rename", "skip", "overwrite":
	default:
		c.ConflictStrategy = DefaultConflictStrategy
	}
	if len(c.Extensions) == 0 {
		if rules, err := defaultRules(); err == nil {
			c.Extensions = append([]string(nil), rules.Extensions...)
		}
	}
	seen := make(map[string]struct{}, len(c.Extensions))
	normalized := make([]string, 0, len(c.Extensions))
	for _, ext := range c.Extensions {
		e := strings.ToLower(strings.TrimSpace(ext))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		normalized = append(normalized, e)
	}
	c.Extensions = normalized
}

// Validate rejects values that would make the pipeline fail at runtime.
func (c *AppConfig) Validate() error {
	if strings.TrimSpace(c.MovieFolder) != "" {
		if _, err := os.Stat(c.MovieFolder); err != nil {
			return fmt.Errorf("电影目录不可用: %w", err)
		}
	}
	if !strings.Contains(c.DirectoryFormat, "{chinese_title}") &&
		!strings.Contains(c.DirectoryFormat, "{english_title}") &&
		!strings.Contains(c.DirectoryFormat, "{year}") {
		return fmt.Errorf("目录名模板至少需要一个变量")
	}
	return nil
}

// CompiledRules caches every regexp used by the scanner. Compiling once avoids
// O(files × patterns) work on the hot path.
type CompiledRules struct {
	Extensions      map[string]bool
	Cleanup         *regexp.Regexp
	SiteMarkers     *regexp.Regexp
	TagSegments     *regexp.Regexp
	Year            *regexp.Regexp
	Separators      *regexp.Regexp
	Spaces          *regexp.Regexp
	ChineseSegments *regexp.Regexp
	NonWordOrSpace  *regexp.Regexp
}

func mustCompile(expr string) (*regexp.Regexp, error) {
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid regexp %q: %w", expr, err)
	}
	return re, nil
}

// CompileRules builds the merged, case-insensitive cleanup regexp plus the
// scanner's fixed regexps.
func CompileRules(c *AppConfig) (*CompiledRules, error) {
	r := &CompiledRules{Extensions: make(map[string]bool, len(c.Extensions))}
	for _, ext := range c.Extensions {
		r.Extensions[strings.ToLower(ext)] = true
	}

	all := make([]string, 0, len(c.CleanupPatterns)+len(c.TechPatterns)+len(c.LanguagePatterns))
	all = append(all, c.CleanupPatterns...)
	all = append(all, c.TechPatterns...)
	all = append(all, c.LanguagePatterns...)

	seen := make(map[string]bool, len(all))
	parts := make([]string, 0, len(all))
	for _, pattern := range all {
		if strings.TrimSpace(pattern) == "" || seen[pattern] {
			continue
		}
		seen[pattern] = true
		if _, err := regexp.Compile(pattern); err != nil {
			return nil, fmt.Errorf("配置中的清洗规则无效 %q: %w", pattern, err)
		}
		parts = append(parts, "("+pattern+")")
	}
	if len(parts) > 0 {
		cleanup, err := mustCompile("(?i:" + strings.Join(parts, "|") + ")")
		if err != nil {
			return nil, err
		}
		r.Cleanup = cleanup
	}

	var err error
	if r.Year, err = mustCompile(`(?:19|20)\d{2}`); err != nil {
		return nil, err
	}
	// Site markers plus their optional domain suffix, e.g. "阳光电影dygod.org".
	// Must run before TagSegments so the domain is consumed as one unit.
	siteMarkers := `阳光电影|最新电影|电影天堂|电影港|迅雷下载|磁力链接|YYeTs|韩剧精灵|人人影视制作`
	if r.SiteMarkers, err = mustCompile(`(?i)(?:` + siteMarkers + `)(?:[a-zA-Z0-9.-]*\.[a-zA-Z]{2,})?`); err != nil {
		return nil, err
	}
	// Chinese-tagged segments (subtitles / audio / version markers) must be
	// removed BEFORE ASCII keywords, otherwise Go's ASCII-only \b lets "HD"
	// match first and splits "HD国语中字无水印" into an undeletable remainder.
	tagWords := `阳光电影|最新电影|电影天堂|电影港|迅雷下载|磁力链接|YYeTs|韩剧精灵|人人影视制作|` +
		`国语|粤语|台配|日语|韩语|英语|法语|德语|俄语|西班牙语|泰语|菲律宾语|他加禄语|音轨|` +
		`中字|中英双字|双语|字幕|无水印|完整版|高清版|修复版|导演版|加长版`
	if r.TagSegments, err = mustCompile(`(?i)[^\s.\[\]()/\\]*(?:` + tagWords + `)[^\s.\[\]()/\\]*`); err != nil {
		return nil, err
	}
	if r.Separators, err = mustCompile(`[.\-_]+`); err != nil {
		return nil, err
	}
	if r.Spaces, err = mustCompile(`\s+`); err != nil {
		return nil, err
	}
	if r.ChineseSegments, err = mustCompile(`[\x{4e00}-\x{9fff}]+`); err != nil {
		return nil, err
	}
	if r.NonWordOrSpace, err = mustCompile(`[^\p{L}\p{N}_\s]`); err != nil {
		return nil, err
	}
	return r, nil
}
