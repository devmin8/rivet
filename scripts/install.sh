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

show_help() {
	cat <<EOF
🚀 Rivet installer

Usage:
  curl -fsSL https://<your-rivet-website>/install.sh | sh
  curl -fsSL https://<your-rivet-website>/install.sh | RIVET_DOMAIN=rivet.example.com sh
  ./scripts/install.sh

Environment variables:
  🌐 RIVET_DOMAIN        Public domain for the Rivet server.
                      Required.
  📦 RIVET_HOME          Persistent state directory.
                      Default: \$HOME/.rivet
  🐳 RIVET_SERVER_IMAGE  Server image to pull.
                      Default: ghcr.io/devmin8/rivet-server:latest
  🖥️  RIVET_CONSOLE_IMAGE Console image to pull.
                      Default: ghcr.io/devmin8/rivet-console:latest
  🔌 CADDY_HTTP_BIND     Host-to-container HTTP port binding.
                      Default: 80:80
  🔐 CADDY_HTTPS_BIND    Host-to-container HTTPS port binding.
                      Default: 443:443
  🕸️  RIVET_NETWORK_NAME  Docker network name.
                      Default: rivet-network
  🧰 RIVET_DOCKER_SOCK   Docker socket path.
                      Default: /var/run/docker.sock
  🧹 RIVET_RESET_DATA    Set to 1 to delete persistent state before install.
                      Default: 0
  💬 RIVET_INTERACTIVE   Set to 0 to skip guided prompts.
                      Default: 1 when a terminal is available
EOF
}

is_interactive() {
	[ "${RIVET_INTERACTIVE:-1}" != "0" ] && [ -r /dev/tty ] && [ -w /dev/tty ]
}

prompt_value() {
	var_name=$1
	label=$2
	default_value=$3

	eval current_value=\${$var_name:-}
	if [ -n "$current_value" ]; then
		return
	fi

	printf '%s [%s]: ' "$label" "$default_value" >/dev/tty
	IFS= read -r input_value </dev/tty || input_value=

	if [ -z "$input_value" ]; then
		input_value=$default_value
	fi

	export "$var_name=$input_value"
}

prompt_required() {
	var_name=$1
	label=$2

	eval current_value=\${$var_name:-}
	if [ -n "$current_value" ]; then
		return
	fi

	while [ -z "$current_value" ]; do
		printf '%s: ' "$label" >/dev/tty
		IFS= read -r current_value </dev/tty || current_value=
	done

	export "$var_name=$current_value"
}

prompt_bool() {
	var_name=$1
	label=$2
	default_value=$3

	eval current_value=\${$var_name:-}
	if [ -n "$current_value" ]; then
		return
	fi

	default_label="y/N"
	if [ "$default_value" = "1" ]; then
		default_label="Y/n"
	fi

	printf '%s [%s]: ' "$label" "$default_label" >/dev/tty
	IFS= read -r input_value </dev/tty || input_value=

	case "$input_value" in
		y|Y|yes|YES)
			export "$var_name=1"
			;;
		n|N|no|NO)
			export "$var_name=0"
			;;
		*)
			export "$var_name=$default_value"
			;;
	esac
}

guided_setup() {
	if ! is_interactive; then
		return
	fi

	log "🚀 Rivet guided setup"
	log "Press Enter to accept the default shown in brackets."
	log ""

	prompt_required RIVET_DOMAIN "🌐 Domain"
	prompt_value RIVET_HOME "📦 Persistent state directory" "$HOME/.rivet"
	prompt_value RIVET_SERVER_IMAGE "🐳 Server image" "ghcr.io/devmin8/rivet-server:latest"
	prompt_value RIVET_CONSOLE_IMAGE "🖥️  Console image" "ghcr.io/devmin8/rivet-console:latest"
	prompt_bool RIVET_RESET_DATA "🧹 Delete existing Rivet data before installing?" "0"

	log ""
}

setup_configuration() {
	guided_setup

	# App environment
	APP_ENV=${APP_ENV:-"prod"}

	# Rivet config
	RIVET_HOME=${RIVET_HOME:-"$HOME/.rivet"}
	RIVET_DOMAIN=${RIVET_DOMAIN:-}
	RIVET_SERVER_DATA_DIR="$RIVET_HOME/server/data"
	RIVET_NETWORK_NAME=${RIVET_NETWORK_NAME:-"rivet-network"}
	RIVET_DOCKER_SOCK=${RIVET_DOCKER_SOCK:-"/var/run/docker.sock"}
	RIVET_SERVER_IMAGE=${RIVET_SERVER_IMAGE:-"ghcr.io/devmin8/rivet-server:latest"}
	RIVET_CONSOLE_IMAGE=${RIVET_CONSOLE_IMAGE:-"ghcr.io/devmin8/rivet-console:latest"}

	# Caddy config
	CADDY_DIR="$RIVET_HOME/caddy"
	CADDY_DATA_DIR="$CADDY_DIR/data"
	CADDY_CONFIG_DIR="$CADDY_DIR/config"
	CADDYFILE="$CADDY_DIR/Caddyfile"
	CADDY_HTTP_BIND=${CADDY_HTTP_BIND:-"80:80"}
	CADDY_HTTPS_BIND=${CADDY_HTTPS_BIND:-"443:443"}

	mkdir -p "$RIVET_SERVER_DATA_DIR" "$CADDY_DATA_DIR" "$CADDY_CONFIG_DIR"
}

ensure_host() {
	[ -n "$RIVET_DOMAIN" ] || fail "RIVET_DOMAIN is required, for example: RIVET_DOMAIN=rivet.example.com ./scripts/install.sh"
	[ -S "$RIVET_DOCKER_SOCK" ] || fail "Docker socket not found at $RIVET_DOCKER_SOCK"
}

cleanup() {
	log "🧹 Cleaning up old Rivet containers"

	docker rm -f rivet-server rivet-console rivet-caddy 2>/dev/null || true
	docker network rm "$RIVET_NETWORK_NAME" 2>/dev/null || true

	if [ "${RIVET_RESET_DATA:-0}" = "1" ]; then
		log "🧹 Removing persistent Rivet state"
		rm -rf "$RIVET_SERVER_DATA_DIR" "$CADDY_DATA_DIR" "$CADDY_CONFIG_DIR"
	fi

	mkdir -p "$RIVET_SERVER_DATA_DIR" "$CADDY_DATA_DIR" "$CADDY_CONFIG_DIR"
}

write_caddyfile() {
	log "📝 Writing $CADDYFILE"

	cat >"$CADDYFILE" <<EOF
{
	admin 0.0.0.0:2019
}
EOF
}

ensure_network() {
	if ! docker network inspect "$RIVET_NETWORK_NAME" >/dev/null 2>&1; then
		log "🕸️  Creating Docker network $RIVET_NETWORK_NAME"
		docker network create "$RIVET_NETWORK_NAME" >/dev/null
	fi
}

pull_rivet_server() {
	log "🐳 Pulling $RIVET_SERVER_IMAGE"
	docker pull "$RIVET_SERVER_IMAGE" >/dev/null
}

pull_rivet_console() {
	log "🐳 Pulling $RIVET_CONSOLE_IMAGE"
	docker pull "$RIVET_CONSOLE_IMAGE" >/dev/null
}

start_rivet_server() {
	log "🚀 Starting rivet-server"

	docker run -d \
		--name rivet-server \
		--network "$RIVET_NETWORK_NAME" \
		--restart unless-stopped \
		-e GO_ENV=production \
		-e PORT=3000 \
		-e DOMAIN="$RIVET_DOMAIN" \
		-e APP_ENV="$APP_ENV" \
		-e DB_PATH=/data/rivet.db \
		-v "$RIVET_SERVER_DATA_DIR:/data" \
		-v "$RIVET_DOCKER_SOCK:/var/run/docker.sock" \
		"$RIVET_SERVER_IMAGE" >/dev/null
}

start_rivet_console() {
	log "🚀 Starting rivet-console"

	docker run -d \
		--name rivet-console \
		--network "$RIVET_NETWORK_NAME" \
		--restart unless-stopped \
		"$RIVET_CONSOLE_IMAGE" >/dev/null
}

start_caddy() {
	log "🌐 Starting rivet-caddy"

	docker run -d \
		--name rivet-caddy \
		--network "$RIVET_NETWORK_NAME" \
		--restart unless-stopped \
		-e CADDY_ADMIN=0.0.0.0:2019 \
		-p "$CADDY_HTTP_BIND" \
		-p "$CADDY_HTTPS_BIND" \
		-v "$CADDYFILE:/etc/caddy/Caddyfile:ro" \
		-v "$CADDY_DATA_DIR:/data" \
		-v "$CADDY_CONFIG_DIR:/config" \
		caddy:2-alpine >/dev/null
}

main() {
	case "${1:-}" in
		-h|--help)
			show_help
			exit 0
			;;
	esac

	setup_configuration
	need_cmd docker
	ensure_host

	cleanup
	pull_rivet_server
	pull_rivet_console
	write_caddyfile
	ensure_network
	# Caddy must be up before rivet-server: the server POSTs to rivet-caddy on startup.
	start_caddy
	start_rivet_console
	start_rivet_server

	log "✅ Rivet is running at https://$RIVET_DOMAIN"
	log "📦 Persistent state is in $RIVET_HOME"
	log ""
	log "💡 Tip: run with RIVET_RESET_DATA=1 only when you want a clean install"
}

main "$@"
