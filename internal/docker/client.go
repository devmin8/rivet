package docker

import (
	"context"
	"fmt"
	"io"

	dockerclient "github.com/moby/moby/client"
)

type Client struct {
	api *dockerclient.Client
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

func (c *Client) Close() error {
	return c.api.Close()
}
