package douban

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOriginalURL(t *testing.T) {
	cases := map[string]string{
		"https://img9.doubanio.com/view/photo/m/public/p123.jpg":            "https://img9.doubanio.com/view/photo/l/public/p123.jpg",
		"https://img1.doubanio.com/view/photo/s_ratio_poster/public/p1.jpg": "https://img1.doubanio.com/view/photo/l/public/p1.jpg",
		"https://img2.doubanio.com/view/photo/m_ratio_poster/public/p2.jpg": "https://img2.doubanio.com/view/photo/l/public/p2.jpg",
		"https://img3.doubanio.com/view/photo/r_ratio_poster/public/p3.jpg": "https://img3.doubanio.com/view/photo/l/public/p3.jpg",
		"//img1.doubanio.com/view/photo/m/public/p4.jpg":                    "https://img1.doubanio.com/view/photo/l/public/p4.jpg",
		"http://img1.doubanio.com/view/photo/m/public/p5.jpg":               "https://img1.doubanio.com/view/photo/l/public/p5.jpg",
	}
	for in, want := range cases {
		if got := OriginalURL(in); got != want {
			t.Errorf("OriginalURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParsePostersThreeFallbacks(t *testing.T) {
	full := `<ul><li><div class="cover"><a href="https://movie.douban.com/photos/photo/1/">
<img src="https://img9.doubanio.com/view/photo/m/public/p1.jpg"></a></div>
<div class="name">正式海报 中国大陆<br/><a href="#">12回应</a></div></li></ul>`
	if got := ParsePosters(full); len(got) != 1 || got[0].Title != "正式海报 中国大陆" {
		t.Fatalf("pattern 1 = %+v", got)
	}

	noAnchor := `<li><div class="cover"><a href="p2"><img src="//img1.doubanio.com/view/photo/m/public/p2.jpg"></a></div>
<div class="name">海报 美国</div></li>`
	if got := ParsePosters(noAnchor); len(got) != 1 || got[0].Title != "海报 美国" {
		t.Fatalf("pattern 2 = %+v", got)
	}

	simple := `<img src="https://img9.doubanio.com/view/photo/m/public/p3.jpg" alt="角色海报 日本" />`
	if got := ParsePosters(simple); len(got) != 1 || got[0].Title != "角色海报 日本" {
		t.Fatalf("pattern 3 = %+v", got)
	}
	if got := ParsePosters("  "); len(got) != 0 {
		t.Fatalf("empty html should yield no posters: %+v", got)
	}
}

func TestAlternativeCDNs(t *testing.T) {
	alts := alternativeCDNs("https://img9.doubanio.com/view/photo/l/public/p1.jpg")
	if len(alts) != 8 {
		t.Fatalf("expected 8 CDN alternatives, got %d: %v", len(alts), alts)
	}
	if alts[0] != "https://img3.doubanio.com/view/photo/l/public/p1.jpg" {
		t.Errorf("first alternative = %q", alts[0])
	}
	if got := alternativeCDNs("https://example.com/a.jpg"); len(got) != 0 {
		t.Errorf("non-douban URL should have no alternatives, got %v", got)
	}
}

func TestChineseRegion(t *testing.T) {
	for _, title := range []string{"正式海报 中国大陆", "海报 中国香港", "Poster China"} {
		if !isChineseRegion(title) {
			t.Errorf("%q should be a Chinese region", title)
		}
	}
	if isChineseRegion("正式海报 美国") {
		t.Error("US poster should not be a Chinese region")
	}
}

func TestValidImageFile(t *testing.T) {
	dir := t.TempDir()
	jpeg := filepath.Join(dir, "a.jpg")
	if err := os.WriteFile(jpeg, []byte{0xff, 0xd8, 0xff, 0xe0, 0, 0, 1, 2}, 0o644); err != nil {
		t.Fatal(err)
	}
	if !validImageFile(jpeg) {
		t.Error("valid JPEG magic should pass")
	}
	html := filepath.Join(dir, "b.jpg")
	if err := os.WriteFile(html, []byte("<!doctype html><html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if validImageFile(html) {
		t.Error("HTML content should be rejected")
	}
	if validImageFile(filepath.Join(dir, "missing.jpg")) {
		t.Error("missing file should be rejected")
	}
}
