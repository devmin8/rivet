# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

- **Persistent Dockerfile volumes** — Rivet now honors image-declared `VOLUME` paths with stable per-project Docker volumes so app data survives redeploys.

## [0.1.0] - 2026-05-26

### Added

- **Self-hosted PaaS** — Run Docker apps on a single VPS with a Go server, Vue dashboard, and Caddy reverse proxy. Plain Docker containers under the hood; no opaque runtime layer.
- **Production installer** — `install.sh` with guided setup for domain, secret key, persistent state, and pulling server/console images from GHCR. Supports interactive and non-interactive installs.
- **Automatic SSL** — Let's Encrypt certificates provisioned and renewed through Caddy for every project domain.
- **Domain routing and wake** — Caddy routes traffic to running containers. Sleeping projects hit Rivet's wake handler first, then route to the app once it is up.
- **Auto-sleep** — Idle projects sleep to free memory and CPU. Configurable per project from the dashboard or CLI during `rivetctl ship`. Background reconciler wakes on demand and sleeps idle containers.
- **Encrypted secrets** — Project env vars and secrets stored encrypted at rest. `RIVET_SECRET_*` keys in `.env` import as secrets during ship; injected into containers at start.
- **Per-app monitoring** — Real-time CPU, memory, and network stats for each container surfaced in the dashboard project list.
- **Web dashboard** — Sign in, list projects, create/deploy/start/stop/delete apps, manage env vars and secrets, and configure auto-sleep.
- **CLI (`rivetctl`)** — `signin`, `signup`, `ship`, `delete`, and `version`. `ship` builds the local Dockerfile for `linux/amd64` or `linux/arm64`, uploads the image, imports `.env`, and deploys.
- **Cross-platform CLI releases** — GitHub Actions workflow to build and publish `rivetctl` binaries for Linux, macOS, and Windows (amd64/arm64 where applicable) with checksums.
- **Release artifact filtering** — GitHub Release publishing attaches only `rivetctl-*` binaries and checksums.
- **Container images** — Server and console images published to GHCR with version, `latest`, and `sha-<commit>` tags, plus provenance, SBOM, and artifact attestations.
- **Marketing site and docs** — Public site at `getrivet.app` with install guide, feature overview, and dark mode.
