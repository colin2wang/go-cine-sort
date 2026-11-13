package config

import (
	"strings"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ArchiveMove {
		t.Error("archive move must default to off")
	}
	if cfg.DirectoryFormat != DefaultDirectoryFormat {
		t.Errorf("DirectoryFormat = %q", cfg.DirectoryFormat)
	}
	if cfg.MovieFolder != "" {
		t.Errorf("MovieFolder should not contain a hard-coded drive letter: %q", cfg.MovieFolder)
	}
	if len(cfg.Extensions) != 7 || len(cfg.CleanupPatterns) < 20 {
		t.Errorf("unexpected default rule counts: %d extensions, %d cleanup patterns",
			len(cfg.Extensions), len(cfg.CleanupPatterns))
	}
	if cfg.ConflictStrategy != "rename" {
		t.Errorf("ConflictStrategy = %q", cfg.ConflictStrategy)
	}
}

func TestCompileRules(t *testing.T) {
	cfg, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	rules, err := CompileRules(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	for name, re := range map[string]bool{
		"cleanup":      rules.Cleanup != nil,
		"site markers": rules.SiteMarkers != nil,
		"tag segments": rules.TagSegments != nil,
		"year":         rules.Year != nil,
		"separators":   rules.Separators != nil,
	} {
		if !re {
			t.Errorf("%s regexp was not compiled", name)
		}
	}
	if !rules.Extensions[".mkv"] || rules.Extensions["mkv"] {
		t.Error("extensions were not normalized to dotted lowercase")
	}
	if !rules.Year.MatchString("2024") {
		t.Error("year regexp does not match 2024")
	}
	if !rules.SiteMarkers.MatchString("阳光电影dygod.org") {
		t.Error("site marker regexp should consume the domain suffix")
	}
	if !rules.TagSegments.MatchString("HD国语中字无水印") {
		t.Error("tag segment regexp should match a whole Chinese-tagged segment")
	}
}

func TestCompileRulesRejectsInvalidPattern(t *testing.T) {
	cfg, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	cfg.CleanupPatterns = append(cfg.CleanupPatterns, "([unclosed")
	if _, err := CompileRules(&cfg); err == nil {
		t.Fatal("invalid regexp should be reported")
	}
}

func TestValidate(t *testing.T) {
	cfg, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("defaults should validate: %v", err)
	}
	cfg.DirectoryFormat = "no-variables-here"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "变量") {
		t.Errorf("template without variables should fail, got %v", err)
	}
	cfg.DirectoryFormat = "{chinese_title}"
	cfg.ConflictStrategy = "unknown"
	cfg.normalize()
	if cfg.ConflictStrategy != "rename" {
		t.Errorf("unknown strategy should fall back to rename, got %q", cfg.ConflictStrategy)
	}
}
