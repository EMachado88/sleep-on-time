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
	"strconv"
	"strings"
	"time"

	"github.com/gen2brain/dlgs"
	"github.com/getlantern/systray"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// Config holds persistent settings
type Config struct {
	TimerMinutes int       `json:"timer_minutes"`
	FixedTime    time.Time `json:"fixed_time"`
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

	// Theme detection
	isDarkMode bool
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
	mTimerSet      *systray.MenuItem
	mTimerActivate *systray.MenuItem
	mFixedSet      *systray.MenuItem
	mTimeActivate  *systray.MenuItem
	mRemaining     *systray.MenuItem
	mCancel        *systray.MenuItem
)

func onReady() {
	systray.SetIcon(iconLight)
	systray.SetTitle("Sleep on Time")
	updateTooltip()

	// Initial theme detection
	isDarkMode = detectDarkMode()
	updateIcon()

	// Menu items
	mTimer := systray.AddMenuItem("Timer", "Timer submenu")
	mTimerSet = mTimer.AddSubMenuItem("Set", "Set timer duration")
	mTimerActivate = mTimer.AddSubMenuItem("Activate", "Activate timer with last duration")
	if config.TimerMinutes <= 0 {
		mTimerActivate.Disable() // disabled until a duration is set
	} else {
		// Show currently set timer
		mTimerSet.SetTitle(fmt.Sprintf("Set (current: %s)", formatRemainingTime(time.Duration(config.TimerMinutes)*time.Minute)))
	}

	mFixed := systray.AddMenuItem("Fixed", "Fixed time submenu")
	mFixedSet = mFixed.AddSubMenuItem("Set", "Set fixed time")
	mTimeActivate = mFixed.AddSubMenuItem("Activate", "Activate fixed time alarm")
	if config.FixedTime.IsZero() {
		mTimeActivate.Disable() // disabled until a time is set
	} else {
		// Show currently set time
		mFixedSet.SetTitle(fmt.Sprintf("Set (current: %s)", config.FixedTime.Format("15:04")))
	}

	systray.AddSeparator()

	mRemaining = systray.AddMenuItem("Remaining: --", "Remaining time until sleep")
	mRemaining.Disable() // always disabled, just informational

	mCancel = systray.AddMenuItem("Cancel", "Cancel active countdown")
	mCancel.Disable() // disabled until a countdown is active

	mExit := systray.AddMenuItem("Exit", "Exit the application")

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-mTimerSet.ClickedCh:
				setTimerDialog()
				// Auto-activate after setting
				activateTimer()
			case <-mTimerActivate.ClickedCh:
				activateTimer()
			case <-mFixedSet.ClickedCh:
				setFixedDialog()
				// Auto-activate after setting
				activateFixed()
			case <-mTimeActivate.ClickedCh:
				activateFixed()
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
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			updateTooltip()
		}
	}()

	// Remaining time updater - update every second
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			updateRemainingMenu()
		}
	}()

	// Theme watcher - check every second
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			newDarkMode := detectDarkMode()
			if newDarkMode != isDarkMode {
				isDarkMode = newDarkMode
				updateIcon()
			}
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

// formatRemainingTime formats a duration into a string with only non-zero components.
// Example: 1h30m45s, 45m20s, 20s, etc.
func formatRemainingTime(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	var parts []string
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	if len(parts) == 0 {
		return "0s"
	}

	return strings.Join(parts, "")
}

// updateTooltip updates the systray tooltip with remaining time if active
// Icon updates are handled separately by updateIcon() via the theme watcher
func updateTooltip() {
	if !activeUntil.IsZero() && time.Now().Before(activeUntil) {
		remaining := time.Until(activeUntil)
		tooltip := fmt.Sprintf("Sleep on Time (%s)", formatRemainingTime(remaining))
		systray.SetTooltip(tooltip)
	} else {
		systray.SetTooltip("Sleep on Time")
	}
}

// updateRemainingMenu updates the disabled menu item with remaining time
func updateRemainingMenu() {
	if mRemaining == nil {
		return
	}
	if !activeUntil.IsZero() && time.Now().Before(activeUntil) {
		remaining := time.Until(activeUntil)
		text := fmt.Sprintf("Remaining: %s", formatRemainingTime(remaining))
		mRemaining.SetTitle(text)
	} else {
		mRemaining.SetTitle("")
	}
}

// updateIcon sets the appropriate icon based on active state and theme
func updateIcon() {
	if !activeUntil.IsZero() && time.Now().Before(activeUntil) {
		systray.SetIcon(iconActive)
	} else if isDarkMode {
		systray.SetIcon(iconDark)
	} else {
		systray.SetIcon(iconLight)
	}
}

// detectDarkMode attempts to detect if the system is using dark mode.
// Returns true if dark mode is detected, false otherwise.
func detectDarkMode() bool {
	switch runtime.GOOS {
	case "linux":
		// Try freedesktop portal first (gdbus)
		if dark, ok := detectDarkModePortal(); ok {
			return dark
		}
		// Fall back to DE-specific methods
		return detectDarkModeLinuxDE()
	case "darwin":
		return detectDarkModeDarwin()
	case "windows":
		return detectDarkModeWindows()
	default:
		return false
	}
}

// detectDarkModePortal uses freedesktop portal to detect color-scheme
// Returns (isDark, true) if successful, (false, false) if failed
func detectDarkModePortal() (bool, bool) {
	// gdbus call --session --dest org.freedesktop.portal.Desktop \
	//   --object-path /org/freedesktop/portal/desktop \
	//   --method org.freedesktop.portal.Settings.Read \
	//   org.freedesktop.appearance color-scheme
	cmd := exec.Command("gdbus", "call",
		"--session",
		"--dest", "org.freedesktop.portal.Desktop",
		"--object-path", "/org/freedesktop/portal/desktop",
		"--method", "org.freedesktop.portal.Settings.Read",
		"org.freedesktop.appearance", "color-scheme")
	output, err := cmd.Output()
	if err != nil {
		return false, false
	}

	// Parse output: expects something like (<<uint32 1>>,)
	// where 0 = no preference (assume light), 1 = dark, 2 = light
	str := strings.TrimSpace(string(output))
	// Find "uint32" in the string
	idx := strings.Index(str, "uint32")
	if idx == -1 {
		return false, false
	}

	// Extract the number after "uint32"
	sub := str[idx+6:] // skip "uint32"
	sub = strings.TrimSpace(sub)

	// Remove any leading non-digit characters (like '<' or '>')
	for len(sub) > 0 && !strings.ContainsAny(string(sub[0]), "0123456789") {
		sub = sub[1:]
	}

	// Get the number before comma, parenthesis, or angle bracket
	end := strings.IndexAny(sub, ",)>")
	if end != -1 {
		sub = sub[:end]
	}
	sub = strings.TrimSpace(sub)

	if val, err := strconv.Atoi(sub); err == nil {
		// 1 = dark, 2 = light, 0 = no preference (assume light)
		return val == 1, true
	}

	return false, false
}

// detectDarkModeLinuxDE detects dark mode using DE-specific methods
func detectDarkModeLinuxDE() bool {
	// Try GNOME gsettings for color-scheme (newer GNOME)
	if output, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme").Output(); err == nil {
		if strings.Contains(strings.ToLower(string(output)), "dark") {
			return true
		}
	}

	// gsettings for gtk-theme
	if output, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "gtk-theme").Output(); err == nil {
		if strings.Contains(strings.ToLower(string(output)), "dark") {
			return true
		}
	}

	// Try KDE 6 (kreadconfig6)
	if output, err := exec.Command("kreadconfig6", "--group", "KDE", "--key", "Theme").Output(); err == nil {
		if strings.Contains(strings.ToLower(string(output)), "dark") {
			return true
		}
	}

	// Try KDE 5 (older)
	if output, err := exec.Command("kreadconfig5", "--group", "KDE", "--key", "Theme").Output(); err == nil {
		if strings.Contains(strings.ToLower(string(output)), "dark") {
			return true
		}
	}

	// Check environment variable
	if theme := os.Getenv("GTK_THEME"); strings.Contains(strings.ToLower(theme), "dark") {
		return true
	}

	return false
}

// detectDarkModeDarwin checks for dark mode on macOS.
func detectDarkModeDarwin() bool {
	// Use defaults command to read global AppleInterfaceStyle
	// If the value is "Dark", then dark mode is active.
	// If the key doesn't exist or is not "Dark", it's light mode.
	cmd := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle")
	output, err := cmd.Output()
	if err != nil {
		// Key doesn't exist -> light mode
		return false
	}
	// Output is usually "Dark\n" if dark mode is on
	return strings.Contains(strings.ToLower(strings.TrimSpace(string(output))), "dark")
}

// detectDarkModeWindows checks for dark mode on Windows.
// It queries the registry key AppsUseLightTheme under
// HKCU\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize.
// Value 0 means dark mode (light theme off), 1 means light mode.
func detectDarkModeWindows() bool {
	cmd := exec.Command("reg", "query",
		"HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Themes\\Personalize",
		"/v", "AppsUseLightTheme")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	// Output format: something like
	// HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize
	//     AppsUseLightTheme    REG_DWORD    0x0
	// We look for "0x0" which indicates dark mode (value 0).
	// Actually, the value is a DWORD. If it's 0, dark mode; if 1, light mode.
	// So we check if the line contains "0x0" (or "0x1" for light).
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "AppsUseLightTheme") {
			// The value is after the last space maybe.
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				valStr := fields[len(fields)-1]
				// valStr might be "0x0" or "0x1"
				if valStr == "0x0" {
					return true
				}
			}
		}
	}
	return false
}

// setTimerDialog opens a dialog to set timer duration (minutes only)
func setTimerDialog() {
	min, ok, err := dlgs.Entry("Timer Minutes", "Enter minutes:", fmt.Sprintf("%d", config.TimerMinutes))
	if err != nil {
		log.Printf("Dialog error: %v", err)
		return
	}
	if !ok {
		return // user cancelled
	}
	var minutes int
	fmt.Sscanf(min, "%d", &minutes)
	if minutes <= 0 || minutes > 999 {
		dlgs.Error("Invalid duration", "Please set at least 1 minute.")
		return
	}

	config.TimerMinutes = minutes
	saveConfig()

	// Enable activate button if not already enabled
	if mTimerActivate != nil {
		mTimerActivate.Enable()
	}

	// Update menu item to show current timer
	if mTimerSet != nil {
		mTimerSet.SetTitle(fmt.Sprintf("Set (current: %s)", formatRemainingTime(time.Duration(config.TimerMinutes)*time.Minute)))
	}
}

// activateTimer starts the countdown based on timer duration (minutes only)
func activateTimer() {
	duration := time.Duration(config.TimerMinutes) * time.Minute
	if duration <= 0 {
		return
	}
	startCountdown(time.Now().Add(duration), "timer")
}

// setFixedDialog opens a dialog to set fixed time (hour + minute)
func setFixedDialog() {
	// Get hour
	hour, ok, err := dlgs.Entry("Fixed Time Hour", "Enter hour (0-23):", fmt.Sprintf("%d", config.FixedTime.Hour()))
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

	// Get minute
	min, ok, err := dlgs.Entry("Fixed Time Minute", "Enter minute (0-59):", fmt.Sprintf("%d", config.FixedTime.Minute()))
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

	// Create time for today
	now := time.Now()
	target := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, now.Location())
	if target.Before(now) {
		target = target.Add(24 * time.Hour) // schedule for tomorrow
	}

	config.FixedTime = target
	saveConfig()

	// Enable activate button
	if mTimeActivate != nil {
		mTimeActivate.Enable()
	}

	// Update menu item to show current fixed time
	if mFixedSet != nil {
		mFixedSet.SetTitle(fmt.Sprintf("Set (current: %s)", config.FixedTime.Format("15:04")))
	}
}

// activateFixed starts the countdown to the fixed time
func activateFixed() {
	if config.FixedTime.IsZero() {
		return
	}

	now := time.Now()
	target := time.Date(now.Year(), now.Month(), now.Day(), config.FixedTime.Hour(), config.FixedTime.Minute(), 0, 0, now.Location())
	if target.Before(now) {
		target = target.Add(24 * time.Hour) // schedule for tomorrow
	}

	if target.Before(now) {
		dlgs.Error("Invalid time", "The set time is in the past.")
		return
	}

	startCountdown(target, "fixed")
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
		var timedOut bool
		select {
		case <-ctx.Done():
			// cancelled
			timedOut = false
		case <-time.After(time.Until(deadline)):
			// Time to sleep
			timedOut = true
		}

		// Common cleanup
		activeUntil = time.Time{}
		updateIcon()
		updateTooltip()
		if mCancel != nil {
			mCancel.Disable()
		}
		if mRemaining != nil {
			mRemaining.SetTitle("")
		}

		if timedOut {
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
