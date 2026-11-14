// Package netclient provides a single browser-like HTTP client with a shared
// cookie jar, unified headers, polite rate limiting and per-request timeouts.
package netclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"go-cine-sort/internal/logging"
)

const (
	UserAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	UserAgentIMDb = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	DoubanReferer = "https://www.douban.com/"
	MovieReferer  = "https://movie.douban.com/"

	maxResponseBytes = 16 << 20
)

type Options struct {
	Timeout       time.Duration
	Referer       string
	Headers       map[string]string
	SkipRateLimit bool
}

func (o Options) timeout() time.Duration {
	if o.Timeout <= 0 {
		return 15 * time.Second
	}
	return o.Timeout
}

type Response struct {
	StatusCode int
	Body       []byte
	URL        string
	Header     http.Header
}

func (r *Response) Text() string { return string(r.Body) }

type Client struct {
	http      *http.Client
	log       *logging.Logger
	rateLimit bool
}

func New(log *logging.Logger) *Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		// cookiejar.New only fails with malformed options; nil options never fail.
		panic(err)
	}
	return &Client{
		http:      &http.Client{Jar: jar},
		log:       log,
		rateLimit: true,
	}
}

// SetRateLimit allows tests to disable the polite delay.
func (c *Client) SetRateLimit(enabled bool) { c.rateLimit = enabled }

func (c *Client) pause(ctx context.Context) error {
	if !c.rateLimit {
		return nil
	}
	d := time.Duration(500+rand.Intn(1001)) * time.Millisecond
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *Client) baseHeaders(opts Options) http.Header {
	h := http.Header{}
	h.Set("User-Agent", UserAgent)
	h.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	h.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	if opts.Referer != "" {
		h.Set("Referer", opts.Referer)
	}
	for k, v := range opts.Headers {
		h.Set(k, v)
	}
	return h
}

// Open performs a request and returns the live response. The caller must close
// the body. Used for streaming poster downloads.
func (c *Client) Open(ctx context.Context, method, rawURL string, body io.Reader, opts Options) (*http.Response, error) {
	if !opts.SkipRateLimit {
		if err := c.pause(ctx); err != nil {
			return nil, err
		}
	}
	reqCtx, cancel := context.WithTimeout(ctx, opts.timeout())
	req, err := http.NewRequestWithContext(reqCtx, method, rawURL, body)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("build request %s: %w", rawURL, err)
	}
	req.Header = c.baseHeaders(opts)

	// DEBUG 级别输出所有外部 API 调用的方法与 URL（GET 的查询参数含在 URL 内）。
	c.log.Debug("net", "%s %s", method, rawURL)

	resp, err := c.http.Do(req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("request %s: %w", rawURL, err)
	}
	// Keep the cancel function for the body lifetime via a wrapper.
	resp.Body = &cancelReadCloser{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

type cancelReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelReadCloser) Close() error {
	defer c.cancel()
	return c.ReadCloser.Close()
}

func readResponse(resp *http.Response) (*Response, error) {
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	finalURL := resp.Request.URL.String()
	return &Response{
		StatusCode: resp.StatusCode,
		Body:       data,
		URL:        finalURL,
		Header:     resp.Header.Clone(),
	}, nil
}

func (c *Client) Get(ctx context.Context, rawURL string, opts Options) (*Response, error) {
	resp, err := c.Open(ctx, http.MethodGet, rawURL, nil, opts)
	if err != nil {
		return nil, err
	}
	out, err := readResponse(resp)
	if err != nil {
		return nil, err
	}
	if out.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d for %s", out.StatusCode, rawURL)
	}
	return out, nil
}

func (c *Client) PostForm(ctx context.Context, rawURL string, values url.Values, opts Options) (*Response, error) {
	if opts.Headers == nil {
		opts.Headers = map[string]string{}
	}
	opts.Headers["Content-Type"] = "application/x-www-form-urlencoded"
	c.log.Debug("net", "POST %s 参数=%s", rawURL, values.Encode())
	resp, err := c.Open(ctx, http.MethodPost, rawURL, strings.NewReader(values.Encode()), opts)
	if err != nil {
		return nil, err
	}
	return readResponse(resp)
}

func (c *Client) PostJSON(ctx context.Context, rawURL string, payload any, opts Options) (*Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal JSON payload: %w", err)
	}
	if opts.Headers == nil {
		opts.Headers = map[string]string{}
	}
	opts.Headers["Content-Type"] = "application/json"
	opts.Headers["User-Agent"] = UserAgentIMDb
	c.log.Debug("net", "POST %s 参数=%s", rawURL, data)
	resp, err := c.Open(ctx, http.MethodPost, rawURL, bytes.NewReader(data), opts)
	if err != nil {
		return nil, err
	}
	out, err := readResponse(resp)
	if err != nil {
		return nil, err
	}
	if out.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d for %s", out.StatusCode, rawURL)
	}
	return out, nil
}

func (c *Client) Head(ctx context.Context, rawURL string, opts Options) (http.Header, error) {
	resp, err := c.Open(ctx, http.MethodHead, rawURL, nil, opts)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return resp.Header.Clone(), nil
}

func (c *Client) Logger() *logging.Logger { return c.log }
