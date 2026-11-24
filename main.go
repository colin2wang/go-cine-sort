package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

// singleInstanceID keys the named mutex that prevents a second app instance.
const singleInstanceID = "go-cine-sort-6f3c2a51"

func main() {
	app := NewApp()

	wailsApp := application.New(application.Options{
		Name: "Cine Sort",
		Services: []application.Service{
			application.NewService(app),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: singleInstanceID,
		},
	})
	app.SetApplication(wailsApp)

	win := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Cine Sort · 电影整理工作台",
		Width:            1440,
		Height:           900,
		MinWidth:         1024,
		MinHeight:        680,
		BackgroundColour: application.NewRGB(11, 15, 26),
		Windows: application.WindowsWindow{
			Theme: application.Dark,
		},
	})
	// 用户关窗时先取消运行中的任务（不阻止关闭）。
	win.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		app.cancelRunningTask()
	})

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
