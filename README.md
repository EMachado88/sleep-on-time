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

### Quick Install (Linux)

For a complete installation on Linux including desktop integration:

```bash
./install-linux.sh
```

This script will:
- Build the binary from source
- Install the binary to `/usr/local/bin/`
- Install the `.desktop` file to `/usr/share/applications/`
- Install icons to standard locations (`/usr/share/pixmaps/` and hicolor theme)
- Update icon caches for KDE/GNOME

### Manual Build

```bash
go build -o sleep-on-time .
```

## Installation (Linux)

### Using the Install Script (Recommended)

The easiest way to install on Linux is using the provided install script:

```bash
./install-linux.sh
```

This will:
1. Check that Go is installed
2. Build the `sleep-on-time` binary
3. Install the binary to `/usr/local/bin/`
4. Install the desktop file to `/usr/share/applications/`
5. Install icons to `/usr/share/pixmaps/` and `/usr/share/icons/hicolor/`
6. Update icon caches for proper desktop integration

After installation, you can run `sleep-on-time` from the terminal or find it in your applications menu.

### Manual Installation

If you prefer to install manually:

1. Build the binary: `go build -o sleep-on-time .`
2. Move the binary to `/usr/local/bin`: `sudo cp sleep-on-time /usr/local/bin/`
3. Copy the desktop file: `sudo cp sleep-on-time.desktop /usr/share/applications/`
4. Install the icon:
   ```bash
   sudo cp assets/icon-light.svg /usr/share/pixmaps/sleep-on-time-icon.svg
   sudo cp assets/icon-light.svg /usr/share/icons/hicolor/scalable/apps/sleep-on-time-icon.svg
   ```
5. Update icon cache (KDE): `kbuildsycoca6 --noincremental` or `kbuildsycoca5 --noincremental`

**Note:** The icons are embedded in the binary, so the application itself doesn't need external icon files to run. The icon files installed above are only for the desktop menu entry.

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

Icons are embedded directly in the binary using Go's `embed` package. This means:
- The application works anywhere the binary is installed without needing external files
- The icons automatically adapt to your system's dark/light theme
- For desktop menu integration, icon files are installed to system locations by the install script

Source SVG files are stored in `assets/` directory:
- `icon-light.svg` - Light theme icon
- `icon-dark.svg` - Dark theme icon  
- `icon-active.svg` - Active countdown icon

If you want to use custom icons, replace the SVG files in `assets/` and rebuild.
