GOCOMMON     := github.com/caicloud/go-common
VERSION      ?= $(shell git describe --tags --always --dirty)
GITREMOTE    ?= $(shell git remote get-url origin)
GITCOMMIT    ?= $(shell git rev-parse HEAD)
GITTREESTATE ?= $(if $(shell git status --porcelain),dirty,clean)
BUILDDATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
DOCKER_LABELS ?= git-describe="$(shell date -u +v%Y%m%d)-$(shell git describe --tags --always --dirty)"
VERSION      ?= $(shell git describe --tags --always --dirty)
# Project output directory.
OUTPUT_DIR := ./bin
# Container registries.
REGISTRY ?= cargo.dev.caicloud.xyz/release

# Container registry for base images.
BASE_REGISTRY ?= cargo.caicloud.xyz/library
ROOT := github.com/caicloud/rudder
TARGETS := release-checker
export SHELLOPTS := errexit
export GOFLAGS ?= -mod=vendor -p=8 -race -count=1

container: build-linux
	@docker build -t $(REGISTRY)/release-checker:$(VERSION)                                        \
	    --label $(DOCKER_LABELS)                                                                   \
	    -f Dockerfile .;

build-linux:
	@docker run --rm -it                                                                           \
	  -v $(shell dirname $(shell dirname `pwd`)):/go/src/$(ROOT)                                   \
	  -w /go/src/$(ROOT)/upgrade/release-checker                                                   \
	  -e GOOS=linux                                                                                \
	  -e GOARCH=amd64                                                                              \
	  -e GOPATH=/go                                                                                \
	  -e SHELLOPTS=$(SHELLOPTS)                                                                    \
	  $(BASE_REGISTRY)/golang:1.10-security                                                        \
	    /bin/bash -c 'go build -v -o $(OUTPUT_DIR)/release-checker                                 \
						-ldflags "-s -w -X $(GOCOMMON)/version.version=$(VERSION)                  \
						  -X $(GOCOMMON)/version.gitRemote=$(GITREMOTE)                            \
						  -X $(GOCOMMON)/version.gitCommit=$(GITCOMMIT)                            \
						  -X $(GOCOMMON)/version.gitTreeState=$(GITTREESTATE)                      \
						  -X $(GOCOMMON)/version.buildDate=$(BUILDDATE)"                           \
						.'