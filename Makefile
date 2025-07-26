.PHONY: all build clean install

BINDIR := $(DESTDIR)/usr/local/bin
BUILD_DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ || echo "Warning: unable to get current date")
GIT_COMMIT_HASH=$(shell git rev-parse HEAD || echo "Warning: unable to get Git commit hash")
LDFLAGS=-X main.BuildDate="$(BUILD_DATE)" -X main.GitCommit="$(GIT_COMMIT_HASH)"

all: build

build:
	go build -o bin/bvm -ldflags "$(LDFLAGS) -w -s" -trimpath ./cmd/bvm

build-debug:
	go build -o bin/bvm -ldflags "$(LDFLAGS)" ./cmd/bvm

clean:
	rm -rf bin/ bvm

install: build
	install -m 755 bin/bvm bvm

install-debug: build-debug
	install -m 755 bin/bvm bvm

test:
	go test -v ./...

fmt:
	go fmt ./...

vet:
	go vet ./... 
