package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go-cine-sort/internal/media"
	"go-cine-sort/internal/model"
)

// MediaFileMeta is one scanned video file with its tags and rename plan.
type MediaFileMeta struct {
	Path      string  `json:"path"`
	Dir       string  `json:"dir"`
	Filename  string  `json:"filename"`
	SizeMB    float64 `json:"sizeMB"`
	Title     string  `json:"title"`
	Comment   string  `json:"comment"`
	NewName   string  `json:"newName"`
	Renamable bool    `json:"renamable"`
	Taggable  bool    `json:"taggable"`
}

// ScanMediaFiles walks dir (optionally recursive), listing video files with
// their current tags and the convention-based rename plan.
func (s *Service) ScanMediaFiles(dir string, recursive bool) ([]MediaFileMeta, error) {
	files, err := collectVideoFiles(dir, recursive)
	if err != nil {
		return nil, err
	}
	result := make([]MediaFileMeta, 0, len(files))
	for _, path := range files {
		meta := buildMediaFileMeta(path)
		if tags, e := media.ReadTags(path); e == nil && tags != nil {
			meta.Title = tags.Title
			meta.Comment = tags.Comment
			meta.Taggable = true
		}
		result = append(result, meta)
	}
	return result, nil
}

// UpdateMediaTags writes title/comment to one video file (MP4 family only).
func (s *Service) UpdateMediaTags(path string, title, comment string) error {
	if title == "" && comment == "" {
		return fmt.Errorf("标题与备注不能同时为空")
	}
	return media.UpdateTags(path, title, comment)
}

// StartMediaRename launches the batch rename task in the background.
func (s *Service) StartMediaRename(dir string, recursive bool) error {
	ctx, cancel, err := s.begin()
	if err != nil {
		return err
	}
	go func() {
		defer s.end(cancel)
		s.runMediaRename(ctx, dir, recursive)
	}()
	return nil
}

func (s *Service) runMediaRename(ctx context.Context, dir string, recursive bool) {
	started := time.Now()
	s.log.Info("media", "批量重命名开始，目录=%s 递归=%v", dir, recursive)

	files, err := collectVideoFiles(dir, recursive)
	if err != nil {
		s.emitError(fmt.Sprintf("扫描目录失败：%v", err))
		s.emitDone(model.TaskSummary{ElapsedMs: time.Since(started).Milliseconds()})
		return
	}
	total := len(files)
	s.emitProgress(model.TaskProgress{Total: total, Stage: "扫描完成", Percent: 0})

	var (
		success, skipped, failed int
		cancelled                bool
	)
	for i, path := range files {
		select {
		case <-ctx.Done():
			cancelled = true
		default:
		}
		if cancelled {
			break
		}

		completed := i + 1
		dirOf, name := filepath.Split(path)
		newName, renamed, rerr := media.RenameFile(dirOf, name)
		progress := model.TaskProgress{
			Total:     total,
			Completed: completed,
			Current:   name,
			Stage:     fmt.Sprintf("重命名 %d/%d", completed, total),
			Percent:   float64(completed) / float64(total) * 100,
		}
		switch {
		case rerr != nil:
			failed++
			progress.Failed = failed
			s.log.Error("media", "%s: %v", name, rerr)
		case renamed:
			success++
			progress.Success = success
			s.log.Info("media", "重命名：%s -> %s", name, newName)
		default:
			skipped++
			progress.Success = success
			s.log.Warn("media", "跳过（格式不匹配）：%s", name)
		}
		s.emitProgress(progress)
	}

	stage := "已完成"
	if cancelled {
		stage = "已取消"
	}
	s.emitProgress(model.TaskProgress{
		Total: total, Completed: success + skipped + failed,
		Success: success, Failed: failed,
		Stage: stage, Percent: 100,
	})
	s.emitDone(model.TaskSummary{
		Total: total, Processed: success + skipped + failed,
		Success: success, Failed: failed,
		ElapsedMs: time.Since(started).Milliseconds(), Cancelled: cancelled,
	})
	s.log.Info("media", "批量重命名结束：成功 %d / 跳过 %d / 失败 %d", success, skipped, failed)
}

func buildMediaFileMeta(path string) MediaFileMeta {
	name := filepath.Base(path)
	meta := MediaFileMeta{
		Path:     path,
		Dir:      filepath.Dir(path),
		Filename: name,
	}
	if st, err := os.Stat(path); err == nil {
		meta.SizeMB = float64(st.Size()) / (1024 * 1024)
	}
	if newBase := media.ParseConventionName(name); newBase != "" {
		ext := filepath.Ext(name)
		meta.NewName = newBase + ext
		meta.Renamable = meta.NewName != name
		meta.Taggable = media.IsMP4(path)
	}
	return meta
}

func collectVideoFiles(dir string, recursive bool) ([]string, error) {
	var files []string
	walk := func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if !recursive && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		switch ext {
		case ".mp4", ".mkv", ".avi", ".mov", ".flv", ".wmv", ".mpeg", ".mpg", ".m4v", ".webm", ".ts":
			files = append(files, path)
		}
		return nil
	}
	if err := filepath.WalkDir(dir, walk); err != nil {
		return nil, err
	}
	return files, nil
}
