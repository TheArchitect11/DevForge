BINARY_NAME=devforge
VERSION?=dev
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build build-all checksums clean test lint vet tidy run

## build: Compile the devforge binary for the current platform
build:
	go build $(LDFLAGS) -o $(BINARY_NAME)

## build-all: Cross-compile for macOS (amd64/arm64), Linux (amd64/arm64), and Windows (amd64)
build-all:
	mkdir -p dist
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe

## checksums: Generate sha256 checksums for all dist/ binaries
checksums:
	cd dist && sha256sum * > checksums.txt

## clean: Remove build artifacts
clean:
	rm -rf dist $(BINARY_NAME)

## test: Run all tests with race detection
test:
	go test -race ./...

## vet: Run go vet on all packages
vet:
	go vet ./...

## tidy: Tidy and verify go.mod / go.sum
tidy:
	go mod tidy
	go mod verify

## lint: Run golangci-lint (requires golangci-lint to be installed)
lint:
	golangci-lint run ./...

## run: Build and run devforge with any extra args (e.g. make run ARGS="version")
run: build
	./$(BINARY_NAME) $(ARGS)

help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
