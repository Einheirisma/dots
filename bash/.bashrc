# ~/.bashrc

# If not running interactively, don't do anything
[[ $- != *i* ]] && return

# Prompt
PS1='\w > '

# Aliases
alias grep="grep --color=auto"
alias v="nvim"

alias cat="bat"
alias vid="mpv"
alias top="btop"
alias cp="cp -r"
alias img="swayimg"
alias rm="rm -rf --"
alias mkdir="mkdir -p"
alias reboot="sudo reboot"
alias shutdown="sudo shutdown -h now"

alias ls='eza --icons --color=always'
alias sl='eza --icons --color=always'
alias tree='eza --all --icons --sort=type -T'
alias ll='eza -laghm@ --all --icons --git --color=always'

# Git
alias ga="git add ."
alias gs="git status"
alias gcm="git commit -m $1"
alias gb="git branch -M master"
alias gp="git push -u origin master"

# Usage: ignores single-word commands
HISTIGNORE=$'*([\t ])+([-%+,./0-9\:@A-Z_a-z])*([\t ])'

# Functions
# Usage: '.. 3' takes you 3 folders up
..() { cd "$(eval printf '../%.0s' {1..$1})" || return 1; }

# Usage: extract file
extract () {
    if [ -f $1 ] ; then
        case $1 in
            *.tar.bz2)   tar xvjf $1                  ;;
            *.tar.gz)    tar xvzf $1                  ;;
            *.bz2)       bunzip2 $1                   ;;
            *.rar)       unrar x $1                   ;;
            *.gz)        gunzip $1                    ;;
            *.tar)       tar xvf $1                   ;;
            *.tbz2)      tar xvjf $1                  ;;
            *.tgz)       tar xvzf $1                  ;;
            *.zip)       unzip $1                     ;;
            *.Z)         uncompress $1                ;;
            *.7z)        7z x $1                      ;;
            *)           echo "can't extract '$1'..." ;;
        esac
    else
        echo "'$1' is not a valid file!"
    fi
}
