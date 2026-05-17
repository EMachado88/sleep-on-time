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
- Set a timer (hours + minutes) to sleep
- Set a fixed time to sleep
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

- **Timer > Set...**: Set timer duration (hours and minutes)
- **Timer > Activate**: Start the timer with last set duration
- **Time > Set...**: Set fixed time (hour and minute)
- **Time > Activate**: Start countdown to fixed time
- **Cancel**: Cancel active countdown
- **Exit**: Quit the application

## Configuration

Settings are saved in `~/.config/sleep-on-time/config.json`.

## Icons

Icons are stored in `assets/` as SVG files. They are converted to PNG at runtime.

If you want to use custom icons, replace the SVG files in `assets/`.
