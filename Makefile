.DEFAULT_GOAL := build

##########
# CONFIG #
##########

ORG                                ?= nirmata
PACKAGE                            ?= github.com/kyverno/reports-server
GIT_SHA                            := $(shell git rev-parse HEAD)
GOOS                               ?= $(shell go env GOOS)
GOARCH                             ?= $(shell go env GOARCH)
REGISTRY                           ?= ghcr.io
REPO                               ?= reports-server
REPO_REPORTS_SERVER	?= 	$(REGISTRY)/$(ORG)/$(REPO)

#########
# TOOLS #
#########

TOOLS_DIR                          := $(PWD)/.tools
REGISTER_GEN                       := $(TOOLS_DIR)/register-gen
OPENAPI_GEN                        := $(TOOLS_DIR)/openapi-gen
CODE_GEN_VERSION                   := v0.28.0
KIND                               := $(TOOLS_DIR)/kind
KIND_VERSION                       := v0.23.0
KO                                 := $(TOOLS_DIR)/ko
KO_VERSION                         := v0.14.1
HELM                               := $(TOOLS_DIR)/helm
HELM_VERSION                       := v3.10.1
TOOLS                              := $(REGISTER_GEN) $(OPENAPI_GEN) $(KIND) $(KO) $(HELM)
ifeq ($(GOOS), darwin)
SED                                := gsed
else
SED                                := sed
endif

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
LOCAL_PLATFORM := linux/$(GOARCH)
KO_REGISTRY    := ko.local
KO_CACHE       ?= /tmp/ko-cache
BIN            := reports-server
ifdef VERSION
LD_FLAGS       := "-s -w -X $(PACKAGE)/pkg/version.BuildVersion=$(VERSION)"
else
LD_FLAGS       := "-s -w"
endif
ifndef VERSION
KO_TAGS             := $(GIT_SHA)
else ifeq ($(VERSION),main)
KO_TAGS             := $(GIT_SHA),latest
else
KO_TAGS             := $(GIT_SHA),$(subst /,-,$(VERSION))
endif

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

########
# TEST #
########

.PHONY: tests
tests: build ## Run tests
	@echo Running tests... >&2
	@go test ./... -race -coverprofile=coverage.out -covermode=atomic

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
	@mkdir -p $(GOPATH_SHIM)/src/github.com/kyverno && ln -s -f ${PWD} $(PACKAGE_SHIM)

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
		-i github.com/kyverno/kyverno/api/reports/v1 \
		-i github.com/kyverno/kyverno/api/policyreport/v1alpha2 \
		-p ./pkg/api/generated/openapi \
		-O zz_generated.openapi \
		-h ./.hack/boilerplate.go.txt

.PHONY: codegen-helm-docs
codegen-helm-docs: ## Generate helm docs
	@echo Generate helm docs... >&2
	@docker run -v ${PWD}/charts:/work -w /work jnorwood/helm-docs:v1.11.0 -s file

.PHONY: codegen-install-manifest
codegen-install-manifest: $(HELM) ## Create install manifest
	@echo Generate latest install manifest... >&2
	@$(HELM) template reports-server --namespace reports-server ./charts/reports-server/ \
		--set image.tag=latest \
		--set templating.enabled=true \
 		| $(SED) -e '/^#.*/d' \
		> ./config/install.yaml

codegen-install-manifest-inmemory: $(HELM) ## Create install manifest without postgres
	@echo Generate latest install manifest... >&2
	@$(HELM) template reports-server --namespace reports-server ./charts/reports-server/ \
		--set image.tag=latest \
		--set config.debug=true \
		--set postgresql.enabled=false \
		--set templating.enabled=true \
 		| $(SED) -e '/^#.*/d' \
		> ./config/install-inmemory.yaml

.PHONY: codegen
codegen: ## Rebuild all generated code and docs
codegen: codegen-helm-docs
codegen: codegen-openapi
codegen: codegen-install-manifest
codegen: codegen-install-manifest-inmemory

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

KIND_IMAGE     ?= kindest/node:v1.30.0
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

.PHONY: kind-install-inmemory
kind-install-inmemory: $(HELM) kind-load ## Build image, load it in kind cluster and deploy helm chart
	@echo Install chart... >&2
	@$(HELM) upgrade --install reports-server --namespace reports-server --create-namespace --wait ./charts/reports-server \
		--set image.registry=$(KO_REGISTRY) \
		--set config.debug=true \
		--set postgresql.enabled=false \
		--set image.repository=$(PACKAGE) \
		--set image.tag=$(GIT_SHA)
 
.PHONY: kind-apply
kind-apply: $(HELM) kind-load ## Build image, load it in kind cluster and deploy helm chart
	@echo Install chart... >&2
	@$(HELM) template reports-server --namespace reports-server ./charts/reports-server \
		--set image.registry=$(KO_REGISTRY) \
		--set image.repository=$(PACKAGE) \
		--set image.tag=$(GIT_SHA) \
			| kubectl apply -f -

.PHONY: kind-migrate
kind-migrate: $(HELM) kind-load ## Build image, load it in kind cluster and deploy helm chart
	@echo Install chart... >&2
	@$(HELM) upgrade --install reports-server --namespace reports-server --create-namespace --wait ./charts/reports-server \
		--set image.registry=$(KO_REGISTRY) \
		--set image.repository=$(PACKAGE) \
		--set image.tag=$(GIT_SHA) \
		--set apiServices.enabled=false

.PHONY: kind-apply-api-services
kind-apply-api-services: $(HELM) kind-load ## Build image, load it in kind cluster and deploy helm chart
	@echo Install api services... >&2
	@$(HELM) template reports-server --namespace reports-server ./charts/reports-server \
		--set image.registry=$(KO_REGISTRY) \
		--set image.repository=$(PACKAGE) \
		--set image.tag=$(GIT_SHA) \
			| kubectl apply -f -

.PHONY: install-pss-policies
install-pss-policies: $(HELM) 
	@echo Install pss policies... >&2
	@$(HELM) repo add kyverno https://kyverno.github.io/kyverno/
	@$(HELM) upgrade --install kyverno-policies kyverno/kyverno-policies \
		--set=podSecurityStandard=restricted \
		--set=background=true \
		--set=validationFailureAction=Audit

########
# HELP #
########

.PHONY: help
help: ## Shows the available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-40s\033[0m %s\n", $$1, $$2}'

################
# PUBLISH (KO) #
################

REGISTRY_USERNAME   ?= dummy
PLATFORMS           := all

.PHONY: ko-login
ko-login: $(KO)
	@$(KO) login $(REGISTRY) --username $(REGISTRY_USERNAME) --password $(REGISTRY_PASSWORD)

.PHONY: ko-publish-reports-server
ko-publish-reports-server: ko-login ## Build and publish reports-server image (with ko)
	@LD_FLAGS=$(LD_FLAGS) KOCACHE=$(KOCACHE) KO_DOCKER_REPO=$(REPO_REPORTS_SERVER) \
		$(KO) build . --bare --tags=$(KO_TAGS) --platform=$(PLATFORMS)
