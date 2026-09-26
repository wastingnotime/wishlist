.PHONY: local dev

# Start the local API and web app together. Stop with Ctrl-C; the API process
# and its temporary executable are cleaned up when the web server exits.
local:
	@set -eu; \
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
		if curl --silent --fail http://127.0.0.1:8080/readyz >/dev/null; then ready=1; break; fi; \
		if ! kill -0 "$$api_pid" 2>/dev/null; then break; fi; \
		sleep 1; \
	done; \
	if [ "$$ready" -ne 1 ]; then echo 'Wishlist API failed to become ready.' >&2; exit 1; fi; \
	cd apps/web; \
	npm run dev -- --host 127.0.0.1 --port 5173

dev: local
