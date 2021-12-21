GOCMD ?= go
GOARGS= -trimpath
REPLICATOR_OUTPUT_DIR ?= bin
CONFIG_TEMPLATE_DIR ?= config
GRAFANA_CONTAINER_OUTPUT_DIR ?= containers/grafana
REPLICATOR_CONTAINER_OUTPUT_DIR ?= containers/replicator

configure-grafana:
	scripts/check_env.sh GCP_GRAFANA_SERVICE PG_USER PG_PASS PG_DATABASE GRAFANA_ADMIN_PASS CLOUD_SQL_INSTANCE
	mkdir -p $(GRAFANA_CONTAINER_OUTPUT_DIR)
	cp -r $(CONFIG_TEMPLATE_DIR)/grafana/dashboards/default $(GRAFANA_CONTAINER_OUTPUT_DIR)/dashboards
	cp -r grafana/* $(GRAFANA_CONTAINER_OUTPUT_DIR)
	cp $(CONFIG_TEMPLATE_DIR)/grafana/Dockerfile $(GRAFANA_CONTAINER_OUTPUT_DIR)/Dockerfile
	envsubst < $(CONFIG_TEMPLATE_DIR)/grafana/app.yaml > $(GRAFANA_CONTAINER_OUTPUT_DIR)/app.yaml

configure-replicator:
	scripts/check_env.sh GCP_PROJECT_ID GCP_DATA_TRANSFER_SERVICE BQ_PROJECT_ID PG_USER PG_PASS PG_DATABASE CLOUD_SQL_INSTANCE
	mkdir -p containers/replicator
	cp $(CONFIG_TEMPLATE_DIR)/postgres_replicator/Dockerfile $(REPLICATOR_CONTAINER_OUTPUT_DIR)/Dockerfile
	envsubst < $(CONFIG_TEMPLATE_DIR)/postgres_replicator/app.yaml > $(REPLICATOR_CONTAINER_OUTPUT_DIR)/app.yaml
	envsubst < $(CONFIG_TEMPLATE_DIR)/postgres_replicator/default/config.yaml > $(REPLICATOR_CONTAINER_OUTPUT_DIR)/config.yaml

replicator: fmt vet
	$(GOCMD) build $(GOARGS) -o $(REPLICATOR_OUTPUT_DIR)/replicator cmd/postgres_replicator/main.go

vet: ## Run go vet against code.
	$(GOCMD) vet ./...

fmt: ## Run go fmt against code.
	$(GOCMD) fmt ./...

test: fmt vet
	$(GOCMD) test ./...
