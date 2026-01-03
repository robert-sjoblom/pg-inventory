.PHONY: start-local stop-local local-certs local-clean local-logs local-status local-pgbackrest-info local-pgbackrest-backup-full local-pgbackrest-backup-incr format-sql fmt lint check build build-all build-extractor build-collector build-api proto clean test

COMPOSE_PROJECT := pginventory
COMPOSE_DIR := local_dev
BIN_DIR := bin

local-certs:
	@echo "Generating TLS certificates..."
	@$(COMPOSE_DIR)/scripts/generate-certs.sh
	@echo "Certificates ready."

start-local: local-certs
	@echo "Starting local environment..."
	cd $(COMPOSE_DIR) && docker compose -p $(COMPOSE_PROJECT) up -d
	@echo ""
	@echo "Waiting for containers to be ready..."
	@sleep 5
	@echo "Creating pgBackRest stanza..."
	@docker exec -u postgres pg-inventory-primary pgbackrest --stanza=main --log-level-console=info stanza-create 2>/dev/null || echo "(stanza already exists or will retry)"
	@echo "Running initial full backup..."
	@docker exec -u postgres pg-inventory-primary pgbackrest --stanza=main --type=full backup 2>/dev/null || echo "(backup will retry later)"
	@echo ""
	@echo "Services:"
	@echo "  Catalog (TimescaleDB): localhost:5432"
	@echo "  Primary:              localhost:5433"
	@echo "  Replica 1:            localhost:5434"
	@echo "  Replica 2:            localhost:5435"
	@echo "  MinIO Console:        http://localhost:9001"
	@echo ""
	@echo "Connect as pgmonitor (with client cert):"
	@echo "  psql \"host=localhost port=5433 dbname=postgres user=pgmonitor sslmode=verify-full sslcert=$(COMPOSE_DIR)/certs/client.crt sslkey=$(COMPOSE_DIR)/certs/client.key sslrootcert=$(COMPOSE_DIR)/certs/ca.crt\""

stop-local:
	@echo "Stopping local environment..."
	cd $(COMPOSE_DIR) && docker compose -p $(COMPOSE_PROJECT) down
	@echo "Stopped."

local-clean:
	@echo "Stopping and removing volumes..."
	cd $(COMPOSE_DIR) && docker compose -p $(COMPOSE_PROJECT) down -v
	@echo "Clean. Run 'make start-local' to recreate."

local-logs:
	cd $(COMPOSE_DIR) && docker compose -p $(COMPOSE_PROJECT) logs -f

local-status:
	@echo "==> Container status"
	@cd $(COMPOSE_DIR) && docker compose -p $(COMPOSE_PROJECT) ps
	@echo ""
	@echo "==> Replication status (primary)"
	@docker exec pg-inventory-primary psql -U postgres -c "SELECT client_addr, state, sent_lsn, write_lsn, flush_lsn, replay_lsn FROM pg_stat_replication;" 2>/dev/null || echo "(primary not ready)"

local-pgbackrest-info:
	@echo "==> pgBackRest backup info"
	@cd $(COMPOSE_DIR) && docker compose -p $(COMPOSE_PROJECT) exec pgbackrest-client pgbackrest --stanza=main --output=json info

local-pgbackrest-stanza-create:
	@echo "==> Creating pgBackRest stanza"
	@docker exec -u postgres pg-inventory-primary pgbackrest --stanza=main --log-level-console=info stanza-create
	@echo "Stanza created successfully."

local-pgbackrest-backup-full:
	@echo "==> Running full backup..."
	@cd $(COMPOSE_DIR) && docker compose -p $(COMPOSE_PROJECT) exec -u postgres pgbackrest-client pgbackrest --stanza=main --type=full backup
	@echo "Full backup complete."

local-pgbackrest-backup-incr:
	@echo "==> Running incremental backup..."
	@cd $(COMPOSE_DIR) && docker compose -p $(COMPOSE_PROJECT) exec -u postgres pgbackrest-client pgbackrest --stanza=main --type=incr backup
	@echo "Incremental backup complete."

format-sql:
	@latest_tag=$$(git describe --tags --abbrev=0 2>/dev/null || echo HEAD~1); \
	for file in $$(git diff --name-only $$latest_tag | grep '\.sql$$' || true); do \
		if [ -f "$$file" ]; then \
			echo "Formatting $$file"; \
			orig_perm=$$(stat -c '%a' "$$file"); \
			chmod u+rw,go-rwx "$$file"; \
			docker run --rm -u $$(id -u):$$(id -g) -v $$(pwd):/workspace sqlfluff/sqlfluff fix "/workspace/$$file" --dialect postgres; \
			chmod $$orig_perm "$$file"; \
		fi; \
	done

fmt:
	@echo "Formatting Go code..."
	go fmt ./...
	@echo "Format complete."

lint:
	@echo "Linting Go code..."
	golangci-lint run ./...
	@echo "Lint complete."

test:
	@echo "Running unit tests..."
	go test -v ./...
	@echo "Tests complete."

test-integration:
	@echo "Running integration tests..."
	go test -tags=integration -v ./...
	@echo "Integration tests complete."

check: fmt lint
	@echo "All checks complete."

build-all: build-extractor build-collector build-api proto
	@echo "All binaries built in $(BIN_DIR)/"

build-extractor:
	@echo "Building extractor..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/extractor ./cmd/extractor

build-extractor-dev:
	@echo "Building extractor (dev mode with reflection)..."
	@mkdir -p $(BIN_DIR)
	go build -tags dev -o $(BIN_DIR)/extractor ./cmd/extractor

build-collector:
	@echo "Building collector..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/collector ./cmd/collector

build-api:
	@echo "Building api-service..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/api-service ./cmd/api-service

proto:
	@echo "Compiling protobuf..."
	@mkdir -p gen/extractor/v1
	protoc \
		--go_out=gen \
		--go_opt=module=github.com/robert-sjoblom/pg-inventory/gen \
		--go-grpc_out=gen \
		--go-grpc_opt=module=github.com/robert-sjoblom/pg-inventory/gen \
		--proto_path=. api/extractor/v1/extractor.proto
	@echo "Proto files generated"

build: build-all

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	@echo "Clean complete."
