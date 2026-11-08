GO ?= go
BIN := bin
# Where `make dev` keeps its daemon, sessions, projects and hosts, apart
# from the installed app's ~/.hustlex-app.
DEV_HOME ?= $(HOME)/.hustlex-dev

.PHONY: build daemons dev test lint fmt run-app run-daemon clean package

build: daemons
	$(GO) build -o $(BIN)/ ./cmd/...

# Linux builds of hustlexd that the app installs on servers.
daemons:
	GO=$(GO) scripts/daemons.sh

test:
	$(GO) test ./...

# Every platform's installers, built here into dist/<version>, not published.
package:
	scripts/package.sh

# The app with hot reload: Go changes rebuild and relaunch it, and the
# relaunched app replaces the daemon an earlier build started. The Linux
# daemons are built first: the app embeds them to install on servers.
dev: daemons
	HUSTLEX_DEV=1 HUSTLEX_HOME=$(DEV_HOME) $(GO) tool mygo dev cmd/hustlex

lint:
	$(GO) vet ./...
	$(GO) run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./...

fmt:
	gofmt -w .

run-app:
	$(GO) run ./cmd/hustlex

run-daemon:
	$(GO) run ./cmd/hustlexd serve

clean:
	rm -rf $(BIN)
