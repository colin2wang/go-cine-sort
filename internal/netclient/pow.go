package netclient

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	rePowTok    = regexp.MustCompile(`name="tok" value="([^"]+)"`)
	rePowCha    = regexp.MustCompile(`name="cha" value="([^"]+)"`)
	rePowRed    = regexp.MustCompile(`name="red" value="([^"]+)"`)
	rePowAction = regexp.MustCompile(`<form[^>]+action="([^"]+)"`)
)

const (
	powDifficulty = 4
	powMaxNonce   = 5_000_000
)

// CalculateNonce solves sha512(challenge+nonce) with `difficulty` leading zero
// hex characters. Mirrors utils/common_util.py:calculate_nonce.
func CalculateNonce(ctx context.Context, challenge string, difficulty int) (string, error) {
	target := strings.Repeat("0", difficulty)
	for nonce := 1; nonce <= powMaxNonce; nonce++ {
		if nonce%1000 == 0 {
			if err := ctx.Err(); err != nil {
				return "", err
			}
		}
		sum := sha512.Sum512([]byte(challenge + strconv.Itoa(nonce)))
		if strings.HasPrefix(hex.EncodeToString(sum[:]), target) {
			return strconv.Itoa(nonce), nil
		}
	}
	return "", fmt.Errorf("PoW 未在 %d 次尝试内找到解", powMaxNonce)
}

func firstMatch(re *regexp.Regexp, text string) (string, bool) {
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return "", false
	}
	return m[1], true
}

// FetchWithVerification loads a Douban page, transparently solving the PoW
// challenge when present. Unlike the Python version it reuses the shared
// cookie jar, so verification is not repeated for every subsequent request.
// purpose is recorded in the response cache for debugging.
func (c *Client) FetchWithVerification(ctx context.Context, targetURL string, maxRetries int, purpose string) (*Response, error) {
	if maxRetries <= 0 {
		maxRetries = 2
	}
	opts := Options{Timeout: 15 * time.Second, Referer: DoubanReferer, Purpose: purpose}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		resp, err := c.Get(ctx, targetURL, opts)
		if err != nil {
			lastErr = err
			c.log.Warn("pow", "第 %d/%d 轮请求失败: %v", attempt, maxRetries, err)
		} else if !strings.Contains(resp.Text(), `<form name="sec" id="sec"`) {
			return resp, nil
		} else {
			c.log.Warn("pow", "检测到豆瓣验证页（第 %d/%d 轮）", attempt, maxRetries)

			tok, okTok := firstMatch(rePowTok, resp.Text())
			cha, okCha := firstMatch(rePowCha, resp.Text())
			red, okRed := firstMatch(rePowRed, resp.Text())
			action, okAction := firstMatch(rePowAction, resp.Text())
			if !okTok || !okCha || !okRed || !okAction {
				lastErr = fmt.Errorf("验证页参数解析失败")
			} else {
				base, err := url.Parse(resp.URL)
				if err != nil {
					lastErr = fmt.Errorf("解析验证页 URL: %w", err)
				} else {
					actionRef, err := url.Parse(strings.TrimSpace(action))
					if err != nil {
						lastErr = fmt.Errorf("解析表单 action: %w", err)
					} else {
						actionURL := base.ResolveReference(actionRef).String()
						sol, err := CalculateNonce(ctx, cha, powDifficulty)
						if err != nil {
							lastErr = err
						} else {
							c.log.Info("pow", "PoW 解算完成 sol=%s，提交验证表单", sol)
							form := url.Values{"tok": {tok}, "cha": {cha}, "sol": {sol}, "red": {red}}
							verify, err := c.PostForm(ctx, actionURL, form, Options{
								Timeout: 20 * time.Second,
								Referer: DoubanReferer,
								Purpose: "豆瓣 PoW 验证",
							})
							if err != nil {
								lastErr = err
							} else if strings.TrimRight(verify.URL, "/") == strings.TrimRight(red, "/") {
								c.log.Info("pow", "验证通过，已跳转 %s", verify.URL)
								return verify, nil
							} else {
								lastErr = fmt.Errorf("验证后跳转异常: %s（期望 %s）", verify.URL, red)
							}
						}
					}
				}
			}
		}

		c.log.Warn("pow", "第 %d/%d 轮未通过: %v", attempt, maxRetries, lastErr)
		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(2 * time.Second):
			}
		}
	}
	return nil, fmt.Errorf("豆瓣验证失败（已重试 %d 轮）: %w", maxRetries, lastErr)
}
