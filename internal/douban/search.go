// Package douban fetches and parses public Douban HTML pages.
package douban

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"go-cine-sort/internal/logging"
	"go-cine-sort/internal/model"
	"go-cine-sort/internal/netclient"
)

var (
	reChineseQuery  = regexp.MustCompile(`[\x{4e00}-\x{9fff}\x{ff1a}\x{ff0c}\x{ff01}\x{ff1f}]+`)
	reResultBlock   = regexp.MustCompile(`(?s)<div class="result">.*?</div>\s*</div>`)
	reSearchTitle   = regexp.MustCompile(`<h3>\s*<span>\[电影\]</span>\s*&nbsp;<a[^>]*>([^<]+)</a>`)
	reSearchSID     = regexp.MustCompile(`sid:\s*(\d+)`)
	reSearchRating  = regexp.MustCompile(`<span[^>]*class="rating_nums"[^>]*>([\d.]+)</span>`)
	reSearchReviews = regexp.MustCompile(`<span>\((\d+)人评价\)</span>`)
	reOriginalTitle = regexp.MustCompile(`原名:([^/]+)`)
	reSubjectCast   = regexp.MustCompile(`(?s)<span[^>]*class="subject-cast"[^>]*>(.*?)</span>`)
	reAnyTag        = regexp.MustCompile(`<[^>]*>`)
	reAliasPrefix   = regexp.MustCompile(`^原名:[^/]*/`)
	reYearOnly      = regexp.MustCompile(`^\d{4}$`)
	reFourDigits    = regexp.MustCompile(`(\d{4})`)
	reAllTitles     = regexp.MustCompile(`(?s)<h3>\s*<span>\[电影\]</span>\s*&nbsp;<a[^>]*href="[^"]*?">([^<]+?)</a>`)
	reSimpleTitle   = regexp.MustCompile(`<a[^>]*>([^<>]{2,50}?)</a>`)
)

var excludedTitleKeywords = []string{"可播放", "预告", "花絮", "回顶部", "↑", "&#8593;"}

// PrepareQuery mirrors utils/douban_search.py:_prepare_search_query: only the
// Chinese segments are used as the keyword when present.
func PrepareQuery(name, year string) string {
	parts := reChineseQuery.FindAllString(name, -1)
	if len(parts) > 0 {
		query := strings.Join(parts, " ")
		if year != "" {
			return query + " " + year
		}
		return query
	}
	if year != "" {
		return name + " " + year
	}
	return name
}

type Search struct {
	net *netclient.Client
	log *logging.Logger
}

func NewSearch(net *netclient.Client, log *logging.Logger) *Search {
	return &Search{net: net, log: log}
}

// Search fetches the Douban search page and returns the first valid movie hit.
func (s *Search) Search(ctx context.Context, name, year string) (model.SearchResult, error) {
	query := PrepareQuery(name, year)
	u, err := url.Parse("https://www.douban.com/search")
	if err != nil {
		return model.SearchResult{}, err
	}
	q := u.Query()
	q.Set("cat", "1002")
	q.Set("q", query)
	u.RawQuery = q.Encode()

	s.log.Info("search", "GET %s", u.String())
	resp, err := s.net.Get(ctx, u.String(), netclient.Options{Timeout: 10 * time.Second})
	if err != nil {
		return model.SearchResult{}, fmt.Errorf("豆瓣搜索请求失败: %w", err)
	}
	result := ParseSearchResult(resp.Text())
	if result.SID == "" {
		return result, fmt.Errorf("豆瓣搜索未找到有效 SID（关键词 %q）", query)
	}
	s.log.Info("search", "命中 %s (SID %s, 评分 %s)", result.Title, result.SID, result.Rating)
	return result, nil
}

func emptySearchResult(errMsg string) model.SearchResult {
	return model.SearchResult{
		Directors: []string{}, Actors: []string{}, Genres: []string{},
		Error: errMsg,
	}
}

// ParseSearchResult is a pure function so it can be unit-tested against the
// offline HTML fixtures in docs/.
func ParseSearchResult(htmlContent string) model.SearchResult {
	info := emptySearchResult("")
	if strings.TrimSpace(htmlContent) == "" {
		return emptySearchResult("解析错误：HTML 内容为空")
	}

	for _, block := range reResultBlock.FindAllString(htmlContent, -1) {
		if !strings.Contains(block, "[电影]") || !strings.Contains(block, "rating_nums") {
			continue
		}
		if m := reSearchTitle.FindStringSubmatch(block); len(m) > 1 {
			info.Title = strings.TrimSpace(m[1])
		}
		if m := reSearchSID.FindStringSubmatch(block); len(m) > 1 {
			info.SID = m[1]
		}
		if m := reSearchRating.FindStringSubmatch(block); len(m) > 1 {
			info.Rating = m[1]
		}
		if m := reSearchReviews.FindStringSubmatch(block); len(m) > 1 {
			info.ReviewCount = m[1]
		}
		if m := reOriginalTitle.FindStringSubmatch(block); len(m) > 1 {
			info.OriginalTitle = strings.TrimSpace(m[1])
		}
		if m := reSubjectCast.FindStringSubmatch(block); len(m) > 1 {
			castText := strings.TrimSpace(reAnyTag.ReplaceAllString(m[1], ""))
			if years := reFourDigits.FindAllString(castText, -1); len(years) > 0 {
				info.Year = years[len(years)-1]
			}
			clean := strings.TrimSpace(reAliasPrefix.ReplaceAllString(castText, ""))
			parts := make([]string, 0)
			for _, p := range strings.Split(clean, "/") {
				if p = strings.TrimSpace(p); p != "" {
					parts = append(parts, p)
				}
			}
			if len(parts) >= 2 {
				info.Directors = []string{parts[0]}
				actors := make([]string, 0, 5)
				for _, p := range parts[1:] {
					if reYearOnly.MatchString(p) {
						break
					}
					actors = append(actors, p)
				}
				if len(actors) > 5 {
					actors = actors[:5]
				}
				info.Actors = actors
			}
		}
		break
	}

	if info.Title == "" {
		if titles := reAllTitles.FindAllStringSubmatch(htmlContent, -1); len(titles) > 0 {
			info.Title = strings.TrimSpace(titles[0][1])
		}
		if info.Title == "" {
			for _, m := range reSimpleTitle.FindAllStringSubmatch(htmlContent, -1) {
				candidate := strings.TrimSpace(m[1])
				skip := false
				for _, kw := range excludedTitleKeywords {
					if strings.Contains(candidate, kw) {
						skip = true
						break
					}
				}
				if candidate != "" && !skip {
					info.Title = candidate
					break
				}
			}
		}
	}

	if info.SID == "" {
		return emptySearchResult("解析错误：未找到电影 SID")
	}
	return info
}
