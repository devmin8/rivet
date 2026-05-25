package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/devmin8/rivet/internal/cli/client"
	"github.com/devmin8/rivet/internal/cli/prompt"
	"github.com/spf13/cobra"
)

func newDeleteCmd(app *app) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <project-id>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			if !yes {
				confirmation, err := prompt.RequiredString(
					cmd.InOrStdin(),
					cmd.OutOrStdout(),
					fmt.Sprintf("Type %s to confirm delete: ", projectID),
				)
				if err != nil {
					return err
				}
				if confirmation != projectID {
					return errors.New("delete cancelled")
				}
			}

			session, err := client.LoadSession()
			if err != nil {
				return err
			}

			if err := app.apiClient().DeleteProject(context.Background(), session, projectID); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Project %s deleted.\n", projectID)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")

	return cmd
}
