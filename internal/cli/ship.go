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

				apiClient := app.apiClient()
				project, err := apiClient.CreateProject(ctx, session, dtos.CreateProjectRequest{
					Name:        req.Name,
					Description: req.Description,
				})
				if err != nil {
					return err
				}

				fmt.Fprintf(cmd.OutOrStdout(), "🚀 Project %s created successfully.\n", project.ID)

				createdApp, err := apiClient.CreateApp(ctx, session, project.ID, dtos.CreateAppRequest{
					Name:        req.Name,
					Domain:      req.Domain,
					Description: req.Description,
					Port:        req.Port,
					Platform:    req.Platform,
				})
				if err != nil {
					return err
				}

				fmt.Fprintf(cmd.OutOrStdout(), "🚀 App %s created successfully.\n", createdApp.ID)

				return buildUploadAndDeploy(ctx, cmd, app, session, createdApp.ID, createdApp.Platform)
			case "2", "existing", "e":
				id, err := prompt.RequiredString(cmd.InOrStdin(), cmd.OutOrStdout(), "Project ID: ")
				if err != nil {
					return err
				}

				project, err := app.apiClient().GetProject(ctx, session, id)
				if err != nil {
					return err
				}

				fmt.Fprintf(cmd.OutOrStdout(), "🚀 Project %s selected.\n", project.ID)

				selectedApp, err := selectApp(ctx, cmd, app, session, project.ID, platform)
				if err != nil {
					return err
				}

				fmt.Fprintf(cmd.OutOrStdout(), "🚀 App %s selected.\n", selectedApp.ID)
				fmt.Fprintf(cmd.OutOrStdout(), "Platform: %s\n", selectedApp.Platform)

				return buildUploadAndDeploy(ctx, cmd, app, session, selectedApp.ID, selectedApp.Platform)
			default:
				return errors.New("project must be new or existing")
			}
		},
	}

	cmd.Flags().StringVar(&platform, "platform", "linux/amd64", "Target image platform: linux/amd64 or linux/arm64")

	return cmd
}

func promptCreateProject(cmd *cobra.Command, platform string) (dtos.CreateAppRequest, error) {
	name, err := prompt.RequiredString(cmd.InOrStdin(), cmd.OutOrStdout(), "Name: ")
	if err != nil {
		return dtos.CreateAppRequest{}, err
	}

	domain, err := prompt.RequiredString(cmd.InOrStdin(), cmd.OutOrStdout(), "Domain: ")
	if err != nil {
		return dtos.CreateAppRequest{}, err
	}

	port, err := prompt.Port(cmd.InOrStdin(), cmd.OutOrStdout(), "Port: ")
	if err != nil {
		return dtos.CreateAppRequest{}, err
	}

	description, err := prompt.String(cmd.InOrStdin(), cmd.OutOrStdout(), "Description: ")
	if err != nil {
		return dtos.CreateAppRequest{}, err
	}

	return dtos.CreateAppRequest{
		Name:        name,
		Domain:      domain,
		Port:        port,
		Description: description,
		Platform:    platform,
	}, nil
}

func selectApp(ctx context.Context, cmd *cobra.Command, app *app, session *client.Session, projectID string, platform string) (*dtos.AppResponse, error) {
	apiClient := app.apiClient()
	apps, err := apiClient.ListProjectApps(ctx, session, projectID)
	if err != nil {
		return nil, err
	}

	switch len(apps.Apps) {
	case 0:
		fmt.Fprintln(cmd.OutOrStdout(), "No apps found for this project. Creating one.")
		req, err := promptCreateProject(cmd, platform)
		if err != nil {
			return nil, err
		}

		return apiClient.CreateApp(ctx, session, projectID, req)
	case 1:
		// for now, only one app is supported for a project
		return &apps.Apps[0], nil
	default:
		appID, err := prompt.RequiredString(cmd.InOrStdin(), cmd.OutOrStdout(), "App ID: ")
		if err != nil {
			return nil, err
		}

		selectedApp, err := apiClient.GetApp(ctx, session, appID)
		if err != nil {
			return nil, err
		}
		if selectedApp.ProjectID != projectID {
			return nil, fmt.Errorf("app %s does not belong to project %s", selectedApp.ID, projectID)
		}

		return selectedApp, nil
	}
}

func buildUploadAndDeploy(ctx context.Context, cmd *cobra.Command, app *app, session *client.Session, appID string, platform string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Building image for %s...\n", platform)
	result, err := docker.BuildImage(ctx, appID, platform, dir, cmd.OutOrStdout())
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
	image, err := client.UploadImage(ctx, session, appID, result.ImageTag, result.TarballPath)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "🚀 Image %s uploaded successfully.\n", result.ImageTag)
	deployment, err := client.CreateDeployment(ctx, session, appID, image)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "🚀 Deployment %s is %s.\n", deployment.ID, deployment.Status)
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
