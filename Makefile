GO_VERSION := $(shell grep '^go ' go.mod | cut -d' ' -f2)

.PHONY: print-go-version
print-go-version: ## Print Go version from go.mod
	@echo $(GO_VERSION)
