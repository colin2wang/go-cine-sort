// Package service orchestrates the two processing pipelines (batch and
// single-SID) and publishes progress/results to the Wails frontend.
package service

import (
	"context"
	"errors"
	"path/filepath"
	"sync"

	"go-cine-sort/internal/cache"
	"go-cine-sort/internal/config"
	"go-cine-sort/internal/douban"
	"go-cine-sort/internal/imdb"
	"go-cine-sort/internal/logging"
	"go-cine-sort/internal/model"
	"go-cine-sort/internal/netclient"
	"go-cine-sort/internal/organizer"
	"go-cine-sort/internal/scanner"
	"go-cine-sort/internal/store"
)

// Events are optional callbacks wired to Wails runtime.EventsEmit by app.go.
type Events struct {
	Progress func(model.TaskProgress)
	Item     func(model.ProcessResult)
	Done     func(model.TaskSummary)
	Error    func(string)
}

type Service struct {
	cfg       *config.AppConfig
	log       *logging.Logger
	net       *netclient.Client
	httpCache *cache.Store
	scanner   *scanner.Scanner
	search    *douban.Search
	details   *douban.Details
	posters   *douban.Posters
	imdb      *imdb.Rating
	org       *organizer.Organizer
	store     *store.Store
	events    Events

	baseCtx context.Context
	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
}

func New(cfg *config.AppConfig, log *logging.Logger, events Events) (*Service, error) {
	net := netclient.New(log)
	httpCache := cache.New()
	net.SetCache(httpCache)
	scan, err := scanner.New(cfg, log)
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(cfg.DataDir, "cine_sort.db")
	st, err := store.Open(dbPath, log)
	if err != nil {
		return nil, err
	}
	posters := douban.NewPosters(net, log)
	posters.SetFormat(cfg.PosterFormat)
	return &Service{
		cfg:       cfg,
		log:       log,
		net:       net,
		httpCache: httpCache,
		scanner:   scan,
		search:    douban.NewSearch(net, log),
		details:   douban.NewDetails(net, log),
		posters:   posters,
		imdb:      imdb.NewRating(net, log),
		org:       organizer.New(cfg, log),
		store:     st,
		events:    events,
		baseCtx:   context.Background(),
	}, nil
}

func (s *Service) Close() error { return s.store.Close() }

// SetContext supplies the Wails application context used for event emission
// and as the parent of every cancellable task.
func (s *Service) SetContext(ctx context.Context) {
	if ctx != nil {
		s.baseCtx = ctx
	}
}

func (s *Service) Config() *config.AppConfig { return s.cfg }

// ReloadConfig applies a saved configuration without restarting the app.
func (s *Service) ReloadConfig(cfg *config.AppConfig) error {
	if err := s.scanner.Refresh(cfg); err != nil {
		return err
	}
	s.cfg = cfg
	s.org.SetConfig(cfg)
	s.posters.SetFormat(cfg.PosterFormat)
	return nil
}

func (s *Service) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Service) ScanDirectory(ctx context.Context, dir string, recursive bool) ([]model.MovieFileInfo, error) {
	return s.scanner.ScanDirectory(ctx, dir, recursive)
}

func (s *Service) InfoForPaths(paths []string) []model.MovieFileInfo {
	return s.scanner.InfoForPaths(paths)
}

func (s *Service) ListHistory(ctx context.Context, keyword string) ([]model.MovieRecord, error) {
	return s.store.ListHistory(ctx, keyword)
}

func (s *Service) GetHistoryDetail(ctx context.Context, sid string) (model.MovieDetail, error) {
	return s.store.GetDetail(ctx, sid)
}

func (s *Service) Stats(ctx context.Context) (model.HistoryStats, error) {
	return s.store.Stats(ctx)
}

// CacheList returns summaries of all cached HTTP responses.
func (s *Service) CacheList() []cache.Summary { return s.httpCache.List() }

// CacheEntry returns one cached HTTP response body by key.
func (s *Service) CacheEntry(key string) (*cache.Entry, bool) {
	return s.httpCache.GetEntry(key)
}

// CacheStats reports cache size counters.
func (s *Service) CacheStats() cache.Stats { return s.httpCache.Stats() }

// CacheClear drops every cached response.
func (s *Service) CacheClear() { s.httpCache.Clear() }

func (s *Service) emitProgress(p model.TaskProgress) {
	if s.events.Progress != nil {
		s.events.Progress(p)
	}
}
func (s *Service) emitItem(r model.ProcessResult) {
	if s.events.Item != nil {
		s.events.Item(r)
	}
}
func (s *Service) emitDone(summary model.TaskSummary) {
	if s.events.Done != nil {
		s.events.Done(summary)
	}
}
func (s *Service) emitError(message string) {
	s.log.Error("service", "%s", message)
	if s.events.Error != nil {
		s.events.Error(message)
	}
}

func (s *Service) begin() (context.Context, context.CancelFunc, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil, nil, errors.New("已有任务正在运行，请先等待完成或取消")
	}
	ctx, cancel := context.WithCancel(s.baseCtx)
	s.running = true
	s.cancel = cancel
	return ctx, cancel, nil
}

func (s *Service) end(cancel context.CancelFunc) {
	s.mu.Lock()
	s.running = false
	s.cancel = nil
	s.mu.Unlock()
	cancel()
}

// CancelTask cancels the running pipeline (if any).
func (s *Service) CancelTask() bool {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

// StartBatch launches the batch pipeline in the background and returns at
// once, so the Wails binding does not block the WebView.
func (s *Service) StartBatch(dir string, recursive bool, filePaths []string) error {
	ctx, cancel, err := s.begin()
	if err != nil {
		return err
	}
	go func() {
		defer s.end(cancel)
		s.runBatch(ctx, dir, recursive, filePaths)
	}()
	return nil
}

// ProcessBySID runs synchronously for the single-movie page.
func (s *Service) ProcessBySID(sid string) (model.ProcessResult, error) {
	ctx, cancel, err := s.begin()
	if err != nil {
		return model.ProcessResult{}, err
	}
	defer s.end(cancel)

	s.emitProgress(model.TaskProgress{Total: 1, Stage: "抓取影片信息"})
	result := s.processSID(ctx, sid, nil)
	s.emitItem(result)
	progress := model.TaskProgress{
		Total: 1, Completed: 1, Stage: "已完成", Percent: 100,
		Current: result.Title,
	}
	if result.Success {
		progress.Success = 1
	} else {
		progress.Failed = 1
	}
	s.emitProgress(progress)
	s.emitDone(model.TaskSummary{
		Total: 1, Processed: 1,
		Success: boolToInt(result.Success),
		Failed:  boolToInt(!result.Success),
	})
	return result, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
