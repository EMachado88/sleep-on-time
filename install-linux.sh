#!/bin/bash

set -e

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go first."
    echo "Visit https://golang.org/dl/ for installation instructions."
    exit 1
fi

echo "Building sleep-on-time..."
go build -o sleep-on-time .

echo "Installing binary to /usr/local/bin..."
sudo cp sleep-on-time /usr/local/bin/sleep-on-time
sudo chmod +x /usr/local/bin/sleep-on-time

echo "Installing desktop file..."
if [ -f "sleep-on-time.desktop" ]; then
    sudo cp sleep-on-time.desktop /usr/share/applications/
    echo "Desktop file installed to /usr/share/applications/"
else
    echo "Warning: sleep-on-time.desktop not found, skipping desktop file installation."
fi

echo "Installing icon..."
if [ -f "assets/icon-light.svg" ]; then
    # Install to multiple locations for compatibility
    sudo mkdir -p /usr/share/pixmaps
    sudo cp assets/icon-light.svg /usr/share/pixmaps/sleep-on-time-icon.svg
    echo "Icon installed to /usr/share/pixmaps/sleep-on-time-icon.svg"

    # Also install to hicolor theme for better integration
    sudo mkdir -p /usr/share/icons/hicolor/scalable/apps
    sudo cp assets/icon-light.svg /usr/share/icons/hicolor/scalable/apps/sleep-on-time-icon.svg
    echo "Icon installed to /usr/share/icons/hicolor/scalable/apps/sleep-on-time-icon.svg"

    # Update icon cache for KDE
    if command -v kbuildsycoca6 &> /dev/null; then
        kbuildsycoca6 --noincremental 2>/dev/null || true
    elif command -v kbuildsycoca5 &> /dev/null; then
        kbuildsycoca5 --noincremental 2>/dev/null || true
    fi

    # Update GTK icon cache if available
    if command -v gtk-update-icon-cache &> /dev/null; then
        sudo gtk-update-icon-cache /usr/share/icons/hicolor/ 2>/dev/null || true
    fi
else
    echo "Warning: assets/icon-light.svg not found, skipping icon installation."
fi

echo "Installation complete!"
echo "You can now run 'sleep-on-time' from the terminal or find it in your applications menu."
