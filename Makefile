PREFIX  ?= $(HOME)/.local
BINDIR  ?= $(PREFIX)/bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  = -X github.com/elee1766/poundai/pkg/cli.Version=$(VERSION)

.PHONY: build test install uninstall clean

build:
	go build -ldflags '$(LDFLAGS)' -o poundai ./cmd/poundai

test:
	go test ./...
	bash tests/bash_plugin_test.bash

install: build
	install -d $(BINDIR)
	install -m 755 poundai $(BINDIR)/poundai

uninstall:
	rm -f $(BINDIR)/poundai

clean:
	rm -f poundai
