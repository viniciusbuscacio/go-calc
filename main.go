package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

const appTitle = "go-Calc"

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     appTitle,
		Width:     380,
		Height:    640,
		MinWidth:  360,
		MinHeight: 640, // fits the tallest view (REST API) without a scrollbar
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Frameless:        true,
		StartHidden:       true, // shown by the frontend once theme/opacity are applied (avoids startup flicker on Linux)
		BackgroundColour: &options.RGBA{R: 20, G: 24, B: 30, A: 0},
		OnStartup:        app.startup,
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			BackdropType:         windows.None,
		},
		Mac: &mac.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  false,
		},
		Linux: &linux.Options{
			WindowIsTranslucent: true,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyOnDemand,
			Icon:                appIcon,
			// ProgramName sets the window's app_id/WMClass. GNOME (esp. on
			// Wayland) matches this to a <ProgramName>.desktop file to show the
			// dock/taskbar icon. Keep it in sync with go-calc.desktop.
			ProgramName: "go-calc",
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
