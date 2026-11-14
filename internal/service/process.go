package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"go-cine-sort/internal/model"
)

func (s *Service) runBatch(ctx context.Context, dir string, recursive bool, filePaths []string) {
	started := time.Now()
	s.log.Info("service", "批量整理开始，目录=%s 递归=%v 指定文件=%d", dir, recursive, len(filePaths))

	var files []model.MovieFileInfo
	if len(filePaths) > 0 {
		files = s.InfoForPaths(filePaths)
	} else {
		var err error
		files, err = s.ScanDirectory(ctx, dir, recursive)
		if err != nil {
			s.emitError(fmt.Sprintf("扫描目录失败：%v", err))
			s.emitDone(model.TaskSummary{ElapsedMs: time.Since(started).Milliseconds()})
			return
		}
	}
	total := len(files)
	s.emitProgress(model.TaskProgress{Total: total, Stage: "扫描完成", Percent: 0})
	if total == 0 {
		s.log.Warn("service", "未发现可处理的视频文件")
		s.emitDone(model.TaskSummary{ElapsedMs: time.Since(started).Milliseconds()})
		return
	}

	success, failed, processed, cancelled := 0, 0, 0, false
	for i, file := range files {
		if ctx.Err() != nil {
			cancelled = true
			s.log.Warn("service", "任务已取消，停止后续文件")
			break
		}
		result := s.processFile(ctx, file)
		processed++
		if result.Success {
			success++
		} else {
			failed++
		}
		s.emitItem(result)
		s.emitProgress(model.TaskProgress{
			Total:     total,
			Completed: processed,
			Success:   success,
			Failed:    failed,
			Current:   file.RawFilename,
			Stage:     fmt.Sprintf("处理中 %d/%d", i+1, total),
			Percent:   float64(processed) / float64(total) * 100,
		})
	}

	stage := "已完成"
	if cancelled {
		stage = "已取消"
	}
	s.emitProgress(model.TaskProgress{
		Total: total, Completed: processed, Success: success, Failed: failed,
		Stage: stage, Percent: float64(processed) / float64(total) * 100,
	})
	s.emitDone(model.TaskSummary{
		Total: total, Processed: processed, Success: success, Failed: failed,
		ElapsedMs: time.Since(started).Milliseconds(), Cancelled: cancelled,
	})
	s.log.Info("service", "批量整理结束：总计 %d / 成功 %d / 失败 %d", total, success, failed)
}

func (s *Service) processFile(ctx context.Context, file model.MovieFileInfo) model.ProcessResult {
	result := model.ProcessResult{
		Filename:  file.RawFilename,
		FilePath:  file.FilePath,
		MovieName: file.MovieName,
		Year:      file.Year,
	}

	s.log.Info("service", "搜索豆瓣：%s (%s)", file.MovieName, file.Year)
	search, err := s.search.Search(ctx, file.MovieName, file.Year)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	details, err := s.details.ParseAndFetch(ctx, search.SID)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	details.SearchTitle = search.Title
	details.SID = search.SID
	if details.Rating == "" {
		details.Rating = search.Rating
	}
	if details.Year == "" {
		details.Year = search.Year
	}
	if details.OriginalTitle == "" {
		details.OriginalTitle = search.OriginalTitle
	}

	outcome, err := s.finish(ctx, details, file.FilePath)
	result.Directory = outcome.dir
	result.VideoPath = outcome.videoPath
	result.PosterCount = outcome.posterCount
	result.SID = details.SID
	result.Title = details.Title
	result.Rating = details.Rating
	for _, warning := range outcome.warnings {
		s.log.Warn("service", "%s", warning)
		result.Error = appendWarning(result.Error, warning)
	}
	if err != nil {
		result.Error = appendWarning(result.Error, err.Error())
		return result
	}
	result.Success = true
	return result
}

func (s *Service) processSID(ctx context.Context, sid string, sourcePath *string) model.ProcessResult {
	result := model.ProcessResult{}
	details, err := s.details.ParseAndFetch(ctx, sid)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	details.SID = sid

	var source string
	if sourcePath != nil {
		source = *sourcePath
	}
	outcome, err := s.finish(ctx, details, source)
	result.Directory = outcome.dir
	result.VideoPath = outcome.videoPath
	result.PosterCount = outcome.posterCount
	result.SID = sid
	result.Title = details.Title
	result.Rating = details.Rating
	result.Filename = details.Title
	for _, warning := range outcome.warnings {
		s.log.Warn("service", "%s", warning)
		result.Error = appendWarning(result.Error, warning)
	}
	if err != nil {
		result.Error = appendWarning(result.Error, err.Error())
		return result
	}
	result.Success = true
	return result
}

type finishOutcome struct {
	dir         string
	videoPath   string
	posterCount int
	warnings    []string
}

// finish creates the directory, writes metadata, downloads posters, archives
// the video and persists everything. Non-critical failures are returned as
// warnings; database and directory failures abort as errors.
func (s *Service) finish(ctx context.Context, details model.MovieDetails, sourcePath string) (finishOutcome, error) {
	out := finishOutcome{warnings: []string{}}

	// prefer_source_dir: the organized directory prefers the source video's own
	// directory; single-SID runs without a source file fall back to MovieFolder
	// or, when it is empty/unavailable, to exeDir/downloads.
	base := ""
	if s.cfg.PreferSourceDir && sourcePath != "" {
		base = filepath.Dir(sourcePath)
	}
	var dir string
	var err error
	if base != "" {
		dir, err = s.org.CreateByDetailAt(base, details)
	} else {
		dir, err = s.org.CreateByDetail(details)
	}
	if err != nil {
		return out, fmt.Errorf("生成整理目录失败：%w", err)
	}
	out.dir = dir

	if s.cfg.CreateRatingFile && details.Rating == "" {
		out.warnings = append(out.warnings, "豆瓣评分为空，未创建豆瓣评分文件")
	}

	if s.cfg.FetchIMDb && details.IMDbID != "" {
		rating, err := s.imdb.Fetch(ctx, details.IMDbID)
		if err != nil {
			out.warnings = append(out.warnings, fmt.Sprintf("IMDb 评分获取失败：%v", err))
		} else if rating != nil {
			if s.cfg.CreateRatingFile {
				if _, err := s.org.WriteIMDbRating(dir, *rating); err != nil {
					out.warnings = append(out.warnings, fmt.Sprintf("写入 IMDb 评分文件失败：%v", err))
				}
			}
		}
	}

	var posterRecords []model.PosterRecord
	if s.cfg.DownloadPosters {
		_, records, err := s.posters.DownloadOfficial(ctx, details.SID, dir, "正式海报")
		if err != nil {
			out.warnings = append(out.warnings, fmt.Sprintf("海报下载失败：%v", err))
		} else {
			posterRecords = records
			out.posterCount = len(records)
		}
	}

	if s.cfg.ArchiveMove && sourcePath != "" {
		videoPath, err := s.org.ArchiveVideo(ctx, sourcePath, dir)
		if err != nil {
			out.warnings = append(out.warnings, fmt.Sprintf("归档视频失败：%v", err))
		} else {
			out.videoPath = videoPath
		}
	}

	movieID, err := s.store.SaveMovie(ctx, details, dir)
	if err != nil {
		return out, fmt.Errorf("写入数据库失败：%w", err)
	}
	if len(posterRecords) > 0 {
		if _, err := s.store.SavePosters(ctx, movieID, posterRecords); err != nil {
			return out, fmt.Errorf("保存海报记录失败：%w", err)
		}
	}
	return out, nil
}

func appendWarning(existing, warning string) string {
	if strings.TrimSpace(existing) == "" {
		return warning
	}
	return existing + "；" + warning
}
