// Package store persists movies and posters in SQLite. The schema is
// compatible with python_cine_sort/utils/db_manager.py.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"go-cine-sort/internal/logging"
	"go-cine-sort/internal/model"
)

const schema = `
CREATE TABLE IF NOT EXISTS movies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sid TEXT UNIQUE NOT NULL,
    title TEXT, original_title TEXT, aliases TEXT, year TEXT, rating TEXT,
    rating_count INTEGER, description TEXT, country TEXT, language TEXT,
    runtime INTEGER, imdb_id TEXT, top250_rank INTEGER,
    short_comments_count INTEGER, reviews_count INTEGER,
    poster_url TEXT, directory_path TEXT, created_at TEXT, updated_at TEXT
);
CREATE TABLE IF NOT EXISTS directors (
    id INTEGER PRIMARY KEY AUTOINCREMENT, movie_id INTEGER NOT NULL, name TEXT NOT NULL,
    FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS actors (
    id INTEGER PRIMARY KEY AUTOINCREMENT, movie_id INTEGER NOT NULL, name TEXT NOT NULL,
    FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS genres (
    id INTEGER PRIMARY KEY AUTOINCREMENT, movie_id INTEGER NOT NULL, name TEXT NOT NULL,
    FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS release_dates (
    id INTEGER PRIMARY KEY AUTOINCREMENT, movie_id INTEGER NOT NULL, date_text TEXT NOT NULL,
    FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS awards (
    id INTEGER PRIMARY KEY AUTOINCREMENT, movie_id INTEGER NOT NULL,
    event TEXT, category TEXT, recipient TEXT,
    FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS posters (
    id INTEGER PRIMARY KEY AUTOINCREMENT, movie_id INTEGER NOT NULL,
    poster_id TEXT, title TEXT, keyword TEXT, region TEXT,
    thumbnail_url TEXT, original_url TEXT, file_size INTEGER,
    is_downloaded INTEGER DEFAULT 0, local_path TEXT, created_at TEXT,
    FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE,
    UNIQUE(movie_id, original_url)
);
CREATE INDEX IF NOT EXISTS idx_movies_sid ON movies(sid);
CREATE INDEX IF NOT EXISTS idx_posters_movie ON posters(movie_id);
`

type Store struct {
	db  *sql.DB
	log *logging.Logger
}

func Open(dbPath string, log *logging.Logger) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 失败: %w", err)
	}
	// SQLite is single-writer; keeping one connection also makes
	// PRAGMA foreign_keys effective for every statement.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.ExecContext(context.Background(), schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("初始化表结构失败: %w", err)
	}
	log.Info("store", "SQLite 已就绪: %s", dbPath)
	return &Store{db: db, log: log}, nil
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SaveMovie upserts by SID inside a transaction; child rows are replaced.
func (s *Store) SaveMovie(ctx context.Context, d model.MovieDetails, directoryPath string) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	now := time.Now().Format(time.RFC3339)
	var movieID int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM movies WHERE sid = ?", d.SID).Scan(&movieID)
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE movies SET title=?, original_title=?, aliases=?, year=?,
            rating=?, rating_count=?, description=?, country=?, language=?, runtime=?,
            imdb_id=?, top250_rank=?, short_comments_count=?, reviews_count=?,
            poster_url=?, directory_path=?, updated_at=? WHERE id=?`,
			d.Title, d.OriginalTitle, d.Aliases, d.Year, d.Rating, d.RatingCount,
			d.Description, d.Country, d.Language, d.Runtime, d.IMDbID, d.Top250Rank,
			d.ShortCommentsCount, d.ReviewsCount, d.PosterURL, directoryPath, now, movieID)
		if err != nil {
			return 0, fmt.Errorf("更新影片失败: %w", err)
		}
		for _, table := range []string{"directors", "actors", "genres", "release_dates", "awards", "posters"} {
			if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE movie_id = ?", movieID); err != nil {
				return 0, fmt.Errorf("清理 %s 失败: %w", table, err)
			}
		}
	} else if err == sql.ErrNoRows {
		res, err := tx.ExecContext(ctx, `INSERT INTO movies
            (sid, title, original_title, aliases, year, rating, rating_count,
             description, country, language, runtime, imdb_id, top250_rank,
             short_comments_count, reviews_count, poster_url, directory_path,
             created_at, updated_at)
            VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			d.SID, d.Title, d.OriginalTitle, d.Aliases, d.Year, d.Rating, d.RatingCount,
			d.Description, d.Country, d.Language, d.Runtime, d.IMDbID, d.Top250Rank,
			d.ShortCommentsCount, d.ReviewsCount, d.PosterURL, directoryPath, now, now)
		if err != nil {
			return 0, fmt.Errorf("插入影片失败: %w", err)
		}
		movieID, err = res.LastInsertId()
		if err != nil {
			return 0, err
		}
	} else {
		return 0, err
	}

	type child struct {
		query string
		items []string
	}
	for _, c := range []child{
		{"INSERT INTO directors (movie_id, name) VALUES (?, ?)", d.Directors},
		{"INSERT INTO actors (movie_id, name) VALUES (?, ?)", d.Actors},
		{"INSERT INTO genres (movie_id, name) VALUES (?, ?)", d.Genres},
		{"INSERT INTO release_dates (movie_id, date_text) VALUES (?, ?)", d.ReleaseDates},
	} {
		for _, item := range c.items {
			if _, err := tx.ExecContext(ctx, c.query, movieID, item); err != nil {
				return 0, err
			}
		}
	}
	for _, a := range d.Awards {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO awards (movie_id, event, category, recipient) VALUES (?,?,?,?)",
			movieID, a.Event, a.Category, a.Recipient); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	s.log.Info("store", "已保存影片 %s (SID %s, ID %d)", d.Title, d.SID, movieID)
	return movieID, nil
}

func (s *Store) SavePosters(ctx context.Context, movieID int64, posters []model.PosterRecord) (int, error) {
	if len(posters) == 0 {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	now := time.Now().Format(time.RFC3339)
	count := 0
	for _, p := range posters {
		_, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO posters
            (movie_id, poster_id, title, keyword, region, thumbnail_url,
             original_url, file_size, is_downloaded, local_path, created_at)
            VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			movieID, p.PosterID, p.Title, p.Keyword, p.Region, p.ThumbnailURL,
			p.OriginalURL, p.FileSize, boolToInt(p.IsDownloaded), p.LocalPath, now)
		if err != nil {
			return 0, fmt.Errorf("保存海报记录失败: %w", err)
		}
		count++
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (s *Store) ListHistory(ctx context.Context, keyword string) ([]model.MovieRecord, error) {
	like := "%" + strings.TrimSpace(keyword) + "%"
	rows, err := s.db.QueryContext(ctx, `SELECT m.id, m.sid, m.title, m.original_title, m.year,
        m.rating, m.rating_count, m.imdb_id, m.directory_path, m.poster_url, m.created_at,
        (SELECT COUNT(*) FROM posters p WHERE p.movie_id = m.id) AS poster_count
        FROM movies m
        WHERE m.title LIKE ? OR m.original_title LIKE ? OR m.sid LIKE ? OR m.directory_path LIKE ?
        ORDER BY m.id DESC`, like, like, like, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.MovieRecord, 0)
	for rows.Next() {
		var r model.MovieRecord
		if err := rows.Scan(&r.ID, &r.SID, &r.Title, &r.OriginalTitle, &r.Year,
			&r.Rating, &r.RatingCount, &r.IMDbID, &r.DirectoryPath, &r.PosterURL,
			&r.CreatedAt, &r.PosterCount); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetDetail(ctx context.Context, sid string) (model.MovieDetail, error) {
	var d model.MovieDetail
	err := s.db.QueryRowContext(ctx, `SELECT id, sid, title, original_title, year, rating,
        rating_count, imdb_id, directory_path, poster_url, created_at,
        COALESCE(aliases,''), COALESCE(description,''), COALESCE(country,''),
        COALESCE(language,''), COALESCE(runtime,0), COALESCE(top250_rank,0),
        COALESCE(short_comments_count,0), COALESCE(reviews_count,0),
        (SELECT COUNT(*) FROM posters p WHERE p.movie_id = movies.id)
        FROM movies WHERE sid = ?`, sid).Scan(
		&d.ID, &d.SID, &d.Title, &d.OriginalTitle, &d.Year, &d.Rating,
		&d.RatingCount, &d.IMDbID, &d.DirectoryPath, &d.PosterURL, &d.CreatedAt,
		&d.Aliases, &d.Description, &d.Country, &d.Language, &d.Runtime,
		&d.Top250Rank, &d.ShortCommentsCount, &d.ReviewsCount, &d.PosterCount)
	if err != nil {
		return d, err
	}
	d.Directors = s.names(ctx, "directors", "name", d.ID)
	d.Actors = s.names(ctx, "actors", "name", d.ID)
	d.Genres = s.names(ctx, "genres", "name", d.ID)
	d.ReleaseDates = s.names(ctx, "release_dates", "date_text", d.ID)

	d.Awards = make([]string, 0)
	if rows, err := s.db.QueryContext(ctx,
		"SELECT event, category, recipient FROM awards WHERE movie_id = ?", d.ID); err == nil {
		for rows.Next() {
			var event, category, recipient sql.NullString
			if rows.Scan(&event, &category, &recipient) == nil {
				parts := make([]string, 0, 3)
				for _, v := range []sql.NullString{event, category, recipient} {
					if v.Valid && v.String != "" {
						parts = append(parts, v.String)
					}
				}
				d.Awards = append(d.Awards, strings.Join(parts, " / "))
			}
		}
		_ = rows.Close()
	}
	d.SimilarMovies = make([]string, 0)

	d.Posters = make([]model.PosterRecord, 0)
	rows, err := s.db.QueryContext(ctx, `SELECT id, COALESCE(poster_id,''), COALESCE(title,''),
        COALESCE(keyword,''), COALESCE(region,''), COALESCE(thumbnail_url,''),
        COALESCE(original_url,''), COALESCE(file_size,0), COALESCE(is_downloaded,0),
        COALESCE(local_path,'') FROM posters WHERE movie_id = ? ORDER BY id`, d.ID)
	if err == nil {
		for rows.Next() {
			var p model.PosterRecord
			var downloaded int
			if rows.Scan(&p.ID, &p.PosterID, &p.Title, &p.Keyword, &p.Region,
				&p.ThumbnailURL, &p.OriginalURL, &p.FileSize, &downloaded, &p.LocalPath) == nil {
				p.IsDownloaded = downloaded != 0
				d.Posters = append(d.Posters, p)
			}
		}
		_ = rows.Close()
	}
	return d, nil
}

func (s *Store) names(ctx context.Context, table, column string, movieID int64) []string {
	out := make([]string, 0)
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf("SELECT %s FROM %s WHERE movie_id = ? ORDER BY id", column, table), movieID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			out = append(out, name)
		}
	}
	return out
}

func (s *Store) Stats(ctx context.Context) (model.HistoryStats, error) {
	var stats model.HistoryStats
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM movies").Scan(&stats.Total); err != nil {
		return stats, err
	}
	var avg sql.NullFloat64
	if err := s.db.QueryRowContext(ctx,
		"SELECT AVG(CAST(rating AS REAL)) FROM movies WHERE rating != ''").Scan(&avg); err != nil {
		return stats, err
	}
	if avg.Valid {
		stats.AverageRating = avg.Float64
	}
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM posters").Scan(&stats.PosterTotal); err != nil {
		return stats, err
	}
	return stats, nil
}
