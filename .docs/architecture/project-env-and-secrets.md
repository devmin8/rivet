# Project Environment Variables And Secrets

Rivet supports project runtime environment variables and secrets. The design follows the simple PaaS pattern used by Dokku, Coolify, CapRover, and Piku: applications receive configuration as normal container environment variables.

This architecture covers runtime environment configuration. Build-time secrets, Docker build arguments, secret file mounts, and CI-provider-specific secret import are adjacent capabilities with different lifecycle and security properties.

The intent is to improve the normal self-hosted flow without requiring users to change their applications:

```txt
common baseline:  edit .env on server -> docker run --env-file .env
Rivet:            import/manage env in Rivet -> encrypted secrets -> Docker runtime env
```

## External Pattern

- Dokku, CapRover, Piku, Tsuru, and Coolify all expose app runtime config as environment variables.
- Coolify also supports Docker BuildKit secrets for build-time secrets, but that is separate from runtime env and outside this architecture.
- Docker/Kubernetes can mount secrets as files, but that requires more app compatibility work and is future work for Rivet.

Refs:

- Dokku env vars: https://dokku.com/docs/configuration/environment-variables/
- Coolify env vars: https://coolify.io/docs/knowledge-base/environment-variables
- Coolify Docker build secrets: https://coolify.io/docs/knowledge-base/environment-variables#docker-build-secrets
- Docker build secrets: https://docs.docker.com/build/building/secrets/
- Docker Swarm secrets: https://docs.docker.com/engine/swarm/secrets/
- Docker Compose secrets: https://docs.docker.com/compose/how-tos/use-secrets/
- Kubernetes Secrets: https://kubernetes.io/docs/concepts/configuration/secret/
- Twelve-Factor config: https://www.12factor.net/config

## Chosen Approach

Rivet is the runtime environment source of truth for a project.

- Normal env vars are visible/editable in Rivet.
- Secrets are encrypted in Rivet, write-only after save, and never returned raw by the API.
- Both normal env vars and secrets are injected into the project container as normal Docker env vars.
- During `rivet ship`, the CLI can import local `.env` values into Rivet before building, uploading, and deploying.

This approach fits Rivet's self-hosted model because:

- It matches how most self-hosted PaaS users already deploy apps: runtime env vars.
- It works with existing apps without code changes.
- It is a step safer than keeping plaintext project `.env` files on the server.
- It avoids the dangerous path of baking `.env` values into Docker images.
- A fully compromised host can usually read runtime env, mounted files, Docker state, or memory anyway. The design primarily reduces accidental leaks through git, image layers, API responses, UI reveal, and logs.

### Decision Rationale

The central tradeoff is compatibility versus stronger delivery isolation. Runtime environment variables are not the strongest possible secret-delivery mechanism: a user with enough host, Docker, or process-level access can usually inspect the effective runtime environment. File-mounted secrets can reduce some environment-specific leak paths, and Docker BuildKit secrets are the right model for build-time credentials because build arguments and build environment variables can persist into image metadata or layers.

Those alternatives solve different problems than Rivet's baseline self-hosted deployment flow. Most applications already read runtime configuration from environment variables, and this is also the model recommended by the Twelve-Factor App for deploy-specific config. Docker and Kubernetes both support runtime environment injection, while Docker's stronger secret primitives either target Swarm services or require application changes such as reading from `/run/secrets/*` or `*_FILE` variables.

For Rivet, the highest-value first architecture is therefore a managed runtime env source of truth:

- It preserves zero-code-change compatibility for existing apps.
- It prevents secrets from being baked into Docker images.
- It removes the need to keep plaintext `.env` files on the server as the operational source of truth.
- It gives Rivet a single project-scoped API/UI surface for review, replacement, deletion, and deploy-time injection.
- It narrows the primary leak channels Rivet controls: git commits, Docker build context, image layers, API responses, Console reveal, and logs.

This does not claim runtime env vars are a complete secret boundary against a compromised host. It is a pragmatic boundary for accidental disclosure and normal self-hosted operations, with clear room to add build secrets and file-mounted runtime secrets later for applications that need stronger isolation.

## `rivet ship` Flow

When a user runs:

```sh
rivet ship
```

the CLI uses the existing project selection/build/upload/deploy flow, with an optional runtime env import step before Docker build starts:

1. Select or create the project using the existing flow.
2. Look for a file named `.env` in the current project directory.
3. If `.env` does not exist, continue with the existing build/upload/deploy flow.
4. If `.env` exists, parse it using dotenv semantics.
5. For each parsed key:
   - If key starts with `RIVET_SECRET_`, strip that prefix and save the value as a secret runtime env var.
   - Otherwise, save the key/value as a normal runtime env var.
6. Call the Rivet project env API before Docker build starts.
7. If the env import succeeds:
   - Run the existing Docker build.
   - Save/upload the image tar.
   - Deploy the project.
   - Server starts the container with imported runtime env attached.
8. If the env import fails:
   - Print a warning and keep the error.
   - Still run the existing Docker build.
   - Still save/upload the image tar.
   - Skip deploy.
   - Print a final message telling the user to update env/secrets in the Console, then deploy/start the project.

Example local `.env`:

```dotenv
APP_URL=https://example.com
DATABASE_URL=postgres://user:pass@example/db
RIVET_SECRET_STRIPE_API_KEY=sk_live_123
```

Imported into Rivet as:

```txt
APP_URL         plain env      value: https://example.com
DATABASE_URL    plain env      value: postgres://user:pass@example/db
STRIPE_API_KEY  secret env     value: sk_live_123
```

Container receives:

```txt
APP_URL=https://example.com
DATABASE_URL=postgres://user:pass@example/db
STRIPE_API_KEY=sk_live_123
```

Important behavior:

- This `.env` import is only to seed/update Rivet runtime env before deploy.
- `RIVET_SECRET_` is only a local `.env` import marker.
- The final runtime key must not include `RIVET_SECRET_`.
- Imported secret values are not printed.
- If import fails after project selection, the command still builds and uploads the image, but skips deploy so the user can fix env/secrets in the Console first.

## Server API

Rivet exposes project-scoped endpoints for environment management:

```txt
GET    /api/v1/projects/:id/env
PUT    /api/v1/projects/:id/env/:key
DELETE /api/v1/projects/:id/env/:key
```

`PUT` request shape:

```json
{
  "kind": "plain",
  "value": "https://example.com"
}
```

or:

```json
{
  "kind": "secret",
  "value": "sk_live_123"
}
```

`GET` responses include normal values but not secret values:

```json
{
  "items": [
    {
      "key": "APP_URL",
      "kind": "plain",
      "value": "https://example.com",
      "has_value": true,
      "updated_at": "..."
    },
    {
      "key": "STRIPE_API_KEY",
      "kind": "secret",
      "value": null,
      "has_value": true,
      "updated_at": "..."
    }
  ]
}
```

API behavior:

- The endpoints use existing auth/session/project ownership checks.
- Keys are validated with `^[A-Za-z_][A-Za-z0-9_]*$`.
- Keys starting with reserved prefix `RIVET_` are rejected, except the CLI import marker `RIVET_SECRET_` before it is stripped.
- `PUT` replaces the existing value for that project/key.
- `DELETE` removes the value for that project/key.
- API responses must never include decrypted secret values.

## Storage

Project environment values are stored in a project-scoped table:

```txt
project_env_vars
- id
- project_id
- key
- kind              # plain | secret
- value            # plaintext only for plain vars
- encrypted_value  # ciphertext only for secrets
- key_version      # start with 1
- created_at
- updated_at
```

Storage behavior:

- `unique(project_id, key)`.
- Multiple projects may use the same env key.
- Plain env vars may be stored in `value`.
- Secret env vars must be encrypted before storage.
- Use a server encryption key from config, for example `RIVET_SECRET_KEY`.
- On Rivet server startup, `RIVET_SECRET_KEY` is read and validated so encryption/decryption is ready before handling secret writes or project starts.
- If `RIVET_SECRET_KEY` is missing or invalid, fail clearly before accepting secret create/update or starting projects that have secrets.
- Secrets are not decrypted in bulk on Rivet server startup.
- Plaintext secrets are not cached in long-lived process state.

## Runtime Container Start

When starting, deploying, or waking a project, Rivet resolves the current project environment and attaches it to the Docker container:

1. Load project env rows by `project_id`.
2. For plain env vars, use `key=value`.
3. For secret env vars, decrypt just-in-time in memory and use `key=value`.
4. Pass the final env list to Docker `ContainerCreate` via `container.Config.Env`.
5. Drop plaintext secret values after the Docker create/start call returns.
6. Avoid logging raw env values.
7. Avoid returning raw secret values in errors.

Changing env values does not update an already-running container. The project must be restarted/redeployed so Docker recreates the container with the new env list.

## Console UI

The Console includes a project env management surface.

Normal env vars:

- Show key and value.
- Allow create, edit, delete.

Secret env vars:

- Show key, kind, updated time, and whether a value exists.
- Hide the raw value after save.
- Allow create, replace, delete.

UI states:

- loading
- error
- empty
- saving
- deleting

After env changes, show that restart/redeploy is required for a running container to receive the update.

## Out Of Scope

The runtime env architecture intentionally excludes:

- Docker build args from `.env`.
- BuildKit `--secret`.
- `.dockerignore` patching.
- Copying or merging `.env` into the Docker build context.
- Runtime secret file mounts.
- `*_FILE` helpers.
- GitHub Actions-specific env/secret import.
- Hot-patching env into already-running containers.

Future work can add BuildKit secrets to `rivet ship`, GitHub Actions workflows, env revision tracking, key rotation, and optional file-mounted secrets.
