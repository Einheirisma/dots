#!/bin/sh

# Default programs
export EDITOR="nvim"
export VISUAL="nvim"
export VIDEOPLAYER="mpv"
export TERMINAL="ghostty"
export IMAGEVIEWER="swayimg"
export BROWSER="brave"

# Follow XDG base dir specification
export XDG_CONFIG_HOME="$HOME/.config"
export XDG_DATA_HOME="$HOME/.local/share"
export XDG_CACHE_HOME="$HOME/.cache"

# Bootstrap .zshrc to $HOME/.config/zsh/.zshrc,
# any other config files can also reside here
export ZDOTDIR="$XDG_CONFIG_HOME/zsh"

# History files
export HISTFILE="$XDG_CACHE_HOME/bash/bash_history"
export LESSHISTFILE="$XDG_CACHE_HOME/less/less_history"
export PYTHON_HISTORY="$XDG_DATA_HOME/python/python_history"

# Moving other files and some other vars
export CARGO_HOME="$XDG_DATA_HOME/cargo"
export RUSTUP_HOME="$XDG_DATA_HOME/rustup"
export GOPATH="$XDG_DATA_HOME/go"
export GOBIN="$GOPATH/bin"
export GOMODCACHE="$XDG_CACHE_HOME/go/mod"
export NPM_CONFIG_PREFIX="$XDG_DATA_HOME/npm"
export NPM_CONFIG_CACHE="$XDG_CACHE_HOME/npm"
export GNUPGHOME="$XDG_DATA_HOME/gnupg"
export DATE=$(date "+%-d %B %Y, %A, %T")
export MANPAGER="nvim +Man!"

export FZF_DEFAULT_OPTS="--style full --color 16 --layout=reverse --height 30% --preview='bat -p --color=always {}'"
export FZF_CTRL_R_OPTS="--style full --color 16 --layout=reverse --height 30% --no-preview"

# Important
export XDG_CURRENT_DESKTOP=sway
export XDG_SESSION_DESKTOP=sway
export XDG_SESSION_TYPE=wayland
export ELECTRON_OZONE_PLATFORM_HINT=wayland
export SDL_VIDEODRIVER=wayland
export QT_QPA_PLATFORM=wayland
export QT_QPA_PLATFORMTHEME=qt6ct
export QT_WAYLAND_DISABLE_WINDOWDECORATION=1
export QT_SCALE_FACTOR=1
export QT_AUTO_SCREEN_SCALE_FACTOR=1
export GDK_SCALE=1
export GDK_BACKEND="wayland,x11,*"
export GDK_USE_PORTAL=1
export WLR_NO_HARDWARE_CURSORS=1
