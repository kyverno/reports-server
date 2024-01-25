.DEFAULT_GOAL := build

##########
# CONFIG #
##########

ORG                                ?= kyverno
PACKAGE                            ?= github.com/$(ORG)/reports-server
GIT_SHA                            := $(shell git rev-parse HEAD)
GOOS                               ?= $(shell go env GOOS)
GOARCH                             ?= $(shell go env GOARCH)
REGISTRY                           ?= ghcr.io
REPO                               ?= reports-server

#########
# TOOLS #
#########

TOOLS_DIR                          := $(PWD)/.tools
REGISTER_GEN                       := $(TOOLS_DIR)/register-gen
OPENAPI_GEN                        := $(TOOLS_DIR)/openapi-gen
CODE_GEN_VERSION                   := v0.28.0
KIND                               := $(TOOLS_DIR)/kind
KIND_VERSION                       := v0.20.0
KO                                 := $(TOOLS_DIR)/ko
KO_VERSION                         := v0.14.1
HELM                               := $(TOOLS_DIR)/helm
HELM_VERSION                       := v3.10.1
TOOLS                              := $(REGISTER_GEN) $(OPENAPI_GEN) $(KIND) $(KO) $(HELM)

$(REGISTER_GEN):
	@echo Install register-gen... >&2
	@GOBIN=$(TOOLS_DIR) go install k8s.io/code-generator/cmd/register-gen@$(CODE_GEN_VERSION)

$(OPENAPI_GEN):
	@echo Install openapi-gen... >&2
	@GOBIN=$(TOOLS_DIR) go install k8s.io/code-generator/cmd/openapi-gen@$(CODE_GEN_VERSION)

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

CGO_ENABLED    ?= 0
LD_FLAGS       := "-s -w"
LOCAL_PLATFORM := linux/$(GOARCH)
KO_REGISTRY    := ko.local
KO_TAGS        := $(GIT_SHA)
KO_CACHE       ?= /tmp/ko-cache
BIN            := reports-server

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
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) go build -o ./$(BIN) -ldflags=$(LD_FLAGS) .

.PHONY: build
build: $(BIN) ## Build

.PHONY: ko-build
ko-build: $(KO) ## Build image (with ko)
	@echo Build image with ko... >&2
	@LDFLAGS=$(LD_FLAGS) KOCACHE=$(KO_CACHE) KO_DOCKER_REPO=$(KO_REGISTRY) \
		$(KO) build . --preserve-import-paths --tags=$(KO_TAGS) --platform=$(LOCAL_PLATFORM)

###########
# CODEGEN #
###########

GOPATH_SHIM     := ${PWD}/.gopath
PACKAGE_SHIM    := $(GOPATH_SHIM)/src/$(PACKAGE)

$(GOPATH_SHIM):
	@echo Create gopath shim... >&2
	@mkdir -p $(GOPATH_SHIM)

.INTERMEDIATE: $(PACKAGE_SHIM)
$(PACKAGE_SHIM): $(GOPATH_SHIM)
	@echo Create package shim... >&2
	@mkdir -p $(GOPATH_SHIM)/src/github.com/$(ORG) && ln -s -f ${PWD} $(PACKAGE_SHIM)

.PHONY: codegen-openapi
codegen-openapi: $(PACKAGE_SHIM) $(OPENAPI_GEN) ## Generate openapi
	@echo Generate openapi... >&2
	@$(OPENAPI_GEN) \
		-i k8s.io/apimachinery/pkg/api/resource \
		-i k8s.io/apimachinery/pkg/apis/meta/v1 \
		-i k8s.io/apimachinery/pkg/version \
		-i k8s.io/apimachinery/pkg/runtime \
		-i k8s.io/apimachinery/pkg/types \
		-i k8s.io/api/core/v1 \
		-i sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2 \
		-p ./pkg/api/generated/openapi \
		-O zz_generated.openapi \
		-h ./.hack/boilerplate.go.txt

.PHONY: codegen-helm-docs
codegen-helm-docs: ## Generate helm docs
	@echo Generate helm docs... >&2
	@docker run -v ${PWD}/charts:/work -w /work jnorwood/helm-docs:v1.11.0 -s file

.PHONY: codegen
codegen: ## Rebuild all generated code and docs
codegen: codegen-helm-docs
codegen: codegen-openapi

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
	@$(HELM) upgrade --install reports-server --namespace reports-server --create-namespace --wait ./charts/reports-server \
		--set image.registry=$(KO_REGISTRY) \
		--set image.repository=$(PACKAGE) \
		--set image.tag=$(GIT_SHA)

########
# HELP #
########

.PHONY: help
help: ## Shows the available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-40s\033[0m %s\n", $$1, $$2}'
