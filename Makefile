.PHONY: all build clean test lint run example linux-amd64 linux-aarch64 windows-amd64 windows-aarch64 darwin-amd64 darwin-aarch64

all: linux-amd64 linux-aarch64 windows-amd64 windows-aarch64 darwin-amd64 darwin-aarch64

build:
	go build -o bin/pikpakapi ./cmd/example

linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o bin/pikpakapi-linux-amd64 ./cmd/example

linux-aarch64:
	GOOS=linux GOARCH=arm64 go build -o bin/pikpakapi-linux-aarch64 ./cmd/example

windows-amd64:
	GOOS=windows GOARCH=amd64 go build -o bin/pikpakapi-windows-amd64.exe ./cmd/example

windows-aarch64:
	GOOS=windows GOARCH=arm64 go build -o bin/pikpakapi-windows-aarch64.exe ./cmd/example

darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -o bin/pikpakapi-darwin-amd64 ./cmd/example

darwin-aarch64:
	GOOS=darwin GOARCH=arm64 go build -o bin/pikpakapi-darwin-aarch64 ./cmd/example

clean:
	rm -rf bin/

test:
	go test -v ./...

lint:
	golangci-lint run ./...

run: build
	./bin/pikpakapi

example:
	go run ./cmd/example/
