package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	appbridge "go-cine-sort/internal/app"
	"go-cine-sort/internal/cache"
	"go-cine-sort/internal/config"
	"go-cine-sort/internal/folder"
	"go-cine-sort/internal/logging"
	"go-cine-sort/internal/model"
	"go-cine-sort/internal/service"
	"go-cine-sort/internal/youtube"
)

// App is the object bound to the Wails frontend. It owns the application
// lifetime and delegates all business logic to internal/service.
type App struct {
	ctx     context.Context
	cfg     *config.AppConfig
	log     *logging.Logger
	svc     *service.Service
	initErr error
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	cfg, err := config.Load()
	if err != nil {
		// A broken user config must not prevent the window from opening.
		fmt.Fprintf(os.Stderr, "配置加载失败，使用内置默认值: %v\n", err)
		def, defErr := config.Defaults()
		if defErr != nil {
			a.initErr = defErr
			return
		}
		def.ConfigPath, _ = config.Path()
		def.DataDir, _ = config.Dir()
		cfg = &def
	}
	a.cfg = cfg

	log, err := logging.New(filepath.Join(cfg.DataDir, "logs"), appbridge.LogSink(ctx))
	if err != nil {
		fmt.Fprintf(os.Stderr, "日志初始化失败: %v\n", err)
		log, _ = logging.New("", appbridge.LogSink(ctx))
	}
	a.log = log

	svc, err := service.New(a.cfg, log, appbridge.Events(ctx))
	if err != nil {
		a.initErr = fmt.Errorf("服务初始化失败: %w", err)
		log.Error("app", "%v", a.initErr)
		return
	}
	svc.SetContext(ctx)
	a.svc = svc
	log.Info("app", "Cine Sort 已启动，配置文件 %s", cfg.ConfigPath)
}

func (a *App) domReady(ctx context.Context) {
	if a.log != nil {
		a.log.Debug("app", "前端 DOM 就绪")
	}
}

func (a *App) beforeClose(ctx context.Context) bool {
	if a.svc != nil && a.svc.IsRunning() {
		_ = a.svc.CancelTask()
	}
	return false
}

func (a *App) shutdown(ctx context.Context) {
	if a.svc != nil {
		_ = a.svc.Close()
	}
	if a.log != nil {
		_ = a.log.Close()
	}
}

func (a *App) ready() error {
	if a.initErr != nil {
		return a.initErr
	}
	if a.svc == nil {
		return errors.New("后端服务尚未就绪")
	}
	return nil
}

// Ping is a connectivity self-check used by the frontend.
func (a *App) Ping() string {
	return "pong"
}

// SetLogLevel adjusts the logger's minimum level at runtime (DEBUG / INFO /
// WARN / ERROR). The choice is session-only; restarts return to INFO.
func (a *App) SetLogLevel(level string) {
	if a.log == nil {
		return
	}
	l := logging.ParseLevel(level)
	a.log.SetLevel(l)
	a.log.Info("app", "日志级别已切换为 %s", l)
}

func (a *App) GetConfig() (config.AppConfig, error) {
	if a.cfg == nil {
		return config.Defaults()
	}
	return *a.cfg, nil
}

func (a *App) GetDefaults() (config.AppConfig, error) {
	return config.Defaults()
}

// SaveConfig validates, persists and hot-reloads the configuration.
func (a *App) SaveConfig(cfg config.AppConfig) (config.AppConfig, error) {
	if err := a.ready(); err != nil {
		return cfg, err
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	// Never trust paths supplied by the frontend.
	dir, err := config.Dir()
	if err != nil {
		return cfg, err
	}
	cfg.DataDir = dir
	cfg.ConfigPath, _ = config.ConfigsDir()
	if err := config.Save(&cfg); err != nil {
		return cfg, err
	}
	if err := a.svc.ReloadConfig(&cfg); err != nil {
		return cfg, err
	}
	a.cfg = &cfg
	a.log.Info("app", "配置已保存并生效")
	return cfg, nil
}

// SelectDirectory opens a native folder picker.
func (a *App) SelectDirectory(current string) (string, error) {
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:            "选择电影目录",
		DefaultDirectory: current,
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(dir) == "" {
		return current, nil
	}
	return dir, nil
}

func (a *App) ScanDirectory(dir string, recursive bool) ([]model.MovieFileInfo, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.ScanDirectory(a.ctx, dir, recursive)
}

func (a *App) StartBatch(dir string, recursive bool, filePaths []string) error {
	if err := a.ready(); err != nil {
		return err
	}
	if strings.TrimSpace(dir) == "" {
		dir = a.cfg.MovieFolder
	}
	return a.svc.StartBatch(dir, recursive, filePaths)
}

func (a *App) ProcessBySID(sid string) (model.ProcessResult, error) {
	if err := a.ready(); err != nil {
		return model.ProcessResult{}, err
	}
	sid = strings.TrimSpace(sid)
	if !onlyDigits(sid) {
		return model.ProcessResult{}, fmt.Errorf("SID 必须是数字，收到 %q", sid)
	}
	return a.svc.ProcessBySID(sid)
}

func onlyDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (a *App) CancelTask() bool {
	if a.svc == nil {
		return false
	}
	return a.svc.CancelTask()
}

// PlanYouTubeSort previews the YouTube archive operation without moving files.
func (a *App) PlanYouTubeSort(sourceDir, targetBase string) (*youtube.PlanResult, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.PlanYouTubeSort(sourceDir, targetBase)
}

// StartYouTubeSort launches the YouTube archive task in the background.
func (a *App) StartYouTubeSort(sourceDir, targetBase, conflictAction string) error {
	if err := a.ready(); err != nil {
		return err
	}
	if conflictAction != string(youtube.ConflictOverwrite) {
		conflictAction = string(youtube.ConflictSkip)
	}
	return a.svc.StartYouTubeSort(sourceDir, targetBase, youtube.ConflictAction(conflictAction))
}

// ScanMediaFiles lists video files with tags and rename plans.
func (a *App) ScanMediaFiles(dir string, recursive bool) ([]service.MediaFileMeta, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.ScanMediaFiles(dir, recursive)
}

// UpdateMediaTags writes title/comment tags to one video file.
func (a *App) UpdateMediaTags(path string, title, comment string) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.svc.UpdateMediaTags(path, title, comment)
}

// StartMediaRename launches the batch rename task in the background.
func (a *App) StartMediaRename(dir string, recursive bool) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.svc.StartMediaRename(dir, recursive)
}

func (a *App) IsRunning() bool {
	return a.svc != nil && a.svc.IsRunning()
}

func (a *App) ListHistory(keyword string) ([]model.MovieRecord, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.ListHistory(a.ctx, keyword)
}

func (a *App) GetHistoryDetail(sid string) (model.MovieDetail, error) {
	if err := a.ready(); err != nil {
		return model.MovieDetail{}, err
	}
	return a.svc.GetHistoryDetail(a.ctx, sid)
}

func (a *App) GetHistoryStats() (model.HistoryStats, error) {
	if err := a.ready(); err != nil {
		return model.HistoryStats{}, err
	}
	return a.svc.Stats(a.ctx)
}

// ListCache returns summaries of every cached HTTP response, newest first.
func (a *App) ListCache() ([]cache.Summary, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.CacheList(), nil
}

// GetCacheEntry returns one cached response (headers + body) by key.
func (a *App) GetCacheEntry(key string) (*cache.Entry, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	entry, ok := a.svc.CacheEntry(key)
	if !ok {
		return nil, errors.New("缓存条目不存在或已过期")
	}
	return entry, nil
}

// GetCacheStats reports cached entry count and total body bytes.
func (a *App) GetCacheStats() (cache.Stats, error) {
	if err := a.ready(); err != nil {
		return cache.Stats{}, err
	}
	return a.svc.CacheStats(), nil
}

// ClearCache drops every cached response.
func (a *App) ClearCache() error {
	if err := a.ready(); err != nil {
		return err
	}
	a.svc.CacheClear()
	a.log.Info("app", "HTTP 响应缓存已清空")
	return nil
}

// ListDirectory lists the files produced inside an organized directory.
func (a *App) ListDirectory(dir string) ([]string, error) {
	if strings.TrimSpace(dir) == "" {
		return []string{}, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

// OpenFolder reveals a directory in the native file manager, reusing an
// existing Explorer window when the directory is already open (Windows).
func (a *App) OpenFolder(path string) error {
	return folder.Open(path)
}
