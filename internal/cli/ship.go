package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/cli/client"
	"github.com/devmin8/rivet/internal/cli/prompt"
	"github.com/devmin8/rivet/internal/docker"
	"github.com/spf13/cobra"
)

const shipUploadTimeout = 30 * time.Minute

func newShipCmd(app *app) *cobra.Command {
	var platform string

	cmd := &cobra.Command{
		Use:   "ship",
		Short: "Ship a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			normalizedPlatform, err := normalizePlatform(platform)
			if err != nil {
				return err
			}
			platform = normalizedPlatform

			session, err := client.LoadSession()
			if err != nil {
				return err
			}

			option, err := prompt.String(cmd.InOrStdin(), cmd.OutOrStdout(), "Project: new (n) / existing (e): ")
			if err != nil {
				return err
			}

			switch strings.ToLower(strings.TrimSpace(option)) {
			case "1", "new", "n":
				req, err := promptCreateProject(cmd, platform)
				if err != nil {
					return err
				}

				project, err := app.apiClient().CreateProject(ctx, session, req)
				if err != nil {
					return err
				}

				fmt.Fprintf(cmd.OutOrStdout(), "🚀 Project %s created successfully.\n", project.ID)

				return buildAndUploadImage(ctx, cmd, app, session, project.ID, project.Platform)
			case "2", "existing", "e":
				id, err := prompt.RequiredString(cmd.InOrStdin(), cmd.OutOrStdout(), "Project ID: ")
				if err != nil {
					return err
				}

				project, err := app.apiClient().GetProject(ctx, session, id)
				if err != nil {
					return err
				}
				if !project.IsActive {
					return fmt.Errorf("project %s is not active", id)
				}

				fmt.Fprintf(cmd.OutOrStdout(), "🚀 Project %s selected.\n", project.ID)
				fmt.Fprintf(cmd.OutOrStdout(), "Platform: %s\n", project.Platform)

				return buildAndUploadImage(ctx, cmd, app, session, project.ID, project.Platform)
			default:
				return errors.New("project must be new or existing")
			}
		},
	}

	cmd.Flags().StringVar(&platform, "platform", "linux/amd64", "Target image platform: linux/amd64 or linux/arm64")

	return cmd
}

func promptCreateProject(cmd *cobra.Command, platform string) (dtos.CreateProjectRequest, error) {
	name, err := prompt.RequiredString(cmd.InOrStdin(), cmd.OutOrStdout(), "Name: ")
	if err != nil {
		return dtos.CreateProjectRequest{}, err
	}

	domain, err := prompt.RequiredString(cmd.InOrStdin(), cmd.OutOrStdout(), "Domain: ")
	if err != nil {
		return dtos.CreateProjectRequest{}, err
	}

	port, err := prompt.Port(cmd.InOrStdin(), cmd.OutOrStdout(), "Port: ")
	if err != nil {
		return dtos.CreateProjectRequest{}, err
	}

	description, err := prompt.String(cmd.InOrStdin(), cmd.OutOrStdout(), "Description: ")
	if err != nil {
		return dtos.CreateProjectRequest{}, err
	}

	return dtos.CreateProjectRequest{
		Name:        name,
		Domain:      domain,
		Port:        port,
		Description: description,
		Platform:    platform,
	}, nil
}

func buildAndUploadImage(ctx context.Context, cmd *cobra.Command, app *app, session *client.Session, projectID string, platform string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Building image for %s...\n", platform)
	result, err := docker.BuildImage(ctx, projectID, platform, dir, cmd.OutOrStdout())
	if err != nil {
		return err
	}

	defer func() {
		if result.TarballPath != "" {
			_ = os.Remove(result.TarballPath)
		}
	}()

	fmt.Fprintf(cmd.OutOrStdout(), "Uploading %s...\n", result.ImageTag)

	client := app.apiClientWithTimeout(shipUploadTimeout)
	if err := client.UploadImage(ctx, session, projectID, result.ImageTag, result.TarballPath); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "🚀 Image %s uploaded successfully.\n", result.ImageTag)
	return nil
}

func normalizePlatform(platform string) (string, error) {
	switch strings.TrimSpace(platform) {
	case "linux/arm64":
		return "linux/arm64", nil
	case "linux/amd64", "":
		return "linux/amd64", nil
	default:
		return "", fmt.Errorf("platform must be linux/amd64 or linux/arm64")
	}
}
