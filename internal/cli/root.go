// Package cli defines the `bit` command-line interface: init, clone, push,
// pull, and remote management. Each command file wires cobra flags and
// .bit/config.json state to the workflows in internal/app.
package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the top-level `bit` command; subcommands register themselves
// onto it via their own init() functions.
var rootCmd = &cobra.Command{
	Use:   "bit",
	Short: "Decentralized version control powered by IPFS and blockchain",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(remoteCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(pullCmd)
}
