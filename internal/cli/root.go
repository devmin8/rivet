package cli

import (
	"time"

	"github.com/devmin8/rivet/internal/cli/client"
	"github.com/spf13/cobra"
)

type app struct {
	cfg *cliConfig
}

func (a *app) apiClient() *client.Client {
	return a.apiClientWithTimeout(a.cfg.Timeout)
}

func (a *app) apiClientWithTimeout(timeout time.Duration) *client.Client {
	return client.New(client.Options{
		BaseURL: a.cfg.ServerURL,
		Timeout: timeout,
	})
}

func newRootCmd(cfg *cliConfig) *cobra.Command {
	app := &app{cfg: cfg}

	var rootCmd = &cobra.Command{
		Use:   "rivet",
		Short: "A cli to manage rivet pass application",
	}

	rootCmd.PersistentFlags().StringVar(
		&cfg.ServerURL,
		"server-url",
		cfg.ServerURL,
		"Rivet server URL; defaults from RIVET_SERVER_URL",
	)
	rootCmd.PersistentFlags().DurationVar(
		&cfg.Timeout,
		"api-timeout",
		cfg.Timeout,
		"Default API request timeout; defaults from RIVET_API_TIMEOUT",
	)

	rootCmd.AddCommand(newSignInCmd(app))
	rootCmd.AddCommand(newSignUpCmd(app))
	rootCmd.AddCommand(newShipCmd(app))

	return rootCmd
}

func Execute() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	rootCmd := newRootCmd(&cfg)
	return rootCmd.Execute()
}
