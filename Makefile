GO_BIN=manage-sw
GO_FILES=$(shell find . -type f -name '*.go' -not -path "./vendor/*")
godeps=$(shell 2>/dev/null go list -mod vendor -deps -f '{{if not .Standard}}{{ $dep := . }}{{range .GoFiles}}{{$dep.Dir}}/{{.}} {{end}}{{end}}' $(1) | sed "s%$(shell pwd)/%%g")

# Install parameters
PREFIX ?= /usr
DESTDIR ?=

.PHONY: all build vendor test format lint clean dist install

all: build

build: $(godeps)
	go build -o $(GO_BIN) -mod=vendor .

vendor:
	go mod tidy
	go mod vendor

test:
	go test ./...

format:
	go fmt $(GO_FILES)

lint:
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint is not installed. Please install it: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi
	golangci-lint run ./...

clean:
	rm -f $(GO_BIN)
	go clean -modcache

dist: build vendor
	tar -czvf $(GO_BIN).tar.gz $(GO_BIN) vendor

install: build
	install -d -m 0755 "$(DESTDIR)$(PREFIX)/bin"
	install -m 0755 "$(GO_BIN)" "$(DESTDIR)$(PREFIX)/bin/$(GO_BIN)"

