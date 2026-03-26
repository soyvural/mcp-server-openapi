.PHONY: build test lint fmt clean

BINARY=mcp-server-openapi

build:
	go build -o bin/$(BINARY) ./cmd/mcp-server-openapi

test:
	go test -v -race -cover ./...

lint:
	golangci-lint run ./...

fmt:
	gofumpt -w .
	goimports -w .

clean:
	rm -rf bin/
