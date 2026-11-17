package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-cine-sort/internal/model"
	"go-cine-sort/internal/youtube"
)

// PlanYouTubeSort previews the archive operation without moving anything.
func (s *Service) PlanYouTubeSort(sourceDir, targetBase string) (*youtube.PlanResult, error) {
	return youtube.Plan(sourceDir, targetBase)
}

// StartYouTubeSort launches the archive task in the background and returns at
// once, mirroring StartBatch so the Wails binding never blocks the WebView.
func (s *Service) StartYouTubeSort(sourceDir, targetBase string, action youtube.ConflictAction) error {
	ctx, cancel, err := s.begin()
	if err != nil {
		return err
	}
	go func() {
		defer s.end(cancel)
		s.runYouTubeSort(ctx, sourceDir, targetBase, action)
	}()
	return nil
}

func (s *Service) runYouTubeSort(ctx context.Context, sourceDir, targetBase string, action youtube.ConflictAction) {
	started := time.Now()
	s.log.Info("youtube", "归档开始，源=%s 目标=%s 冲突策略=%s", sourceDir, targetBase, action)

	plan, err := youtube.Plan(sourceDir, targetBase)
	if err != nil {
		s.emitError(fmt.Sprintf("扫描目录失败：%v", err))
		s.emitDone(model.TaskSummary{ElapsedMs: time.Since(started).Milliseconds()})
		return
	}
	total := len(plan.Moves)
	s.emitProgress(model.TaskProgress{Total: total, Stage: "扫描完成", Percent: 0})
	if total == 0 {
		s.log.Warn("youtube", "未发现可归档的文件")
		s.emitDone(model.TaskSummary{ElapsedMs: time.Since(started).Milliseconds()})
		return
	}

	var (
		success, skipped, failed int
		cancelled                bool
	)
	_, _, _, _ = youtube.Sort(plan.Moves, plan.Conflicts, action, func(i, n int, mv youtube.MovePlan, moveErr string) {
		select {
		case <-ctx.Done():
			cancelled = true
			return
		default:
		}
		if moveErr == "" {
			success++
			s.log.Info("youtube", "移动：%s -> %s/", mv.Filename, mv.Uploader)
		} else if strings.Contains(moveErr, "已跳过") {
			skipped++
			s.log.Warn("youtube", "%s: %s", mv.Filename, moveErr)
		} else {
			failed++
			s.log.Error("youtube", "%s: %s", mv.Filename, moveErr)
		}
		s.emitProgress(model.TaskProgress{
			Total:     total,
			Completed: i,
			Success:   success,
			Failed:    failed,
			Current:   mv.Filename,
			Stage:     fmt.Sprintf("归档中 %d/%d", i, total),
			Percent:   float64(i) / float64(total) * 100,
		})
	})

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
	s.log.Info("youtube", "归档结束：成功 %d / 跳过 %d / 失败 %d", success, skipped, failed)
}
