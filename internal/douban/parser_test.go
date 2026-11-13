package douban

import (
	"os"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("读取夹具 %s 失败: %v", name, err)
	}
	return string(data)
}

func TestParseSearchResultFromRealHTML(t *testing.T) {
	got := ParseSearchResult(fixture(t, "search_result.html"))
	t.Logf("search => %+v", got)
	if got.SID != "1292052" {
		t.Errorf("SID = %q, want 1292052", got.SID)
	}
	if got.Title != "肖申克的救赎" {
		t.Errorf("Title = %q, want 肖申克的救赎", got.Title)
	}
	if got.Rating != "9.7" {
		t.Errorf("Rating = %q, want 9.7", got.Rating)
	}
	if got.Year != "1994" {
		t.Errorf("Year = %q, want 1994", got.Year)
	}
	if got.OriginalTitle != "The Shawshank Redemption" {
		t.Errorf("OriginalTitle = %q, want The Shawshank Redemption", got.OriginalTitle)
	}
	if len(got.Directors) == 0 || got.Directors[0] != "弗兰克·德拉邦特" {
		t.Errorf("Directors = %v", got.Directors)
	}
	if len(got.Actors) == 0 || got.Actors[0] != "蒂姆·罗宾斯" {
		t.Errorf("Actors = %v", got.Actors)
	}
	if got.Error != "" {
		t.Errorf("unexpected error: %s", got.Error)
	}
}

func TestParseSearchResultEmpty(t *testing.T) {
	if got := ParseSearchResult(""); got.Error == "" {
		t.Fatal("空 HTML 应返回解析错误")
	}
	if got := ParseSearchResult("<html>no results</html>"); got.SID != "" || got.Error == "" {
		t.Fatalf("无结果页应返回空 SID 与错误: %+v", got)
	}
}

func TestPrepareQuery(t *testing.T) {
	cases := []struct {
		name, year, want string
	}{
		{"肖申克的救赎", "1994", "肖申克的救赎 1994"},
		{"腿", "", "腿"},
		{"Chainsaw Man Reze Arc", "2025", "Chainsaw Man Reze Arc 2025"},
		{"The Shawshank Redemption", "", "The Shawshank Redemption"},
	}
	for _, c := range cases {
		if got := PrepareQuery(c.name, c.year); got != c.want {
			t.Errorf("PrepareQuery(%q,%q) = %q, want %q", c.name, c.year, got, c.want)
		}
	}
}

func TestParseDetailsFromRealHTML(t *testing.T) {
	got := ParseDetails(fixture(t, "details_result.html"))
	t.Logf("details => title=%q rating=%q year=%q original=%q imdb=%q runtime=%d votes=%d directors=%v actors=%v genres=%v descLen=%d",
		got.Title, got.Rating, got.Year, got.OriginalTitle, got.IMDbID,
		got.Runtime, got.RatingCount, got.Directors, got.Actors, got.Genres, len(got.Description))

	if got.Title != "肖申克的救赎 The Shawshank Redemption" {
		t.Errorf("Title = %q", got.Title)
	}
	if got.Rating != "9.7" {
		t.Errorf("Rating = %q, want 9.7", got.Rating)
	}
	if got.Year != "1994" {
		t.Errorf("Year = %q, want 1994", got.Year)
	}
	if got.IMDbID != "tt0111161" {
		t.Errorf("IMDbID = %q, want tt0111161", got.IMDbID)
	}
	if got.Runtime != 142 {
		t.Errorf("Runtime = %d, want 142", got.Runtime)
	}
	if got.Description == "" {
		t.Error("Description 为空，简介四级 fallback 未生效")
	}
	if len(got.Genres) != 2 {
		t.Errorf("Genres = %v", got.Genres)
	}
	if len(got.Directors) == 0 || got.Directors[0] != "弗兰克·德拉邦特" {
		t.Errorf("Directors = %v", got.Directors)
	}
	if len(got.Actors) < 5 || got.Actors[0] != "蒂姆·罗宾斯" {
		t.Errorf("Actors = %v", got.Actors)
	}
}
