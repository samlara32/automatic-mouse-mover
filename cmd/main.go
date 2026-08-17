package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Resousse/automatic-mouse-mover/pkg/mousemover"
	"github.com/getlantern/systray"
	"github.com/go-vgo/robotgo"
	"github.com/kirsle/configdir"
	log "github.com/sirupsen/logrus"
)

type AppSettings struct {
	Icon         string `json:"icon"`
	Color        string `json:"color"`
	Interval     int    `json:"interval"`      // in seconds: 30, 60, 120, 300, 600
	MoveMode     string `json:"move_mode"`     // "standard", "micro", "jiggle"
	TimerMinutes int    `json:"timer_minutes"` // 0 = continuous, 30, 60, 120, 240
}

var configPath = configdir.LocalConfig("amm")
var configFile = filepath.Join(configPath, "settings.json")

const alphaInactive = 0.6

var (
	colorBlue  = color.RGBA{30, 144, 255, 255}
	colorRed   = color.RGBA{255, 0, 0, 255}
	colorWhite = color.RGBA{255, 255, 255, 255}
)

var (
	timerCancelCh chan struct{}
	timerMutex    sync.Mutex
)

func main() {
	systray.Run(onReady, onExit)
}

func loadIconFile(iconName string) []byte {
	if iconName != "mouse" && iconName != "cloud" && iconName != "geometric" && iconName != "man" {
		iconName = "mouse"
	}
	iconPaths := []string{}
	if base := os.Getenv("AMM_ICON_DIR"); base != "" {
		iconPaths = append(iconPaths, filepath.Join(base, "assets", "icon", iconName+".png"))
	}
	ex, _ := os.Executable()
	exPath := filepath.Dir(ex)
	iconPaths = append(iconPaths,
		exPath+"/../Resources/assets/icon/"+iconName+".png",
		exPath+"/../assets/icon/"+iconName+".png",
		"./assets/icon/"+iconName+".png",
	)

	for _, iconPath := range iconPaths {
		if _, err := os.Stat(iconPath); err == nil {
			b, err := os.ReadFile(iconPath)
			if err == nil && len(b) > 0 {
				return b
			}
		}
	}
	panic("Failed to load icon: " + iconName + ".png")
}

func getMenuIcon(iconName string) []byte {
	return loadIconFile(iconName)
}

func getTrayIcon(iconName string, active bool, col string) []byte {
	b := loadIconFile(iconName)

	if active && col != "" {
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			log.Fatalln(err)
		}
		var dimg *image.RGBA = image.NewRGBA(img.Bounds())
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				r, g, b, a := img.At(x, y).RGBA()
				if a != 0 {
					switch col {
					case "white":
						dimg.Set(x, y, colorWhite)
					case "red":
						dimg.Set(x, y, colorRed)
					default:
						dimg.Set(x, y, colorBlue)
					}
				} else {
					dimg.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
				}
			}
		}
		var c bytes.Buffer
		png.Encode(&c, dimg)
		return c.Bytes()
	}

	if !active {
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			log.Fatalln(err)
		}
		var dimg *image.RGBA = image.NewRGBA(img.Bounds())
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				r, g, b, a := img.At(x, y).RGBA()
				newAlpha := uint8(float64(a>>8) * alphaInactive)
				dimg.Set(x, y, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), newAlpha})
			}
		}
		var c bytes.Buffer
		png.Encode(&c, dimg)
		return c.Bytes()
	}

	return b
}

func saveSettings(settings *AppSettings) {
	fh, err := os.Create(configFile)
	if err != nil {
		log.Errorf("Failed to save settings: %v", err)
		return
	}
	defer fh.Close()
	encoder := json.NewEncoder(fh)
	encoder.Encode(settings)
}

func setIcon(iconName string, color string, configFile string, settings *AppSettings, active ...bool) {
	isActive := len(active) != 0 && active[0]
	var iconData []byte
	if isActive && color != "" {
		iconData = getTrayIcon(iconName, true, color)
		systray.SetIcon(iconData)
	} else {
		iconData = getTrayIcon(iconName, isActive, "")
		systray.SetTemplateIcon(iconData, iconData)
	}
	if configFile != "" {
		settings.Icon = iconName
		settings.Color = color
		saveSettings(settings)
	}
}

func updateChecks[T comparable](items map[T]*systray.MenuItem, selected T) {
	for k, item := range items {
		if k == selected {
			item.Check()
		} else {
			item.Uncheck()
		}
	}
}

func startAppWithTimer(mouseMover *mousemover.MouseMover, settings *AppSettings, ammStart, ammStop *systray.MenuItem) {
	timerMutex.Lock()
	if timerCancelCh != nil {
		close(timerCancelCh)
		timerCancelCh = nil
	}
	timerMutex.Unlock()

	mode := mousemover.MovementMode(settings.MoveMode)
	if mode == "" {
		mode = mousemover.ModeStandard
	}
	interval := settings.Interval
	if interval <= 0 {
		interval = 60
	}

	mouseMover.StartWithConfig(mousemover.Config{
		IntervalSeconds: interval,
		MovementMode:    mode,
	})
	ammStart.Disable()
	ammStop.Enable()
	setIcon(settings.Icon, settings.Color, "", settings, true)

	if settings.TimerMinutes > 0 {
		cancelCh := make(chan struct{})
		timerMutex.Lock()
		timerCancelCh = cancelCh
		timerMutex.Unlock()

		duration := time.Duration(settings.TimerMinutes) * time.Minute
		go func(cancel <-chan struct{}, d time.Duration) {
			select {
			case <-time.After(d):
				log.Infof("Auto-stop timer elapsed (%v), stopping app", d)
				mouseMover.Quit()
				ammStart.Enable()
				ammStop.Disable()
				setIcon(settings.Icon, settings.Color, "", settings, false)
				go robotgo.Alert("Automatic Mouse Mover", fmt.Sprintf("Auto-stop timer finished (%d minutes). App is now stopped.", settings.TimerMinutes))
			case <-cancel:
				return
			}
		}(cancelCh, duration)
	}
}

func stopApp(mouseMover *mousemover.MouseMover, settings *AppSettings, ammStart, ammStop *systray.MenuItem) {
	timerMutex.Lock()
	if timerCancelCh != nil {
		close(timerCancelCh)
		timerCancelCh = nil
	}
	timerMutex.Unlock()

	mouseMover.Quit()
	ammStart.Enable()
	ammStop.Disable()
	setIcon(settings.Icon, settings.Color, "", settings, false)
}

func onReady() {
	go func() {
		_ = configdir.MakePath(configPath)
		settings := AppSettings{
			Icon:         "mouse",
			Color:        "blue",
			Interval:     60,
			MoveMode:     string(mousemover.ModeStandard),
			TimerMinutes: 0,
		}

		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			saveSettings(&settings)
		} else {
			fh, err := os.Open(configFile)
			if err == nil {
				_ = json.NewDecoder(fh).Decode(&settings)
				fh.Close()
			}
		}

		if settings.Interval <= 0 {
			settings.Interval = 60
		}
		if settings.MoveMode == "" {
			settings.MoveMode = string(mousemover.ModeStandard)
		}

		about := systray.AddMenuItem("About Automatic Mouse Mover", "")
		systray.AddSeparator()
		ammStart := systray.AddMenuItem("Start", "")
		ammStop := systray.AddMenuItem("Stop", "")
		systray.AddSeparator()

		// Interval Submenu adhering to macOS HIG
		intervalMenu := systray.AddMenuItem("Interval", "")
		intervalItems := map[int]*systray.MenuItem{
			30:  intervalMenu.AddSubMenuItemCheckbox("30 Seconds", "", settings.Interval == 30),
			60:  intervalMenu.AddSubMenuItemCheckbox("1 Minute (Default)", "", settings.Interval == 60),
			120: intervalMenu.AddSubMenuItemCheckbox("2 Minutes", "", settings.Interval == 120),
			300: intervalMenu.AddSubMenuItemCheckbox("5 Minutes", "", settings.Interval == 300),
			600: intervalMenu.AddSubMenuItemCheckbox("10 Minutes", "", settings.Interval == 600),
		}

		// Movement Mode Submenu adhering to macOS HIG
		modeMenu := systray.AddMenuItem("Movement Mode", "")
		modeItems := map[string]*systray.MenuItem{
			string(mousemover.ModeStandard): modeMenu.AddSubMenuItemCheckbox("Standard (10 px)", "", settings.MoveMode == string(mousemover.ModeStandard)),
			string(mousemover.ModeMicro):    modeMenu.AddSubMenuItemCheckbox("Micro-Nudge (1 px)", "", settings.MoveMode == string(mousemover.ModeMicro)),
			string(mousemover.ModeJiggle):   modeMenu.AddSubMenuItemCheckbox("Jiggle (Natural)", "", settings.MoveMode == string(mousemover.ModeJiggle)),
		}

		// Auto-Stop Timer Submenu adhering to macOS HIG
		timerMenu := systray.AddMenuItem("Auto-Stop Timer", "")
		timerItems := map[int]*systray.MenuItem{
			0:   timerMenu.AddSubMenuItemCheckbox("Continuous (No Timer)", "", settings.TimerMinutes == 0),
			30:  timerMenu.AddSubMenuItemCheckbox("30 Minutes", "", settings.TimerMinutes == 30),
			60:  timerMenu.AddSubMenuItemCheckbox("1 Hour", "", settings.TimerMinutes == 60),
			120: timerMenu.AddSubMenuItemCheckbox("2 Hours", "", settings.TimerMinutes == 120),
			240: timerMenu.AddSubMenuItemCheckbox("4 Hours", "", settings.TimerMinutes == 240),
		}

		systray.AddSeparator()

		// Icons Submenu adhering to macOS HIG
		iconMenu := systray.AddMenuItem("Icon", "")
		iconItems := map[string]*systray.MenuItem{
			"mouse":     iconMenu.AddSubMenuItemCheckbox("Mouse", "", settings.Icon == "mouse"),
			"cloud":     iconMenu.AddSubMenuItemCheckbox("Cloud", "", settings.Icon == "cloud"),
			"man":       iconMenu.AddSubMenuItemCheckbox("Figure", "", settings.Icon == "man"),
			"geometric": iconMenu.AddSubMenuItemCheckbox("Geometric", "", settings.Icon == "geometric"),
		}

		// Set template icons for icon previews in submenu
		mouseIcon := getMenuIcon("mouse")
		iconItems["mouse"].SetTemplateIcon(mouseIcon, mouseIcon)
		cloudIcon := getMenuIcon("cloud")
		iconItems["cloud"].SetTemplateIcon(cloudIcon, cloudIcon)
		manIcon := getMenuIcon("man")
		iconItems["man"].SetTemplateIcon(manIcon, manIcon)
		geoIcon := getMenuIcon("geometric")
		iconItems["geometric"].SetTemplateIcon(geoIcon, geoIcon)

		// Colors Submenu adhering to macOS HIG
		colorMenu := systray.AddMenuItem("Icon Color", "")
		colorItems := map[string]*systray.MenuItem{
			"":      colorMenu.AddSubMenuItemCheckbox("System Default", "", settings.Color == ""),
			"blue":  colorMenu.AddSubMenuItemCheckbox("Blue", "", settings.Color == "blue"),
			"red":   colorMenu.AddSubMenuItemCheckbox("Red", "", settings.Color == "red"),
			"white": colorMenu.AddSubMenuItemCheckbox("White", "", settings.Color == "white"),
		}

		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit Automatic Mouse Mover", "")

		mouseMover := mousemover.GetInstance()
		startAppWithTimer(mouseMover, &settings, ammStart, ammStop)

		restartIfRunning := func() {
			if ammStart.Disabled() {
				mouseMover.Quit()
				startAppWithTimer(mouseMover, &settings, ammStart, ammStop)
			}
		}

		for {
			select {
			case <-ammStart.ClickedCh:
				log.Infof("starting the app")
				startAppWithTimer(mouseMover, &settings, ammStart, ammStop)

			case <-ammStop.ClickedCh:
				log.Infof("stopping the app")
				stopApp(mouseMover, &settings, ammStart, ammStop)

			case <-mQuit.ClickedCh:
				log.Infof("Requesting quit")
				stopApp(mouseMover, &settings, ammStart, ammStop)
				systray.Quit()
				return

			// Intervals
			case <-intervalItems[30].ClickedCh:
				settings.Interval = 30
				updateChecks(intervalItems, 30)
				saveSettings(&settings)
				restartIfRunning()
			case <-intervalItems[60].ClickedCh:
				settings.Interval = 60
				updateChecks(intervalItems, 60)
				saveSettings(&settings)
				restartIfRunning()
			case <-intervalItems[120].ClickedCh:
				settings.Interval = 120
				updateChecks(intervalItems, 120)
				saveSettings(&settings)
				restartIfRunning()
			case <-intervalItems[300].ClickedCh:
				settings.Interval = 300
				updateChecks(intervalItems, 300)
				saveSettings(&settings)
				restartIfRunning()
			case <-intervalItems[600].ClickedCh:
				settings.Interval = 600
				updateChecks(intervalItems, 600)
				saveSettings(&settings)
				restartIfRunning()

			// Modes
			case <-modeItems[string(mousemover.ModeStandard)].ClickedCh:
				settings.MoveMode = string(mousemover.ModeStandard)
				updateChecks(modeItems, string(mousemover.ModeStandard))
				saveSettings(&settings)
				restartIfRunning()
			case <-modeItems[string(mousemover.ModeMicro)].ClickedCh:
				settings.MoveMode = string(mousemover.ModeMicro)
				updateChecks(modeItems, string(mousemover.ModeMicro))
				saveSettings(&settings)
				restartIfRunning()
			case <-modeItems[string(mousemover.ModeJiggle)].ClickedCh:
				settings.MoveMode = string(mousemover.ModeJiggle)
				updateChecks(modeItems, string(mousemover.ModeJiggle))
				saveSettings(&settings)
				restartIfRunning()

			// Timers
			case <-timerItems[0].ClickedCh:
				settings.TimerMinutes = 0
				updateChecks(timerItems, 0)
				saveSettings(&settings)
				restartIfRunning()
			case <-timerItems[30].ClickedCh:
				settings.TimerMinutes = 30
				updateChecks(timerItems, 30)
				saveSettings(&settings)
				restartIfRunning()
			case <-timerItems[60].ClickedCh:
				settings.TimerMinutes = 60
				updateChecks(timerItems, 60)
				saveSettings(&settings)
				restartIfRunning()
			case <-timerItems[120].ClickedCh:
				settings.TimerMinutes = 120
				updateChecks(timerItems, 120)
				saveSettings(&settings)
				restartIfRunning()
			case <-timerItems[240].ClickedCh:
				settings.TimerMinutes = 240
				updateChecks(timerItems, 240)
				saveSettings(&settings)
				restartIfRunning()

			// Icons
			case <-iconItems["mouse"].ClickedCh:
				updateChecks(iconItems, "mouse")
				setIcon("mouse", settings.Color, configFile, &settings, ammStart.Disabled())
			case <-iconItems["cloud"].ClickedCh:
				updateChecks(iconItems, "cloud")
				setIcon("cloud", settings.Color, configFile, &settings, ammStart.Disabled())
			case <-iconItems["man"].ClickedCh:
				updateChecks(iconItems, "man")
				setIcon("man", settings.Color, configFile, &settings, ammStart.Disabled())
			case <-iconItems["geometric"].ClickedCh:
				updateChecks(iconItems, "geometric")
				setIcon("geometric", settings.Color, configFile, &settings, ammStart.Disabled())

			// Colors
			case <-colorItems[""].ClickedCh:
				updateChecks(colorItems, "")
				setIcon(settings.Icon, "", configFile, &settings, ammStart.Disabled())
			case <-colorItems["blue"].ClickedCh:
				updateChecks(colorItems, "blue")
				setIcon(settings.Icon, "blue", configFile, &settings, ammStart.Disabled())
			case <-colorItems["red"].ClickedCh:
				updateChecks(colorItems, "red")
				setIcon(settings.Icon, "red", configFile, &settings, ammStart.Disabled())
			case <-colorItems["white"].ClickedCh:
				updateChecks(colorItems, "white")
				setIcon(settings.Icon, "white", configFile, &settings, ammStart.Disabled())

			case <-about.ClickedCh:
				log.Infof("Requesting about")
				robotgo.Alert("Automatic Mouse Mover", "Automatic Mouse Mover v1.5\n\nFeatures:\n• Native Apple Silicon & Intel Universal binary\n• Configurable intervals (30s to 10m)\n• Invisible Micro-Nudge & Jiggle modes\n• Auto-Stop Countdown Timers\n• macOS HIG compliant menus & icon options\n\nGitHub: https://github.com/samlara32/automatic-mouse-mover", "OK", "")
			}
		}
	}()
}

func onExit() {
	log.Infof("Finished quitting")
}
