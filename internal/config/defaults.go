package config

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed defaults.yml
var defaultRulesYAML []byte

const (
	DefaultDirectoryFormat  = "{chinese_title}.{english_title}.{year}"
	DefaultConflictStrategy = "rename"
)

type ruleFile struct {
	Extensions       []string `yaml:"extensions"`
	CleanupPatterns  []string `yaml:"cleanup_patterns"`
	TechPatterns     []string `yaml:"tech_patterns"`
	LanguagePatterns []string `yaml:"language_patterns"`
}

func defaultRules() (ruleFile, error) {
	var r ruleFile
	if err := yaml.Unmarshal(defaultRulesYAML, &r); err != nil {
		return r, fmt.Errorf("parse embedded default rules: %w", err)
	}
	return r, nil
}

// Defaults returns a configuration entirely populated with embedded defaults.
// No drive-letter paths are baked in; MovieFolder is left empty on purpose.
func Defaults() (AppConfig, error) {
	rules, err := defaultRules()
	if err != nil {
		return AppConfig{}, err
	}
	return AppConfig{
		MovieFolder:      "",
		DirectoryFormat:  DefaultDirectoryFormat,
		Recursive:        true,
		ArchiveMove:      false,
		ConflictStrategy: DefaultConflictStrategy,
		CreateRatingFile: true,
		CreateSIDFile:    true,
		DownloadPosters:  true,
		FetchIMDb:        true,
		Extensions:       append([]string(nil), rules.Extensions...),
		CleanupPatterns:  append([]string(nil), rules.CleanupPatterns...),
		TechPatterns:     append([]string(nil), rules.TechPatterns...),
		LanguagePatterns: append([]string(nil), rules.LanguagePatterns...),
	}, nil
}
