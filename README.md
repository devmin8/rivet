# rivet

A simple self hosted paas in Go.

## Local dev

Run the Docker Compose dev setup from the repo root:

```bash
docker compose up --build
```

To use a different local HTTP port:

```bash
CADDY_PORT=8080 docker compose up --build
```

Then open:

```text
http://rivet-server.localhost
```

Dev state is stored under `~/.rivet`. To stop and remove the dev containers:

```bash
docker compose down
```

To also remove local dev data:

```bash
rm -rf ~/.rivet
```

The dev compose file uses a static `RIVET_SECRET_KEY` for local development only. Production installs should use the hosted installer with a real secret key.
