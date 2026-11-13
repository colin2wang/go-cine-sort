package organizer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-cine-sort/internal/config"
	"go-cine-sort/internal/logging"
	"go-cine-sort/internal/model"
)

func newTestOrganizer(t *testing.T) (*Organizer, string) {
	t.Helper()
	cfg, err := config.Defaults()
	if err != nil {
		t.Fatal(err)
	}
	cfg.MovieFolder = t.TempDir()
	log, err := logging.New("", nil)
	if err != nil {
		t.Fatal(err)
	}
	log.SetLevel(logging.LevelError)
	return New(&cfg, log), cfg.MovieFolder
}

// Mirrors python_cine_sort/tests/test_movie_org_util.py:100-116.
func TestCreateSimpleDirectory(t *testing.T) {
	org, _ := newTestOrganizer(t)
	dir, err := org.CreateSimple(SimpleInfo{
		Title:         "7号房的礼物",
		OriginalTitle: "Gift From Room",
		Year:          "2013",
		SID:           "25746891",
		Rating:        "8.9",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := filepath.Base(dir), "7号房的礼物.Gift.From.Room.2013"; got != want {
		t.Fatalf("目录名 = %q, want %q", got, want)
	}
	sidContent, err := os.ReadFile(filepath.Join(dir, "25746891.txt"))
	if err != nil || string(sidContent) != "25746891" {
		t.Errorf("SID 文件 = %q, err=%v", sidContent, err)
	}
	ratingContent, err := os.ReadFile(filepath.Join(dir, "8.9.txt"))
	if err != nil || len(ratingContent) != 0 {
		t.Errorf("评分文件 = %q, err=%v", ratingContent, err)
	}
}

func TestCreateByDetailWithEnglishInTitle(t *testing.T) {
	org, _ := newTestOrganizer(t)
	dir, err := org.CreateByDetail(model.MovieDetails{
		Title:       "肖申克的救赎 The Shawshank Redemption",
		SearchTitle: "肖申克的救赎",
		Year:        "1994",
		SID:         "1292052",
		Rating:      "9.7",
		Directors:   []string{"弗兰克·德拉邦特"},
		Description: "一场谋杀案…",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := filepath.Base(dir), "肖申克的救赎.The.Shawshank.Redemption.1994"; got != want {
		t.Fatalf("目录名 = %q, want %q", got, want)
	}
	content, err := os.ReadFile(filepath.Join(dir, "1292052.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"电影基本信息", "中文名称: 肖申克的救赎", "导演: 弗兰克·德拉邦特", "剧情简介"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("SID 文件缺少 %q", want)
		}
	}
}

func TestCleanFilename(t *testing.T) {
	cases := map[string]string{
		`A/B:C*D?E"F<G>H|I`:        "A.B.C.D.E.F.G.H.I",
		"The Shawshank Redemption": "The.Shawshank.Redemption",
		"  ..7号房的礼物.. ":            "7号房的礼物",
	}
	for in, want := range cases {
		if got := CleanFilename(in); got != want {
			t.Errorf("CleanFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWriteIMDbRating(t *testing.T) {
	org, _ := newTestOrganizer(t)
	dir := t.TempDir()
	path, err := org.WriteIMDbRating(dir, 7.0)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := filepath.Base(path), "IMDB 7.0.txt"; got != want {
		t.Errorf("文件名 = %q, want %q", got, want)
	}
}

func TestArchiveVideo(t *testing.T) {
	org, _ := newTestOrganizer(t)
	root := t.TempDir()
	source := filepath.Join(root, "movie.mkv")
	if err := os.WriteFile(source, []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "organized")
	dest, err := org.ArchiveVideo(t.Context(), source, target)
	if err != nil {
		t.Fatal(err)
	}
	if dest == "" {
		t.Fatal("归档目标为空")
	}
	data, err := os.ReadFile(dest)
	if err != nil || string(data) != "0123456789" {
		t.Fatalf("归档文件内容 = %q, err=%v", data, err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatal("源文件应已移走")
	}
}

func TestArchiveSkipConflict(t *testing.T) {
	org, _ := newTestOrganizer(t)
	org.cfg.ConflictStrategy = "skip"
	root := t.TempDir()
	source := filepath.Join(root, "movie.mkv")
	_ = os.WriteFile(source, []byte("new"), 0o644)
	target := filepath.Join(root, "organized")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(target, "movie.mkv"), []byte("old"), 0o644)
	dest, err := org.ArchiveVideo(t.Context(), source, target)
	if err != nil {
		t.Fatal(err)
	}
	if dest != "" {
		t.Fatalf("skip 策略应返回空目标，实际 %q", dest)
	}
}
