BINARY := tfp
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build test lint fmt vet tidy clean run

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/tfp

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

vet:
	go vet ./...

tidy:
	go mod tidy

run: build
	./bin/$(BINARY)

clean:
	rm -rf bin/ dist/
