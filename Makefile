SHELL := /bin/bash
OS := $(shell uname)

GO_VARS := GO111MODULE=on GO15VENDOREXPERIMENT=1 CGO_ENABLED=0
BUILDFLAGS := ''

APP_NAME := jx-app-jacoco
MAIN := cmd/jacoco/main.go

BUILD_DIR=build
PACKAGE_DIRS := $(shell go list ./...)
PKGS := $(subst  :,_,$(PACKAGE_DIRS))
PLATFORMS := windows linux darwin
os = $(word 1, $@)

VERSION ?= $(shell cat VERSION)

# setting some defaults for skaffold
DOCKER_REGISTRY ?= $(shell kubectl get service jenkins-x-docker-registry -o go-template --template='{{index .metadata.annotations "fabric8.io/exposeUrl"}}' |  sed 's/http:\/\///')
JENKINS_X_DOCKER_REGISTRY_INTERNAL ?= $(shell kubectl get service jenkins-x-docker-registry -o go-template --template="{{.spec.clusterIP}}":5000)

FGT := $(GOPATH)/bin/fgt
GOLINT := $(GOPATH)/bin/golint

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
	JENKINS_X_DOCKER_REGISTRY_INTERNAL=$(JENKINS_X_DOCKER_REGISTRY_INTERNAL) DOCKER_REGISTRY=$(DOCKER_REGISTRY) VERSION=$(VERSION) skaffold build -f skaffold.yaml

.PHONY: skaffold-run
skaffold-run: linux ## Runs 'skaffold run'
	JENKINS_X_DOCKER_REGISTRY_INTERNAL=$(JENKINS_X_DOCKER_REGISTRY_INTERNAL) DOCKER_REGISTRY=$(DOCKER_REGISTRY) VERSION=$(VERSION) skaffold run -f skaffold.yaml -p dev

.PHONY: help
help: ## Prints this help
	@grep -E '^[^.]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-40s\033[0m %s\n", $$1, $$2}'	

.PHONY: update-release-version
update-release-version: ## Updates the release version
ifeq ($(OS),Darwin)
	sed -i "" -e "s/version:.*/version: $(VERSION)/" ./charts/jx-app-jacoco/Chart.yaml
	sed -i "" -e "s/tag: .*/tag: $(VERSION)/" ./charts/jx-app-jacoco/values.yaml
else ifeq ($(OS),Linux)
	sed -i -e "s/version:.*/version: $(VERSION)/" ./charts/jx-app-jacoco/Chart.yaml
	sed -i -e "s/tag: .*/tag: $(VERSION)/" ./charts/jx-app-jacoco/values.yaml
else
	echo "platform $(OS) not supported to tag with"
	exit -1
endif

.PHONY: release-branch
release-branch: update-release-version ## Creates release branch and pushes release
	git checkout -b release-v$(VERSION)
	git add --all
	git commit -m "release $(VERSION)" --allow-empty # if first release then no version update is performed
	git tag -fa v$(VERSION) -m "Release version $(VERSION)"
	git push origin HEAD v$(VERSION)

# Targets to get some Go tools
$(FGT):
	@$(GO_VARS) go get github.com/GeertJohan/fgt

$(GOLINT):
	@$(GO_VARS) go get github.com/golang/lint/golint
