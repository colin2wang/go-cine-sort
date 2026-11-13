package store

import (
	"path/filepath"
	"testing"

	"go-cine-sort/internal/logging"
	"go-cine-sort/internal/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	log, err := logging.New("", nil)
	if err != nil {
		t.Fatal(err)
	}
	log.SetLevel(logging.LevelError)
	st, err := Open(filepath.Join(t.TempDir(), "test.db"), log)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestSaveMovieUpsertAndQuery(t *testing.T) {
	st := newTestStore(t)
	ctx := t.Context()
	details := model.MovieDetails{
		SID:       "1292052",
		Title:     "肖申克的救赎",
		Year:      "1994",
		Rating:    "9.7",
		IMDbID:    "tt0111161",
		Directors: []string{"弗兰克·德拉邦特"},
		Actors:    []string{"蒂姆·罗宾斯", "摩根·弗里曼"},
		Genres:    []string{"剧情", "犯罪"},
		Runtime:   142,
	}
	id, err := st.SaveMovie(ctx, details, `D:\movies\肖申克的救赎.1994`)
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Fatalf("invalid movie id %d", id)
	}

	// Same SID must update instead of inserting a second row.
	details.Rating = "9.8"
	details.Directors = []string{"弗兰克·德拉邦特", "第二导演"}
	if _, err := st.SaveMovie(ctx, details, `D:\movies\new`); err != nil {
		t.Fatal(err)
	}
	stats, err := st.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Total != 1 {
		t.Fatalf("upsert created a duplicate row: total=%d", stats.Total)
	}

	list, err := st.ListHistory(ctx, "肖申克")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Rating != "9.8" {
		t.Fatalf("history = %+v", list)
	}
	if _, err := st.ListHistory(ctx, "不存在的片名"); err != nil {
		t.Fatal(err)
	}

	detail, err := st.GetDetail(ctx, "1292052")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Rating != "9.8" || detail.Runtime != 142 || detail.IMDbID != "tt0111161" {
		t.Errorf("detail = %+v", detail)
	}
	if len(detail.Directors) != 2 || len(detail.Actors) != 2 || len(detail.Genres) != 2 {
		t.Errorf("child rows lost: %+v", detail)
	}
}

func TestSavePostersAndStats(t *testing.T) {
	st := newTestStore(t)
	ctx := t.Context()
	details := model.MovieDetails{SID: "1", Title: "测试", Rating: "8.0"}
	id, err := st.SaveMovie(ctx, details, `D:\movies\test`)
	if err != nil {
		t.Fatal(err)
	}
	posters := []model.PosterRecord{
		{PosterID: "p1", Title: "正式海报 中国大陆", OriginalURL: "https://img1.doubanio.com/p1.jpg", FileSize: 2 << 20, IsDownloaded: true, LocalPath: `D:\movies\test\p1.jpg`},
		{PosterID: "p2", Title: "海报 美国", OriginalURL: "https://img1.doubanio.com/p2.jpg", FileSize: 3 << 20, IsDownloaded: true},
	}
	count, err := st.SavePosters(ctx, id, posters)
	if err != nil || count != 2 {
		t.Fatalf("SavePosters = %d, %v", count, err)
	}
	// Unique(movie_id, original_url) makes a repeat an upsert, not a duplicate.
	if _, err := st.SavePosters(ctx, id, posters[:1]); err != nil {
		t.Fatal(err)
	}
	stats, err := st.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.PosterTotal != 2 || stats.Total != 1 || stats.AverageRating != 8.0 {
		t.Errorf("stats = %+v", stats)
	}
	detail, err := st.GetDetail(ctx, "1")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Posters) != 2 || detail.PosterCount != 2 {
		t.Errorf("detail posters = %+v", detail.Posters)
	}
}
