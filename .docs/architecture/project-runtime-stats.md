# Project Runtime Stats

Rivet shows lightweight runtime stats in the project list: CPU, memory, network I/O, and process count for running projects.

This is dashboard data, not billing data and not a full observability system. The goal is: "is this project alive, idle, busy, or suspicious?" The endpoint must stay fast enough for the project list.

## API

```http
GET /api/v1/projects/stats
GET /api/v1/projects/stats?ids=<project-id>,<project-id>
```

Example:

```json
{
  "as_of": "2026-05-08T14:30:00Z",
  "stale": false,
  "items": [
    {
      "project_id": "project-uuid",
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

`items` is sparse. A stopped project or missing container means that project is omitted. If Docker stats for a project fail but Rivet has a cached row, the cached row may be returned with `stale: true`; otherwise the project is omitted. The console treats missing stats as "not available" and shows a small stale warning when any requested live stat could not be refreshed.

## Why A Separate Endpoint

`GET /projects` should stay a fast database read. Runtime stats come from Docker, have different failure behavior, and can be slower or stale. The console should call `/projects` and `/projects/stats` in parallel and merge by `project_id`.

Do not expose Docker container IDs or names in public API responses. Docker does not know Rivet authorization. The service first loads projects owned by the authenticated user, then reads stats only for those project containers.

## Data Source

Stats come from Docker Engine's `/containers/{id}/stats` API through the Go Docker client.

The Docker lookup key is `Project.ContainerName`, not `ContainerID`. The name is stable for a project (`rivet-<project-id>`). The ID can change when reconciliation recreates a container.

## Collection Strategy

Use Docker one-shot stats:

```go
Stream: false
IncludePreviousSample: false
```

This returns one immediate sample. Do not use Docker's previous-sample option. It makes Docker wait roughly one second so it can compute CPU for us. In local measurement, the old path took about `1.96s`; one-shot took about `0.03s`.

For 50 projects, the service fans out container stats reads concurrently and waits once for all results. It should not read project 1, then project 2, then project 3.

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

All memory and network values in the API are bytes. The console can format bytes as MiB/GiB for display.

## Caching

Use an in-memory TTL cache.

- Key: project ID
- TTL: 5 seconds
- Value: normalized stats row plus Docker CPU sample counters
- Guard refresh with one mutex to avoid dashboard request stampedes

The mutex serializes concurrent HTTP refreshes, but collection inside one refresh is concurrent per project.

This is enough for the MVP. A background sampler and SQLite history can come later if we add charts or long-term metrics.

## Failure Behavior

Stats are best-effort.

- One project fails: return its cached row if available, set `stale: true`, and otherwise omit that project.
- Docker unavailable: return available cached stats and `stale: true`.
- No cached stats: return an empty `items` array and `stale: true`.
- Log failures server-side with project ID and container name.

The project list must not fail because optional runtime stats failed.

## Why This Is Acceptable

This style is normal for PaaS/runtime dashboards.

- Docker exposes live container CPU, memory, network, block I/O, and PID stats. Docker CLI also subtracts memory cache before showing `MEM USAGE`: https://docs.docker.com/reference/cli/docker/container/stats/
- Kubernetes `kubectl top` explicitly says its numbers may not match OS tools because they are recent, stable metrics rather than pinpoint-accurate live truth: https://kubernetes.io/docs/reference/kubectl/generated/kubectl_top/
- Kubernetes Metrics Server collects CPU and memory every 15 seconds for autoscaling and quick debugging, not exact monitoring: https://kubernetes-sigs.github.io/metrics-server/
- Coolify uses Sentinel, a lightweight agent for server/container CPU and RAM metrics, with a configurable collection interval and local storage: https://github.com/coollabsio/sentinel
- Dokploy exposes configurable server/container refresh rates and warns that lower intervals improve precision but increase load: https://docs.dokploy.com/docs/core/monitoring
- CapRover delegates richer monitoring to Netdata instead of blocking its main app APIs on exact stats collection: https://caprover.com/docs/resource-monitoring.html

The tradeoff is deliberate: fast, recent, explainable stats in the project list. For detailed historical observability, use a dedicated metrics stack later.

## Future Work

- Add historical charts with a background sampler.
- Add disk I/O when we have a clear UI story for Docker's block I/O semantics.
- Add per-node collection if Rivet runs projects across multiple hosts.
- Add deeper project-detail stats without coupling them to `GET /projects`.
