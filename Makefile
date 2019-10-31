# vim: sw=8

define USAGE
Usage:
  make
  make binaries
  make static  # DEPRECATED
  make deb
  make clean
  make vendor  # DEPRECATED
  make vendor-status  # DEPRECATED
  make vendor-upgrade  # DEPRECATED
  make vet
  make test
  make errcheck
  make images
  make gc
  make down
  make down-state
  make up-godoc

`make` runs `go install`.  `make clean` removes files that `make` creates.

`make binaries` builds Linux binaries in `product/bin`.

`make static` (DEPRECATED) is an alias for `make binaries`.

`make deb` builds Linux binaries and packs them into deb files.

`make vendor` is DEPRECATED.  Use Go module commands instead, like:

    ddev go list -m all
    ddev go mod tidy
    ddev go get ...
    ddev go get -u=patch ./...  # upgrade all to latest patch
    ddev go get -u ./...        # upgrade all to latest minor

`make vet`, `make test`, and `make errcheck` run the respective Go tools.

`make images` builds Docker images explicitly.

`make gc` removes exited containers.  `make down` removes Docker objects that
can be re-created.  `make down-state` also removes volumes that contain state
that cannot be re-created.

`make up-godoc` starts the godoc server and on Mac opens the project doc in
Chrome.

See CONTRIBUTING.md for details.

endef

IS_CONTAINER := $(shell test -d /go && echo isContainer)
IS_GIT_FILE := $(shell test -f .git && echo isGitFile)

ifdef IS_CONTAINER
    $(error "This Makefile must be used outside the dev container.")
endif

ifdef IS_GIT_FILE
    $(error "`.git` is a file.  It must be the git dir.  See `tools/env.sh`.")
endif

# The build tag encodes information about the Git commit and the build time.
# It is determined outside the container, so that the same Git version is used
# that is also used for managing the worktree.
GIT_COMMIT_TAG := $(shell \
    TZ=UTC git show -s \
	 --date=format-local:%Y%m%dT%H%M%SZ --abbrev=6 --pretty=%cd-g%h \
)
GIT_DIRTY := $(shell \
    if [ -n "$$(git status -s)" ]; then echo "-dirty"; fi \
)
BUILD_DATE := $(shell \
    date -u +%s \
)
BUILD_TAG := $(GIT_COMMIT_TAG)-b$(BUILD_DATE)$(GIT_DIRTY)

OS := $(shell uname)
DC := docker-compose
DDEV := $(DC) run --rm godev
DMAKE := $(DDEV) make -f Makefile.docker BUILD_TAG=$(BUILD_TAG)


.PHONY: all
all: install

.PHONY: help
export USAGE
help:
	@echo "$${USAGE}"

.PHONY: install
install:
	@echo '    DMAKE install'
	$(DMAKE) install

.PHONY: binaries
binaries:
	@echo '    DMAKE binaries'
	$(DMAKE) binaries

.PHONY: deb
deb:
	@echo '    DMAKE deb'
	$(DMAKE) deb

.PHONY: static
static:
	@echo '    DMAKE static'
	@echo 'WARNING: Target `static` is deprecated.  Use `binaries` instead.'
	$(DMAKE) binaries

.PHONY: clean
clean:
	@echo '    DMAKE clean'
	$(DMAKE) clean
	$(MAKE) gc
	docker volume rm -f bcpfs_go

.PHONY: vendor
vendor:
	@echo '    DEPRECATED vendor'
	@echo '`vendor` is no longer used, since Go modules have been enabled.'

.PHONY: vendor-status
vendor-status:
	@echo '    DEPRECATED vendor-status'
	@echo '`vendor-status` is no longer used, since Go modules have been enabled.'
	@echo 'Use Go module commands instead, like:'
	@echo
	@echo '    ddev go list -m all'
	@echo

.PHONY: vendor-upgrade
vendor-upgrade:
	@echo '    DEPRECATED vendor-upgrade'
	@echo '`vendor-upgrade` is no longer used, since Go modules have been enabled.'
	@echo 'Use Go module commands instead, like:'
	@echo
	@echo '    ddev go mod tidy'
	@echo '    ddev go get ...'
	@echo

.PHONY: vet
vet:
	@echo '    DMAKE vet'
	$(DMAKE) vet

.PHONY: test
test:
	@echo '    DMAKE test'
	$(DMAKE) test

.PHONY: errcheck
errcheck:
	@echo '    DMAKE errcheck'
	$(DMAKE) errcheck

.PHONY: images
images:
	@echo '    DOCKER BUILD'
	$(DC) build

.PHONY: gc
gc:
	@echo '    DOCKER RM exited containers'
	$(DC) rm -f

.PHONY: down
down:
	@echo '    DOCKER COMPOSE down'
	$(DC) down

.PHONY: down-state
down-state:
	@echo '    DOCKER COMPOSE down stateful'
	$(DC) down --volumes

.PHONY: up-godoc
up-godoc:
	@echo '    DOCKER COMPOSE up godoc http://localhost:6060'
	$(DC) up -d godoc
ifeq ($(OS),Darwin)
	@sleep 1
	open -b com.google.Chrome http://localhost:6060/pkg/github.com/nogproject/?m=all
else
	@echo
	@echo open http://localhost:6060/pkg/github.com/nogproject/?m=all
endif
