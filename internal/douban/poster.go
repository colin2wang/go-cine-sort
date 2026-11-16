package douban

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	// 注册 image.Decode 支持的解码器；豆瓣海报以 webp 为主。
	_ "golang.org/x/image/webp"
	_ "image/gif"

	"go-cine-sort/internal/logging"
	"go-cine-sort/internal/model"
	"go-cine-sort/internal/netclient"
)

var (
	rePosterPattern1 = regexp.MustCompile(`(?s)<li[^>]*>.*?<div class="cover">.*?<a[^>]+href="([^"]+)".*?<img[^>]+(?:src|data-img)="([^"]+)".*?</div>.*?<div class="name">\s*(.*?)\s*<a`)
	rePosterPattern2 = regexp.MustCompile(`(?s)<li[^>]*>.*?<div class="cover">.*?<a[^>]+href="([^"]+)".*?<img[^>]+(?:src|data-img)="([^"]+)".*?</div>.*?<div class="name">\s*(.*?)\s*</div>`)
	rePosterPattern3 = regexp.MustCompile(`(?s)<img[^>]*>`)
	rePosterImgSrc   = regexp.MustCompile(`(?:src|data-img)="([^"]+)"`)
	rePosterImgAlt   = regexp.MustCompile(`alt="([^"]*)"`)
	rePosterReply    = regexp.MustCompile(`\d+回应$`)
	rePosterCDN      = regexp.MustCompile(`(https?://)img(\d+)(\.doubanio\.com/.*)`)
	reInvalidPoster  = regexp.MustCompile(`[<>:"/\\|?*]`)
)

const (
	minQualifiedSize   = 1 << 20 // 1 MB
	minKeepSize        = 10 << 10
	maxPostersKeep     = 4
	maxPosterRetries   = 3
	maxPosterDownloads = 12
	defaultPosterWord  = "正式海报"
)

var chineseRegions = []string{
	"中国大陆", "中国香港", "中国台湾", "香港", "台湾", "澳门",
	"China", "Hong Kong", "Taiwan",
}

func isChineseRegion(title string) bool {
	for _, region := range chineseRegions {
		if strings.Contains(title, region) {
			return true
		}
	}
	return false
}

// OriginalURL converts a Douban thumbnail URL to its large/original variant.
func OriginalURL(thumbnail string) string {
	if thumbnail == "" {
		return thumbnail
	}
	url := thumbnail
	url = strings.Replace(url, "/view/photo/m/", "/view/photo/l/", 1)
	url = strings.Replace(url, "/s_ratio_poster/", "/l/", 1)
	url = strings.Replace(url, "/m_ratio_poster/", "/l/", 1)
	url = strings.Replace(url, "/r_ratio_poster/", "/l/", 1)
	if strings.HasPrefix(url, "//") {
		url = "https:" + url
	} else if strings.HasPrefix(url, "http://") {
		url = "https://" + strings.TrimPrefix(url, "http://")
	}
	return url
}

func cleanPosterTitle(raw string) string {
	title := strings.TrimSpace(reAnyTag.ReplaceAllString(raw, ""))
	return strings.TrimSpace(rePosterReply.ReplaceAllString(title, ""))
}

// ParsePosters tries three increasingly loose HTML shapes, mirroring
// utils/douban_poster.py:DoubanPosterParser.parse.
func ParsePosters(content string) []model.Poster {
	if strings.TrimSpace(content) == "" {
		return []model.Poster{}
	}
	makePoster := func(photoURL, thumbnail, title string) model.Poster {
		return model.Poster{
			PhotoURL:  photoURL,
			Thumbnail: thumbnail,
			Original:  OriginalURL(thumbnail),
			Title:     title,
		}
	}

	posters := make([]model.Poster, 0)
	for _, shape := range []*regexp.Regexp{rePosterPattern1, rePosterPattern2} {
		for _, m := range shape.FindAllStringSubmatch(content, -1) {
			if len(m) < 4 {
				continue
			}
			posters = append(posters, makePoster(m[1], m[2], cleanPosterTitle(m[3])))
		}
		if len(posters) > 0 {
			return posters
		}
	}
	for _, tag := range rePosterPattern3.FindAllString(content, -1) {
		src := rePosterImgSrc.FindStringSubmatch(tag)
		if len(src) < 2 || !strings.Contains(src[1], "doubanio.com") {
			continue
		}
		alt := ""
		if m := rePosterImgAlt.FindStringSubmatch(tag); len(m) > 1 {
			alt = strings.TrimSpace(m[1])
		}
		posters = append(posters, makePoster("", src[1], alt))
	}
	return posters
}

func alternativeCDNs(rawURL string) []string {
	m := rePosterCDN.FindStringSubmatch(rawURL)
	if len(m) < 4 {
		return nil
	}
	out := make([]string, 0, 8)
	for _, n := range []string{"3", "1", "2", "4", "5", "6", "7", "8"} {
		candidate := m[1] + "img" + n + m[3]
		if candidate != rawURL {
			out = append(out, candidate)
		}
	}
	return out
}

func validImageFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	header := make([]byte, 16)
	n, _ := io.ReadFull(f, header)
	header = header[:max(n, 0)]
	if len(header) < 4 {
		return false
	}
	switch {
	case bytes.HasPrefix(header, []byte{0xff, 0xd8, 0xff}):
		return true
	case bytes.Equal(header[:4], []byte{0x89, 'P', 'N', 'G'}):
		return true
	case len(header) >= 6 && (string(header[:6]) == "GIF87a" || string(header[:6]) == "GIF89a"):
		return true
	case len(header) >= 12 && string(header[:4]) == "RIFF" && string(header[8:12]) == "WEBP":
		return true
	case header[0] == 'B' && header[1] == 'M':
		return true
	case len(header) >= 8 && string(header[4:8]) == "ftyp":
		return true
	}
	return false
}

type Posters struct {
	net    *netclient.Client
	log    *logging.Logger
	format string // "none"/"png"/"jpg"; empty keeps originals
}

func NewPosters(net *netclient.Client, log *logging.Logger) *Posters {
	return &Posters{net: net, log: log}
}

// SetFormat sets the target conversion format ("none"/"png"/"jpg").
func (p *Posters) SetFormat(format string) { p.format = format }

func (p *Posters) fetchPage(ctx context.Context, sid, photoType string) (string, error) {
	url := fmt.Sprintf("https://movie.douban.com/subject/%s/photos?type=%s", sid, photoType)
	resp, err := p.net.FetchWithVerification(ctx, url, 2, "豆瓣海报列表页")
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("海报页 HTTP %d", resp.StatusCode)
	}
	return resp.Text(), nil
}

// DownloadOfficial always returns a fixed (paths, metadata, error) triple,
// fixing the Python bug where failure returned a bare list that could not be
// unpacked by the two callers.
func (p *Posters) DownloadOfficial(ctx context.Context, sid, saveDir, keyword string) ([]string, []model.PosterRecord, error) {
	empty := []string{}
	emptyMeta := []model.PosterRecord{}
	if keyword == "" {
		keyword = defaultPosterWord
	}
	content, err := p.fetchPage(ctx, sid, "R")
	if err != nil {
		return empty, emptyMeta, fmt.Errorf("获取海报页失败: %w", err)
	}
	posters := ParsePosters(content)
	if len(posters) == 0 {
		p.log.Warn("poster", "SID %s 未解析到海报", sid)
		return empty, emptyMeta, nil
	}

	// Three-level keyword filter: exact keyword -> 海报 -> everything.
	filter := func(word string) []model.Poster {
		out := make([]model.Poster, 0)
		for _, poster := range posters {
			if word == "" || strings.Contains(poster.Title, word) {
				out = append(out, poster)
			}
		}
		return out
	}
	filtered := filter(keyword)
	if len(filtered) == 0 {
		filtered = filter("海报")
	}
	if len(filtered) == 0 {
		filtered = filter("")
	}
	if len(filtered) > maxPosterDownloads {
		// Pre-rank by HEAD Content-Length so we do not download every poster.
		p.fillSizes(ctx, filtered)
		sort.SliceStable(filtered, func(i, j int) bool { return filtered[i].Size > filtered[j].Size })
		filtered = filtered[:maxPosterDownloads]
	}
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		return empty, emptyMeta, fmt.Errorf("创建海报目录失败: %w", err)
	}

	type downloaded struct {
		path   string
		poster model.Poster
		size   int64
	}
	var all []downloaded
	seen := make(map[string]bool)
	qualifiedCN, qualifiedOther := 0, 0

	for _, poster := range filtered {
		if err := ctx.Err(); err != nil {
			return empty, emptyMeta, err
		}
		if seen[poster.Original] {
			continue
		}
		seen[poster.Original] = true

		dest := filepath.Join(saveDir, posterFileName(poster))
		size, finalPath, err := p.download(ctx, poster.Original, dest)
		if err != nil {
			p.log.Warn("poster", "下载失败 %s: %v", path.Base(poster.Original), err)
			continue
		}
		all = append(all, downloaded{path: finalPath, poster: poster, size: size})
		if size >= minQualifiedSize {
			if isChineseRegion(poster.Title) {
				qualifiedCN++
			} else {
				qualifiedOther++
			}
		}
		if qualifiedCN >= 1 && qualifiedOther >= 3 {
			p.log.Info("poster", "早停：已获得 1 张中文区 + 3 张非中文区高清海报")
			break
		}
	}

	// Final selection: largest Chinese-region poster first, then largest of the
	// rest, up to four posters.
	var cn, other []downloaded
	for _, d := range all {
		if isChineseRegion(d.poster.Title) {
			cn = append(cn, d)
		} else {
			other = append(other, d)
		}
	}
	bySize := func(items []downloaded) {
		sort.SliceStable(items, func(i, j int) bool { return items[i].size > items[j].size })
	}
	bySize(cn)
	bySize(other)

	selected := make([]downloaded, 0, maxPostersKeep)
	if len(cn) > 0 {
		selected = append(selected, cn[0])
		cn = cn[1:]
	}
	remaining := append(cn, other...)
	bySize(remaining)
	for _, d := range remaining {
		if len(selected) >= maxPostersKeep {
			break
		}
		selected = append(selected, d)
	}

	selectedPaths := make(map[string]bool, len(selected))
	for _, d := range selected {
		selectedPaths[d.path] = true
	}
	for _, d := range all {
		if d.size < minKeepSize || !selectedPaths[d.path] {
			_ = os.Remove(d.path)
		}
	}

	paths := make([]string, 0, len(selected))
	metadata := make([]model.PosterRecord, 0, len(selected))
	for _, d := range selected {
		if _, err := os.Stat(d.path); err != nil {
			continue
		}
		base, kw, region := posterNameParts(d.poster)
		paths = append(paths, d.path)
		metadata = append(metadata, model.PosterRecord{
			PosterID:     base,
			Title:        d.poster.Title,
			Keyword:      kw,
			Region:       region,
			ThumbnailURL: d.poster.Thumbnail,
			OriginalURL:  d.poster.Original,
			FileSize:     d.size,
			IsDownloaded: true,
			LocalPath:    d.path,
		})
	}
	p.log.Info("poster", "SID %s 最终保留 %d 张官方海报", sid, len(paths))
	return paths, metadata, nil
}

func (p *Posters) fillSizes(ctx context.Context, posters []model.Poster) {
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for i := range posters {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			posters[idx].Size = p.imageSize(ctx, posters[idx].Original)
		}(i)
	}
	wg.Wait()
}

func (p *Posters) imageSize(ctx context.Context, rawURL string) int64 {
	header, err := p.net.Head(ctx, rawURL, netclient.Options{
		Timeout:       10 * time.Second,
		Referer:       netclient.MovieReferer,
		SkipRateLimit: true,
	})
	if err != nil {
		return 0
	}
	value := strings.TrimSpace(header.Get("Content-Length"))
	if value == "" {
		return 0
	}
	size, err := strconv.ParseInt(value, 10, 64)
	if err != nil || size < 0 {
		return 0
	}
	return size
}

func (p *Posters) download(ctx context.Context, rawURL, dest string) (int64, string, error) {
	urls := append([]string{rawURL}, alternativeCDNs(rawURL)...)
	opts := netclient.Options{
		Timeout:       30 * time.Second,
		Referer:       netclient.MovieReferer,
		SkipRateLimit: true,
	}

	for _, url := range urls {
		for attempt := 1; attempt <= maxPosterRetries; attempt++ {
			if err := ctx.Err(); err != nil {
				return 0, "", err
			}
			resp, err := p.net.Open(ctx, http.MethodGet, url, nil, opts)
			if err != nil {
				p.log.Debug("poster", "下载失败(%d/%d) %s: %v", attempt, maxPosterRetries, url, err)
				continue
			}
			contentType := strings.ToLower(resp.Header.Get("Content-Type"))
			if resp.StatusCode >= 400 || strings.Contains(contentType, "text/html") || strings.Contains(contentType, "javascript") {
				_ = resp.Body.Close()
				break // try the next CDN
			}
			err = p.streamToFile(resp.Body, dest)
			_ = resp.Body.Close()
			if err != nil {
				_ = os.Remove(dest)
				continue
			}
			if !validImageFile(dest) {
				_ = os.Remove(dest)
				continue
			}
			finalPath, err := p.convertPoster(dest)
			if err != nil {
				p.log.Warn("poster", "格式转换失败 %s: %v", path.Base(dest), err)
				_ = os.Remove(dest)
				continue
			}
			stat, err := os.Stat(finalPath)
			if err != nil {
				return 0, "", err
			}
			return stat.Size(), finalPath, nil
		}
	}
	_ = os.Remove(dest)
	return 0, "", fmt.Errorf("所有 CDN 均下载失败: %s", path.Base(rawURL))
}

// convertPoster re-encodes the downloaded image into the configured format and
// returns the final path (possibly with a new extension). A format of
// "none"/"" keeps the original file untouched.
func (p *Posters) convertPoster(src string) (string, error) {
	switch p.format {
	case "png", "jpg":
	default:
		return src, nil
	}

	f, err := os.Open(src)
	if err != nil {
		return "", err
	}
	img, _, err := image.Decode(f)
	_ = f.Close()
	if err != nil {
		return "", fmt.Errorf("解码图片失败: %w", err)
	}

	targetExt := "." + p.format
	dst := strings.TrimSuffix(src, filepath.Ext(src)) + targetExt
	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	switch p.format {
	case "jpg":
		err = jpeg.Encode(out, img, &jpeg.Options{Quality: 90})
	default:
		err = png.Encode(out, img)
	}
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(dst)
		return "", fmt.Errorf("编码 %s 失败: %w", p.format, err)
	}
	if dst != src {
		_ = os.Remove(src)
		p.log.Debug("poster", "已转换为 %s: %s", p.format, filepath.Base(dst))
	}
	return dst, nil
}

func (p *Posters) streamToFile(body io.Reader, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	buf := make([]byte, 8192)
	for {
		n, readErr := body.Read(buf)
		if n > 0 {
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
				_ = f.Close()
				return writeErr
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = f.Close()
			return readErr
		}
	}
	return f.Close()
}

func posterNameParts(poster model.Poster) (base, keyword, region string) {
	fileName := path.Base(poster.Original)
	ext := filepath.Ext(fileName)
	base = strings.TrimSuffix(fileName, ext)
	if ext == "" {
		ext = ".jpg"
	}
	parts := strings.Fields(poster.Title)
	keyword = defaultPosterWord
	if len(parts) > 0 {
		keyword = parts[0]
	}
	if len(parts) > 1 {
		region = parts[1]
	}
	return base, keyword, region
}

func posterFileName(poster model.Poster) string {
	fileName := path.Base(poster.Original)
	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(fileName, ext)
	if ext == "" {
		ext = ".jpg"
	}
	parts := strings.Fields(poster.Title)
	keyword := defaultPosterWord
	if len(parts) > 0 {
		keyword = parts[0]
	}
	name := base + "_" + keyword
	if len(parts) > 1 {
		name += "_" + parts[1]
	}
	name += ext
	return reInvalidPoster.ReplaceAllString(name, "_")
}
