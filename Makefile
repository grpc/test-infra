# Version tag for all images
TEST_INFRA_VERSION ?= "latest"
# Version of the gRPC driver
DRIVER_VERSION ?= "master"
# Prefix for all images used as init containers, enabling use with registries
# other than DockerHub
INIT_IMAGE_PREFIX ?= ""
# Prefix for all images used as runtime containers, enabling use with registries
# other than DockerHub
IMAGE_PREFIX ?= ""
# Image URL to use all building/pushing image targets
IMG ?= ${IMAGE_PREFIX}controller:${TEST_INFRA_VERSION}
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

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
manager-image:
	docker build . -t ${IMG}

# Push the manager image to a docker registry
push-manager-image:
	docker push ${IMG}

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
	docker push ${IMAGE_PREFIX}driver:${TEST_INFRA_VERSION}-grpc${DRIVER_VERSION}

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

# Build the Java runtime image
java-image:
	docker build -t ${IMAGE_PREFIX}java:${TEST_INFRA_VERSION} \
		containers/runtime/java

# Push the Java runtime image to a docker registry
push-java-image:
	docker push ${IMAGE_PREFIX}java:${TEST_INFRA_VERSION}

# Build the Ruby runtime image
ruby-image:
	docker build -t ${IMAGE_PREFIX}ruby:${TEST_INFRA_VERSION} \
		containers/runtime/ruby

# Push the Ruby runtime image to a docker registry
push-ruby-image:
	docker push ${IMAGE_PREFIX}ruby:${TEST_INFRA_VERSION}

# Build the Python runtime image
python-image:
	docker build -t ${IMAGE_PREFIX}python:${TEST_INFRA_VERSION} \
		containers/runtime/python

# Push the Python runtime image to a docker registry
push-python-image:
	docker push ${IMAGE_PREFIX}python:${TEST_INFRA_VERSION}

# Build all init container and runtime container images
all-images: \
	clone-image \
	ready-image \
	driver-image \
	cxx-image \
	go-image \
	java-image \
	ruby-image \
	python-image \
	manager-image

# Push all init container and runtime container images to a docker registry
push-all-images: \
	push-clone-image \
	push-ready-image \
	push-driver-image \
	push-cxx-image \
	push-go-image \
	push-java-image \
	push-ruby-image \
	push-python-image \
	push-manager-image

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
