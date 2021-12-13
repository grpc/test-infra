GOCMD ?= go
GOARGS= -trimpath

postgres_replicator: fmt vet
	$(GOCMD) build $(GOARGS) -o bin/postgres_replicator cmd/postgres_replicator/main.go

vet: ## Run go vet against code.
	$(GOCMD) vet ./...

fmt: ## Run go fmt against code.
	$(GOCMD) fmt ./...
