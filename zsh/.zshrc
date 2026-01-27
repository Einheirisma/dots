#
# /home/$USER/.config/zsh/.zshrc
#

# Source files
[ -f "$XDG_CONFIG_HOME/shell/alias" ] && source "$XDG_CONFIG_HOME/shell/alias"
[ -f "$XDG_CONFIG_HOME/shell/fn" ] && source "$XDG_CONFIG_HOME/shell/fn"

# Load modules
zmodload zsh/complist
autoload -Uz colors && colors
autoload -Uz compinit && compinit -d "${XDG_CACHE_HOME:-$HOME/.cache}/zsh/zcompdump"
autoload -Uz up-line-or-beginning-search down-line-or-beginning-search
autoload -Uz add-zsh-hook

# Binds
bindkey -e

# Zle widgets
zle -N up-line-or-beginning-search
zle -N down-line-or-beginning-search
bindkey '^[[A' up-line-or-beginning-search
bindkey '^[[B' down-line-or-beginning-search

# Hook-functions
function rehash_on_cd() {
	rehash
}
add-zsh-hook chpwd rehash_on_cd

# Completion options
zstyle ':completion:*' completer _extensions _complete _approximate
zstyle ':completion:*' menu no
zstyle ':completion:*' matcher-list '' 'm:{a-zA-Z}={A-Za-z}' 'r:|[._-]=* r:|=*'
zstyle ':completion:*' use-cache on
zstyle ':completion:*' cache-path "${XDG_CACHE_HOME:-$HOME/.cache}/zsh/zcompcache"
zstyle ':completion:*' group-name ''
#zstyle ':completion:*:*:-command-:*:*' group-order alias builtins functions commands
zstyle ':completion:*' file-sort modification reverse
zstyle ':completion:*' ignore-parents parent pwd
zstyle ':completion:*' squeeze-slashes true
zstyle ':completion:*' verbose yes
#zstyle ':completion:*' filename-completion

# Formatting
zstyle ':completion:*:warnings' format ' %F{red}-- no matches found --%f'

# Fzf-tab
zstyle ':completion:*:git-checkout:*' sort false
zstyle ':fzf-tab:complete:cd:*' fzf-preview 'eza -1 --color=always $realpath'
zstyle ':fzf-tab:*' switch-group '<' '>'

# Main options
setopt GLOBDOTS
setopt EXTENDED_GLOB
setopt INTERACTIVE_COMMENTS
setopt CDABLE_VARS
setopt PUSHD_IGNORE_DUPS
unsetopt BEEP LIST_BEEP

# History
setopt HIST_IGNORE_ALL_DUPS
setopt HIST_IGNORE_SPACE
setopt HIST_SAVE_NO_DUPS
setopt HIST_FIND_NO_DUPS
setopt HIST_EXPIRE_DUPS_FIRST
setopt SHARE_HISTORY
setopt INC_APPEND_HISTORY
setopt APPEND_HISTORY

# History options
HISTSIZE=1000000
SAVEHIST=1000000

# Prompt
PROMPT='%F{cyan}%~%f $ '

# FZF setup
if command -v fzf &>/dev/null; then
	source <(fzf --zsh)
fi

# Plugins
plugins=(
	/usr/share/zsh/plugins/fzf-tab-git/fzf-tab.plugin.zsh
	/usr/share/zsh/plugins/fast-syntax-highlighting/fast-syntax-highlighting.plugin.zsh
)

for plugin in "${plugins[@]}"; do
	[ -f "$plugin" ] && source "$plugin"
done
