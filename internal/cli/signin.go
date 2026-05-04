package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/devmin8/rivet/internal/cli/client"
	"github.com/devmin8/rivet/internal/cli/prompt"
	"github.com/spf13/cobra"
)

func newSignInCmd(app *app) *cobra.Command {
	return &cobra.Command{
		Use:   "signin",
		Short: "Sign in to the application",
		RunE: func(cmd *cobra.Command, args []string) error {
			username, err := prompt.String(cmd.InOrStdin(), cmd.OutOrStdout(), "Username: ")
			if err != nil {
				return err
			}

			password, err := prompt.Password(os.Stdin, cmd.OutOrStdout(), "Password: ")
			if err != nil {
				return err
			}

			apiClient := app.apiClient()
			session, err := apiClient.SignIn(context.Background(), username, password)
			if err != nil {
				return err
			}

			if err := client.StoreSession(session); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Signed in as %s.\n", username)
			return nil
		},
	}
}
