# Version tag for all images
TEST_INFRA_VERSION ?= "latest"

# Version of the gRPC driver
DRIVER_VERSION ?= "master"

# Prefix for all images used as clone and ready containers, enabling use with registries
# other than DockerHub
INIT_IMAGE_PREFIX ?= ""

# Prefix for all images used as build containers, enabling use with registries
# other than DockerHub
BUILD_IMAGE_PREFIX ?= ""

# Prefix for all images used as runtime containers, enabling use with registries
# other than DockerHub
IMAGE_PREFIX ?= ""

# Image URL to use all building/pushing image targets
CONTROLLER_IMG ?= ${IMAGE_PREFIX}controller:${TEST_INFRA_VERSION}

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GOVERSION=$(shell go version | cut -f3 -d' ' | cut -c3-)

# Make tools build compatible with go 1.12
ifeq (1.13,$(shell echo -e "1.13\n$(GOVERSION)" | sort -V | head -n1))
GOARGS=-trimpath
TOOLSPREREQ=fmt vet
else
GOARGS=
TOOLSPREREQ=
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: controller all-tools

all-tools: runner prepare_prebuilt_workers delete_prebuilt_workers

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

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out -race -v

##@ Build executables

controller: generate fmt vet ## Build load test controller binary.
	go build $(GOARGS) -o bin/controller cmd/controller/main.go

runner: $(TOOLSPREREQ) ## Build the runner tool binary.
	go build $(GOARGS) -o bin/runner tools/cmd/runner/main.go

prepare_prebuilt_workers: $(TOOLSPREREQ) ## Build the prepare_prebuilt_workers tool binary.
	go build $(GOARGS) -o bin/prepare_prebuilt_workers tools/cmd/prepare_prebuilt_workers/main.go

delete_prebuilt_workers: $(TOOLSPREREQ) ## Build the delete_prebuilt_workers tool binary.
	go build $(GOARGS) -o bin/delete_prebuilt_workers tools/cmd/delete_prebuilt_workers/main.go

##@ Build container images

all-images: clone-image controller-image csharp-build-image cxx-image driver-image go-image java-image node-build-image node-image php7-build-image php7-image python-image ready-image ruby-build-image ruby-image ## Build all container images.

clone-image: ## Build the clone init container image.
	docker build -t ${INIT_IMAGE_PREFIX}clone:${TEST_INFRA_VERSION} containers/init/clone

controller-image: ## Build the load test controller container image.
	docker build -t ${CONTROLLER_IMG} -f containers/runtime/controller/Dockerfile .

csharp-build-image: ## Build the C# build-time container image.
	docker build -t ${BUILD_IMAGE_PREFIX}csharp:${TEST_INFRA_VERSION} containers/init/build/csharp

cxx-image: ## Build the C++ test runtime container image.
	docker build -t ${IMAGE_PREFIX}cxx:${TEST_INFRA_VERSION} containers/runtime/cxx

driver-image: ## Build the driver container image.
	docker build --build-arg GITREF=${DRIVER_VERSION} -t ${IMAGE_PREFIX}driver:${TEST_INFRA_VERSION} containers/runtime/driver

go-image: ## Build the Go test runtime container image.
	docker build -t ${IMAGE_PREFIX}go:${TEST_INFRA_VERSION} containers/runtime/go

java-image: ## Build the Java test runtime container image.
	docker build -t ${IMAGE_PREFIX}java:${TEST_INFRA_VERSION} containers/runtime/java

node-build-image: ## Build the Node.js build image
	docker build -t ${BUILD_IMAGE_PREFIX}node:${TEST_INFRA_VERSION} containers/init/build/node
	
node-image: ## Build the Node.js test runtime container image.
	docker build -t ${IMAGE_PREFIX}node:${TEST_INFRA_VERSION} containers/runtime/node

php7-build-image: ## Build the PHP7 build-time container image.
	docker build -t ${BUILD_IMAGE_PREFIX}php7:${TEST_INFRA_VERSION} containers/init/build/php7

php7-image: ## Build the PHP7 test runtime container image.
	docker build -t ${IMAGE_PREFIX}php7:${TEST_INFRA_VERSION} containers/runtime/php7

python-image: ## Build the Python test runtime container image.
	docker build -t ${IMAGE_PREFIX}python:${TEST_INFRA_VERSION} containers/runtime/python

ready-image: ## Build the ready init container image.
	docker build -t ${INIT_IMAGE_PREFIX}ready:${TEST_INFRA_VERSION} -f containers/init/ready/Dockerfile .

ruby-build-image: ## Build the Ruby build-time container image.
	docker build -t ${BUILD_IMAGE_PREFIX}ruby:${TEST_INFRA_VERSION} containers/init/build/ruby

ruby-image: ## Build the Ruby test runtime container image.
	docker build -t ${IMAGE_PREFIX}ruby:${TEST_INFRA_VERSION} containers/runtime/ruby

##@ Publish container images

push-all-images: push-clone-image push-controller-image push-csharp-build-image push-cxx-image push-driver-image push-go-image push-java-image push-node-build-image push-node-image push-php7-build-image push-php7-image push-python-image push-ready-image push-ruby-build-image push-ruby-image ## Push all container images to a registry.

push-clone-image: ## Push the clone init container image to a registry.
	docker push ${INIT_IMAGE_PREFIX}clone:${TEST_INFRA_VERSION}

push-controller-image: ## Push the load test controller container image to a registry.
	docker push ${CONTROLLER_IMG}

push-csharp-build-image: ## Push the C# build-time container image to a registry.
	docker push ${BUILD_IMAGE_PREFIX}csharp:${TEST_INFRA_VERSION}

push-cxx-image: ## Push the C++ test runtime container image to a registry.
	docker push ${IMAGE_PREFIX}cxx:${TEST_INFRA_VERSION}

push-driver-image: ## Push the driver container image to a registry.
	docker push ${IMAGE_PREFIX}driver:${TEST_INFRA_VERSION}

push-go-image: ## Push the Go test runtime container image to a registry.
	docker push ${IMAGE_PREFIX}go:${TEST_INFRA_VERSION}

push-java-image: ## Push the Java test runtime container image to a registry.
	docker push ${IMAGE_PREFIX}java:${TEST_INFRA_VERSION}

push-node-build-image: ## Push the Node.js build image to a docker registry
	docker push ${BUILD_IMAGE_PREFIX}node:${TEST_INFRA_VERSION}

push-node-image: ## Push the Node.js test runtime container image to a registry.
	docker push ${IMAGE_PREFIX}node:${TEST_INFRA_VERSION}

push-php7-build-image: ## Push the PHP7 build-time container image to a registry.
	docker push ${BUILD_IMAGE_PREFIX}php7:${TEST_INFRA_VERSION}

push-php7-image: ## Push the PHP7 test runtime container image to a registry.
	docker push ${IMAGE_PREFIX}php7:${TEST_INFRA_VERSION}

push-python-image: ## Push the Python test runtime container image to a registry.
	docker push ${IMAGE_PREFIX}python:${TEST_INFRA_VERSION}

push-ready-image: ## Push the ready init container image to a registry.
	docker push ${INIT_IMAGE_PREFIX}ready:${TEST_INFRA_VERSION}

push-ruby-build-image: ## Push the Ruby build-time container image to a registry.
	docker push ${BUILD_IMAGE_PREFIX}ruby:${TEST_INFRA_VERSION}

push-ruby-image: ## Push the Ruby test runtime container image to a registry.
	docker push ${IMAGE_PREFIX}ruby:${TEST_INFRA_VERSION}

##@ Deployment

install: ## Install both CRDs and RBACs
	install-crd install-rbac

uninstall: ## Uninstall both CRDs and RBACs
	uninstall-crd uninstall-rbac

install-crd: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall-crd: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=true -f -

install-rbac: manifests kustomize ## Install RBACs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/rbac | kubectl apply -f -

uninstall-rbac: manifests kustomize ## Uninstall RBACs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/rbac | kubectl delete --ignore-not-found=true -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${CONTROLLER_IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
