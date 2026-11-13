package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"go-cine-sort/internal/config"
	"go-cine-sort/internal/logging"
)

func newTestScanner(t *testing.T) *Scanner {
	t.Helper()
	cfg, err := config.Defaults()
	if err != nil {
		t.Fatal(err)
	}
	log, err := logging.New("", nil)
	if err != nil {
		t.Fatal(err)
	}
	log.SetLevel(logging.LevelError)
	s, err := New(&cfg, log)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Assertions are taken from python_cine_sort/tests/test_movie_file_util.py.
func TestCleanMovieName(t *testing.T) {
	s := newTestScanner(t)
	cases := []struct {
		filename string
		want     string
	}{
		{"[Haruhana] Chainsaw Man - Reze Arc [WebRip][HEVC-10bit 1080p][CHS_JPN]", "Chainsaw Man Reze Arc"},
		{"猪猪侠.一只老猪的逆袭.1080p.HD国语中字无水印[最新电影www.5266ys.com]", "猪猪侠 一只老猪的逆袭"},
		{"电影.2024", "电影"},
		{"纯电影名称", "纯电影名称"},
		{"阳光电影dygod.org.测试电影.2024.HD.1080P", "测试电影"},
		{"电影VS名称.2024.HD", "电影VS名称"},
		{"阳光电影dygod.org.我的1919.2020.BD.HD1080P.国英双语双字", "我的1919"},
	}
	for _, c := range cases {
		got := s.CleanMovieName(c.filename)
		if got != c.want {
			t.Errorf("CleanMovieName(%q) = %q, want %q", c.filename, got, c.want)
		}
	}
}

func TestExtractYear(t *testing.T) {
	s := newTestScanner(t)
	cases := []struct {
		filename string
		want     string
	}{
		{"阳光电影dygod.org.睡觉的笨蛋.2025.BD.1080P.日语中字", "2025"},
		{"[Haruhana] Chainsaw Man - Reze Arc [WebRip]", ""},
		{"电影.2024", "2024"},
		{"纯电影名称", ""},
		{"阳光电影dygod.org.测试电影.2024.HD.1080P", "2024"},
		{"阳光电影dygod.org.我的1919.2020.BD.HD1080P.国英双语双字", "2020"},
	}
	for _, c := range cases {
		if got := s.ExtractYear(c.filename); got != c.want {
			t.Errorf("ExtractYear(%q) = %q, want %q", c.filename, got, c.want)
		}
	}
}

func TestSingleChineseTitlePreserved(t *testing.T) {
	// 2026-06-07 fix: a single Chinese character title must survive step 4.
	s := newTestScanner(t)
	if got := s.CleanMovieName("腿.2024.1080p"); got == "" {
		t.Fatal("单字中文片名被丢弃")
	}
}

func TestScanDirectorySkipsOrganized(t *testing.T) {
	s := newTestScanner(t)
	root := t.TempDir()
	writeFile(t, root, "阿凡达.2009.1080p.mkv", "dummy")
	nested := filepath.Join(root, "肖申克的救赎.The.Shawshank.Redemption.1994")
	writeFile(t, nested, "9.7.txt", "")
	writeFile(t, nested, "1292052.txt", "1292052")
	writeFile(t, nested, "阿凡达.2009.mkv", "dummy")

	results, err := s.ScanDirectory(t.Context(), root, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("应跳过已整理目录，期望 1 个文件，实际 %d: %+v", len(results), results)
	}
	if results[0].MovieName == "" {
		t.Fatal("片名为空")
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
