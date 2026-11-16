// Package cache provides a process-local cache of every external HTTP
// response (Douban / IMDb API and page calls) so the frontend can inspect and
// debug raw payloads without re-fetching.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// defaultTTL keeps responses replayable for repeated runs within an hour;
// janitor runs at half the TTL so expired entries do not pile up.
const (
	defaultTTL      = time.Hour
	cleanupInterval = 30 * time.Minute
)

// Entry is an immutable snapshot of one HTTP exchange stored in the cache.
type Entry struct {
	Key        string            `json:"key"`
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	Source     string            `json:"source"`
	Purpose    string            `json:"purpose"`
	StatusCode int               `json:"statusCode"`
	Header     map[string]string `json:"header"`
	Body       string            `json:"body"`
	BodyBytes  int               `json:"bodyBytes"`
	CachedAt   time.Time         `json:"cachedAt"`
	ExpiresAt  time.Time         `json:"expiresAt"`
}

// Summary is the list-view projection of an Entry (without the body).
type Summary struct {
	Key        string    `json:"key"`
	Method     string    `json:"method"`
	URL        string    `json:"url"`
	Source     string    `json:"source"`
	Purpose    string    `json:"purpose"`
	StatusCode int       `json:"statusCode"`
	BodyBytes  int       `json:"bodyBytes"`
	CachedAt   time.Time `json:"cachedAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// Stats describes the current cache size.
type Stats struct {
	Count     int   `json:"count"`
	TotalBody int64 `json:"totalBodyBytes"`
}

// Store wraps go-cache with response recording and lookup helpers. All
// entries are immutable once stored, so storing values (not shared mutable
// state) is safe under go-cache's reference semantics.
type Store struct {
	c   *gocache.Cache
	mu  sync.RWMutex // guards bodyTotal for Stats
	sum int64
}

// New creates the singleton store. Expiration must be non-zero and the
// cleanup interval must stay positive so the janitor keeps memory bounded.
func New() *Store {
	return &Store{c: gocache.New(defaultTTL, cleanupInterval)}
}

// Key derives a stable cache key from method, URL and optional request body.
func Key(method, url string, body []byte) string {
	h := sha256.Sum256(append([]byte(method+"\x00"+url+"\x00"), body...))
	return hex.EncodeToString(h[:16])
}

// Source guesses the upstream site from the URL for display purposes.
func Source(rawURL string) string {
	switch {
	case strings.Contains(rawURL, "douban.com"):
		return "douban"
	case strings.Contains(rawURL, "imdb.com"):
		return "imdb"
	default:
		return "other"
	}
}

// PutResponse records a successful HTTP exchange under key. It takes raw
// fields instead of *netclient.Response to keep this package dependency-free.
func (s *Store) PutResponse(key, purpose, method string, statusCode int, header http.Header, url string, body []byte) {
	if s == nil {
		return
	}
	headerMap := make(map[string]string, len(header))
	for k, v := range header {
		if len(v) > 0 {
			headerMap[k] = v[len(v)-1]
		}
	}
	entry := &Entry{
		Key:        key,
		Method:     method,
		URL:        url,
		Source:     Source(url),
		Purpose:    purpose,
		StatusCode: statusCode,
		Header:     headerMap,
		Body:       string(body),
		BodyBytes:  len(body),
		CachedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(defaultTTL),
	}
	s.mu.Lock()
	s.sum += int64(len(body))
	s.mu.Unlock()
	s.c.Set(key, entry, gocache.DefaultExpiration)
}

// GetEntry returns a cached entry by key. The entry is immutable; callers
// must not mutate it.
func (s *Store) GetEntry(key string) (*Entry, bool) {
	if s == nil {
		return nil, false
	}
	v, found := s.c.Get(key)
	if !found {
		return nil, false
	}
	entry, ok := v.(*Entry)
	return entry, ok
}

// CachedResponse is the replay payload returned for a cache hit.
type CachedResponse struct {
	StatusCode int
	Body       []byte
	URL        string
	Header     http.Header
}

// Lookup returns a cached response for the given request fingerprint.
func (s *Store) Lookup(key string) (*CachedResponse, bool) {
	entry, ok := s.GetEntry(key)
	if !ok {
		return nil, false
	}
	header := make(http.Header, len(entry.Header))
	for k, v := range entry.Header {
		header[k] = []string{v}
	}
	return &CachedResponse{
		StatusCode: entry.StatusCode,
		Body:       []byte(entry.Body),
		URL:        entry.URL,
		Header:     header,
	}, true
}

// List returns summaries of all live entries, newest first. It reflects the
// logical view (expired-but-not-yet-swept entries are excluded).
func (s *Store) List() []Summary {
	if s == nil {
		return []Summary{}
	}
	now := time.Now()
	var out []Summary
	for _, item := range s.c.Items() {
		entry, ok := item.Object.(*Entry)
		if !ok {
			continue
		}
		expiresAt := time.Unix(0, item.Expiration)
		if !expiresAt.After(now) {
			continue
		}
		out = append(out, Summary{
			Key:        entry.Key,
			Method:     entry.Method,
			URL:        entry.URL,
			Source:     entry.Source,
			Purpose:    entry.Purpose,
			StatusCode: entry.StatusCode,
			BodyBytes:  entry.BodyBytes,
			CachedAt:   entry.CachedAt,
			ExpiresAt:  time.Unix(0, item.Expiration),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CachedAt.After(out[j].CachedAt) })
	if out == nil {
		out = []Summary{}
	}
	return out
}

// Stats reports the live entry count and total cached body bytes.
func (s *Store) Stats() Stats {
	if s == nil {
		return Stats{}
	}
	count := 0
	now := time.Now()
	for _, item := range s.c.Items() {
		if _, ok := item.Object.(*Entry); ok && time.Unix(0, item.Expiration).After(now) {
			count++
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Stats{Count: count, TotalBody: s.sum}
}

// Clear drops every entry and resets the byte counter.
func (s *Store) Clear() {
	if s == nil {
		return
	}
	s.c.Flush()
	s.mu.Lock()
	s.sum = 0
	s.mu.Unlock()
}
