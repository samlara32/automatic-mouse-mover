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

		about := systray.AddMenuItem("About AMM", "Information about the app")
		systray.AddSeparator()
		ammStart := systray.AddMenuItem("Start", "start the app")
		ammStop := systray.AddMenuItem("Stop", "stop the app")
		systray.AddSeparator()

		// Interval Submenu
		intervalMenu := systray.AddMenuItem("Interval", "Interval between idle checks")
		i30 := intervalMenu.AddSubMenuItem("30 Seconds ⚡", "Check every 30 seconds")
		i60 := intervalMenu.AddSubMenuItem("1 Minute (Default) ⏱", "Check every 1 minute")
		i120 := intervalMenu.AddSubMenuItem("2 Minutes ⏱", "Check every 2 minutes")
		i300 := intervalMenu.AddSubMenuItem("5 Minutes ⏱", "Check every 5 minutes")
		i600 := intervalMenu.AddSubMenuItem("10 Minutes ⏱", "Check every 10 minutes")

		// Movement Mode Submenu
		modeMenu := systray.AddMenuItem("Movement Mode", "Mouse movement style")
		mStandard := modeMenu.AddSubMenuItem("Standard (10px) 🟢", "10px shift back and forth")
		mMicro := modeMenu.AddSubMenuItem("Micro-nudge (1px) 🔍", "1px invisible subtle nudge")
		mJiggle := modeMenu.AddSubMenuItem("Jiggle (Natural) 🔀", "Random subtle human-like motion")

		// Auto-stop Timer Submenu
		timerMenu := systray.AddMenuItem("Auto-Stop Timer", "Automatically stop after duration")
		tContinuous := timerMenu.AddSubMenuItem("Continuous ♾️", "Run until manually stopped")
		t30m := timerMenu.AddSubMenuItem("30 Minutes ⏳", "Run for 30 minutes")
		t1h := timerMenu.AddSubMenuItem("1 Hour ⏳", "Run for 1 hour")
		t2h := timerMenu.AddSubMenuItem("2 Hours ⏳", "Run for 2 hours")
		t4h := timerMenu.AddSubMenuItem("4 Hours ⏳", "Run for 4 hours")

		systray.AddSeparator()

		// Icons Submenu
		icons := systray.AddMenuItem("Icons", "icon of the app")
		mouse := icons.AddSubMenuItem("Mouse", "Mouse icon")
		mouseIcon := getMenuIcon("mouse")
		mouse.SetTemplateIcon(mouseIcon, mouseIcon)
		cloud := icons.AddSubMenuItem("Cloud", "Cloud icon")
		cloudIcon := getMenuIcon("cloud")
		cloud.SetTemplateIcon(cloudIcon, cloudIcon)
		man := icons.AddSubMenuItem("Man", "Man icon")
		manIcon := getMenuIcon("man")
		man.SetTemplateIcon(manIcon, manIcon)
		geometric := icons.AddSubMenuItem("Geometric", "Geometric")
		geometricIcon := getMenuIcon("geometric")
		geometric.SetTemplateIcon(geometricIcon, geometricIcon)

		// Colors Submenu
		colors := systray.AddMenuItem("Icon Colors", "")
		system := colors.AddSubMenuItem("System", "System default color")
		blue := colors.AddSubMenuItem("Blue 🔵", "Blue")
		white := colors.AddSubMenuItem("White ⚪️", "White")
		red := colors.AddSubMenuItem("Red 🔴", "Red")

		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

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
			case <-i30.ClickedCh:
				settings.Interval = 30
				saveSettings(&settings)
				restartIfRunning()
			case <-i60.ClickedCh:
				settings.Interval = 60
				saveSettings(&settings)
				restartIfRunning()
			case <-i120.ClickedCh:
				settings.Interval = 120
				saveSettings(&settings)
				restartIfRunning()
			case <-i300.ClickedCh:
				settings.Interval = 300
				saveSettings(&settings)
				restartIfRunning()
			case <-i600.ClickedCh:
				settings.Interval = 600
				saveSettings(&settings)
				restartIfRunning()

			// Modes
			case <-mStandard.ClickedCh:
				settings.MoveMode = string(mousemover.ModeStandard)
				saveSettings(&settings)
				restartIfRunning()
			case <-mMicro.ClickedCh:
				settings.MoveMode = string(mousemover.ModeMicro)
				saveSettings(&settings)
				restartIfRunning()
			case <-mJiggle.ClickedCh:
				settings.MoveMode = string(mousemover.ModeJiggle)
				saveSettings(&settings)
				restartIfRunning()

			// Timers
			case <-tContinuous.ClickedCh:
				settings.TimerMinutes = 0
				saveSettings(&settings)
				restartIfRunning()
			case <-t30m.ClickedCh:
				settings.TimerMinutes = 30
				saveSettings(&settings)
				restartIfRunning()
			case <-t1h.ClickedCh:
				settings.TimerMinutes = 60
				saveSettings(&settings)
				restartIfRunning()
			case <-t2h.ClickedCh:
				settings.TimerMinutes = 120
				saveSettings(&settings)
				restartIfRunning()
			case <-t4h.ClickedCh:
				settings.TimerMinutes = 240
				saveSettings(&settings)
				restartIfRunning()

			// Icons
			case <-mouse.ClickedCh:
				setIcon("mouse", settings.Color, configFile, &settings, ammStart.Disabled())
			case <-cloud.ClickedCh:
				setIcon("cloud", settings.Color, configFile, &settings, ammStart.Disabled())
			case <-man.ClickedCh:
				setIcon("man", settings.Color, configFile, &settings, ammStart.Disabled())
			case <-geometric.ClickedCh:
				setIcon("geometric", settings.Color, configFile, &settings, ammStart.Disabled())

			// Colors
			case <-system.ClickedCh:
				setIcon(settings.Icon, "", configFile, &settings, ammStart.Disabled())
			case <-blue.ClickedCh:
				setIcon(settings.Icon, "blue", configFile, &settings, ammStart.Disabled())
			case <-red.ClickedCh:
				setIcon(settings.Icon, "red", configFile, &settings, ammStart.Disabled())
			case <-white.ClickedCh:
				setIcon(settings.Icon, "white", configFile, &settings, ammStart.Disabled())

			case <-about.ClickedCh:
				log.Infof("Requesting about")
				robotgo.Alert("Automatic Mouse Mover", "Automatic Mouse Mover v1.5\n\nFeatures:\n- Native Apple Silicon & Intel Universal binary\n- Configurable intervals (30s to 10m)\n- Invisible Micro-nudge & Jiggle modes\n- Auto-Stop Countdown Timers\n- Custom icons & colors\n\nGitHub: https://github.com/samlara32/automatic-mouse-mover", "OK", "")
			}
		}
	}()
}

func onExit() {
	log.Infof("Finished quitting")
}
