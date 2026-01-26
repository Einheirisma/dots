#!/usr/bin/env sh

# 
# /home/$USER/.config/mango/scripts/autostart.sh
#

set +e

# Screen Sharing 
dbus-update-activation-environment --systemd WAYLAND_DISPLAY XDG_CURRENT_DESKTOP=wlroots >/dev/null 2>&1

# Wallpaper daemon
awww-daemon >/dev/null 2>&1 & 
awww img ~/.pictures/background/anime-skull.png &

# Launcher
vicinae server >/dev/null 2>&1 &

# Bar
waybar >/dev/null 2>&1 &

# Brightness control
brightnessctl s 255 >/dev/null 2>&1
