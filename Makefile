.PHONY: start-local stop-local format-sql fmt lint check build build-all build-extractor build-collector build-api proto clean

COMPOSE_PROJECT := pginventory
COMPOSE_DIR := local_dev
BIN_DIR := bin

start-local:
	@echo "Starting local PostgreSQL environment..."
	cd $(COMPOSE_DIR) && \
		chmod 644 init-db.sql setup-script.sh && \
		docker-compose build && \
		docker-compose -p $(COMPOSE_PROJECT) down && \
		docker-compose -p $(COMPOSE_PROJECT) up -d --force-recreate --remove-orphans
	@echo "PostgreSQL is running!"

stop-local:
	@echo "Stopping local PostgreSQL environment..."
	cd $(COMPOSE_DIR) && docker-compose -p $(COMPOSE_PROJECT) down
	@echo "PostgreSQL stopped."

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
