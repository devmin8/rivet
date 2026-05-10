#!/usr/bin/env sh
# Exit on error (-e) and fail on undefined variables (-u)
set -eu

log() {
	printf '%s\n' "$*"
}

fail() {
	printf 'error: %s\n' "$*" >&2
	exit 1
}

need_cmd() {
	command -v "$1" >/dev/null 2>&1 || fail "$1 is required"
}

setup_configuration() {
	REPO_ROOT=$(pwd)

	# App environment
	APP_ENV=${APP_ENV:-"dev"}

	# Rivet config
	RIVET_HOME=${RIVET_HOME:-"$HOME/.rivet"}
	RIVET_DOMAIN=${RIVET_DOMAIN:-"rivet-server.localhost"}
	RIVET_SERVER_DATA_DIR="$RIVET_HOME/server/data"
	RIVET_NETWORK_NAME=${RIVET_NETWORK_NAME:-"rivet-network"}
	RIVET_DOCKER_SOCK=${RIVET_DOCKER_SOCK:-"/var/run/docker.sock"}

	# Caddy config
	CADDY_DIR="$RIVET_HOME/caddy"
	CADDY_DATA_DIR="$CADDY_DIR/data"
	CADDY_CONFIG_DIR="$CADDY_DIR/config"
	CADDY_LOG_DIR="$CADDY_DIR/logs"
	CADDYFILE="$CADDY_DIR/Caddyfile"

	mkdir -p "$RIVET_SERVER_DATA_DIR" "$CADDY_DATA_DIR" "$CADDY_CONFIG_DIR" "$CADDY_LOG_DIR"
}

ensure_repository() {
	[ -f "$REPO_ROOT/go.mod" ] || fail "go.mod not found at $REPO_ROOT"
	[ -d "$REPO_ROOT/cmd/rivet-server" ] || fail "cmd/rivet-server not found at $REPO_ROOT"
	[ -f "$REPO_ROOT/Dockerfile" ] || fail "Dockerfile not found at $REPO_ROOT"
	[ -f "$REPO_ROOT/console/Dockerfile" ] || fail "console/Dockerfile not found at $REPO_ROOT"
	[ -f "$REPO_ROOT/console/package.json" ] || fail "console/package.json not found at $REPO_ROOT"
	[ -S "$RIVET_DOCKER_SOCK" ] || fail "Docker socket not found at $RIVET_DOCKER_SOCK"
}

cleanup() {
	log "🧹 Cleaning up"

	# Remove Rivet containers
	docker rm -f rivet-server rivet-console rivet-caddy 2>/dev/null || true

	# Remove Rivet network
	docker network rm "$RIVET_NETWORK_NAME" 2>/dev/null || true

	# Remove local state
	rm -rf "$RIVET_SERVER_DATA_DIR" "$CADDY_DATA_DIR" "$CADDY_CONFIG_DIR" "$CADDY_LOG_DIR"

	# Optional full Docker prune (ONLY when explicitly enabled)
	if [ "${RIVET_FULL_PRUNE:-0}" = "1" ]; then
		log "⚠️  Running full Docker prune (dev only)"
		docker system prune -a --volumes -f >/dev/null
	fi

	# Recreate dirs
	mkdir -p "$RIVET_SERVER_DATA_DIR" "$CADDY_DATA_DIR" "$CADDY_CONFIG_DIR" "$CADDY_LOG_DIR"
}

write_caddyfile() {
	log "Writing $CADDYFILE"

	cat >"$CADDYFILE" <<EOF
{
	admin 0.0.0.0:2019
}
EOF
}

ensure_network() {
	if ! docker network inspect "$RIVET_NETWORK_NAME" >/dev/null 2>&1; then
		log "Creating Docker network $RIVET_NETWORK_NAME"
		docker network create "$RIVET_NETWORK_NAME" >/dev/null
	fi
}

start_rivet_server() {
	log "Starting rivet-server"

	docker run -d \
		--name rivet-server \
		--network "$RIVET_NETWORK_NAME" \
		--restart unless-stopped \
		-e PORT=3000 \
		-e DOMAIN=$RIVET_DOMAIN \
		-e APP_ENV=$APP_ENV \
		-e DB_PATH=/data/rivet.db \
		-e CADDY_ACCESS_LOG_PATH=/var/log/rivet-caddy/access.log \
		-v "$RIVET_SERVER_DATA_DIR:/data" \
		-v "$CADDY_LOG_DIR:/var/log/rivet-caddy:ro" \
		-v "$RIVET_DOCKER_SOCK:/var/run/docker.sock" \
		rivet-server:$APP_ENV >/dev/null
}

start_caddy() {
	log "Starting rivet-caddy"

	docker run -d \
		--name rivet-caddy \
		--network "$RIVET_NETWORK_NAME" \
		--restart unless-stopped \
		-e CADDY_ADMIN=0.0.0.0:2019 \
		-p "127.0.0.1:80:80" \
		-v "$CADDYFILE:/etc/caddy/Caddyfile:ro" \
		-v "$CADDY_DATA_DIR:/data" \
		-v "$CADDY_CONFIG_DIR:/config" \
		-v "$CADDY_LOG_DIR:/var/log/caddy" \
		caddy:2-alpine >/dev/null
}

build_rivet_server() {
	log "Building rivet-server:$APP_ENV from $REPO_ROOT"
	docker build -t rivet-server:$APP_ENV "$REPO_ROOT"
}

build_rivet_console() {
	log "Building rivet-console:$APP_ENV from $REPO_ROOT/console"
	docker build \
		-t rivet-console:$APP_ENV \
		"$REPO_ROOT/console"
}

start_rivet_console() {
	log "Starting rivet-console"

	docker run -d \
		--name rivet-console \
		--network "$RIVET_NETWORK_NAME" \
		--restart unless-stopped \
		rivet-console:$APP_ENV >/dev/null
}

main() {
	setup_configuration
	ensure_repository
	need_cmd docker

	cleanup
	build_rivet_server
	build_rivet_console
	write_caddyfile
	ensure_network
	# Caddy must be up before rivet-server: the server POSTs to rivet-caddy on startup.
	start_caddy
	start_rivet_console
	start_rivet_server

	log "✅ Rivet is running at http://$RIVET_DOMAIN"
	log "📦 Persistent state is in $RIVET_HOME"
	log ""
	log "Tip: run with RIVET_FULL_PRUNE=1 for a completely clean Docker state"
}

main "$@"
