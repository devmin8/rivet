# GHA - Publish Server Image

This workflow builds the server Docker image and publishes it to GitHub Container Registry (GHCR).

Workflow file: `.github/workflows/publish-server-image.yml`

## When It Runs

The workflow only runs manually:

```yaml
on:
  workflow_dispatch:
```

Run it from the GitHub Actions UI when a server image needs to be published.

## What It Publishes

The image is pushed to:

```text
ghcr.io/${{ github.repository }}-server
```

For this repo, the image name follows the repository name and adds `-server`.

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

Because the group includes `${{ github.ref }}`, runs from different branches can still run at the same time. Run this workflow from the intended release branch, since the `latest` tag is shared for the image.

## Build Context And Dockerfile

```yaml
context: .
file: ./Dockerfile
```

The Docker build uses the repository root as the build context and the root `Dockerfile`.

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
