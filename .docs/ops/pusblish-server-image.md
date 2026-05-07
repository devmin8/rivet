# GHA - Publish Images

Rivet publishes the server and console as separate Docker images in GitHub Container Registry (GHCR).

Workflow files:

- `.github/workflows/publish-server-image.yml`
- `.github/workflows/publish-console-image.yml`

The workflows stay separate because the server and console have different build contexts, runtime dependencies, and release cadence. Publishing them independently keeps rebuilds scoped to the artifact that changed, reduces release blast radius, and matches the installer contract where `RIVET_SERVER_IMAGE` and `RIVET_CONSOLE_IMAGE` are configured separately.

## When It Runs

The workflow only runs manually:

```yaml
on:
  workflow_dispatch:
```

Run the matching workflow from the GitHub Actions UI when an image needs to be published.

## What It Publishes

The image is pushed to:

```text
ghcr.io/${{ github.repository }}-server
```

For this repo, the image name follows the repository name and adds `-server`.

The console workflow pushes to:

```text
ghcr.io/${{ github.repository }}-console
```

For this repo, the image name follows the repository name and adds `-console`.

## Console Image Workflow

Workflow file: `.github/workflows/publish-console-image.yml`

This workflow builds the Vue console as static assets and packages them behind Nginx.

It uses:

```yaml
context: ./console
file: ./console/Dockerfile
```

The console Dockerfile has two stages:

| Stage | Purpose |
| --- | --- |
| `oven/bun` build stage | Installs console dependencies and runs the Vite production build |
| `nginx:alpine` runtime stage | Serves the built static files with the SPA fallback config |

The workflow passes:

```yaml
build-args: |
  VITE_RIVET_API_URL=/api/v1
```

That value is intentionally relative. In production, the browser loads the console and calls the API on the same public origin, for example `https://console.example.com/api/v1`. Caddy handles `/api` by proxying to `rivet-server`, so the console image does not need to know the internal server container name or a public API subdomain.

The console image is published with the same tag policy as the server image:

| Tag | Meaning |
| --- | --- |
| `latest` | Convenient moving tag for the newest published console |
| `sha-<commit>` | Stable tag tied to the exact commit that produced the console image |

Production installs use the image through:

```sh
RIVET_CONSOLE_IMAGE=ghcr.io/devmin8/rivet-console:latest
```

Use the `sha-<commit>` tag instead when a deployment needs repeatable image pinning.

## Tags

The workflow creates two tags:

| Tag | Meaning |
| --- | --- |
| `latest` | Convenient moving tag for the newest published image |
| `sha-<commit>` | Stable tag tied to the exact commit that produced the image |

Use `latest` when convenience matters. Use the `sha-<commit>` tag when deployments need a repeatable, traceable image.

## Build Platforms

```yaml
platforms: linux/amd64,linux/arm64
```

The workflow publishes a multi-architecture image for both `amd64` and `arm64`.

QEMU is configured for `arm64`:

```yaml
platforms: arm64
```

This lets the GitHub-hosted `amd64` runner build the `arm64` image variant.

Buildx is also enabled before the image build:

```yaml
uses: docker/setup-buildx-action@v4
```

Buildx is what makes the multi-platform build, cache export, SBOM, and provenance options work cleanly in one Docker build step.

## Registry Login

```yaml
registry: ${{ env.REGISTRY }}
username: ${{ github.actor }}
password: ${{ secrets.GITHUB_TOKEN }}
```

The workflow logs in to GHCR with the built-in `GITHUB_TOKEN`. No personal access token is needed.

## Permissions

```yaml
permissions:
  contents: read
  packages: write
  attestations: write
  id-token: write
```

| Permission | Why it is needed |
| --- | --- |
| `contents: read` | Checks out the repository |
| `packages: write` | Pushes the image to GHCR |
| `attestations: write` | Uploads artifact attestations |
| `id-token: write` | Allows GitHub to issue the OIDC token used for verifiable attestations |

Keep these permissions scoped to the job. They are enough for publishing the image and generating the supply-chain metadata.

## Concurrency

```yaml
concurrency:
  group: publish-server-image-${{ github.ref }}
  cancel-in-progress: false
```

Only one publish run per branch or ref can run at a time.

`cancel-in-progress: false` means an older run is allowed to finish instead of being cancelled. This avoids half-published image work and keeps tag updates predictable.

The console workflow uses the same pattern with `publish-console-image-${{ github.ref }}`.

Because the group includes `${{ github.ref }}`, runs from different branches can still run at the same time. Run workflows from the intended release branch, since the `latest` tag is shared for each image.

## Build Context And Dockerfile

Server workflow:

```yaml
context: .
file: ./Dockerfile
```

The Docker build uses the repository root as the build context and the root `Dockerfile`.

Console workflow:

```yaml
context: ./console
file: ./console/Dockerfile
```

The Docker build uses the console folder as the build context and bakes `VITE_RIVET_API_URL=/api/v1` into the static frontend so production requests stay same-origin behind the reverse proxy.

## Push Behavior

```yaml
push: true
```

The workflow pushes the built image to GHCR. It does not only build locally inside the runner.

## Metadata

```yaml
images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
tags: |
  type=raw,value=latest
  type=sha,prefix=sha-
```

`docker/metadata-action` creates the final image name, tags, and OCI labels.

| Flag | Meaning |
| --- | --- |
| `images` | Sets the full GHCR image path |
| `type=raw,value=latest` | Always creates the `latest` tag |
| `type=sha,prefix=sha-` | Creates a commit-based tag like `sha-abc1234` |

```yaml
tags: ${{ steps.meta.outputs.tags }}
labels: ${{ steps.meta.outputs.labels }}
```

The build step applies those generated tags and labels to the pushed image.

## Build Cache

```yaml
cache-from: type=gha,scope=rivet-server
cache-to: type=gha,mode=max,scope=rivet-server
```

The workflow uses the GitHub Actions cache backend for Docker layers.

| Flag | Meaning |
| --- | --- |
| `type=gha` | Stores and reads cache through GitHub Actions cache |
| `scope=rivet-server` | Keeps this cache separate from other Docker builds |
| `mode=max` | Saves as much reusable layer data as BuildKit can export |

The console workflow uses the same pattern with `scope=rivet-console`.

This makes later image builds faster, especially when dependency layers have not changed.

## Provenance

```yaml
provenance: true
```

BuildKit attaches provenance metadata to the image. It records where and how the image was built, including the source repository, commit, workflow, and build environment.

This is useful when a deployment or security tool needs to verify the image origin.

## SBOM

```yaml
sbom: true
```

BuildKit attaches a Software Bill of Materials to the image.

An SBOM lists the software and dependencies found in the image. It helps with vulnerability scanning, audits, and dependency review.

## Artifact Attestation

```yaml
subject-name: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
subject-digest: ${{ steps.push.outputs.digest }}
push-to-registry: true
```

After the image is pushed, GitHub creates an artifact attestation for the image digest.

| Flag | Meaning |
| --- | --- |
| `subject-name` | The image name being attested |
| `subject-digest` | The immutable digest returned by the Docker build step |
| `push-to-registry` | Stores the attestation with the image in GHCR |

The digest matters because tags can move. The attestation points to the exact image artifact that was produced.

## Quick Runbook

1. Open GitHub Actions.
2. Select **Publish server image**.
3. Run the workflow from the intended release branch.
4. After it finishes, pull either `latest` or the `sha-<commit>` tag from GHCR.

## Notes

- This workflow publishes production-ready multi-architecture images.
- The `sha-<commit>` tag is the safest choice for pinned deployments.
- The cache, SBOM, provenance, and attestation settings are intentional. Remove them only if the publish process no longer needs build speed or supply-chain metadata.

## Further Reading

- [GitHub Actions workflow syntax](https://docs.github.com/en/actions/writing-workflows/workflow-syntax-for-github-actions)
- [GitHub Actions concurrency](https://docs.github.com/en/actions/using-jobs/using-concurrency)
- [GitHub artifact attestations](https://docs.github.com/en/actions/security-guides/using-artifact-attestations-to-establish-provenance-for-builds)
- [Docker multi-platform builds](https://docs.docker.com/build/building/multi-platform/)
- [Docker GitHub Actions cache backend](https://docs.docker.com/build/cache/backends/gha/)
- [Docker SBOM attestations](https://docs.docker.com/build/metadata/attestations/sbom/)
- [Docker provenance attestations](https://docs.docker.com/build/metadata/attestations/provenance/)
