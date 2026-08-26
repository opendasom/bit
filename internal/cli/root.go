// Package cli defines Bit's command-line interface.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bit",
	Short: "Decentralized version control powered by IPFS and blockchain",
	// Execute formats runtime errors consistently with Git.
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// Argument errors opt into usage because runtime errors suppress it globally.
func argsWithUsage(validate cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := validate(cmd, args); err != nil {
			cmd.Println(cmd.UsageString())
			return err
		}
		return nil
	}
}

// Required-flag errors also opt into usage.
func requiredFlagsUsage(cmd *cobra.Command, _ []string) error {
	if err := cmd.ValidateRequiredFlags(); err != nil {
		cmd.Println(cmd.UsageString())
		return err
	}
	return nil
}

func init() {
	rootCmd.AddCommand(remoteCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(pullCmd)
}
