#!make

# run from the repo root

BIN_PATH=build/gibber-server
BUILD_PATH=cmd/server/main.go
MK_BUILD_PATH=test -d build || mkdir -p build
GO_CMD=go
GO_BUILD=CGO_ENABLED=0 $(GO_CMD) build -ldflags '-s -w' -o $(BIN_PATH) $(BUILD_PATH)
GO_TEST=$(GO_CMD) test ./... -count=1
GO_TEST_COVER=$(GO_TEST) -coverprofile=coverage.txt
GIT_HOOKS=git config --local core.hooksPath .githooks/

include gibber.env
export $(shell sed 's/=.*//' gibber.env)

default: fmt build test test_cover

all: bootstrap default

fmt:
	gofmt -l -s -w service user cmd datastore

bootstrap: githooks

build: 
	$(mk_build_path)
	$(GO_BUILD)

test: clean
	$(GO_TEST)

test_cover: clean
	$(GO_TEST_COVER)

githooks:
	$(GIT_HOOKS)

clean:
	rm -rf generated

.PHONY: .githooks all build test clean .env
