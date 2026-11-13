package organizer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ArchiveVideo moves the source video into the freshly created directory.
// It returns an empty destination when the conflict strategy is "skip".
// Same-volume moves use os.Rename; cross-volume moves fall back to a verified
// copy followed by removing the source. The generated directory is never
// removed on failure.
func (o *Organizer) ArchiveVideo(ctx context.Context, source, targetDir string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	info, err := os.Stat(source)
	if err != nil {
		return "", fmt.Errorf("源视频不可访问: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("源路径是目录: %s", source)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("目标目录不可用: %w", err)
	}

	name := filepath.Base(source)
	dest := filepath.Join(targetDir, name)
	if _, err := os.Stat(dest); err == nil {
		switch o.cfg.ConflictStrategy {
		case "skip":
			o.log.Info("archive", "目标已存在，按策略跳过: %s", dest)
			return "", nil
		case "overwrite":
			if err := os.Remove(dest); err != nil {
				return "", fmt.Errorf("覆盖目标失败: %w", err)
			}
		default: // rename
			stem := strings.TrimSuffix(name, filepath.Ext(name))
			ext := filepath.Ext(name)
			for i := 1; ; i++ {
				candidate := filepath.Join(targetDir, fmt.Sprintf("%s_%d%s", stem, i, ext))
				if _, err := os.Stat(candidate); os.IsNotExist(err) {
					dest = candidate
					break
				}
			}
		}
	}

	if err := os.Rename(source, dest); err == nil {
		o.log.Info("archive", "已归档 %s -> %s", filepath.Base(source), dest)
		return dest, nil
	} else {
		o.log.Debug("archive", "同盘移动失败（%v），改用跨盘复制: %s", err, filepath.Base(source))
	}

	if err := copyVerified(ctx, source, dest, info.Size()); err != nil {
		_ = os.Remove(dest)
		return "", fmt.Errorf("跨盘归档失败: %w", err)
	}
	if err := os.Remove(source); err != nil {
		o.log.Warn("archive", "复制成功但删除源文件失败（保留源文件）: %v", err)
	} else {
		o.log.Info("archive", "已跨盘归档 %s -> %s", filepath.Base(source), dest)
	}
	return dest, nil
}

func copyVerified(ctx context.Context, src, dst string, wantSize int64) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	buf := make([]byte, 128*1024)
	for {
		if err := ctx.Err(); err != nil {
			_ = out.Close()
			return err
		}
		n, readErr := in.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				_ = out.Close()
				return writeErr
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = out.Close()
			return readErr
		}
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	stat, err := os.Stat(dst)
	if err != nil {
		return err
	}
	if stat.Size() != wantSize {
		return fmt.Errorf("复制后大小不一致: %d != %d", stat.Size(), wantSize)
	}
	return nil
}
