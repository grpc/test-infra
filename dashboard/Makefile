GOCMD ?= go
GOARGS= -trimpath
REPLICATOR_OUTPUT_DIR ?= bin

configure_replicator:
	git clone --depth 1 https://github.com/grpc/test-infra postgres_replicator/test-infra
	cp ../config/dashboard/postgres_replicator/config.yaml postgres_replicator

replicator: fmt vet
	$(GOCMD) build $(GOARGS) -o $(REPLICATOR_OUTPUT_DIR)/replicator cmd/postgres_replicator/main.go

configure_grafana:
	cp -r ../config/dashboard/dashboards grafana

vet: ## Run go vet against code.
	$(GOCMD) vet ./...

fmt: ## Run go fmt against code.
	$(GOCMD) fmt ./...
