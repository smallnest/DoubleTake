BINARY := doubletake
GO := go
BUILD_DIR := ./cmd/doubletake/

.PHONY: build test clean lint snapshot release

build:
	$(GO) build -o $(BINARY) $(BUILD_DIR)

test:
	$(GO) test -race ./...

cover:
	$(GO) test -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY) coverage.out coverage.html
	rm -rf dist/

snapshot:
	goreleaser build --snapshot --clean

release:
	goreleaser release --clean
