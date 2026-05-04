package docker

import (
	"context"
	"fmt"
	"io"
	"strings"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	dockerclient "github.com/moby/moby/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const rivetNetworkName = "rivet-network"
const LabelAppID = "rivet.app_id"
const LabelDeploymentID = "rivet.deployment_id"

type Client struct {
	api *dockerclient.Client
}

type StartContainerOptions struct {
	Image        string
	Platform     string
	Env          []string
	AppID        string
	DeploymentID string
}

type Container struct {
	ID           string
	AppID        string
	DeploymentID string
	Running      bool
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

func (c *Client) ContainerRunning(ctx context.Context, containerID string) (bool, error) {
	if err := c.CheckRunning(ctx); err != nil {
		return false, err
	}

	res, err := c.api.ContainerInspect(ctx, containerID, dockerclient.ContainerInspectOptions{})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("inspect docker container: %w", err)
	}
	if res.Container.State == nil {
		return false, nil
	}

	return res.Container.State.Running, nil
}

func (c *Client) ListRivetContainers(ctx context.Context) ([]Container, error) {
	if err := c.CheckRunning(ctx); err != nil {
		return nil, err
	}

	filters := make(dockerclient.Filters).Add("label", LabelAppID)
	res, err := c.api.ContainerList(ctx, dockerclient.ContainerListOptions{
		All:     true,
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("list docker containers: %w", err)
	}

	containers := make([]Container, 0, len(res.Items))
	for i := range res.Items {
		item := res.Items[i]
		containers = append(containers, Container{
			ID:           item.ID,
			AppID:        item.Labels[LabelAppID],
			DeploymentID: item.Labels[LabelDeploymentID],
			Running:      item.State == container.StateRunning,
		})
	}

	return containers, nil
}

func (c *Client) StartContainer(ctx context.Context, opts StartContainerOptions) (string, error) {
	if err := c.CheckRunning(ctx); err != nil {
		return "", err
	}

	res, err := c.api.ContainerCreate(ctx, dockerclient.ContainerCreateOptions{
		Config: &container.Config{
			Image:  opts.Image,
			Env:    opts.Env,
			Labels: rivetLabels(opts.AppID, opts.DeploymentID),
		},
		HostConfig: &container.HostConfig{
			NetworkMode: container.NetworkMode(rivetNetworkName),
			RestartPolicy: container.RestartPolicy{
				Name: "unless-stopped",
			},
		},
		Platform: parsePlatform(opts.Platform),
	})
	if err != nil {
		return "", fmt.Errorf("create docker container: %w", err)
	}

	if _, err := c.api.ContainerStart(ctx, res.ID, dockerclient.ContainerStartOptions{}); err != nil {
		return "", fmt.Errorf("start docker container: %w", err)
	}

	return res.ID, nil
}

func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	if err := c.CheckRunning(ctx); err != nil {
		return err
	}

	if _, err := c.api.ContainerStop(ctx, containerID, dockerclient.ContainerStopOptions{}); err != nil {
		if cerrdefs.IsNotFound(err) || cerrdefs.IsNotModified(err) {
			return nil
		}
		return fmt.Errorf("stop docker container: %w", err)
	}

	return nil
}

func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
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

// parsePlatform converts a platform string to an ocispec.Platform.
// Example: "linux/amd64" -> &ocispec.Platform{OS: "linux", Architecture: "amd64"}
func parsePlatform(platform string) *ocispec.Platform {
	parts := strings.Split(strings.TrimSpace(platform), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil
	}

	return &ocispec.Platform{
		OS:           parts[0],
		Architecture: parts[1],
	}
}

func rivetLabels(appID string, deploymentID string) map[string]string {
	labels := map[string]string{}
	if appID != "" {
		labels[LabelAppID] = appID
	}
	if deploymentID != "" {
		labels[LabelDeploymentID] = deploymentID
	}

	return labels
}
