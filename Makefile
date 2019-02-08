SHELL := /bin/bash

GO_VARS := GO111MODULE=on GO15VENDOREXPERIMENT=1 CGO_ENABLED=0
BUILDFLAGS := ''

APP_NAME := jx-app-jacoco
MAIN := cmd/jacoco/main.go

BUILD_DIR=build
PACKAGE_DIRS := $(shell go list ./...)
PKGS := $(subst  :,_,$(PACKAGE_DIRS))
PLATFORMS := windows linux darwin
os = $(word 1, $@)

# setting some defaults for skaffold 
DOCKER_REGISTRY ?= localhost:5000
VERSION ?= latest

.PHONY : all
all: linux test check ## Compiles, test and verifies source

.PHONY: $(PLATFORMS)
$(PLATFORMS):	
	$(GO_VARS) GOOS=$(os) GOARCH=amd64 go build -ldflags $(BUILDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN)

.PHONY : test
test: ## Runs unit tests
	$(GO_VARS) go test -v $(PACKAGE_DIRS) 

.PHONY : fmt
fmt: ## Re-formates Go source files according to standard
	@$(GO_VARS) go fmt $(PACKAGE_DIRS)

.PHONY : clean
clean: ## Deletes the build directory with all generated artefacts
	rm -rf $(BUILD_DIR)

check: $(GOLINT) $(FGT)
	@echo "LINTING"
	@$(FGT) $(GOLINT) $(PACKAGE_DIRS)
	@echo "VETTING"
	@$(GO_VARS) $(FGT) go vet $(PACKAGE_DIRS)

.PHONY: watch
watch: ## Watches for file changes in Go source files and re-runs 'skaffold build'. Requires entr
	find . -name "*.go" | entr -s 'make skaffold-build' 

.PHONY: skaffold-build
skaffold-build: linux ## Runs 'skaffold build'
	DOCKER_REGISTRY=$(DOCKER_REGISTRY) VERSION=$(VERSION) skaffold build -f skaffold.yaml

.PHONY: skaffold-run
skaffold-run: linux ## Runs 'skaffold run'
	DOCKER_REGISTRY=$(DOCKER_REGISTRY) VERSION=$(VERSION) skaffold run -f skaffold.yaml -p dev 

.PHONY: help
help: ## Prints this help
	@grep -E '^[^.]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-40s\033[0m %s\n", $$1, $$2}'	

# Targets to get some Go tools
FGT := $(GOPATH)/bin/fgt
$(FGT):
	go get github.com/GeertJohan/fgt

GOLINT := $(GOPATH)/bin/golint
$(GOLINT):
	go get github.com/golang/lint/golint	
