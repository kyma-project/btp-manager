# Semantic Module Version used for identifying the build
MODULE_VERSION ?= 0.0.1

# Operating system architecture
OS_ARCH ?= $(shell uname -m)

# Operating system type
OS_TYPE ?= $(shell uname)

# This value is used only if SUITE_TIMEOUT is not exported in set-env-vars.sh and is not specified by the user during the make execution
SUITE_TIMEOUT ?= $${SUITE_TIMEOUT:-30s}

# btp-manager manifest file
MANIFEST_PATH=./manifests/btp-operator
MANIFEST_FILE=btp-manager.yaml

# Image URL to use all building/pushing image targets
IMG_REGISTRY_PORT ?= 5001
IMG_REGISTRY ?= k3d-kyma-registry:$(IMG_REGISTRY_PORT)
IMG ?= $(IMG_REGISTRY)/btp-manager:$(MODULE_VERSION)

COMPONENT_CLI_VERSION ?= latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

GOLINT_VER = v2.1.5
ifeq (,$(GOLINT_TIMEOUT))
GOLINT_TIMEOUT=2m
endif

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	@echo "Adding nonResourceURLs to ClusterRole..."
	@if ! grep -q "nonResourceURLs:" config/rbac/role.yaml; then \
		echo "- nonResourceURLs:" >> config/rbac/role.yaml; \
		echo "  - /metrics" >> config/rbac/role.yaml; \
		echo "  verbs:" >> config/rbac/role.yaml; \
		echo "  - get" >> config/rbac/role.yaml; \
	fi

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: test
test: manifests kustomize generate fmt vet envtest ginkgo test-docs ## Run tests.
	@. ./scripts/testing/set-env-vars.sh; \
	go test -skip=TestAPIs ./... -timeout $(SUITE_TIMEOUT) -coverprofile cover.out -v; \
	if [ "$(USE_EXISTING_CLUSTER)" == "true" ]; then $(GINKGO) controllers; else $(GINKGO) $(GINKGO_PARALLEL_FLAG) controllers; fi

.PHONY: test-docs
test-docs:
	go run cmd/autodoc/main.go

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	IMG=$(IMG) docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
ifneq (,$(GCR_DOCKER_PASSWORD))
	docker login $(IMG_REGISTRY) -u oauth2accesstoken --password $(GCR_DOCKER_PASSWORD)
endif
	docker push ${IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: create-manifest
create-manifest: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	mkdir -p ${MANIFEST_PATH}
	$(KUSTOMIZE) build config/default > ${MANIFEST_PATH}/${MANIFEST_FILE}
	@grep "image:" ${MANIFEST_PATH}/${MANIFEST_FILE}
	@grep "^kind:" ${MANIFEST_PATH}/${MANIFEST_FILE}|sort

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Module

.PHONY: module-image
module-image: docker-build docker-push ## Build the Module Image and push it to a registry defined in IMG_REGISTRY
	echo "built and pushed module image $(IMG)"

##@ Tools

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

########## Kustomize ###########
KUSTOMIZE ?= $(LOCALBIN)/kustomize
KUSTOMIZE_VERSION ?= v5.4.3
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download & Build kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

########## controller-gen ###########
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
CONTROLLER_TOOLS_VERSION ?= v0.19.0
.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download & Build controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

########## envtest ###########
ENVTEST ?= $(LOCALBIN)/setup-envtest
.PHONY: envtest
envtest: $(ENVTEST) ## Download & Build envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.0.0-20240820183333-e6c3d139d2b6

########## ginkgo ###########
GINKGO ?= $(LOCALBIN)/ginkgo
.PHONY: ginkgo
ginkgo: $(GINKGO) ## Download & Build ginkgo locally if necessary.
$(GINKGO): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/v2/ginkgo@v2.23.4

##@ Checks

########## static code checks ###########
.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

go-lint-install: ## linter config in file at root of project -> '.golangci.yaml'
	@if [ "$(shell command golangci-lint version --short)" != "$(GOLINT_VER)" ]; then \
  		echo golangci in version $(GOLINT_VER) not found. will be downloaded; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLINT_VER); \
		echo golangci installed with version: $(shell command golangci-lint version --short); \
	fi;

.PHONY: go-lint
go-lint: go-lint-install ## linter config in file at root of project -> '.golangci.yaml'
	golangci-lint run --timeout=$(GOLINT_TIMEOUT)

.PHONY: fix
fix: go-lint-install ## try to fix automatically issues
	go mod tidy
	golangci-lint run --fix
