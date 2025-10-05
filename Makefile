.PHONY: dependencies build local

start-local:
	cd local_dev && chmod 644 init-db.sql && docker-compose build && docker-compose -p pginventory down && docker-compose -f docker-compose.yml -p pginventory up -d --force-recreate --remove-orphans

stop-local:
	cd local_dev && docker-compose -p pginventory down

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
