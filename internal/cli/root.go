package cli

import (
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "rivet",
		Short: "A cli to manage rivet pass application",
	}

	return rootCmd
}

func Execute() error {
	rootCmd := newRootCmd()
	return rootCmd.Execute()
}
