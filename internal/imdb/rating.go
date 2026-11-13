// Package imdb fetches a title rating from IMDb's public GraphQL endpoint.
package imdb

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"go-cine-sort/internal/logging"
	"go-cine-sort/internal/netclient"
)

const graphqlURL = "https://graphql.imdb.com/"

var reIMDbID = regexp.MustCompile(`^tt\d+$`)

type Rating struct {
	net *netclient.Client
	log *logging.Logger
}

func NewRating(net *netclient.Client, log *logging.Logger) *Rating {
	return &Rating{net: net, log: log}
}

type graphqlResponse struct {
	Data struct {
		Title *struct {
			RatingsSummary *struct {
				AggregateRating *float64 `json:"aggregateRating"`
				VoteCount       *int     `json:"voteCount"`
			} `json:"ratingsSummary"`
		} `json:"title"`
	} `json:"data"`
}

// Fetch returns nil (without an error) when IMDb has no rating for the title,
// matching utils/imdb_rating.py:get_imdb_rating semantics.
func (r *Rating) Fetch(ctx context.Context, imdbID string) (*float64, error) {
	if !reIMDbID.MatchString(imdbID) {
		return nil, fmt.Errorf("无效的 IMDb ID: %q", imdbID)
	}
	query := fmt.Sprintf(`query { title(id: "%s") { ratingsSummary { aggregateRating voteCount } } }`, imdbID)
	payload := map[string]string{"query": query}

	resp, err := r.net.PostJSON(ctx, graphqlURL, payload, netclient.Options{Timeout: 15 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("IMDb 评分请求失败: %w", err)
	}
	var out graphqlResponse
	if err := json.Unmarshal(resp.Body, &out); err != nil {
		return nil, fmt.Errorf("IMDb 响应解析失败: %w", err)
	}
	if out.Data.Title == nil || out.Data.Title.RatingsSummary == nil ||
		out.Data.Title.RatingsSummary.AggregateRating == nil {
		r.log.Warn("imdb", "%s 无评分数据", imdbID)
		return nil, nil
	}
	value := *out.Data.Title.RatingsSummary.AggregateRating
	r.log.Info("imdb", "%s 评分 %.1f", imdbID, value)
	return &value, nil
}
