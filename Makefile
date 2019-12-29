BIN_PATH=build/gibber-server
BUILD_PATH=cmd/server/main.go
MK_BUILD_PATH=test -d build || mkdir -p build
LOAD_TEST_CONFIG=${PWD}/gibber_test.conf
LOAD_PROD_CONFIG=${PWD}/gibber_prod.conf
GO_CMD=go
GO_BUILD=$(GO_CMD) build -o $(BIN_PATH) $(BUILD_PATH)
GO_TEST=$(GO_CMD) test ./... -count=1
GO_TEST_COVER=$(GO_TEST) --cover cover.out
GIT_HOOKS=git config --local core.hooksPath .githooks/

default: build test test_cover

all: bootstrap default

bootstrap: githooks

build: 
	$(MK_BUILD_PATH)
	$(GO_BUILD)

test:	
	$(LOAD_TEST_CONFIG)
	$(GO_TEST)

test_cover:
	$(GO_TEST_COVER)

githooks:
	$(GIT_HOOKS)

.PHONY: .githooks all build test clean
