override GO_VERSION := $(shell grep '^go ' go.mod | cut -d' ' -f2)
override GOLANGCI_LINT_VERSION := v2.6.2

# Use the exact Go toolchain declared in go.mod for all Make commands.
override GOTOOLCHAIN := go$(GO_VERSION)
export GOTOOLCHAIN

LOCALBIN ?= $(shell pwd)/bin
override GOLANGCI_LINT := $(LOCALBIN)/golangci-lint

.PHONY: print-go-version
print-go-version:
	@echo $(GO_VERSION)

.PHONY: print-golangci-lint-version
print-golangci-lint-version:
	@echo $(GOLANGCI_LINT_VERSION)

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: golangci-lint
golangci-lint: $(LOCALBIN) ## Download the pinned golangci-lint version if necessary
	@if test "v$$($(GOLANGCI_LINT) version --short 2>/dev/null)" != "$(GOLANGCI_LINT_VERSION)"; then \
		curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(LOCALBIN) $(GOLANGCI_LINT_VERSION); \
	fi

.PHONY: build
build:
	rm --recursive --force dist/
	GOOS=linux GOARCH=amd64 go build -o dist/linux-x64
	GOOS=linux GOARCH=arm64 go build -o dist/linux-arm64
	GOOS=darwin GOARCH=amd64 go build -o dist/macos-x64
	GOOS=darwin GOARCH=arm64 go build -o dist/macos-arm64
	GOOS=windows GOARCH=amd64 go build -o dist/windows-x64

.PHONY: update-readme-version
update-readme-version:
ifndef VERSION
	$(error VERSION is required, e.g. make update-readme-version VERSION=v3)
endif
	sed --in-place --regexp-extended 's|action-s3-cache@v[0-9]+|action-s3-cache@$(VERSION)|g' README.md
