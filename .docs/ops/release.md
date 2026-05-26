# Release Process

Rivet publishes OSS releases from one GitHub Actions workflow:

```text
.github/workflows/release.yml
```

Run the workflow manually from the default branch with a semver version such as `0.1.0`. The workflow creates the release tag and publishes every public artifact from that same tag.

## What It Does

The workflow:

- validates the requested version and creates a `v<version>` tag
- moves the current `CHANGELOG.md` Unreleased notes under that version
- commits the changelog update with the GitHub Actions bot identity
- builds and publishes the server and console images to GHCR
- builds `rivetctl` binaries for Linux, macOS, and Windows
- creates checksums for the CLI binaries
- publishes a GitHub Release using the changelog notes

The changelog commit happens before the release artifacts are built. The tag points at that changelog commit, so the release, images, and CLI binaries all come from the same source revision.

## Version And Tag Flow

The workflow accepts either `0.1.0` or `v0.1.0`.

Internally it stores:

| Value | Example | Used for |
| --- | --- | --- |
| version | `0.1.0` | Changelog heading, Go linker version, image tag |
| tag | `v0.1.0` | Git tag, GitHub Release name, image tag |

The workflow refuses to run from a non-default branch, rejects invalid semver values, and stops if the tag already exists.

## Published Artifacts

Server image:

```text
ghcr.io/devmin8/rivet-server
```

Console image:

```text
ghcr.io/devmin8/rivet-console
```

Image tags:

| Tag | Meaning |
| --- | --- |
| `v0.1.0` | Git tag style version |
| `0.1.0` | Semver version |
| `latest` | Moving tag for the newest release |
| `sha-<commit>` | Commit-based traceability tag |

CLI binaries are attached to the GitHub Release as `rivetctl-<os>-<arch>` files with `checksums.txt`.

## Changelog

Before building artifacts, the workflow moves the current `Unreleased` notes into a dated release section.

For a `0.1.0` release on `2026-05-26`, this:

```md
## [Unreleased]
```

becomes:

```md
## [Unreleased]

## [0.1.0] - 2026-05-26
```

The workflow commits that changelog update and tags the resulting commit. The GitHub Release body is generated from that version's changelog section.

## Image Security

Release images include supply-chain metadata for traceability and verification:

- Docker Buildx provenance
- Docker Buildx SBOM
- GitHub artifact attestations pushed to GHCR

These require:

| Permission | Why it is needed |
| --- | --- |
| `packages: write` | Pushes images to GHCR |
| `attestations: write` | Uploads artifact attestations |
| `id-token: write` | Issues the OIDC token used for verifiable attestations |

Provenance records where and how the image was built. The SBOM records the software found in the image. The artifact attestation ties the published image digest back to this GitHub Actions workflow run.

## Bot Commit Identity

The workflow commits the changelog with:

```text
github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>
```

This is GitHub's standard Actions bot identity. It makes the release commit clearly machine-generated instead of attributing it to a maintainer's local Git identity.

The bot commit is expected. It is part of the release, not a post-release follow-up.

## Runbook

1. Make sure `CHANGELOG.md` has the notes you want under `Unreleased`.
2. Commit all intended source changes to the default branch.
3. Open **Actions** in GitHub.
4. Run **Release** from the default branch.
5. Enter the version without or with a leading `v`, for example `0.1.0` or `v0.1.0`.
6. Mark `prerelease` only for prerelease versions.

After the workflow finishes, GitHub will have a release tag, release notes, CLI binaries, checksums, and GHCR images for that version.
