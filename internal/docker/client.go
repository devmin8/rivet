package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	dockerclient "github.com/moby/moby/client"
)

const RivetNetworkName = "rivet-network"

var ErrContainerNotFound = errors.New("container not found")
var ErrContainerNotRunning = errors.New("container is not running")

// Client wraps the Docker Engine API used by Rivet's local runtime.
type Client struct {
	api *dockerclient.Client
}

// ContainerInfo is the small subset of Docker inspect data Rivet needs.
type ContainerInfo struct {
	ID      string
	Image   string
	Running bool
}

// ContainerStats is Rivet's normalized view of one Docker stats sample.
type ContainerStats struct {
	CPUPercent             float64
	CPUSampleWindowSeconds float64
	MemoryUsageBytes       uint64
	MemoryLimitBytes       uint64
	MemoryPercent          float64
	NetworkRxBytes         uint64
	NetworkTxBytes         uint64
	Pids                   uint64
	Sample                 ContainerStatsSample
}

// ContainerStatsSample stores the raw CPU counters needed for the next CPU delta.
type ContainerStatsSample struct {
	Read        time.Time
	CPUUsage    uint64
	SystemUsage uint64
}

// NewClient creates a Docker client configured from the local Docker environment.
func NewClient() (*Client, error) {
	api, err := dockerclient.New(dockerclient.FromEnv)
	if err != nil {
		return nil, err
	}

	return &Client{api: api}, nil
}

// CheckRunning verifies that the Docker daemon is reachable.
func (c *Client) CheckRunning(ctx context.Context) error {
	if _, err := c.api.Ping(ctx, dockerclient.PingOptions{}); err != nil {
		return fmt.Errorf("docker is not running: %w", err)
	}

	return nil
}

// LoadImage imports a Docker image tar stream into the local daemon.
func (c *Client) LoadImage(ctx context.Context, imageTar io.Reader) error {
	if err := c.CheckRunning(ctx); err != nil {
		return err
	}

	res, err := c.api.ImageLoad(ctx, imageTar, dockerclient.ImageLoadWithQuiet(true))
	if err != nil {
		return fmt.Errorf("load docker image: %w", err)
	}
	defer res.Close()

	if _, err := io.Copy(io.Discard, res); err != nil {
		return fmt.Errorf("read docker image load response: %w", err)
	}

	return nil
}

// InspectImageID returns Docker's immutable image ID for an image tag or reference.
func (c *Client) InspectImageID(ctx context.Context, tag string) (string, error) {
	image, err := c.api.ImageInspect(ctx, tag)
	if err != nil {
		return "", fmt.Errorf("inspect docker image: %w", err)
	}

	return image.ID, nil
}

// TagImage adds a new tag to an existing local Docker image.
func (c *Client) TagImage(ctx context.Context, source string, target string) error {
	if err := c.CheckRunning(ctx); err != nil {
		return err
	}

	if _, err := c.api.ImageTag(ctx, dockerclient.ImageTagOptions{
		Source: source,
		Target: target,
	}); err != nil {
		return fmt.Errorf("tag docker image: %w", err)
	}

	return nil
}

// EnsureNetwork creates Rivet's shared Docker network if it does not exist.
func (c *Client) EnsureNetwork(ctx context.Context) error {
	if err := c.CheckRunning(ctx); err != nil {
		return err
	}

	_, err := c.api.NetworkInspect(ctx, RivetNetworkName, dockerclient.NetworkInspectOptions{})
	if err == nil {
		return nil
	}
	if !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("inspect docker network: %w", err)
	}

	_, err = c.api.NetworkCreate(ctx, RivetNetworkName, dockerclient.NetworkCreateOptions{
		Driver: "bridge",
		Scope:  "local",
		Labels: map[string]string{
			"rivet.managed": "true",
		},
	})
	if err != nil && !cerrdefs.IsAlreadyExists(err) {
		return fmt.Errorf("create docker network: %w", err)
	}

	return nil
}

// StartContainer creates and starts one Rivet-managed project container.
func (c *Client) StartContainer(ctx context.Context, containerName string, projectID string, image string) (string, error) {
	if err := c.EnsureNetwork(ctx); err != nil {
		return "", err
	}

	created, err := c.api.ContainerCreate(ctx, dockerclient.ContainerCreateOptions{
		Name: containerName,
		Config: &container.Config{
			Image: image,
			Labels: map[string]string{
				"rivet.project_id":     projectID,
				"rivet.container_name": containerName,
				"rivet.managed":        "true",
			},
		},
		NetworkingConfig: &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				RivetNetworkName: {
					Aliases: []string{containerName},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("create docker container: %w", err)
	}

	if _, err := c.api.ContainerStart(ctx, created.ID, dockerclient.ContainerStartOptions{}); err != nil {
		return "", fmt.Errorf("start docker container: %w", err)
	}

	return created.ID, nil
}

// RemoveContainer deletes a project container by name and ignores missing containers.
func (c *Client) RemoveContainer(ctx context.Context, containerName string) error {
	opts := dockerclient.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}

	if _, err := c.api.ContainerRemove(ctx, containerName, opts); err != nil {
		if cerrdefs.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("remove docker container: %w", err)
	}

	return nil
}

// InspectContainer reads the container ID, image, and running state for a project container.
func (c *Client) InspectContainer(ctx context.Context, containerName string) (ContainerInfo, error) {
	res, err := c.api.ContainerInspect(ctx, containerName, dockerclient.ContainerInspectOptions{})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return ContainerInfo{}, ErrContainerNotFound
		}
		return ContainerInfo{}, fmt.Errorf("inspect docker container: %w", err)
	}

	info := ContainerInfo{
		ID: res.Container.ID,
	}
	if res.Container.Config != nil {
		info.Image = res.Container.Config.Image
	}
	if res.Container.State != nil {
		info.Running = res.Container.State.Running
	}

	return info, nil
}

// ContainerStats returns one immediate Docker stats sample for a running container.
func (c *Client) ContainerStats(ctx context.Context, containerName string, previous *ContainerStatsSample) (ContainerStats, error) {
	res, err := c.api.ContainerStats(ctx, containerName, dockerclient.ContainerStatsOptions{
		Stream: false,
		// Docker can collect a previous CPU sample for us, but that blocks the
		// request for roughly a second. Rivet needs a fast dashboard endpoint, so
		// we take one immediate sample and compare it with our cached prior sample.
		IncludePreviousSample: false,
	})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return ContainerStats{}, ErrContainerNotFound
		}
		return ContainerStats{}, fmt.Errorf("get docker container stats: %w", err)
	}
	defer res.Body.Close()

	var raw container.StatsResponse
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return ContainerStats{}, fmt.Errorf("decode docker container stats: %w", err)
	}
	if raw.Read.IsZero() {
		return ContainerStats{}, ErrContainerNotRunning
	}

	return normalizeContainerStats(raw, previous), nil
}

// Close releases Docker client resources.
func (c *Client) Close() error {
	return c.api.Close()
}

// normalizeContainerStats converts Docker's raw stats payload into Rivet's API shape.
func normalizeContainerStats(raw container.StatsResponse, previous *ContainerStatsSample) ContainerStats {
	memoryUsage := memoryUsage(raw.MemoryStats)
	memoryLimit := raw.MemoryStats.Limit
	stats := ContainerStats{
		CPUPercent:             cpuPercent(raw, previous),
		CPUSampleWindowSeconds: cpuSampleWindowSeconds(raw, previous),
		MemoryUsageBytes:       memoryUsage,
		MemoryLimitBytes:       memoryLimit,
		NetworkRxBytes:         networkRxBytes(raw.Networks),
		NetworkTxBytes:         networkTxBytes(raw.Networks),
		Pids:                   raw.PidsStats.Current,
		Sample:                 containerStatsSample(raw),
	}
	if memoryLimit > 0 {
		stats.MemoryPercent = float64(memoryUsage) / float64(memoryLimit) * 100
	}

	return stats
}

// containerStatsSample keeps only the raw counters needed for the next CPU calculation.
func containerStatsSample(raw container.StatsResponse) ContainerStatsSample {
	return ContainerStatsSample{
		Read:        raw.Read,
		CPUUsage:    raw.CPUStats.CPUUsage.TotalUsage,
		SystemUsage: raw.CPUStats.SystemUsage,
	}
}

// cpuPercent calculates recent CPU usage from the current and previous samples.
func cpuPercent(raw container.StatsResponse, previous *ContainerStatsSample) float64 {
	if previous == nil || previous.Read.IsZero() {
		return 0
	}
	// Docker CPU counters are cumulative since container start, not "current CPU".
	// A percentage only exists after comparing two samples: how much container CPU
	// time changed divided by how much host CPU time changed during the same window.
	if raw.CPUStats.CPUUsage.TotalUsage < previous.CPUUsage ||
		raw.CPUStats.SystemUsage < previous.SystemUsage {
		return 0
	}

	cpuDelta := raw.CPUStats.CPUUsage.TotalUsage - previous.CPUUsage
	systemDelta := raw.CPUStats.SystemUsage - previous.SystemUsage
	if cpuDelta == 0 || systemDelta == 0 {
		return 0
	}

	onlineCPUs := raw.CPUStats.OnlineCPUs
	if onlineCPUs == 0 {
		onlineCPUs = uint32(len(raw.CPUStats.CPUUsage.PercpuUsage))
	}
	if onlineCPUs == 0 {
		onlineCPUs = raw.NumProcs
	}
	if onlineCPUs == 0 {
		onlineCPUs = 1
	}

	return float64(cpuDelta) / float64(systemDelta) * float64(onlineCPUs) * 100
}

// cpuSampleWindowSeconds reports how many seconds the CPU percent was averaged over.
func cpuSampleWindowSeconds(raw container.StatsResponse, previous *ContainerStatsSample) float64 {
	if previous == nil || previous.Read.IsZero() || raw.Read.Before(previous.Read) {
		return 0
	}

	return raw.Read.Sub(previous.Read).Seconds()
}

// memoryUsage returns Docker CLI-style memory usage by subtracting reclaimable cache.
func memoryUsage(stats container.MemoryStats) uint64 {
	usage := stats.Usage
	if usage == 0 {
		return 0
	}

	// Docker's raw API includes file cache in memory usage. That cache is often
	// reclaimable by the kernel, so Docker CLI subtracts it for the human-facing
	// "MEM USAGE" value. The field name depends on cgroup/Docker version.
	if cache := reclaimableMemoryCache(stats.Stats); cache > 0 && usage > cache {
		return usage - cache
	}

	return usage
}

// reclaimableMemoryCache finds the Docker/cgroup field that represents file cache.
func reclaimableMemoryCache(stats map[string]uint64) uint64 {
	for _, key := range []string{"total_inactive_file", "inactive_file", "cache"} {
		if value := stats[key]; value > 0 {
			return value
		}
	}

	return 0
}

// networkRxBytes sums received bytes across all container network interfaces.
func networkRxBytes(networks map[string]container.NetworkStats) uint64 {
	var total uint64
	for _, network := range networks {
		total += network.RxBytes
	}
	return total
}

// networkTxBytes sums transmitted bytes across all container network interfaces.
func networkTxBytes(networks map[string]container.NetworkStats) uint64 {
	var total uint64
	for _, network := range networks {
		total += network.TxBytes
	}
	return total
}
