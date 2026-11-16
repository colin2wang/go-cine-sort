package douban

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"go-cine-sort/internal/logging"
	"go-cine-sort/internal/model"
	"go-cine-sort/internal/netclient"
)

var (
	reDetailTitle     = regexp.MustCompile(`<span property="v:itemreviewed">([^<]+)</span>`)
	reDetailRating    = regexp.MustCompile(`<strong class="ll rating_num" property="v:average">([\d.]+)</strong>`)
	reDetailYear      = regexp.MustCompile(`<span class="year">\((\d{4})\)</span>`)
	reDetailDirectors = regexp.MustCompile(`导演:</span>.*?<a[^>]*>([^<]+)</a>`)
	reDetailActors    = regexp.MustCompile(`主演:</span>.*?<a[^>]*>([^<]+)</a>`)
	reDirectorBlock   = regexp.MustCompile(`(?s)<span class=['"]pl['"]>导演</span>:\s*<span class=['"]attrs['"]>(.*?)</span>`)
	reActorBlock      = regexp.MustCompile(`(?s)<span class=['"]pl['"]>主演</span>:\s*<span class=['"]attrs['"]>(.*?)</span>`)
	reAnchorText      = regexp.MustCompile(`<a[^>]*>([^<]+)</a>`)
	reGenres          = regexp.MustCompile(`<span property="v:genre">([^<]+)</span>`)
	reDescFull        = regexp.MustCompile(`(?s)<span property="v:summary" class="all hidden">\s*(.*?)\s*</span>`)
	reDescShort       = regexp.MustCompile(`(?s)<span property="v:summary">\s*(.*?)\s*(?:<br />|</span>)`)
	reMetaDesc        = regexp.MustCompile(`<meta name="description" content="([^">]+)"`)
	reJSONDesc        = regexp.MustCompile(`"description":\s*"([^"]+)"`)
	reAliases         = regexp.MustCompile(`又名:</span>([^<]+)`)
	reHanChar         = regexp.MustCompile(`[\x{4e00}-\x{9fff}]`)
	reParenHan        = regexp.MustCompile(`[(（][^)）]*[\x{4e00}-\x{9fff}][^)）]*[)）]`)
	reNonEnglish      = regexp.MustCompile(`[^a-zA-Z0-9\s.\-]`)
	reRatingPer       = regexp.MustCompile(`<span class="rating_per">([\d.]+)%</span>`)
	reAwardBlock      = regexp.MustCompile(`(?s)<ul class="award">.*?<li>(.*?)</li>.*?<li>(.*?)</li>.*?<li>(.*?)</li>`)
	reSimilar         = regexp.MustCompile(`(?s)<a href="https://movie\.douban\.com/subject/(\d+)/[^>]*>.*?<img src="[^"]+" alt="([^"]+)" />.*?<span class="subject-rate">([\d.]+)</span>`)
	reCountry         = regexp.MustCompile(`制片国家/地区:</span>([^<]+)`)
	reDetailLanguage  = regexp.MustCompile(`语言:</span>([^<]+)`)
	reReleaseDate     = regexp.MustCompile(`<span property="v:initialReleaseDate"[^>]*>([^<]+)</span>`)
	reRuntime         = regexp.MustCompile(`<span property="v:runtime"[^>]*>(\d+)`)
	// IMDb 渲染为超链接（<a href="...imdb.com/title/tt…">tt…</a>）时，纯文本
	// 形式 `[^<\s]+` 会静默失配，导致 details.IMDbID 为空、评分文件被跳过。
	reIMDbID        = regexp.MustCompile(`IMDb:</span>\s*(?:<a[^>]*>)?(tt\d+)`)
	reVotes         = regexp.MustCompile(`<span property="v:votes">(\d+)</span>`)
	rePosterURL     = regexp.MustCompile(`<img[^>]+src="([^"]+)"[^>]*title="点击看更多海报"`)
	reShortComments = regexp.MustCompile(`<a href="https://movie\.douban\.com/subject/\d+/comments[^>]*>全部 (\d+) 条</a>`)
	reReviews       = regexp.MustCompile(`<a href="reviews">全部 (\d+) 条</a>`)
	reTop250        = regexp.MustCompile(`<div class="top250"><span class="top250-no">No\.(\d+)</span>`)
)

type Details struct {
	net *netclient.Client
	log *logging.Logger
}

func NewDetails(net *netclient.Client, log *logging.Logger) *Details {
	return &Details{net: net, log: log}
}

// Fetch loads the subject page, transparently handling Douban verification.
func (d *Details) Fetch(ctx context.Context, sid string) (string, error) {
	target := fmt.Sprintf("https://movie.douban.com/subject/%s/", sid)
	d.log.Info("details", "抓取详情页 SID=%s", sid)
	resp, err := d.net.FetchWithVerification(ctx, target, 2, "豆瓣详情页")
	if err != nil {
		return "", fmt.Errorf("豆瓣详情页获取失败 (SID %s): %w", sid, err)
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("豆瓣详情页 HTTP %d (SID %s)", resp.StatusCode, sid)
	}
	return resp.Text(), nil
}

func extractSingle(re *regexp.Regexp, content string) string {
	m := re.FindStringSubmatch(content)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func extractMultiple(re *regexp.Regexp, content string, limit int) []string {
	out := make([]string, 0)
	for _, m := range re.FindAllStringSubmatch(content, -1) {
		if len(m) < 2 {
			continue
		}
		out = append(out, strings.TrimSpace(m[1]))
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func cleanHTML(text string) string {
	if text == "" {
		return ""
	}
	text = reAnyTag.ReplaceAllString(text, "")
	text = html.UnescapeString(text)
	return strings.Join(strings.Fields(text), " ")
}

func toInt(text string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(text))
	return n
}

func toFloat(text string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(text), 64)
	return f
}

// ParseDetails is a pure function mirroring
// utils/douban_details.py:DoubanMovieDetailsParser.parse().
func ParseDetails(content string) model.MovieDetails {
	result := model.MovieDetails{
		Directors:          []string{},
		Actors:             []string{},
		Genres:             []string{},
		ReleaseDates:       []string{},
		RatingDistribution: map[string]float64{},
		Awards:             []model.Award{},
		SimilarMovies:      []model.SimilarMovie{},
	}
	if strings.TrimSpace(content) == "" {
		return result
	}

	result.Title = html.UnescapeString(extractSingle(reDetailTitle, content))
	result.Rating = extractSingle(reDetailRating, content)
	result.Year = extractSingle(reDetailYear, content)

	// Directors / actors: keep the original regex results, and fall back to the
	// common "attrs" block so multi-director / full cast lists are not lost.
	result.Directors = extractMultiple(reDetailDirectors, content, 0)
	result.Actors = extractMultiple(reDetailActors, content, 10)
	if len(result.Directors) == 0 {
		if block := extractSingle(reDirectorBlock, content); block != "" {
			result.Directors = extractMultiple(reAnchorText, block, 0)
		}
	}
	if len(result.Actors) <= 1 {
		if block := extractSingle(reActorBlock, content); block != "" {
			if all := extractMultiple(reAnchorText, block, 10); len(all) > len(result.Actors) {
				result.Actors = all
			}
		}
	}
	result.Genres = extractMultiple(reGenres, content, 0)

	// Description: 4-level fallback, identical priority to Python.
	switch {
	case reDescFull.MatchString(content):
		result.Description = cleanHTML(extractSingle(reDescFull, content))
	case reDescShort.MatchString(content):
		result.Description = cleanHTML(extractSingle(reDescShort, content))
	case reMetaDesc.MatchString(content):
		result.Description = strings.TrimSpace(extractSingle(reMetaDesc, content))
	case reJSONDesc.MatchString(content):
		result.Description = strings.TrimSpace(extractSingle(reJSONDesc, content))
	}

	// Original title from aliases: first segment without Chinese characters.
	if aliases := extractSingle(reAliases, content); aliases != "" {
		result.Aliases = aliases
		parts := make([]string, 0)
		for _, p := range strings.Split(aliases, "/") {
			if p = strings.TrimSpace(p); p != "" {
				parts = append(parts, p)
			}
		}
		found := false
		for _, p := range parts {
			if !reHanChar.MatchString(p) {
				result.OriginalTitle = p
				found = true
				break
			}
		}
		if !found && len(parts) > 0 {
			english := strings.TrimSpace(reParenHan.ReplaceAllString(parts[0], ""))
			if reHanChar.MatchString(english) {
				english = strings.TrimSpace(reNonEnglish.ReplaceAllString(parts[0], ""))
			}
			result.OriginalTitle = english
		}
	}

	result.Country = extractSingle(reCountry, content)
	result.Language = extractSingle(reDetailLanguage, content)
	result.ReleaseDates = extractMultiple(reReleaseDate, content, 0)
	result.Runtime = toInt(extractSingle(reRuntime, content))
	result.IMDbID = extractSingle(reIMDbID, content)
	result.RatingCount = toInt(extractSingle(reVotes, content))

	if per := extractMultiple(reRatingPer, content, 5); len(per) >= 5 {
		result.RatingDistribution = map[string]float64{
			"5_star": toFloat(per[0]), "4_star": toFloat(per[1]),
			"3_star": toFloat(per[2]), "2_star": toFloat(per[3]),
			"1_star": toFloat(per[4]),
		}
	}

	result.PosterURL = extractSingle(rePosterURL, content)

	for _, m := range reAwardBlock.FindAllStringSubmatch(content, -1) {
		if len(m) < 4 {
			continue
		}
		award := model.Award{
			Event:     cleanHTML(m[1]),
			Category:  cleanHTML(m[2]),
			Recipient: cleanHTML(m[3]),
		}
		if award.Event != "" || award.Category != "" || award.Recipient != "" {
			result.Awards = append(result.Awards, award)
		}
	}

	for _, m := range reSimilar.FindAllStringSubmatch(content, -1) {
		if len(m) < 4 {
			continue
		}
		result.SimilarMovies = append(result.SimilarMovies, model.SimilarMovie{
			SID:    m[1],
			Title:  strings.TrimSpace(m[2]),
			Rating: toFloat(m[3]),
		})
		if len(result.SimilarMovies) >= 10 {
			break
		}
	}

	result.ShortCommentsCount = toInt(extractSingle(reShortComments, content))
	result.ReviewsCount = toInt(extractSingle(reReviews, content))
	result.Top250Rank = toInt(extractSingle(reTop250, content))
	return result
}

// ParseAndFetch is kept for callers that want the combined operation.
func (d *Details) ParseAndFetch(ctx context.Context, sid string) (model.MovieDetails, error) {
	content, err := d.Fetch(ctx, sid)
	if err != nil {
		return model.MovieDetails{}, err
	}
	details := ParseDetails(content)
	details.SID = sid
	return details, nil
}
