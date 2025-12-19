GO_VERSION := $(shell grep '^go ' go.mod | cut -d' ' -f2)

.PHONY: print-go-version
print-go-version:
	@echo $(GO_VERSION)

.PHONY: build
build:
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
