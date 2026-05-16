package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gen2brain/dlgs"
	"github.com/getlantern/systray"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// Config holds persistent settings
type Config struct {
	TimerHours   int `json:"timer_hours"`
	TimerMinutes int `json:"timer_minutes"`
	FixedHour    int `json:"fixed_hour"`
	FixedMinute  int `json:"fixed_minute"`
}

var (
	configPath string
	config     Config

	// Active countdown
	activeCancel context.CancelFunc
	activeUntil  time.Time
	activeType   string // "timer" or "time"

	// Icons
	iconLight  []byte
	iconDark   []byte
	iconActive []byte
)

func main() {
	// Determine config path
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	configPath = filepath.Join(configDir, "sleep-on-time", "config.json")
	loadConfig()

	// Load icons
	iconLight, err = loadIcon("assets/icon-light.svg")
	if err != nil {
		log.Fatal(err)
	}
	iconDark, err = loadIcon("assets/icon-dark.svg")
	if err != nil {
		log.Fatal(err)
	}
	iconActive, err = loadIcon("assets/icon-active.svg")
	if err != nil {
		log.Fatal(err)
	}

	systray.Run(onReady, onExit)
}

var (
	mTimerActivate *systray.MenuItem
	mTimeActivate  *systray.MenuItem
	mCancel        *systray.MenuItem
)

func onReady() {
	systray.SetIcon(iconLight)
	systray.SetTitle("Sleep on Time")
	updateTooltip()

	// Menu items
	mTimer := systray.AddMenuItem("Timer", "Timer submenu")
	mTimerSet := mTimer.AddSubMenuItem("Set...", "Set timer duration")
	mTimerActivate = mTimer.AddSubMenuItem("Activate", "Activate timer with last duration")
	if config.TimerHours == 0 && config.TimerMinutes == 0 {
		mTimerActivate.Disable() // disabled until a duration is set
	}

	mTime := systray.AddMenuItem("Time", "Fixed time submenu")
	mTimeSet := mTime.AddSubMenuItem("Set...", "Set fixed time")
	mTimeActivate = mTime.AddSubMenuItem("Activate", "Activate fixed time alarm")
	if config.FixedHour == 0 && config.FixedMinute == 0 {
		mTimeActivate.Disable() // disabled until a time is set
	}

	systray.AddSeparator()

	mCancel = systray.AddMenuItem("Cancel", "Cancel active countdown")
	mCancel.Disable() // disabled until a countdown is active

	mExit := systray.AddMenuItem("Exit", "Exit the application")

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-mTimerSet.ClickedCh:
				setTimerDialog()
			case <-mTimerActivate.ClickedCh:
				activateTimer()
			case <-mTimeSet.ClickedCh:
				setTimeDialog()
			case <-mTimeActivate.ClickedCh:
				activateTime()
			case <-mCancel.ClickedCh:
				cancelCountdown()
			case <-mExit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()

	// Tooltip updater
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			updateTooltip()
		}
	}()
}

func onExit() {
	saveConfig()
	if activeCancel != nil {
		activeCancel()
	}
}

// loadIcon reads an SVG file and converts it to PNG bytes
func loadIcon(svgPath string) ([]byte, error) {
	// Read SVG file
	svgData, err := os.ReadFile(svgPath)
	if err != nil {
		return nil, err
	}

	// Parse SVG
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svgData))
	if err != nil {
		return nil, err
	}

	// Set viewport to SVG's viewBox
	w, h := int(icon.ViewBox.W), int(icon.ViewBox.H)
	if w <= 0 || h <= 0 {
		w, h = 64, 64
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	icon.SetTarget(0, 0, float64(w), float64(h))
	raster := rasterx.NewDasher(w, h, rasterx.NewScannerGV(w, h, img, img.Bounds()))
	icon.Draw(raster, 1.0)

	// Encode to PNG
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// updateTooltip updates the systray tooltip with remaining time if active
func updateTooltip() {
	if !activeUntil.IsZero() && time.Now().Before(activeUntil) {
		remaining := time.Until(activeUntil)
		hours := int(remaining.Hours())
		minutes := int(remaining.Minutes()) % 60
		tooltip := fmt.Sprintf("Sleep on Time (%dh%02d)", hours, minutes)
		systray.SetTooltip(tooltip)
		systray.SetIcon(iconActive)
	} else {
		systray.SetTooltip("Sleep on Time")
		systray.SetIcon(iconLight)
	}
}

// setTimerDialog opens a dialog to set timer duration
func setTimerDialog() {
	// We'll ask for hours and minutes separately
	hour, ok, err := dlgs.Entry("Timer Hours", "Enter hours (0-23):", fmt.Sprintf("%d", config.TimerHours))
	if err != nil {
		log.Printf("Dialog error: %v", err)
		return
	}
	if !ok {
		return // user cancelled
	}
	var hours int
	fmt.Sscanf(hour, "%d", &hours)
	if hours < 0 || hours > 23 {
		dlgs.Error("Invalid hour", "Hour must be between 0 and 23.")
		return
	}

	min, ok, err := dlgs.Entry("Timer Minutes", "Enter minutes (0-59):", fmt.Sprintf("%d", config.TimerMinutes))
	if err != nil {
		log.Printf("Dialog error: %v", err)
		return
	}
	if !ok {
		return
	}
	var minutes int
	fmt.Sscanf(min, "%d", &minutes)
	if minutes < 0 || minutes > 59 {
		dlgs.Error("Invalid minute", "Minute must be between 0 and 59.")
		return
	}

	if hours == 0 && minutes == 0 {
		dlgs.Error("Invalid duration", "Please set at least 1 minute.")
		return
	}

	config.TimerHours = hours
	config.TimerMinutes = minutes
	saveConfig()

	// Enable activate button if not already enabled
	if mTimerActivate != nil {
		mTimerActivate.Enable()
	}
}

// activateTimer starts the countdown based on timer duration
func activateTimer() {
	duration := time.Duration(config.TimerHours)*time.Hour + time.Duration(config.TimerMinutes)*time.Minute
	if duration <= 0 {
		return
	}
	startCountdown(time.Now().Add(duration), "timer")
}

// setTimeDialog opens a dialog to set fixed time
func setTimeDialog() {
	hour, ok, err := dlgs.Entry("Fixed Time Hour", "Enter hour (0-23):", fmt.Sprintf("%d", config.FixedHour))
	if err != nil {
		log.Printf("Dialog error: %v", err)
		return
	}
	if !ok {
		return
	}
	var h int
	fmt.Sscanf(hour, "%d", &h)
	if h < 0 || h > 23 {
		dlgs.Error("Invalid hour", "Hour must be between 0 and 23.")
		return
	}

	min, ok, err := dlgs.Entry("Fixed Time Minute", "Enter minute (0-59):", fmt.Sprintf("%d", config.FixedMinute))
	if err != nil {
		log.Printf("Dialog error: %v", err)
		return
	}
	if !ok {
		return
	}
	var m int
	fmt.Sscanf(min, "%d", &m)
	if m < 0 || m > 59 {
		dlgs.Error("Invalid minute", "Minute must be between 0 and 59.")
		return
	}

	config.FixedHour = h
	config.FixedMinute = m
	saveConfig()

	// Enable activate button
	if mTimeActivate != nil {
		mTimeActivate.Enable()
	}
}

// activateTime starts the countdown to the fixed time
func activateTime() {
	now := time.Now()
	target := time.Date(now.Year(), now.Month(), now.Day(), config.FixedHour, config.FixedMinute, 0, 0, now.Location())
	if target.Before(now) {
		target = target.Add(24 * time.Hour)
	}
	startCountdown(target, "time")
}

// startCountdown begins a countdown to deadline
func startCountdown(deadline time.Time, countdownType string) {
	// Cancel any existing countdown
	if activeCancel != nil {
		activeCancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	activeCancel = cancel
	activeUntil = deadline
	activeType = countdownType

	systray.SetIcon(iconActive)
	updateTooltip()

	// Enable cancel menu item
	if mCancel != nil {
		mCancel.Enable()
	}

	go func() {
		select {
		case <-ctx.Done():
			// cancelled
			activeUntil = time.Time{}
			systray.SetIcon(iconLight)
			updateTooltip()
			if mCancel != nil {
				mCancel.Disable()
			}
			return
		case <-time.After(time.Until(deadline)):
			// Time to sleep
			activeUntil = time.Time{}
			systray.SetIcon(iconLight)
			updateTooltip()
			if mCancel != nil {
				mCancel.Disable()
			}
			sleep()
		}
	}()
}

// cancelCountdown cancels the active countdown
func cancelCountdown() {
	if activeCancel != nil {
		activeCancel()
	}
}

// sleep puts the system to sleep
func sleep() {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("systemctl", "suspend")
		if err := cmd.Run(); err != nil {
			cmd = exec.Command("loginctl", "suspend")
			if err := cmd.Run(); err != nil {
				log.Printf("Failed to suspend: %v", err)
			}
		}
	case "darwin":
		cmd = exec.Command("pmset", "sleepnow")
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to sleep: %v", err)
		}
	case "windows":
		cmd = exec.Command("rundll32.exe", "powrprof.dll,SetSuspendState", "0,1,0")
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to sleep: %v", err)
		}
	default:
		log.Printf("Unsupported OS: %s", runtime.GOOS)
	}
}

// loadConfig reads config from file
func loadConfig() {
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Config doesn't exist, use defaults
		config = Config{}
		return
	}
	json.Unmarshal(data, &config)
}

// saveConfig writes config to file
func saveConfig() {
	dir := filepath.Dir(configPath)
	os.MkdirAll(dir, 0755)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal config: %v", err)
		return
	}
	os.WriteFile(configPath, data, 0644)
}
