# Sleep on Time

A simple system tray application that puts your computer to sleep after a timer or at a fixed time.

## Features

- System tray icon with active/inactive states
- **Automatic dark/light mode detection** (cross-platform)
  - **Linux**: Uses freedesktop portal (`gdbus`), then GNOME/KDE fallbacks
  - **macOS**: Detects via `defaults read -g AppleInterfaceStyle`
  - **Windows**: Queries registry `AppsUseLightTheme` value
  - Checks every second for theme changes
- **Remaining time display** in system tray menu (disabled entry showing countdown)
  - Shows only non-zero components (e.g., 1h30m45s, 45m, 20s)
  - Same format used in tooltip and menu entry
- **Timer mode**: Set minutes only, auto-activates after setting
  - Shows currently set duration in menu
- **Fixed mode**: Set time via hour + minute entry
  - Schedules for next occurrence of that time (today or tomorrow)
  - Auto-activates after setting
  - Shows currently set time in menu (e.g., "Set... (current: 14:30)")
- Countdown shown in tooltip
- Persists last settings
- Cross-platform (Linux, macOS, Windows)

## Dependencies

- Go 1.18+
- On Linux: `libayatana-appindicator` and `zenity` (or `kdialog`) for dialogs
- On macOS: `pmset` (built-in)
- On Windows: built-in

## Build

```bash
go build -o sleep-on-time .
```

## Installation (Linux)

1. Build the binary: `go build -o sleep-on-time .`
2. Move the binary to `/usr/local/bin`: `sudo mv sleep-on-time /usr/local/bin/`
3. Create config directory: `mkdir -p ~/.config/sleep-on-time/assets`
4. Copy assets: `cp -r assets ~/.config/sleep-on-time/`
5. (Optional) Copy `sleep-on-time.desktop` to `~/.local/share/applications/` and update paths.

## Usage

Run the executable. The app will appear in your system tray.

- **Timer > Set...**: Set timer duration (minutes only, 1-999)
  - Auto-activates after setting
  - Shows current duration in menu
- **Timer > Activate**: Start the timer with last set duration
- **Fixed > Set...**: Set fixed time (hour 0-23, minute 0-59)
  - Schedules for next occurrence (today or tomorrow)
  - Auto-activates after setting
  - Shows current time in menu (e.g., "Set... (current: 14:30)")
- **Fixed > Activate**: Start countdown to fixed time
- **Remaining: --**: Disabled entry showing countdown (when active)
- **Cancel**: Cancel active countdown
- **Exit**: Quit the application

## Configuration

Settings are saved in `~/.config/sleep-on-time/config.json`.

## Icons

Icons are stored in `assets/` as SVG files. They are converted to PNG at runtime.

If you want to use custom icons, replace the SVG files in `assets/`.
