.PHONY: start-local stop-local format-sql

COMPOSE_PROJECT := pginventory
COMPOSE_DIR := local_dev

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
