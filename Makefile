BIN_PATH=build/gibber-server
BUILD_PATH=cmd/server/main.go
MK_BUILD_PATH=test -d build || mkdir -p build
GO_CMD=go
GO_BUILD=$(GO_CMD) build -o $(BIN_PATH) $(BUILD_PATH)
GO_TEST=$(GO_CMD) test ./...
GO_TEST_COVER=$(GO_TEST) --cover cover.out
GIT_HOOKS=git config --local core.hooksPath .githooks/

default: build test test_cover

all: bootstrap default

bootstrap: githooks

build: 
	$(MK_BUILD_PATH)
	$(GO_BUILD)

test:	
	$(GO_TEST)

test_cover:
	$(GO_TEST_COVER)

.PHONY: .githooks
githooks:
	GIT_HOOKS
