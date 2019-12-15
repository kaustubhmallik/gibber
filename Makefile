default: build test

all: bootstrap build test

bootstrap: githooks

clean:
	rm -rf build

build: clean
	mkdir -p build
	go build -o build/gibber-server  cmd/server/main.go

test:	
	go test ./...

.PHONY: .githooks
githooks:
	git config --local core.hooksPath .githooks/
