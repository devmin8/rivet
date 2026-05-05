package docker

import (
	"context"
	"errors"
	"fmt"
	"io"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	dockerclient "github.com/moby/moby/client"
)

const RivetNetworkName = "rivet-network"

var ErrContainerNotFound = errors.New("container not found")

type Client struct {
	api *dockerclient.Client
}

type ContainerInfo struct {
	ID      string
	Image   string
	Running bool
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

func (c *Client) Close() error {
	return c.api.Close()
}
