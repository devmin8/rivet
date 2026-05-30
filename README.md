# Rivet

<p align="center">
  <a href="https://getrivet.app">
    <img src="./site/public/favicon.svg" alt="Rivet logo" width="96" height="96" />
  </a>
</p>

Open-source, self-hosted PaaS for a single VPS.

Website: [getrivet.app](https://getrivet.app)

Rivet turns a plain Linux server into a small platform for Docker apps. Install the control plane once, then deploy Dockerfile-based apps from the CLI or run public images from the console. Rivet handles HTTPS, routing, runtime env vars, encrypted secrets, auto-sleep, and lightweight per-app monitoring.

It is intentionally not a managed cloud replacement. You still own the server, Docker runtime, data, and cost model. Rivet adds the platform layer that makes one VPS feel closer to Heroku or Railway without hiding the infrastructure underneath.

## What Rivet Does

- Deploys Dockerfile apps from your machine with `rivet ship`
- Runs public Docker images from the web console
- Routes app domains through Caddy
- Provisions and renews HTTPS certificates automatically
- Stores project secrets encrypted at rest
- Injects runtime env vars into containers
- Sleeps idle containers and wakes them on HTTP traffic
- Shows CPU, memory, network, and process stats for running projects
- Keeps project lifecycle state reconciled with Docker reality

## When To Use It

Rivet is a good fit when you want:

- one VPS instead of a fleet
- a self-hosted deployment workflow
- Docker-native apps with normal environment variables
- predictable infrastructure costs
- a dashboard and CLI without adopting Kubernetes
- infrastructure you can inspect over SSH

It is not trying to be a multi-region scheduler, hosted database provider, or full observability platform. Those may arrive as integrations or later features, but the current design is deliberately small.

## Quick Start

You need a Linux VPS with Docker Engine installed, ports `80` and `443` open, and a domain pointed at the server.

Install Rivet on the VPS:

```bash
curl -fsSL https://getrivet.app/install.sh | sh
```

The installer starts the Rivet API server, web console, and Caddy router. When it finishes, open your Rivet domain in the browser and create the first user. The first user is the admin.

Install the CLI on your local machine from the GitHub releases page, or build it from source:

```bash
git clone https://github.com/devmin8/rivet.git
cd rivet
go build -o rivet ./cmd/rivetctl
sudo mv rivet /usr/local/bin/
```

Point the CLI at your server:

```bash
export RIVET_SERVER_URL=https://rivet.example.com
```

Sign in and deploy an app from a directory with a `Dockerfile`:

```bash
rivet signin
cd my-app
rivet ship
```

During `rivet ship`, the CLI asks for the project name, public domain, container port, and optional auto-sleep setting. Your app domain must resolve to the same VPS IP as the Rivet control plane.

Full user docs live at [getrivet.app/docs.html](https://getrivet.app/docs.html).

## Environment Variables And Secrets

Rivet treats runtime configuration as project-scoped environment variables.

If your app has a `.env` file, `rivet ship` can import it before deploy:

```dotenv
APP_URL=https://api.example.com
DATABASE_URL=postgres://user:pass@example/db
RIVET_SECRET_STRIPE_API_KEY=sk_live_123
```

Normal keys are stored as plain runtime env vars. Keys prefixed with `RIVET_SECRET_` are stored as encrypted secrets, and the prefix is stripped before injection:

```text
STRIPE_API_KEY=sk_live_123
```

Add `.env` to `.dockerignore` so secrets are not copied into your Docker build context.

## Local Development

Run the Docker Compose dev setup from the repo root:

```bash
docker compose up --build
```

Then open:

```text
http://rivet-server.localhost
```

Use another local HTTP port if needed:

```bash
CADDY_PORT=8080 docker compose up --build
```

Dev state is stored under `~/.rivet`.

Stop the dev stack:

```bash
docker compose down
```

Remove local dev data:

```bash
rm -rf ~/.rivet
```

The dev compose file uses a static `RIVET_SECRET_KEY` for local development only. Production installs must use a real secret key.

## Repository Layout

```text
cmd/
  rivet-server/      # API server entrypoint
  rivetctl/          # CLI entrypoint

internal/
  api/               # shared API DTOs/constants
  caddy/             # Caddy config and routing helpers
  cli/               # rivet CLI commands
  docker/            # Docker client/runtime integration
  server/            # config, database, services, web routes
  validation/        # validation helpers

console/             # Vue 3 web console
site/                # public marketing/docs site
.docs/               # architecture and ops notes
test-apps/           # local test apps and seed scripts
```

## Architecture Notes

Rivet has three main runtime pieces:

- `rivet-server`: Go API server, reconciler, project runtime services, auth, env/secrets, stats, and Caddy coordination
- `rivet-console`: Vue web console for projects, env vars, secrets, lifecycle actions, and stats
- `rivet-caddy`: Caddy reverse proxy for the control plane and deployed apps

Project lifecycle state is stored in the database. Docker is the runtime reality. A reconciler loop compares the persisted project state with Docker containers and marks projects `running`, `stopped`, `sleeping`, `deploying`, or `failed` as appropriate.

Runtime stats are intentionally lightweight. The dashboard reads project metadata and Docker stats separately so the project list remains fast even when Docker metrics are stale or unavailable.

Useful design docs:

- [project runtime reconciliation](./.docs/architecture/project-runtime-reconciliation.md)
- [project runtime stats](./.docs/architecture/project-runtime-stats.md)
- [project env and secrets](./.docs/architecture/project-env-and-secrets.md)
- [console structure](./.docs/architecture/rivet-console-structure.md)
- [release process](./.docs/ops/release.md)

## Development Notes

Backend:

```bash
go build ./cmd/rivet-server
go build ./cmd/rivetctl
```

Console:

```bash
bun install --cwd console
bun run --cwd console dev
```

The marketing site lives in `site/`. It is a Vite + TypeScript + Tailwind v4 static site for the public landing page and docs.

After adding, removing, or renaming shared console components under `console/src/components/**`, update component declarations:

```bash
bun run --cwd console components:dts
```

## Release

Rivet publishes releases through `.github/workflows/release.yml`. The workflow creates a version tag, updates the changelog, builds server and console images, builds CLI binaries, generates checksums, and publishes a GitHub release.

Release notes should be prepared under `CHANGELOG.md` before running the workflow.

## License

MIT
