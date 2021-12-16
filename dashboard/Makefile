GOCMD ?= go
GOARGS= -trimpath
REPLICATOR_OUTPUT_DIR ?= bin

replicator: fmt vet
	$(GOCMD) build $(GOARGS) -o $(REPLICATOR_OUTPUT_DIR)/replicator cmd/postgres_replicator/main.go

configure_replicator:
	scripts/check_env.sh GCP_PROJECT_ID GCP_DATA_TRANSFER_SERVICE BQ_PROJECT_ID PG_USER PG_PASS PG_DATABASE CLOUD_SQL_INSTANCE
	cp ../config/dashboard/postgres_replicator/config.yaml postgres_replicator
	scripts/inplace_envsubst.sh \
	  "postgres_replicator/app.yaml" \
	  "postgres_replicator/config.yaml"

configure_grafana:
	scripts/check_env.sh GCP_GRAFANA_SERVICE PG_USER PG_PASS PG_DATABASE GRAFANA_ADMIN_PASS CLOUD_SQL_INSTANCE
	cp -r ../config/dashboard/dashboards grafana
	scripts/inplace_envsubst.sh "grafana/app.yaml"

vet: ## Run go vet against code.
	$(GOCMD) vet ./...

fmt: ## Run go fmt against code.
	$(GOCMD) fmt ./...
