package docker

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	dockerclient "github.com/moby/moby/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type Client struct {
	api *dockerclient.Client
}

const rivetNetworkName = "rivet-network"

type StartContainerOptions struct {
	Image     string
	Platform  string
	Env       map[string]string
	ProjectID string
}

type ContainerInfo struct {
	Exists   bool
	Running  bool
	Status   string
	Error    string
	ExitCode int
}

func NewClient() (*Client, error) {
	api, err := dockerclient.New(dockerclient.FromEnv)
	if err != nil {
		return nil, err
	}

	return &Client{api: api}, nil
}

func (c *Client) CheckRunning(ctx context.Context) error {
	if _, err := c.api.Ping(ctx, dockerclient.PingOptions{}); err != nil {
		return fmt.Errorf("docker is not running: %w", err)
	}

	return nil
}

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

func (c *Client) InspectImageID(ctx context.Context, tag string) (string, error) {
	image, err := c.api.ImageInspect(ctx, tag)
	if err != nil {
		return "", fmt.Errorf("inspect docker image: %w", err)
	}

	return image.ID, nil
}

func (c *Client) InspectContainer(ctx context.Context, containerID string) (ContainerInfo, error) {
	if strings.TrimSpace(containerID) == "" {
		return ContainerInfo{}, nil
	}
	if err := c.CheckRunning(ctx); err != nil {
		return ContainerInfo{}, err
	}

	res, err := c.api.ContainerInspect(ctx, containerID, dockerclient.ContainerInspectOptions{})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return ContainerInfo{}, nil
		}
		return ContainerInfo{}, fmt.Errorf("inspect docker container: %w", err)
	}
	if res.Container.State == nil {
		return ContainerInfo{Exists: true}, nil
	}

	return ContainerInfo{
		Exists:   true,
		Running:  res.Container.State.Running,
		Status:   string(res.Container.State.Status),
		Error:    res.Container.State.Error,
		ExitCode: res.Container.State.ExitCode,
	}, nil
}

func (c *Client) StartContainer(ctx context.Context, opts StartContainerOptions) (string, error) {
	if err := c.EnsureNetwork(ctx); err != nil {
		return "", err
	}

	res, err := c.api.ContainerCreate(ctx, dockerclient.ContainerCreateOptions{
		Config: &container.Config{
			Image: opts.Image,
			Env:   envList(opts.Env),
			Labels: map[string]string{
				"rivet.project_id": opts.ProjectID,
			},
		},
		HostConfig: &container.HostConfig{
			NetworkMode:   container.NetworkMode(rivetNetworkName),
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
		},
		Platform: platformSpec(opts.Platform),
		Name:     containerName(opts.ProjectID),
	})
	if err != nil {
		return "", fmt.Errorf("create docker container: %w", err)
	}

	if _, err := c.api.ContainerStart(ctx, res.ID, dockerclient.ContainerStartOptions{}); err != nil {
		_, _ = c.api.ContainerRemove(ctx, res.ID, dockerclient.ContainerRemoveOptions{Force: true})
		return "", fmt.Errorf("start docker container: %w", err)
	}

	return res.ID, nil
}

func (c *Client) EnsureNetwork(ctx context.Context) error {
	if err := c.CheckRunning(ctx); err != nil {
		return err
	}

	if _, err := c.api.NetworkInspect(ctx, rivetNetworkName, dockerclient.NetworkInspectOptions{}); err == nil {
		return nil
	} else if !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("inspect docker network %s: %w", rivetNetworkName, err)
	}

	if _, err := c.api.NetworkCreate(ctx, rivetNetworkName, dockerclient.NetworkCreateOptions{
		Driver: "bridge",
		Labels: map[string]string{
			"rivet.managed": "true",
		},
	}); err != nil && !cerrdefs.IsAlreadyExists(err) {
		return fmt.Errorf("create docker network %s: %w", rivetNetworkName, err)
	}

	return nil
}

func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	if strings.TrimSpace(containerID) == "" {
		return nil
	}
	if err := c.CheckRunning(ctx); err != nil {
		return err
	}

	timeout := 10
	if _, err := c.api.ContainerStop(ctx, containerID, dockerclient.ContainerStopOptions{Timeout: &timeout}); err != nil {
		if cerrdefs.IsNotFound(err) || cerrdefs.IsNotModified(err) {
			return nil
		}
		return fmt.Errorf("stop docker container: %w", err)
	}

	return nil
}

func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
	if strings.TrimSpace(containerID) == "" {
		return nil
	}
	if err := c.CheckRunning(ctx); err != nil {
		return err
	}

	if _, err := c.api.ContainerRemove(ctx, containerID, dockerclient.ContainerRemoveOptions{Force: true}); err != nil {
		if cerrdefs.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("remove docker container: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	return c.api.Close()
}

func envList(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}

	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	res := make([]string, 0, len(keys))
	for _, key := range keys {
		res = append(res, key+"="+env[key])
	}

	return res
}

func platformSpec(value string) *ocispec.Platform {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	osName, arch, _ := strings.Cut(value, "/")

	return &ocispec.Platform{
		OS:           osName,
		Architecture: arch,
	}
}

func containerName(projectID string) string {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return ""
	}

	return "rivet-" + projectID
}
