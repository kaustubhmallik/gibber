all: build

default: build

clean:
	rm -rf build

build: clean
	mkdir -p build
	go build -o build/gibber-server  cmd/server/main.go
