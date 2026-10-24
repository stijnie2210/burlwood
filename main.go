package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	appMenu := menu.NewMenu()
	gitgoodMenu := appMenu.AddSubmenu("gitgood")
	gitgoodMenu.AddText("Preferences…", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
		if app.ctx != nil {
			runtime.EventsEmit(app.ctx, "open-preferences")
		}
	})
	gitgoodMenu.AddSeparator()
	gitgoodMenu.AddText("Quit gitgood", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		if app.ctx != nil {
			runtime.Quit(app.ctx)
		}
	})

	// The frontend handles these too; listing them here makes every shortcut discoverable in the menu bar.
	emit := func(action string) menu.Callback {
		return func(_ *menu.CallbackData) {
			if app.ctx != nil {
				runtime.EventsEmit(app.ctx, "menu", action)
			}
		}
	}

	repoMenu := appMenu.AddSubmenu("Repository")
	repoMenu.AddText("Open Repository…", keys.CmdOrCtrl("o"), emit("open-repo"))
	repoMenu.AddText("Close Tab", keys.CmdOrCtrl("w"), emit("close-tab"))
	repoMenu.AddSeparator()
	repoMenu.AddText("Fetch All", keys.CmdOrCtrl("r"), emit("fetch"))
	repoMenu.AddSeparator()
	repoMenu.AddText("Previous Tab", keys.CmdOrCtrl("["), emit("prev-tab"))
	repoMenu.AddText("Next Tab", keys.CmdOrCtrl("]"), emit("next-tab"))

	appMenu.Append(menu.EditMenu())

	viewMenu := appMenu.AddSubmenu("View")
	viewMenu.AddText("Commits", keys.CmdOrCtrl("1"), emit("view-commits"))
	viewMenu.AddText("Working Tree", keys.CmdOrCtrl("2"), emit("view-staging"))
	viewMenu.AddSeparator()
	viewMenu.AddText("Find Commit…", keys.CmdOrCtrl("f"), emit("find"))

	err := wails.Run(&options.App{
		Title:     "gitgood",
		Width:     1440,
		Height:    900,
		MinWidth:  960,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 27, B: 29, A: 255},
		// The toolbar row doubles as the title bar; the frontend marks it draggable and insets it for the traffic lights.
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
		},
		Menu:          appMenu,
		OnStartup:     app.startup,
		OnBeforeClose: app.beforeClose,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
