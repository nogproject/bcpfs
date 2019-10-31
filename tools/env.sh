# Source the file in bash.

alias dc="docker-compose"
alias ddev="docker-compose run --rm godev"
alias dfake="docker-compose run --rm fake"

export COMPOSE_PROJECT_NAME='bcpfs'

if [ -f '.git' ]; then
    cat <<\EOF
******************************************************************************
Warning: `.git` is a file.  Move the git dir, so that Git commands work in the
container:

    cat .git \
    && gitdir="$(git rev-parse --git-dir)" \
    && git config --unset core.worktree \
    && rm .git \
    && mv "${gitdir}" .git \
    && git status

EOF
fi
