package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/devmin8/rivet/internal/cli/client"
	"github.com/devmin8/rivet/internal/cli/prompt"
	"github.com/spf13/cobra"
)

func newSignUpCmd(app *app) *cobra.Command {
	return &cobra.Command{
		Use:   "signup",
		Short: "Sign up for the application",
		RunE: func(cmd *cobra.Command, args []string) error {
			username, err := prompt.String(cmd.InOrStdin(), cmd.OutOrStdout(), "Username: ")
			if err != nil {
				return err
			}

			email, err := prompt.String(cmd.InOrStdin(), cmd.OutOrStdout(), "Email: ")
			if err != nil {
				return err
			}

			password, err := prompt.Password(os.Stdin, cmd.OutOrStdout(), "Password: ")
			if err != nil {
				return err
			}

			confirmPassword, err := prompt.Password(os.Stdin, cmd.OutOrStdout(), "Confirm password: ")
			if err != nil {
				return err
			}
			if password != confirmPassword {
				return errors.New("passwords do not match")
			}

			apiClient := app.apiClient()
			if _, err := apiClient.Register(context.Background(), username, email, password); err != nil {
				return err
			}

			session, err := apiClient.SignIn(context.Background(), username, password)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Account created for %s, but automatic signin failed.\n", username)
				return err
			}

			if err := client.StoreSession(session); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Account created and signed in as %s.\n", username)
			return nil
		},
	}
}
