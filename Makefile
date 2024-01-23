.DEFAULT_GOAL := build

##########
# CONFIG #
##########

ORG                                ?= kyverno
PACKAGE                            ?= github.com/$(ORG)/policy-reports
GIT_SHA                            := $(shell git rev-parse HEAD)
GOOS                               ?= $(shell go env GOOS)
GOARCH                             ?= $(shell go env GOARCH)
REGISTRY                           ?= ghcr.io
REPO                               ?= policy-reports
BUILD_DATE                         := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

#########
# TOOLS #
#########

TOOLS_DIR                          := $(PWD)/.tools
KIND                               := $(TOOLS_DIR)/kind
KIND_VERSION                       := v0.20.0
KO                                 := $(TOOLS_DIR)/ko
KO_VERSION                         := v0.14.1
HELM                               := $(TOOLS_DIR)/helm
HELM_VERSION                       := v3.10.1
TOOLS                              := $(KIND) $(KO) $(HELM)

$(KIND):
	@echo Install kind... >&2
	@GOBIN=$(TOOLS_DIR) go install sigs.k8s.io/kind@$(KIND_VERSION)

$(KO):
	@echo Install ko... >&2
	@GOBIN=$(TOOLS_DIR) go install github.com/google/ko@$(KO_VERSION)

$(HELM):
	@echo Install helm... >&2
	@GOBIN=$(TOOLS_DIR) go install helm.sh/helm/v3/cmd/helm@$(HELM_VERSION)

.PHONY: install-tools
install-tools: $(TOOLS) ## Install tools

.PHONY: clean-tools
clean-tools: ## Remove installed tools
	@echo Clean tools... >&2
	@rm -rf $(TOOLS_DIR)

#########
# BUILD #
#########

CGO_ENABLED     ?= 0
CLIENT_GO_PKG   := k8s.io/client-go/pkg
VERSION_LDFLAGS := -X $(CLIENT_GO_PKG)/version.gitVersion=$(GIT_TAG) -X $(CLIENT_GO_PKG)/version.gitCommit=$(GIT_SHA) -X $(CLIENT_GO_PKG)/version.buildDate=$(BUILD_DATE)
LD_FLAGS        := -s -w $(VERSION_LDFLAGS)
LOCAL_PLATFORM  := linux/$(GOARCH)
KO_REGISTRY     := ko.local
KO_TAGS         := $(GIT_SHA)
KO_CACHE        ?= /tmp/ko-cache
BIN             := policy-reports

.PHONY: fmt
fmt: ## Run go fmt
	@echo Go fmt... >&2
	@go fmt ./...

.PHONY: vet
vet: ## Run go vet
	@echo Go vet... >&2
	@go vet ./...

$(BIN): fmt vet
	@echo Build cli binary... >&2
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) go build -o ./$(BIN) -ldflags="$(LD_FLAGS)" .

.PHONY: build
build: $(BIN) ## Build

.PHONY: ko-build
ko-build: $(KO) ## Build image (with ko)
	@echo Build image with ko... >&2
	@LDFLAGS="$(LD_FLAGS)" KOCACHE=$(KO_CACHE) KO_DOCKER_REPO=$(KO_REGISTRY) \
		$(KO) build . --preserve-import-paths --tags=$(KO_TAGS) --platform=$(LOCAL_PLATFORM)

#########
# TEMPO #
#########

.PHONY: verify
verify: verify-licenses
# verify: verify-lint
# verify: verify-toc
# verify: verify-deps
# verify: verify-scripts-deps
# verify: verify-generated
# verify: verify-structured-logging

.PHONY: update
update: update-licenses
# update: update-lint
# update: update-toc
# update: update-deps
# update: update-generated

.PHONY: verify-structured-logging
verify-structured-logging: logcheck
	$(TOOLS_DIR)/logcheck ./... || (echo 'Fix structured logging' && exit 1)

HAS_LOGCHECK := $(shell command -v logcheck)

.PHONY: logcheck
logcheck:
ifndef HAS_LOGCHECK
	@GOBIN=$(TOOLS_DIR) go install -mod=readonly -modfile=scripts/go.mod sigs.k8s.io/logtools/logcheck
endif

.PHONY: update-deps
update-deps:
	go mod tidy
	cd scripts && go mod tidy

.PHONY: verify-deps
verify-deps:
	go mod verify
	go mod tidy
	@git diff --exit-code -- go.mod go.sum

.PHONY: verify-scripts-deps
verify-scripts-deps:
	make -C scripts -f ../Makefile verify-deps

############
# LICENSES #
############

HAS_ADDLICENSE := $(shell command -v addlicense)

.PHONY: verify-licenses
verify-licenses: addlicense
	find -type f -name "*.go" ! -path "*/vendor/*" | xargs $(GOPATH)/bin/addlicense -check || (echo 'Run "make update"' && exit 1)

.PHONY: update-licenses
update-licenses: addlicense
	find -type f -name "*.go" ! -path "*/vendor/*" | xargs $(GOPATH)/bin/addlicense -c "The Kubernetes Authors."

.PHONY: addlicense
addlicense:
ifndef HAS_ADDLICENSE
	go install -mod=readonly -modfile=scripts/go.mod github.com/google/addlicense
endif

###########
# CODEGEN #
###########

REPO_DIR        := $(shell pwd)
generated_files  = pkg/api/generated/openapi/zz_generated.openapi.go

.PHONY: verify-generated
verify-generated: update-generated
	@git diff --exit-code -- $(generated_files)

.PHONY: update-generated
update-generated:
	# pkg/api/generated/openapi/zz_generated.openapi.go
	@GOBIN=$(TOOLS_DIR) go install -mod=readonly -modfile=scripts/go.mod k8s.io/kube-openapi/cmd/openapi-gen
	$(TOOLS_DIR)/openapi-gen \
		-i sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2,k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/version,k8s.io/api/core/v1.ObjectReference \
		-p pkg/api/generated/openapi \
		-O zz_generated.openapi \
		-o $(REPO_DIR) \
		-h $(REPO_DIR)/scripts/boilerplate.go.txt \
		-r /dev/null

.PHONY: codegen-helm-docs
codegen-helm-docs: ## Generate helm docs
	@echo Generate helm docs... >&2
	@docker run -v ${PWD}/charts:/work -w /work jnorwood/helm-docs:v1.11.0 -s file

.PHONY: codegen
codegen: ## Rebuild all generated code and docs
codegen: codegen-helm-docs

.PHONY: verify-codegen
verify-codegen: codegen ## Verify all generated code and docs are up to date
	@echo Checking codegen is up to date... >&2
	@git --no-pager diff -- .
	@echo 'If this test fails, it is because the git diff is non-empty after running "make codegen".' >&2
	@echo 'To correct this, locally run "make codegen", commit the changes, and re-run tests.' >&2
	@git diff --quiet --exit-code -- .

########
# KIND #
########

KIND_IMAGE     ?= kindest/node:v1.28.0
KIND_NAME      ?= kind

.PHONY: kind-create
kind-create: $(KIND) ## Create kind cluster
	@echo Create kind cluster... >&2
	@$(KIND) create cluster --name $(KIND_NAME) --image $(KIND_IMAGE) --wait 1m

.PHONY: kind-delete
kind-delete: $(KIND) ## Delete kind cluster
	@echo Delete kind cluster... >&2
	@$(KIND) delete cluster --name $(KIND_NAME)

.PHONY: kind-load
kind-load: $(KIND) ko-build ## Build image and load in kind cluster
	@echo Load image... >&2
	@$(KIND) load docker-image --name $(KIND_NAME) $(KO_REGISTRY)/$(PACKAGE):$(GIT_SHA)

.PHONY: kind-install
kind-install: $(HELM) kind-load ## Build image, load it in kind cluster and deploy helm chart
	@echo Install chart... >&2
	@$(HELM) upgrade --install policy-reports --namespace policy-reports --create-namespace --wait ./charts/policy-reports \
		--set image.registry=$(KO_REGISTRY) \
		--set image.repository=$(PACKAGE) \
		--set image.tag=$(GIT_SHA)

########
# HELP #
########

.PHONY: help
help: ## Shows the available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-40s\033[0m %s\n", $$1, $$2}'
