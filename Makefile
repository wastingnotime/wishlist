.PHONY: local dev test import-sqlite

# Start the local PostgreSQL database, API, and web app. Stop the app with
# Ctrl-C; the PostgreSQL container and its data stay available for the next run.
local:
	@set -eu; \
	if curl --silent --fail http://127.0.0.1:8080/healthz >/dev/null || curl --silent --fail http://127.0.0.1:5173/ >/dev/null; then \
		echo 'A local Wishlist process is already using port 8080 or 5173; stop it before running make local.' >&2; exit 1; \
	fi; \
	docker compose up -d --wait postgres; \
	export WISHLIST_DATABASE_URL="$${WISHLIST_DATABASE_URL:-postgres://wishlist:local-wishlist-only@127.0.0.1:5439/wishlist?sslmode=disable}"; \
	tmpdir=$$(mktemp -d); \
	api_pid=; \
	cleanup() { \
		if [ -n "$$api_pid" ]; then kill "$$api_pid" 2>/dev/null || true; wait "$$api_pid" 2>/dev/null || true; fi; \
		rm -rf "$$tmpdir"; \
	}; \
	trap cleanup EXIT; \
	trap 'exit 130' INT; \
	trap 'exit 143' TERM; \
	(cd apps/api && go build -o "$$tmpdir/wishlist-api" ./cmd/api); \
	(cd apps/api && exec "$$tmpdir/wishlist-api") & \
	api_pid=$$!; \
	ready=0; \
	for attempt in $$(seq 1 60); do \
		if ! kill -0 "$$api_pid" 2>/dev/null; then break; fi; \
		if curl --silent --fail http://127.0.0.1:8080/readyz >/dev/null; then ready=1; break; fi; \
		sleep 1; \
	done; \
	if [ "$$ready" -ne 1 ]; then echo 'Wishlist API failed to become ready.' >&2; exit 1; fi; \
	cd apps/web; \
	npm run dev -- --host 127.0.0.1 --port 5173

dev: local

# One-time import of the previous local SQLite database. The source is kept.
import-sqlite:
	@docker compose up -d --wait postgres
	@cd apps/api && WISHLIST_DATABASE_URL='postgres://wishlist:local-wishlist-only@127.0.0.1:5439/wishlist?sslmode=disable' go run ./cmd/initdb
	@python3 scripts/import-sqlite.py

# Run API and browser tests against a disposable PostgreSQL container.
test:
	@set -eu; \
	cleanup() { docker compose -f compose.test.yaml down --volumes; }; \
	trap cleanup EXIT; \
	docker compose -f compose.test.yaml up -d --wait postgres; \
	export WISHLIST_TEST_DATABASE_URL='postgres://wishlist_test:wishlist_test@127.0.0.1:55439/wishlist_test?sslmode=disable'; \
	(cd apps/api && go test ./...); \
	export WISHLIST_DATABASE_URL="$$WISHLIST_TEST_DATABASE_URL"; \
	export WISHLIST_API_URL='http://127.0.0.1:18080'; \
	export WISHLIST_API_ADDR='127.0.0.1:18080'; \
	export WNT_WEB_E2E_API_PORT=18080; \
	export WNT_WEB_E2E_WEB_PORT=15173; \
	(cd apps/web && npm run typecheck && npm run build && npm run test:e2e)
