// Command icongen renders the single SVG brand asset into every PNG size
// required by Wails and the Windows executable.
//
// Usage: go run ./tools/icongen [-root <project root>]
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

var icoSizes = []int{16, 24, 32, 48, 64, 128, 256}

const appIconSize = 512

func main() {
	rootFlag := flag.String("root", "", "项目根目录（默认自动从当前目录向上查找 wails.json）")
	flag.Parse()

	root, err := resolveRoot(*rootFlag)
	if err != nil {
		fail(err)
	}
	svgPath := filepath.Join(root, "frontend", "src", "assets", "logo.svg")
	svgData, err := os.ReadFile(svgPath)
	if err != nil {
		fail(fmt.Errorf("读取 SVG 源图标 %s 失败: %w", svgPath, err))
	}

	icon, err := oksvg.ReadIconStream(bytes.NewReader(svgData), oksvg.WarnErrorMode)
	if err != nil {
		fail(fmt.Errorf("解析 SVG 失败: %w", err))
	}
	if icon.ViewBox.W <= 0 || icon.ViewBox.H <= 0 {
		fail(fmt.Errorf("SVG 缺少有效的 viewBox"))
	}

	windowsDir := filepath.Join(root, "build", "windows")
	if err := os.MkdirAll(windowsDir, 0o755); err != nil {
		fail(err)
	}

	frames := make([]ICOImage, 0, len(icoSizes))
	for _, size := range icoSizes {
		data, err := render(icon, size)
		if err != nil {
			fail(err)
		}
		frames = append(frames, ICOImage{Size: size, PNG: data})
		fmt.Printf("  · %3d x %-3d  %6.1f KB\n", size, size, float64(len(data))/1024)
	}
	icoBytes, err := EncodeICO(frames)
	if err != nil {
		fail(err)
	}
	icoPath := filepath.Join(windowsDir, "icon.ico")
	if err := os.WriteFile(icoPath, icoBytes, 0o644); err != nil {
		fail(err)
	}

	appIconPNG, err := render(icon, appIconSize)
	if err != nil {
		fail(err)
	}
	appIconPath := filepath.Join(root, "build", "appicon.png")
	if err := os.WriteFile(appIconPath, appIconPNG, 0o644); err != nil {
		fail(err)
	}

	// Keep the favicon copy byte-identical with the single SVG source.
	publicPath := filepath.Join(root, "frontend", "public", "logo.svg")
	if err := os.MkdirAll(filepath.Dir(publicPath), 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(publicPath, svgData, 0o644); err != nil {
		fail(err)
	}

	if err := verifyVisible(appIconPNG, appIconSize); err != nil {
		fail(fmt.Errorf("图标渲染校验失败: %w", err))
	}

	fmt.Println("---")
	fmt.Printf("SVG 源       : %s\n", svgPath)
	fmt.Printf("应用图标     : %s (%d x %d)\n", appIconPath, appIconSize, appIconSize)
	fmt.Printf("可执行程序   : %s (%d 个尺寸, %.1f KB)\n", icoPath, len(frames), float64(len(icoBytes))/1024)
	fmt.Printf("页面 favicon : %s\n", publicPath)
}

func render(icon *oksvg.SvgIcon, size int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
	raster := rasterx.NewDasher(size, size, scanner)
	icon.SetTarget(0, 0, float64(size), float64(size))
	icon.Draw(raster, 1.0)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// verifyVisible guards against a silently empty render (e.g. an unsupported
// SVG element): the center of the reel must be a non-transparent gold pixel.
func verifyVisible(pngData []byte, size int) error {
	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return err
	}
	_, _, _, alpha := img.At(size/2, size/2).RGBA()
	if alpha == 0 {
		return fmt.Errorf("图标中心像素完全透明")
	}
	return nil
}

func resolveRoot(explicit string) (string, error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", err
		}
		return abs, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "wails.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("未找到 wails.json，请使用 -root 指定项目根目录")
		}
		dir = parent
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "icongen:", err)
	os.Exit(1)
}
