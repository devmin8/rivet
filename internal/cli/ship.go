package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/cli/client"
	"github.com/devmin8/rivet/internal/cli/prompt"
	"github.com/spf13/cobra"
)

func newShipCmd(app *app) *cobra.Command {
	return &cobra.Command{
		Use:   "ship",
		Short: "Ship a project",
		RunE: func(cmd *cobra.Command, args []string) error {
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
				req, err := promptCreateProject(cmd)
				if err != nil {
					return err
				}

				id, err := app.apiClient().CreateProject(context.Background(), session, req)
				if err != nil {
					return err
				}

				fmt.Fprintf(cmd.OutOrStdout(), "🚀 Project %s created successfully.\n", id)
			case "2", "existing", "e":
				id, err := prompt.RequiredString(cmd.InOrStdin(), cmd.OutOrStdout(), "Project ID: ")
				if err != nil {
					return err
				}

				fmt.Fprintf(cmd.OutOrStdout(), "🚀 Project %s selected.\n", id)
			default:
				return errors.New("project must be new or existing")
			}

			return nil
		},
	}
}

func promptCreateProject(cmd *cobra.Command) (dtos.CreateProjectRequest, error) {
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
	}, nil
}
