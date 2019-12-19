default: build test

all: bootstrap build test

bootstrap: githooks

build: 
	mkdir -p build
	go build -o build/gibber-server  cmd/server/main.go

test:	
	go test ./...

.PHONY: .githooks
githooks:
	git config --local core.hooksPath .githooks/
