PREFIX  ?= $(HOME)/.local
BINDIR  ?= $(PREFIX)/bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  = -X github.com/elee1766/zsh_poundai/pkg/cli.Version=$(VERSION)

.PHONY: build test install uninstall clean

build:
	go build -ldflags '$(LDFLAGS)' -o zsh_poundai ./cmd/zsh_poundai

test:
	go test ./...

install: build
	install -d $(BINDIR)
	install -m 755 zsh_poundai $(BINDIR)/zsh_poundai

uninstall:
	rm -f $(BINDIR)/zsh_poundai

clean:
	rm -f zsh_poundai
