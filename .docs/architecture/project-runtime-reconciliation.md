# Project Runtime Reconciliation

Rivet stores one lifecycle status on each project row. That status is the project state shown to users.

The current status values are:

- `stopped`: user intentionally stopped the project
- `starting`: reserved for async start; start requested, container not confirmed yet
- `running`: container is confirmed running
- `sleeping`: reserved for sleep/wake; container intentionally removed because the project is idle
- `waking`: reserved for sleep/wake; traffic or user action requested a sleeping project to start
- `deploying`: new image is being rolled out
- `failed`: expected runtime action failed, or a running container is gone/exited

`desired_status` was removed. Keep lifecycle decisions in one status field unless there is a strong reason to split intent and observation later.

## Source Of Truth

The project row is the persisted source of truth for API and UI reads.

Docker is the runtime reality. The reconciler checks Docker and updates the project row when runtime reality disagrees with the persisted state.

`GET /projects` reads the database only. It does not inspect Docker. This keeps the project list fast, side-effect-free, and resilient when Docker is slow or unavailable.

## Reconciler Loop

The reconciler runs in the server process on a fixed interval.

For each active project whose status claims a container should exist and is safe to inspect outside a synchronous handler:

1. Choose the expected image.
   - `running`, `starting`, and `waking` expect `current_image_ref`, falling back to `target_image_ref` if needed.
2. Inspect the Docker container by stable `container_name`.
3. If the container is missing, mark the project `failed`, clear `container_id`, and store `last_error`.
4. If the container exists but is not running, mark the project `failed`, keep the container ID, and store `last_error`.
5. If the container is running with the expected image, mark the project `running`, update `container_id`, clear `last_error`, and refresh `last_active_at`.
6. If the container is running with the wrong image, mark the project `failed`.

The reconciler is not a blind auto-healer. If a running container crashes, users should see `failed`, not a silent restart hidden behind `running`.

## Synchronous Actions

Start, stop, delete, and deploy handlers still perform their Docker work synchronously.

- `start`: remove any old container, start the chosen image, then mark `running`
- `stop`: remove the container, then mark `stopped`
- `delete`: remove the container, then mark inactive and `stopped`
- `deploy`: mark `deploying`, replace the container with `target_image_ref`, then mark `running`

If a start/deploy runtime action fails, mark the row `failed`. If stop/delete fails, return the error and do not pretend the operation succeeded.

While deploy is synchronous, the reconciler does not scan `deploying` projects. Otherwise it can race the small window where deploy has removed the old container but has not started the new one yet.

## Failure Updates

Failure updates are keyed by project ID, not image reference.

This matters because a project can have:

- `current_image_ref`: the image currently serving
- `target_image_ref`: a newly uploaded image waiting for deploy

If the current container crashes while a new target image exists, the row must still become `failed`. Guarding failure updates by `target_image_ref` can miss that case.

When Docker says a container is missing, write `container_id = ""`. Stale container IDs should not remain after Rivet knows the container is gone.

## UI Freshness Tradeoff

The dashboard polls project rows and runtime stats while it is open. This gives close-to-current status without making every read hit Docker.

Accepted tradeoff:

- Status is eventually correct, not recalculated on every read.
- A crashed container may show `running` until the reconciler or stats endpoint marks it `failed` and the project query refetches.
- This keeps reads simple and avoids coupling project-list availability to Docker health.

If this delay becomes painful, prefer one of these in order:

1. Shorten the reconciler interval if Docker inspect load is still small.
2. Invalidate the project query when stats discovers a runtime failure.
3. Add SSE/WebSocket runtime events.
4. Only then consider read-through Docker inspection in `GET /projects`.

## Sleep/Wake Fit

The same status model supports Heroku-style sleep/wake later:

- idle `running` project transitions to `sleeping`
- sleeping project routes to Rivet's wake handler
- wake request transitions `sleeping -> waking -> running`

Sleeping is intentionally different from stopped. `stopped` means user does not want the app running. `sleeping` means Rivet intentionally removed the container, but the app is still meant to be available through wake-on-request.

## External Reading

- Kubernetes kubelet sync loop is a mature example of reconciling desired/persisted state with container runtime reality: https://kubernetes.io/docs/reference/node/kubelet-sync-loop/
- Docker stats docs are useful background on running vs stopped container stats behavior: https://docs.docker.com/reference/cli/docker/container/stats/
