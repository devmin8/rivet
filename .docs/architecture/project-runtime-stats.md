# Project Runtime Stats

Rivet shows lightweight runtime stats in the project list: CPU, memory, network I/O, and process count for projects that are currently `running`.

These stats are dashboard data, not billing data and not a full observability system. The goal is to answer: "is this project alive, idle, busy, or suspicious?" The endpoint must stay fast enough for the project list.

## API

```http
GET /api/v1/projects/stats
GET /api/v1/projects/stats?ids=<project-id>,<project-id>
```

Example:

```json
{
  "as_of": "2026-05-09T14:30:00Z",
  "items": [
    {
      "project_id": "project-uuid",
      "captured_at": "2026-05-09T14:29:57Z",
      "stale": false,
      "cpu_percent": 2.4,
      "cpu_sample_window_seconds": 5.01,
      "memory_usage_bytes": 73400320,
      "memory_limit_bytes": 8589934592,
      "memory_percent": 0.85,
      "network_rx_bytes": 128300,
      "network_tx_bytes": 92210,
      "pids": 8
    }
  ]
}
```

`items` is sparse. Stopped, sleeping, failed, or otherwise non-running projects are omitted. If a running project's fresh Docker stats read fails and Rivet has a cached row, that row is returned with `stale: true` and the original `captured_at`. If there is no cached row, the project is omitted and the console shows stats unavailable.

There is no response-level `stale` flag. Staleness is row-wise because one project can have stale metrics while another project's metrics are fresh.

## Why A Separate Endpoint

`GET /projects` should stay a fast database read for persisted project metadata and lifecycle status. Runtime stats come from Docker, have different failure behavior, and can be slower or stale. The console calls `/projects` and `/projects/stats` in parallel and merges stats by `project_id`.

Do not expose Docker container IDs or names in public stats API responses. Docker does not know Rivet authorization. The service first loads projects owned by the authenticated user, then reads stats only for those project containers.

## Data Source

Stats come from Docker Engine's `/containers/{id}/stats` API through the Go Docker client.

The Docker lookup key is `Project.ContainerName`, not `ContainerID`. The name is stable for a project (`rivet-<project-id>`). The ID can change when Rivet recreates a container.

Only projects with `status = running` are eligible for runtime stats. Lifecycle truth lives on the project row and is maintained by the reconciler; stats are optional metrics layered on top.

## Collection Strategy

Use Docker one-shot stats:

```go
Stream: false
IncludePreviousSample: false
```

This returns one immediate sample. Do not use Docker's previous-sample option. It can block while Docker gathers a second sample. Rivet keeps the endpoint fast by collecting one immediate sample and comparing it with the previous sample stored in Rivet's cache.

For multiple projects, the service fans out container stats reads concurrently and waits once for all results. It should not read project 1, then project 2, then project 3.

## CPU Calculation

Docker CPU counters are cumulative since container start. A single sample cannot answer "CPU percent right now."

Rivet stores the previous CPU sample in the in-memory stats cache:

- previous container CPU total
- previous host/system CPU total
- previous sample timestamp

On the next refresh:

```text
cpu_delta = current_container_cpu_total - previous_container_cpu_total
system_delta = current_host_cpu_total - previous_host_cpu_total
cpu_percent = cpu_delta / system_delta * online_cpus * 100
cpu_sample_window_seconds = current_sample_time - previous_sample_time
```

The first uncached sample returns `cpu_percent: 0` and `cpu_sample_window_seconds: 0` because there is no previous sample yet. That is intentional. The next refresh has a real window.

`cpu_sample_window_seconds` exists so the UI and engineers can understand the number. For example, `2.4% over 5.0s` is a recent average, not an instant reading.

## Memory Calculation

Docker's raw API memory usage includes file cache. File cache is usually reclaimable by the Linux kernel, so it can make an app look like it is using more memory than it truly needs.

Docker CLI subtracts this cache for its human-facing `MEM USAGE` value. Rivet follows that style:

```text
memory_usage_bytes = raw_usage - reclaimable_cache
memory_percent = memory_usage_bytes / memory_limit_bytes * 100
```

The cache field name depends on cgroup and Docker version:

- `total_inactive_file` for cgroup v1
- `inactive_file` for cgroup v2
- `cache` for older Docker behavior

This is why the code checks those field names. They are Docker/Linux memory-stat names, not Rivet concepts.

All memory and network values in the API are bytes. The console formats bytes as MiB/GiB for display.

## Caching And Staleness

Rivet uses an in-memory TTL cache.

- Key: project ID
- TTL: 5 seconds
- Value: normalized stats row plus Docker CPU sample counters
- Guard refresh with one mutex to avoid dashboard request stampedes

The mutex serializes concurrent HTTP refreshes, but collection inside one refresh is concurrent per project.

Stale behavior is row-wise:

- Fresh cache entry exists inside the TTL: return the cached row with `stale: false` and its existing `captured_at`.
- Docker stats read succeeds after a cache miss/expiry: return the row with `stale: false` and `captured_at = now`.
- Fresh Docker stats read fails and a cached row exists: return the cached row with `stale: true` and the cached `captured_at`.
- Fresh Docker stats read fails and no cached row exists: omit that project from `items`.
- Docker daemon unavailable: return any cached rows as stale; omit projects without cache.

The console polls project status and runtime stats while the dashboard is open. Polling is intentionally simple for now. Later, we can replace or supplement it with stats-driven query invalidation, SSE, or WebSocket updates.

## Failure Behavior

Stats are best-effort, but some stats failures reveal runtime truth.

- If Docker stats fails because the container is missing or not running, Rivet marks the project `failed` and records `last_error`.
- If Docker itself is unavailable or another non-container-specific error occurs, Rivet does not mark all projects failed. It returns cached stats where possible and logs the error.
- The project list must not fail because optional runtime stats failed.

This means a project may briefly show stale metrics before the project row refreshes to `failed`. That is acceptable. Runtime status is persisted truth maintained by reconciliation; stats are recent metrics layered on top.

## Why This Tradeoff

`GET /projects` does not inspect Docker on every read. It returns database truth. The reconciler updates that truth from Docker reality in the background, and the dashboard polls frequently enough to keep the UI close to current.

This is deliberate:

- Reads stay fast and side-effect-free.
- Docker outages do not make the main project list slow or destructive.
- Reconciler owns runtime state transitions.
- Stats can be stale without pretending project metadata is stale.

The tradeoff is eventual consistency: if a container crashes, the UI may show `running` until the reconciler or stats endpoint marks the project `failed` and the project list refetches. We accept that for now to keep the architecture simple and robust.

## External Reading

- Docker `docker stats` docs explain live container stats, stopped containers returning no data, and memory cache subtraction: https://docs.docker.com/reference/cli/docker/container/stats/
- Kubernetes kubelet sync loop explains the common controller pattern of reconciling desired state with actual runtime state: https://kubernetes.io/docs/reference/node/kubelet-sync-loop/
- Kubernetes Metrics Server collects recent resource metrics for autoscaling/debugging and is not a full monitoring pipeline: https://kubernetes-sigs.github.io/metrics-server/
- `kubectl top` documents that displayed metrics are recent, stable resource metrics rather than exact OS-tool parity: https://kubernetes.io/docs/reference/kubectl/generated/kubectl_top/

## Future Work

- Add historical charts with a background sampler.
- Add disk I/O when we have a clear UI story for Docker's block I/O semantics.
- Add per-node collection if Rivet runs projects across multiple hosts.
- Add deeper project-detail stats without coupling them to `GET /projects`.
- Consider replacing polling with stats-driven invalidation, SSE, or WebSocket project events.
