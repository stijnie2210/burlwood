package main

import (
	"embed"
	"log"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"burlwood/internal/desktop"
	"burlwood/internal/repo"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	go desktop.IntegrateAppImage(appIcon)
	svc := NewApp()

	app := application.New(application.Options{
		Name: "Burlwood",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Linux: application.LinuxOptions{
			ApplicationID: desktop.AppID,
		},
	})

	app.Menu.Set(buildMenu(app))

	opts := application.WebviewWindowOptions{
		Title:            "Burlwood",
		Width:            1440,
		Height:           900,
		MinWidth:         960,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(27, 27, 29),
		// The toolbar row doubles as the title bar; the frontend marks it draggable and insets it for the traffic lights.
		// The webview only turns transparent above the backdrop with -tags private_mac_apis (see build/darwin/Taskfile.yml).
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHiddenInset,
			Backdrop: application.MacBackdropTranslucent,
			// The UI is dark-only, so the vibrancy material must be the dark one regardless of system appearance.
			Appearance: application.NSAppearanceNameDarkAqua,
		},
		UseApplicationMenu: true,
		URL:                "/",
	}
	if runtime.GOOS == "darwin" {
		// Transparent so the sidebar can show the window's vibrancy; every other area paints its own background.
		opts.BackgroundColour = application.NewRGBA(27, 27, 29, 0)
	}
	if prefs, err := repo.LoadWindowPrefs(); err == nil && prefs != nil {
		opts.Width, opts.Height = prefs.Width, prefs.Height
		opts.InitialPosition = application.WindowXY
		opts.X, opts.Y = prefs.X, prefs.Y
	}
	window := app.Window.NewWithOptions(opts)
	// Close() emits WindowClosing too, so this also runs when the app quits with the window open.
	window.RegisterHook(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		w, h := window.Size()
		x, y := window.Position()
		_ = repo.SaveWindowPrefs(w, h, x, y)
	})
	svc.attach(app, window)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

const repoURL = "https://github.com/stijnie2210/burlwood"

func buildMenu(app *application.App) *application.Menu {
	appMenu := app.NewMenu()
	appNameMenu := appMenu.AddSubmenu("Burlwood")
	appNameMenu.AddRole(application.About)
	appNameMenu.AddSeparator()
	appNameMenu.Add("Settings…").SetAccelerator("CmdOrCtrl+,").OnClick(func(_ *application.Context) {
		app.Event.Emit("open-preferences")
	})
	appNameMenu.AddSeparator()
	if runtime.GOOS == "darwin" {
		appNameMenu.AddRole(application.Hide)
		appNameMenu.AddRole(application.HideOthers)
		appNameMenu.AddRole(application.ShowAll)
		appNameMenu.AddSeparator()
	}
	appNameMenu.Add("Quit Burlwood").SetAccelerator("CmdOrCtrl+q").OnClick(func(_ *application.Context) {
		app.Quit()
	})

	// The frontend handles these too; listing them here makes every shortcut discoverable in the menu bar.
	emit := func(action string) func(*application.Context) {
		return func(_ *application.Context) {
			app.Event.Emit("menu", action)
		}
	}

	repoMenu := appMenu.AddSubmenu("Repository")
	repoMenu.Add("Open Repository…").SetAccelerator("CmdOrCtrl+o").OnClick(emit("open-repo"))
	repoMenu.Add("Close Tab").SetAccelerator("CmdOrCtrl+w").OnClick(emit("close-tab"))
	repoMenu.AddSeparator()
	repoMenu.Add("Fetch All").SetAccelerator("CmdOrCtrl+r").OnClick(emit("fetch"))
	repoMenu.Add("Pull").SetAccelerator("CmdOrCtrl+Shift+p").OnClick(emit("pull"))
	repoMenu.Add("Push").SetAccelerator("CmdOrCtrl+p").OnClick(emit("push"))
	repoMenu.AddSeparator()
	repoMenu.Add("New Branch…").SetAccelerator("CmdOrCtrl+Shift+n").OnClick(emit("new-branch"))
	repoMenu.Add("Stash Changes").SetAccelerator("CmdOrCtrl+Shift+s").OnClick(emit("stash"))
	repoMenu.Add("Pop Latest Stash").OnClick(emit("pop-stash"))
	repoMenu.AddSeparator()
	repoMenu.Add("Previous Tab").SetAccelerator("CmdOrCtrl+[").OnClick(emit("prev-tab"))
	repoMenu.Add("Next Tab").SetAccelerator("CmdOrCtrl+]").OnClick(emit("next-tab"))

	appMenu.AddRole(application.EditMenu)

	viewMenu := appMenu.AddSubmenu("View")
	viewMenu.Add("Commits").SetAccelerator("CmdOrCtrl+1").OnClick(emit("view-commits"))
	viewMenu.Add("Working Tree").SetAccelerator("CmdOrCtrl+2").OnClick(emit("view-staging"))
	viewMenu.AddSeparator()
	viewMenu.Add("Find Commit…").SetAccelerator("CmdOrCtrl+f").OnClick(emit("find"))

	// Not the WindowMenu role: off macOS it adds Close with Ctrl+W, which takes the shortcut from
	// Close Tab and quits the app.
	windowMenu := appMenu.AddSubmenu("Window")
	windowMenu.AddRole(application.Minimise)
	windowMenu.AddRole(application.Zoom)
	if runtime.GOOS == "darwin" {
		windowMenu.AddSeparator()
		windowMenu.AddRole(application.Front)
	}

	open := func(url string) func(*application.Context) {
		return func(_ *application.Context) {
			_ = app.Browser.OpenURL(url)
		}
	}
	helpMenu := appMenu.AddSubmenu("Help")
	helpMenu.Add("Burlwood on GitHub").OnClick(open(repoURL))
	helpMenu.Add("Release Notes").OnClick(open(repoURL + "/releases"))
	helpMenu.Add("Report an Issue…").OnClick(open(repoURL + "/issues/new"))

	return appMenu
}
