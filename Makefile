# Module Name used for bundling the OCI Image and later on for referencing in the Kyma Modules
MODULE_NAME ?= btp-operator

# Semantic Module Version used for identifying the build
MODULE_VERSION ?= 0.0.1

# Module Registry used for pushing the image
MODULE_REGISTRY_PORT ?= 60770
MODULE_REGISTRY ?= op-kcp-registry.localhost:$(MODULE_REGISTRY_PORT)/unsigned

# Desired Channel of the Generated Module Template
MODULE_CHANNEL ?= alpha

# Operating system architecture
OS_ARCH ?= $(shell uname -m)

# Operating system type
OS_TYPE ?= $(shell uname)

# This value is used only if SUITE_TIMEOUT is not exported in set-env-vars.sh and is not specified by the user during the make execution
SUITE_TIMEOUT ?= $${SUITE_TIMEOUT:-30s}

# Credentials used for authenticating into the module registry
# see `kyma alpha mod create --help for more info`
# MODULE_CREDENTIALS ?= testuser:testpw

# Image URL to use all building/pushing image targets
IMG_REGISTRY_PORT ?= 60765
IMG_REGISTRY ?= op-skr-registry.localhost:$(IMG_REGISTRY_PORT)/unsigned/operator-images
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

# This will change the flags of the `kyma alpha module create` command in case we spot credentials
# Otherwise we will assume http-based local registries without authentication (e.g. for k3d)
ifneq (,$(PROW_JOB_ID))
GCP_ACCESS_TOKEN=$(shell gcloud auth application-default print-access-token)
MODULE_CREATION_FLAGS=--registry $(MODULE_REGISTRY) -c oauth2accesstoken:$(GCP_ACCESS_TOKEN)
else ifeq (,$(MODULE_CREDENTIALS))
MODULE_CREATION_FLAGS=--registry $(MODULE_REGISTRY) --insecure
else
MODULE_CREATION_FLAGS=--registry $(MODULE_REGISTRY) -c $(MODULE_CREDENTIALS)
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

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: test
test: manifests kustomize generate fmt vet envtest ginkgo  test-docs ## Run tests.
	@. ./scripts/testing/set-env-vars.sh; \
	go test -skip=TestAPIs ./... -timeout $(SUITE_TIMEOUT) -coverprofile cover.out -v; \
	if [ "$(USE_EXISTING_CLUSTER)" == "true" ]; then $(GINKGO) -v controllers; else $(GINKGO) -v -p controllers; fi

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

.PHONY: module-build
module-build: kyma kustomize ## Build the Module and push it to a registry defined in MODULE_REGISTRY
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KYMA) alpha create module $(SECURITY_SCAN_OPTIONS) --channel=${MODULE_CHANNEL} --name kyma.project.io/module/$(MODULE_NAME) --version $(MODULE_VERSION) --path . $(MODULE_CREATION_FLAGS)

.PHONY: module-template-push
module-template-push: ## Pushes the ModuleTemplate referencing the Image on MODULE_REGISTRY
	sh hack/local-template.sh
	kubectl apply -f template.yaml

##@ Tools

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

########## Kustomize ###########
KUSTOMIZE ?= $(LOCALBIN)/kustomize
KUSTOMIZE_VERSION ?= v4.5.6
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download & Build kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/kustomize/kustomize/v4@$(KUSTOMIZE_VERSION)

########## Kyma CLI ###########
KYMA_STABILITY ?= unstable

# $(call os_error, os-type, os-architecture)
define os_error
$(error Error: unsuported platform OS_TYPE:$1, OS_ARCH:$2; to mitigate this problem set variable KYMA with absolute path to kyma-cli binary compatible with your operating system and architecture)
endef

KYMA_FILE_NAME ?= $(shell ./hack/get_kyma_file_name.sh ${OS_TYPE} ${OS_ARCH})

KYMA ?= $(LOCALBIN)/kyma-$(KYMA_STABILITY)
kyma: $(LOCALBIN) $(KYMA) ## Download kyma locally if necessary.
$(KYMA):
	## Detect if operating system
	$(if $(KYMA_FILE_NAME),,$(call os_error, ${OS_TYPE}, ${OS_ARCH}))
	test -f $@ || curl -s -Lo $(KYMA) https://storage.googleapis.com/kyma-cli-$(KYMA_STABILITY)/$(KYMA_FILE_NAME)
	chmod 0100 $(KYMA)


########## controller-gen ###########
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
CONTROLLER_TOOLS_VERSION ?= v0.9.2
.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download & Build controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

########## envtest ###########
ENVTEST ?= $(LOCALBIN)/setup-envtest
.PHONY: envtest
envtest: $(ENVTEST) ## Download & Build envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.0.0-20230403212152-53057ba616d1

########## ginkgo ###########
GINKGO ?= $(LOCALBIN)/ginkgo
.PHONY: ginkgo
ginkgo: $(GINKGO) ## Download & Build ginkgo locally if necessary.
$(GINKGO): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/v2/ginkgo@v2.9.2

##@ Checks

########## static code checks ###########
.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

GOLANG_CI_LINT = $(LOCALBIN)/golangci-lint
GOLANG_CI_LINT_VERSION ?= v1.50.1
.PHONY: lint
lint: ## Download & Build & Run golangci-lint against code.
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANG_CI_LINT_VERSION)
	$(LOCALBIN)/golangci-lint run

