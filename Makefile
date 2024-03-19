
# Image URL to use all building/pushing image targets
CONTROLLER_IMG ?= $(RUN_IMAGE_PREFIX)controller:$(TEST_INFRA_VERSION)
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.29.0

# Golang command for build
GOCMD ?= go

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell $(GOCMD) env GOBIN))
GOBIN=$(shell $(GOCMD) env GOPATH)/bin
else
GOBIN=$(shell $(GOCMD) env GOBIN)
endif

# Golang build arguments
GOARGS = -trimpath

# Project directory.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Make all targets PHONY.
MAKEFLAGS += --always-make
all: build all-tools
all-tools: runner prepare_prebuilt_workers delete_prebuilt_workers

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	$(GOCMD) fmt ./...

vet: ## Run go vet against code.
	$(GOCMD) vet ./...

test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GOCMD) test $$($(GOCMD) list ./... | grep -v /e2e) -coverprofile cover.out

# Utilize Kind or modify the e2e tests to load the image locally, enabling compatibility with other vendors.
# Run the e2e tests against a Kind k8s instance that is spun up.
test-e2e:
	$(GOCMD) test ./test/e2e/ -v -ginkgo.v

lint: golangci-lint ## Run golangci-lint linter & yamllint
	$(GOLANGCI_LINT) run

lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

##@ Build

build: manifests generate fmt vet ## Build manager binary.
	$(GOCMD) build -o bin/manager cmd/main.go

run: manifests generate fmt vet ## Run a controller from your host.
	$(GOCMD) run ./cmd/main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t $(CONTROLLER_IMG) .

docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push $(CONTROLLER_IMG)

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le

docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name project-v3-builder
	$(CONTAINER_TOOL) buildx use project-v3-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm project-v3-builder
	rm Dockerfile.cross

build-installer: manifests generate $(KUSTOMIZE) ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	@if [ -d "config/crd" ]; then \
		$(KUSTOMIZE) build config/crd > dist/install.yaml; \
	fi
	echo "---" >> dist/install.yaml  # Add a document separator before appending
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default >> dist/install.yaml

##@ Build tool executables

runner: fmt vet ## Build the runner tool binary.
	$(GOCMD) build $(GOARGS) -o $(LOCALBIN)/runner tools/cmd/runner/main.go

prepare_prebuilt_workers: fmt vet ## Build the prepare_prebuilt_workers tool binary.
	$(GOCMD) build $(GOARGS) -o $(LOCALBIN)/prepare_prebuilt_workers tools/cmd/prepare_prebuilt_workers/main.go

delete_prebuilt_workers: fmt vet ## Build the delete_prebuilt_workers tool binary.
	$(GOCMD) build $(GOARGS) -o $(LOCALBIN)/delete_prebuilt_workers tools/cmd/delete_prebuilt_workers/main.go

##@ Build container images

all-images: clone-image controller-image csharp-build-image cxx-image dotnet-build-image dotnet-image driver-image go-image java-image node-build-image node-image php7-build-image php7-image python-image ready-image ruby-build-image ruby-image ## Build all container images.

clone-image: ## Build the clone init container image.
	docker build -t $(INIT_IMAGE_PREFIX)clone:$(TEST_INFRA_VERSION) containers/init/clone

controller-image: ## Build the load test controller container image.
	$(CONTAINER_TOOL) build -t $(CONTROLLER_IMG) -f containers/runtime/controller/Dockerfile .

csharp-build-image: ## Build the C# build-time container image.
	$(CONTAINER_TOOL) build -t $(BUILD_IMAGE_PREFIX)csharp:$(TEST_INFRA_VERSION) containers/init/build/csharp

cxx-image: ## Build the C++ test runtime container image.
	$(CONTAINER_TOOL) build -t $(RUN_IMAGE_PREFIX)cxx:$(TEST_INFRA_VERSION) containers/runtime/cxx

dotnet-build-image: ## Build the grpc-dotnet build-time container image.
	$(CONTAINER_TOOL) build -t $(BUILD_IMAGE_PREFIX)dotnet:$(TEST_INFRA_VERSION) containers/init/build/dotnet

dotnet-image: ## Build the grpc-dotnet test runtime container image.
	$(CONTAINER_TOOL) build -t $(RUN_IMAGE_PREFIX)dotnet:$(TEST_INFRA_VERSION) containers/runtime/dotnet

driver-image: ## Build the driver container image.
	$(CONTAINER_TOOL) build --build-arg GITREF=$(DRIVER_VERSION) --build-arg BREAK_CACHE="$(date +%Y%m%d%H%M%S)" -t $(RUN_IMAGE_PREFIX)driver:$(TEST_INFRA_VERSION) containers/runtime/driver

go-image: ## Build the Go test runtime container image.
	$(CONTAINER_TOOL) build -t $(RUN_IMAGE_PREFIX)go:$(TEST_INFRA_VERSION) containers/runtime/go

java-image: ## Build the Java test runtime container image.
	$(CONTAINER_TOOL) build -t $(RUN_IMAGE_PREFIX)java:$(TEST_INFRA_VERSION) containers/runtime/java

node-build-image: ## Build the Node.js build image
	$(CONTAINER_TOOL) build -t $(BUILD_IMAGE_PREFIX)node:$(TEST_INFRA_VERSION) containers/init/build/node

node-image: ## Build the Node.js test runtime container image.
	$(CONTAINER_TOOL) build -t $(RUN_IMAGE_PREFIX)node:$(TEST_INFRA_VERSION) containers/runtime/node

php7-build-image: ## Build the PHP7 build-time container image.
	$(CONTAINER_TOOL) build -t $(BUILD_IMAGE_PREFIX)php7:$(TEST_INFRA_VERSION) containers/init/build/php7

php7-image: ## Build the PHP7 test runtime container image.
	$(CONTAINER_TOOL) build -t $(RUN_IMAGE_PREFIX)php7:$(TEST_INFRA_VERSION) containers/runtime/php7

python-image: ## Build the Python test runtime container image.
	$(CONTAINER_TOOL) build -t $(RUN_IMAGE_PREFIX)python:$(TEST_INFRA_VERSION) containers/runtime/python

ready-image: ## Build the ready init container image.
	$(CONTAINER_TOOL) build -t $(INIT_IMAGE_PREFIX)ready:$(TEST_INFRA_VERSION) -f containers/init/ready/Dockerfile .

ruby-build-image: ## Build the Ruby build-time container image.
	$(CONTAINER_TOOL) build -t $(BUILD_IMAGE_PREFIX)ruby:$(TEST_INFRA_VERSION) containers/init/build/ruby

ruby-image: ## Build the Ruby test runtime container image.
	$(CONTAINER_TOOL) build -t $(RUN_IMAGE_PREFIX)ruby:$(TEST_INFRA_VERSION) containers/runtime/ruby

##@ Publish container images

push-all-images: push-clone-image push-controller-image push-csharp-build-image push-cxx-image push-dotnet-build-image push-dotnet-image push-driver-image push-go-image push-java-image push-node-build-image push-node-image push-php7-build-image push-php7-image push-python-image push-ready-image push-ruby-build-image push-ruby-image  ## Push all container images to a registry.

push-clone-image: ## Push the clone init container image to a registry.
	$(CONTAINER_TOOL) push $(INIT_IMAGE_PREFIX)clone:$(TEST_INFRA_VERSION)

push-controller-image: ## Push the load test controller container image to a registry.
	$(CONTAINER_TOOL) push $(CONTROLLER_IMG)

push-csharp-build-image: ## Push the C# build-time container image to a registry.
	$(CONTAINER_TOOL) push $(BUILD_IMAGE_PREFIX)csharp:$(TEST_INFRA_VERSION)

push-cxx-image: ## Push the C++ test runtime container image to a registry.
	$(CONTAINER_TOOL) push $(RUN_IMAGE_PREFIX)cxx:$(TEST_INFRA_VERSION)

push-dotnet-build-image: ## Push the grpc-dotnet build image to a docker registry
	$(CONTAINER_TOOL) push $(BUILD_IMAGE_PREFIX)dotnet:$(TEST_INFRA_VERSION)

push-dotnet-image: ## Push the grpc-dotnet.js test runtime container image to a registry.
	$(CONTAINER_TOOL) push $(RUN_IMAGE_PREFIX)dotnet:$(TEST_INFRA_VERSION)

push-driver-image: ## Push the driver container image to a registry.
	$(CONTAINER_TOOL) push $(RUN_IMAGE_PREFIX)driver:$(TEST_INFRA_VERSION)

push-go-image: ## Push the Go test runtime container image to a registry.
	$(CONTAINER_TOOL) push $(RUN_IMAGE_PREFIX)go:$(TEST_INFRA_VERSION)

push-java-image: ## Push the Java test runtime container image to a registry.
	$(CONTAINER_TOOL) push $(RUN_IMAGE_PREFIX)java:$(TEST_INFRA_VERSION)

push-node-build-image: ## Push the Node.js build image to a docker registry
	$(CONTAINER_TOOL) push $(BUILD_IMAGE_PREFIX)node:$(TEST_INFRA_VERSION)

push-node-image: ## Push the Node.js test runtime container image to a registry.
	$(CONTAINER_TOOL) push $(RUN_IMAGE_PREFIX)node:$(TEST_INFRA_VERSION)

push-php7-build-image: ## Push the PHP7 build-time container image to a registry.
	$(CONTAINER_TOOL) push $(BUILD_IMAGE_PREFIX)php7:$(TEST_INFRA_VERSION)

push-php7-image: ## Push the PHP7 test runtime container image to a registry.
	$(CONTAINER_TOOL) push $(RUN_IMAGE_PREFIX)php7:$(TEST_INFRA_VERSION)

push-python-image: ## Push the Python test runtime container image to a registry.
	$(CONTAINER_TOOL) push $(RUN_IMAGE_PREFIX)python:$(TEST_INFRA_VERSION)

push-ready-image: ## Push the ready init container image to a registry.
	$(CONTAINER_TOOL) push $(INIT_IMAGE_PREFIX)ready:$(TEST_INFRA_VERSION)

push-ruby-build-image: ## Push the Ruby build-time container image to a registry.
	$(CONTAINER_TOOL) push $(BUILD_IMAGE_PREFIX)ruby:$(TEST_INFRA_VERSION)

push-ruby-image: ## Push the Ruby test runtime container image to a registry.
	$(CONTAINER_TOOL) push $(RUN_IMAGE_PREFIX)ruby:$(TEST_INFRA_VERSION)

##@ Build PSM related container images

all-psm-images: sidecar-image xds-server-image ## Build all psm related container images to a registry.

sidecar-image: ## Build the sidecar runtime container image.
	$(CONTAINER_TOOL) build --no-cache -t ${PSM_IMAGE_PREFIX}sidecar:${TEST_INFRA_VERSION} containers/runtime/sidecar/

xds-server-image: ## Build the xds server runtime container image.
	$(CONTAINER_TOOL) build --no-cache -t ${PSM_IMAGE_PREFIX}xds-server:${TEST_INFRA_VERSION} -f containers/runtime/xds-server/Dockerfile .

##@ Publish PSM related container images
push-all-psm-images: push-sidecar-image push-xds-server-image ## Push all psm related container images to a registry.

push-sidecar-image: ## Push the sidecar image to container registry.
	$(CONTAINER_TOOL) push ${PSM_IMAGE_PREFIX}sidecar:${TEST_INFRA_VERSION}

push-xds-server-image: ## Push the xds-server image to container registry.
	$(CONTAINER_TOOL) push ${PSM_IMAGE_PREFIX}xds-server:${TEST_INFRA_VERSION}

##@ Deployment

install: install-crd install-rbac ## Install both CRDs and RBACs

uninstall: uninstall-crd uninstall-rbac ## Uninstall both CRDs and RBACs

install-crd: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

uninstall-crd: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=true -f -

install-rbac: manifests kustomize ## Install RBACs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/rbac | $(KUBECTL) apply -f -

uninstall-rbac: manifests kustomize ## Uninstall RBACs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/rbac | $(KUBECTL) delete --ignore-not-found=true -f -

install-prometheus-crds: ## Install Prometheus and Prometheus Operator related CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUBECTL) create -f config/prometheus/crds/bases/crds.yaml

uninstall-prometheus-crds: ## Uninstall Prometheus and Prometheus Operator related CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUBECTL) delete --ignore-not-found=true -f config/prometheus/crds/bases/crds.yaml

deploy-prometheus: kustomize ## Deploy Prometheus Operator and Prometheus into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/prometheus/ | $(KUBECTL) apply -f -

undeploy-prometheus: kustomize ## Delete Prometheus Operator and Prometheus deployment from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/prometheus/ | $(KUBECTL) delete --ignore-not-found=true -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= $(KUBECTL)
KUSTOMIZE ?= $(LOCALBIN)/kustomize-$(KUSTOMIZE_VERSION)
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen-$(CONTROLLER_TOOLS_VERSION)
ENVTEST ?= $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION)
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)

## Tool Versions
KUSTOMIZE_VERSION ?= v5.3.0
CONTROLLER_TOOLS_VERSION ?= v0.14.0
ENVTEST_VERSION ?= latest
GOLANGCI_LINT_VERSION ?= v1.54.2

$(KUSTOMIZE): $(KUSTOMIZE) ## Download $(KUSTOMIZE) locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/$(KUSTOMIZE)/$(KUSTOMIZE)/v5,$(KUSTOMIZE_VERSION))

controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,${GOLANGCI_LINT_VERSION})

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
GOBIN=$(LOCALBIN) $(GOCMD) install $${package} ;\
mv "$$(echo "$(1)" | sed "s/-$(3)$$//")" $(1) ;\
}
endef
