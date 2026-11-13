// Package model holds the domain types shared by the backend layers and the
// Wails bindings. JSON names are kept in camelCase to match frontend TS types.
package model

type MovieFileInfo struct {
	FilePath    string `json:"filePath"`
	MovieName   string `json:"movieName"`
	Year        string `json:"year"`
	Extension   string `json:"extension"`
	RawFilename string `json:"rawFilename"`
}

type SearchResult struct {
	Title         string   `json:"title"`
	Rating        string   `json:"rating"`
	Directors     []string `json:"directors"`
	Actors        []string `json:"actors"`
	Year          string   `json:"year"`
	Genres        []string `json:"genres"`
	OriginalTitle string   `json:"originalTitle"`
	ReviewCount   string   `json:"reviewCount"`
	SID           string   `json:"sid"`
	Error         string   `json:"error"`
}

type Award struct {
	Event     string `json:"event"`
	Category  string `json:"category"`
	Recipient string `json:"recipient"`
}

type SimilarMovie struct {
	SID    string  `json:"sid"`
	Title  string  `json:"title"`
	Rating float64 `json:"rating"`
}

// MovieDetails mirrors utils/douban_details.py parse_movie_details_result().
type MovieDetails struct {
	SID                string             `json:"sid"`
	Title              string             `json:"title"`
	OriginalTitle      string             `json:"originalTitle"`
	Year               string             `json:"year"`
	Rating             string             `json:"rating"`
	Directors          []string           `json:"directors"`
	Actors             []string           `json:"actors"`
	Genres             []string           `json:"genres"`
	Description        string             `json:"description"`
	Aliases            string             `json:"aliases"`
	Country            string             `json:"country"`
	Language           string             `json:"language"`
	ReleaseDates       []string           `json:"releaseDates"`
	Runtime            int                `json:"runtime"`
	IMDbID             string             `json:"imdbId"`
	RatingCount        int                `json:"ratingCount"`
	RatingDistribution map[string]float64 `json:"ratingDistribution"`
	PosterURL          string             `json:"posterUrl"`
	Awards             []Award            `json:"awards"`
	SimilarMovies      []SimilarMovie     `json:"similarMovies"`
	ShortCommentsCount int                `json:"shortCommentsCount"`
	ReviewsCount       int                `json:"reviewsCount"`
	Top250Rank         int                `json:"top250Rank"`
	SearchTitle        string             `json:"searchTitle"`
}

type Poster struct {
	PhotoURL  string `json:"photoUrl"`
	Thumbnail string `json:"thumbnail"`
	Original  string `json:"original"`
	Title     string `json:"title"`
	Size      int64  `json:"size"`
	Region    string `json:"region"`
	PosterID  string `json:"posterId"`
}

type PosterRecord struct {
	ID           int64  `json:"id"`
	PosterID     string `json:"posterId"`
	Title        string `json:"title"`
	Keyword      string `json:"keyword"`
	Region       string `json:"region"`
	ThumbnailURL string `json:"thumbnailUrl"`
	OriginalURL  string `json:"originalUrl"`
	FileSize     int64  `json:"fileSize"`
	IsDownloaded bool   `json:"isDownloaded"`
	LocalPath    string `json:"localPath"`
}

type ProcessResult struct {
	Filename    string `json:"filename"`
	FilePath    string `json:"filePath"`
	MovieName   string `json:"movieName"`
	Year        string `json:"year"`
	Directory   string `json:"directory"`
	VideoPath   string `json:"videoPath"`
	Error       string `json:"error"`
	Success     bool   `json:"success"`
	Skipped     bool   `json:"skipped"`
	SID         string `json:"sid"`
	Title       string `json:"title"`
	Rating      string `json:"rating"`
	PosterCount int    `json:"posterCount"`
}

type TaskProgress struct {
	Total     int     `json:"total"`
	Completed int     `json:"completed"`
	Success   int     `json:"success"`
	Failed    int     `json:"failed"`
	Current   string  `json:"current"`
	Stage     string  `json:"stage"`
	Percent   float64 `json:"percent"`
}

type TaskSummary struct {
	Total     int   `json:"total"`
	Processed int   `json:"processed"`
	Success   int   `json:"success"`
	Failed    int   `json:"failed"`
	ElapsedMs int64 `json:"elapsedMs"`
	Cancelled bool  `json:"cancelled"`
}

type MovieRecord struct {
	ID            int64  `json:"id"`
	SID           string `json:"sid"`
	Title         string `json:"title"`
	OriginalTitle string `json:"originalTitle"`
	Year          string `json:"year"`
	Rating        string `json:"rating"`
	RatingCount   int    `json:"ratingCount"`
	IMDbID        string `json:"imdbId"`
	DirectoryPath string `json:"directoryPath"`
	PosterURL     string `json:"posterUrl"`
	CreatedAt     string `json:"createdAt"`
	PosterCount   int    `json:"posterCount"`
}

type MovieDetail struct {
	MovieRecord
	Aliases            string         `json:"aliases"`
	Description        string         `json:"description"`
	Country            string         `json:"country"`
	Language           string         `json:"language"`
	Runtime            int            `json:"runtime"`
	Top250Rank         int            `json:"top250Rank"`
	ShortCommentsCount int            `json:"shortCommentsCount"`
	ReviewsCount       int            `json:"reviewsCount"`
	Directors          []string       `json:"directors"`
	Actors             []string       `json:"actors"`
	Genres             []string       `json:"genres"`
	ReleaseDates       []string       `json:"releaseDates"`
	Awards             []string       `json:"awards"`
	SimilarMovies      []string       `json:"similarMovies"`
	Posters            []PosterRecord `json:"posters"`
}

type HistoryStats struct {
	Total         int     `json:"total"`
	AverageRating float64 `json:"averageRating"`
	PosterTotal   int     `json:"posterTotal"`
}
