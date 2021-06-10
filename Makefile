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
# Image URL to use all building/pushing image targets
CLEAN_IMG ?= ${IMAGE_PREFIX}cleanup:${TEST_INFRA_VERSION}
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

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

all: controller cleanup-agent all-tools

all-tools: runner prepare_prebuilt_workers delete_prebuilt_workers

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build controller manager binary
controller: generate fmt vet
	go build $(GOARGS) -o bin/controller cmd/controller/main.go

# Build cleanup_agent manager binary
cleanup-agent: generate fmt vet
	go build $(GOARGS) -o bin/cleanup_agent cmd/cleanup_agent/main.go

# Build test runner tool
runner: $(TOOLSPREREQ)
	go build $(GOARGS) -o bin/runner cmd/runner/main.go

# Build prepare_prebuilt_workers tool
prepare_prebuilt_workers: $(TOOLSPREREQ)
	go build $(GOARGS) -o bin/prepare_prebuilt_workers tools/prepare_prebuilt_workers/prepare_prebuilt_workers.go

# Build delete_prebuilt_workers tool
delete_prebuilt_workers: $(TOOLSPREREQ)
	go build $(GOARGS) -o bin/delete_prebuilt_workers tools/delete_prebuilt_workers/delete_prebuilt_workers.go

# Install both CRDs and RBACs
install: installCRD installRBAC

# Uninstall both CRDs and RBACs
uninstall: uninstallCRD uninstallRBAC

# Install CRDs into a cluster
installCRD: manifests
	kustomize build config/crd | kubectl apply -f - 

# Uninstall CRDs from a cluster
uninstallCRD: manifests
	kustomize build config/crd | kubectl delete -f -


# Install RBACs into a cluster
installRBAC: manifests
	kustomize build config/rbac | kubectl apply -f -

# Uninstall RBACs from a cluster
uninstallRBAC: manifests
	kustomize build config/rbac | kubectl delete -f -

# Deploy both controller and cleanup_agent to the cluster
deploy: deploy-controller deploy-cleanup-agent

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy-controller: manifests
	cd config/manager && kustomize edit set image controller=${CONTROLLER_IMG}
	kustomize build config/default | kubectl apply -f -

# Deploy cleanup_agent in the configured Kubernetes cluster in ~/.kube/config
deploy-cleanup-agent: manifests
	cd config/cleanup_agent && kustomize edit set image cleanup_agent=${CLEAN_IMG}
	kustomize build config/cleanup_agent | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." \
		output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the manager image with the controller
controller-image:
	docker build -t ${CONTROLLER_IMG} -f containers/runtime/controller/Dockerfile .

# Build the manager image with the cleanup_agent
cleanup-agent-image:
	docker build -t ${CLEAN_IMG} -f containers/runtime/cleanup_agent/Dockerfile .

# Push the controller manager image to a docker registry
push-controller-image:
	docker push ${CONTROLLER_IMG}

# Push the cleanup-agent manager image to a docker registry
push-cleanup-agent-image:
	docker push ${CLEAN_IMG}

# Build the clone init container image
clone-image:
	docker build -t ${INIT_IMAGE_PREFIX}clone:${TEST_INFRA_VERSION} \
		containers/init/clone

# Push the clone init container image to a docker registry
push-clone-image:
	docker push ${INIT_IMAGE_PREFIX}clone:${TEST_INFRA_VERSION}

# Build the ready init container image
ready-image:
	docker build -t ${INIT_IMAGE_PREFIX}ready:${TEST_INFRA_VERSION} \
		-f containers/init/ready/Dockerfile .

# Push the ready init container image to a docker registry
push-ready-image:
	docker push ${INIT_IMAGE_PREFIX}ready:${TEST_INFRA_VERSION}

# Build the driver container image at the $DRIVER_VERSION
driver-image:
	docker build --build-arg GITREF=${DRIVER_VERSION} \
		-t ${IMAGE_PREFIX}driver:${TEST_INFRA_VERSION} \
		containers/runtime/driver

# Push the driver container image to a docker regisry
push-driver-image:
	docker push ${IMAGE_PREFIX}driver:${TEST_INFRA_VERSION}

# Build the C++ runtime image
cxx-image:
	docker build -t ${IMAGE_PREFIX}cxx:${TEST_INFRA_VERSION} containers/runtime/cxx

# Push the C++ runtime image to a docker registry
push-cxx-image:
	docker push ${IMAGE_PREFIX}cxx:${TEST_INFRA_VERSION}

# Build the Go runtime image
go-image:
	docker build -t ${IMAGE_PREFIX}go:${TEST_INFRA_VERSION} containers/runtime/go

# Push the Go runtime image to a docker registry
push-go-image:
	docker push ${IMAGE_PREFIX}go:${TEST_INFRA_VERSION}

# Build the Node.js build image
node-build-image:
	docker build -t ${BUILD_IMAGE_PREFIX}node:${TEST_INFRA_VERSION} containers/init/build/node

# Push the Node.js build image to a docker registry
push-node-build-image:
	docker push ${BUILD_IMAGE_PREFIX}node:${TEST_INFRA_VERSION}

# Build the Node.js runtime image
node-image:
	docker build -t ${IMAGE_PREFIX}node:${TEST_INFRA_VERSION} containers/runtime/node

# Push the Node.js runtime image to a docker registry
push-node-image:
	docker push ${IMAGE_PREFIX}node:${TEST_INFRA_VERSION}

# Build the Java runtime image
java-image:
	docker build -t ${IMAGE_PREFIX}java:${TEST_INFRA_VERSION} \
		containers/runtime/java

# Push the Java runtime image to a docker registry
push-java-image:
	docker push ${IMAGE_PREFIX}java:${TEST_INFRA_VERSION}

# Build the Ruby build image
ruby-build-image:
	docker build -t ${BUILD_IMAGE_PREFIX}ruby:${TEST_INFRA_VERSION} \
		containers/init/build/ruby

# Push the Ruby runtime image to a docker registry
push-ruby-build-image:
	docker push ${BUILD_IMAGE_PREFIX}ruby:${TEST_INFRA_VERSION}

# Build the Ruby runtime image
ruby-image:
	docker build -t ${IMAGE_PREFIX}ruby:${TEST_INFRA_VERSION} \
		containers/runtime/ruby

# Push the Ruby runtime image to a docker registry
push-ruby-image:
	docker push ${IMAGE_PREFIX}ruby:${TEST_INFRA_VERSION}

# Build the PHP7 build image
php7-build-image:
	docker build -t ${BUILD_IMAGE_PREFIX}php7:${TEST_INFRA_VERSION} \
		containers/init/build/php7

# Push the PHP7 runtime image to a docker registry
push-php7-build-image:
	docker push ${BUILD_IMAGE_PREFIX}php7:${TEST_INFRA_VERSION}

# Build the PHP7 runtime image
php7-image:
	docker build -t ${IMAGE_PREFIX}php7:${TEST_INFRA_VERSION} \
		containers/runtime/php7

# Push the PHP7 runtime image to a docker registry
push-php7-image:
	docker push ${IMAGE_PREFIX}php7:${TEST_INFRA_VERSION}

# Build the Python runtime image
python-image:
	docker build -t ${IMAGE_PREFIX}python:${TEST_INFRA_VERSION} \
		containers/runtime/python

# Push the Python runtime image to a docker registry
push-python-image:
	docker push ${IMAGE_PREFIX}python:${TEST_INFRA_VERSION}

# Build the csharp build image
csharp-build-image:
	docker build -t ${BUILD_IMAGE_PREFIX}csharp:${TEST_INFRA_VERSION} containers/init/build/csharp

# Push the csharp build image to a docker registry
push-csharp-build-image:
	docker push ${BUILD_IMAGE_PREFIX}csharp:${TEST_INFRA_VERSION}

# Build all init container and runtime container images
all-images: \
	clone-image \
	ready-image \
	driver-image \
	cxx-image \
	go-image \
	java-image \
	node-build-image \
	node-image \
	python-image \
	php7-build-image \
	php7-image \
	ruby-build-image \
	ruby-image \
	csharp-build-image \
	controller-image\
	cleanup-agent-image

# Push all init container and runtime container images to a docker registry
push-all-images: \
	push-clone-image \
	push-ready-image \
	push-driver-image \
	push-cxx-image \
	push-go-image \
	push-node-build-image \
	push-node-image \
	push-java-image \
	push-php7-build-image \
	push-php7-image \
	push-python-image \
	push-ruby-build-image \
	push-ruby-image \
	push-csharp-build-image \
	push-controller-image \
	push-cleanup-agent-image

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
