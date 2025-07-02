# ~~~~~~~~~~~~~~~ Sourcing global shell aliases ~~~~~~~~~~~~~~~


[ -f "$HOME/.config/zsh/aliases.zsh" ] && source "$HOME/.config/zsh/aliases.zsh"


# ~~~~~~~~~~~~~~~ Path configuration ~~~~~~~~~~~~~~~


setopt extended_glob null_glob

path=(
    $path                  # Keep existing PATH entries
    /bin
    /sbin
    /usr/sbin
    /usr/bin
    /usr/local/bin
    /usr/local/sbin
    $XDG_DATA_HOME/go
    $XDG_DATA_HOME/cargo
)

# Remove duplicate entries and non-existent directories
typeset -U path
path=($^path(N-/))

export PATH


# ~~~~~~~~~~~~~~~ History ~~~~~~~~~~~~~~~


HISTFILE="$XDG_CACHE_HOME/zsh/zsh_history" #Mmove .histfile to cache
HISTSIZE=1000000
SAVEHIST=1000000

setopt HIST_IGNORE_DUPS     # Don't save duplicate lines
setopt HIST_IGNORE_SPACE    # Don't save when prefixed with space
setopt SHARE_HISTORY        # Share history between sessions


# ~~~~~~~~~~~~~~~ Load modules ~~~~~~~~~~~~~~~


autoload -U compinit && compinit
autoload -U colors && colors
zmodload zsh/complist
source /usr/share/zsh/plugins/fzf-tab/fzf-tab.plugin.zsh
autoload -Uz vcs_info


# ~~~~~~~~~~~~~~~ Completion options ~~~~~~~~~~~~~~~


zstyle ':completion:*' menu no # tab opens cmp menu
zstyle ':completion:*' special-dirs false # force . and .. to show in cmp menu
zstyle ':completion:*' mather-list 'm{a-z}={A-Za-z}'
zstyle ':completion:*' list-colors ${(s.:.)LS_COLORS}
zstyle ':fzf-tab:complete:cd:*' fzf-preview 'eza -1 --color=always $realpath'
zstyle ':completion:*' file-list true # more detailed list
zstyle ':completion:*' squeeze-slashes false # explicit disable to allow /*/ expansion


# ~~~~~~~~~~~~~~~ Main options ~~~~~~~~~~~~~~~


setopt append_history inc_append_history share_history # better history
setopt auto_menu menu_complete # autocmp first menu match
setopt autocd # type a dir to cd
setopt auto_param_slash # when a dir is completed, add a / instead of a trailing space
setopt no_case_glob no_case_match # make cmp case insensitive
setopt globdots # include dotfiles
setopt extended_glob # match ~ # ^
setopt interactive_comments # allow comments in shell
unsetopt prompt_sp # don't autoclean blanklines
stty stop undef # disable accidental ctrl s


# ~~~~~~~~~~~~~~~ Binds ~~~~~~~~~~~~~~~


bindkey "^a" beginning-of-line
bindkey "^e" end-of-line
bindkey "^k" kill-line
bindkey "^j" backward-word
bindkey "^k" forward-word
bindkey "^H" backward-kill-word
# ctrl J & K for going up and down in prev commands
bindkey "^J" history-search-forward
bindkey "^K" history-search-backward
bindkey '^R' fzf-history-widget


# ~~~~~~~~~~~~~~~ Prompt ~~~~~~~~~~~~~~~


zstyle ':vcs_info:git:*' formats '%b'
setopt PROMPT_SUBST

typeset -g EMPTY_LINE_NEEDED=0

preexec() {
    if [[ "$1" != "clear" ]]; then
        EMPTY_LINE_NEEDED=1
    fi
}

precmd() {
    vcs_info
    if [[ $EMPTY_LINE_NEEDED -eq 1 ]]; then
        echo ""
        EMPTY_LINE_NEEDED=0
    fi
}

git_color() {
  local git_status="$(git status 2> /dev/null)"
  local star_color="%F{#FFC0CB}"
  local star=""

  if ! [[ $git_status =~ "nothing to commit, working tree clean" ]]; then
    star="${star_color}*%f"
  fi

  echo "%F{#808080}$1%f${star}"
}

PROMPT='%F{blue}%~%f ${vcs_info_msg_0_:+$(git_color ${vcs_info_msg_0_})}%F{#D8BFD8}
; %f'


# ~~~~~~~~~~~~~~~ Sourcing ~~~~~~~~~~~~~~~


source <(fzf --zsh)
source /usr/share/zsh/plugins/zsh-autosuggestions/zsh-autosuggestions.zsh
source /usr/share/zsh/plugins/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh


# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
