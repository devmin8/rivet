package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type BuildResult struct {
	ImageTag    string
	TarballPath string
}

func BuildImage(ctx context.Context, projectID string, platform string, dir string, out io.Writer) (BuildResult, error) {
	client, err := NewClient()
	if err != nil {
		return BuildResult{}, err
	}
	if err := client.CheckRunning(ctx); err != nil {
		return BuildResult{}, err
	}
	defer client.Close()

	binary, err := exec.LookPath("docker")
	if err != nil {
		return BuildResult{}, fmt.Errorf("docker command not found: %w", err)
	}

	imageTag := fmt.Sprintf("rivet/%s:latest", projectID)
	tarballPath := filepath.Join(dir, fmt.Sprintf("rivet-%s.tar", projectID))

	if err := runDocker(ctx, binary, dir, out, "build", "--platform", platform, "-t", imageTag, "."); err != nil {
		return BuildResult{}, err
	}
	if err := runDocker(ctx, binary, dir, out, "save", "-o", tarballPath, imageTag); err != nil {
		if ctx.Err() != nil {
			_ = os.Remove(tarballPath)
		}
		return BuildResult{}, err
	}

	return BuildResult{
		ImageTag:    imageTag,
		TarballPath: tarballPath,
	}, nil
}

func runDocker(ctx context.Context, binary string, dir string, out io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Stdout = out
	cmd.Stderr = out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %s: %w", args[0], err)
	}

	return nil
}
