#   Copyright 2020 Docker Hub Tool authors

#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at

#       http://www.apache.org/licenses/LICENSE-2.0

#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.
include vars.mk

NULL:=/dev/null
PKG_NAME:=github.com/docker/hub-tool
STATIC_FLAGS:=CGO_ENABLED=0
GOLANGCI_LINT_CACHE?=$(CURDIR)/.cache/golangci-lint
GOCACHE?=$(CURDIR)/.cache/go-build
UNIX_PLATFORMS:=linux/amd64 linux/arm linux/arm64 darwin/amd64 darwin/arm64
TMPDIR_WIN_PKG := $(shell mktemp -d)

ifeq ($(COMMIT),)
    COMMIT:=$(shell git rev-parse HEAD 2> $(NULL))
endif
ifeq ($(TAG_NAME),)
    TAG_NAME:=$(shell git describe --tags --match "v[0-9]*" 2> $(NULL))
endif
ifneq ($(strip $(E2E_TEST_NAME)),)
    RUN_TEST=-run $(E2E_TEST_NAME)
endif

LDFLAGS:=-s -w \
    -X $(PKG_NAME)/internal.GitCommit=$(COMMIT) \
    -X $(PKG_NAME)/internal.Version=$(TAG_NAME)
GO_BUILD:=go build -trimpath -ldflags="$(LDFLAGS)"

.PHONY: all
all: build

.PHONY: build
build: ## Build the tool
	mkdir -p bin
	$(STATIC_FLAGS) $(GO_BUILD) -o bin/$(PLATFORM_BINARY) .
	cp bin/$(PLATFORM_BINARY) bin/$(BINARY)

.PHONY: mod-tidy
mod-tidy: ## Update go.mod and go.sum
	go mod tidy

.PHONY: generate-api
generate-api: ## Generate Docker Hub API types, requires providing a file
	go generate ./pkg/hub/api

.PHONY: cross
cross: ## Cross compile the tool binaries
	mkdir -p bin
	GOOS=linux   GOARCH=amd64 $(STATIC_FLAGS) $(GO_BUILD) -o bin/$(BINARY_NAME)_linux_amd64 .
	GOOS=linux   GOARCH=arm64 $(STATIC_FLAGS) $(GO_BUILD) -o bin/$(BINARY_NAME)_linux_arm64 .
	GOOS=linux   GOARCH=arm   $(STATIC_FLAGS) $(GO_BUILD) -o bin/$(BINARY_NAME)_linux_arm .
	GOOS=darwin  GOARCH=amd64 $(STATIC_FLAGS) $(GO_BUILD) -o bin/$(BINARY_NAME)_darwin_amd64 .
	GOOS=darwin  GOARCH=arm64 $(STATIC_FLAGS) $(GO_BUILD) -o bin/$(BINARY_NAME)_darwin_arm64 .
	GOOS=windows GOARCH=amd64 $(STATIC_FLAGS) $(GO_BUILD) -o bin/$(BINARY_NAME)_windows_amd64.exe .
	GOOS=windows GOARCH=arm64 $(STATIC_FLAGS) $(GO_BUILD) -o bin/$(BINARY_NAME)_windows_arm64.exe .

.PHONY: package-cross
package-cross: cross ## Package the cross compiled binaries in tarballs for *nix and a zip for Windows
	mkdir -p dist
	$(foreach plat,$(UNIX_PLATFORMS),mkdir -p $(TMPDIR_WIN_PKG)/$(BINARY_NAME) && \
		cp packaging/LICENSE $(TMPDIR_WIN_PKG)/$(BINARY_NAME)/LICENSE && \
		cp bin/$(BINARY_NAME)_$(subst /,_,$(plat)) $(TMPDIR_WIN_PKG)/$(BINARY_NAME)/$(BINARY_NAME) && \
		tar -C $(TMPDIR_WIN_PKG) -czf dist/$(BINARY_NAME)-$(subst /,-,$(plat)).tar.gz $(BINARY_NAME) && \
		rm -rf $(TMPDIR_WIN_PKG)/$(BINARY_NAME) ;)
	cp bin/$(BINARY_NAME)_windows_amd64.exe $(TMPDIR_WIN_PKG)/$(BINARY_NAME).exe
	rm -f dist/$(BINARY_NAME)-windows-amd64.zip && zip dist/$(BINARY_NAME)-windows-amd64.zip -j packaging/LICENSE $(TMPDIR_WIN_PKG)/$(BINARY_NAME).exe
	cp bin/$(BINARY_NAME)_windows_arm64.exe $(TMPDIR_WIN_PKG)/$(BINARY_NAME).exe
	rm -f dist/$(BINARY_NAME)-windows-arm64.zip && zip dist/$(BINARY_NAME)-windows-arm64.zip -j packaging/LICENSE $(TMPDIR_WIN_PKG)/$(BINARY_NAME).exe
	rm -r $(TMPDIR_WIN_PKG)

.PHONY: install
install: build ## Install the tool to your /usr/local/bin/
	cp bin/$(BINARY_NAME) /usr/local/bin/$(BINARY)

.PHONY: test
test: test-unit e2e

.PHONY: e2e
e2e: build ## Run the end-to-end tests
	BINARY=$(BINARY) go test ./e2e $(RUN_TEST) -ldflags="$(LDFLAGS)"

.PHONY: test-unit
test-unit: ## Run unit tests
	$(STATIC_FLAGS) go test $(shell go list ./... | grep -vE '/e2e')

.PHONY: lint
lint: ## Run the go linter
	mkdir -p $(GOLANGCI_LINT_CACHE) $(GOCACHE)
	GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) GOCACHE=$(GOCACHE) $(STATIC_FLAGS) golangci-lint run --timeout 10m ./...

.PHONY: validate-go-mod
validate-go-mod: ## Validate go.mod and go.sum are up-to-date
	@scripts/validate/check-go-mod

.PHONY: validate
validate: validate-go-mod lint test-unit ## Validate sources

.PHONY: help
help: ## Show help
	@echo Please specify a build target. The choices are:
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":| ## "}; {printf "\033[36m%-30s\033[0m %s\n", $$2, $$NF}'
