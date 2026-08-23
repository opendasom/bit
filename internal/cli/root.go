// Package cli defines the `bit` command-line interface: init, clone, push,
// pull, and remote management. Each command file wires cobra flags and
// .bit/config.json state to the workflows in internal/app.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the top-level `bit` command; subcommands register themselves
// onto it via their own init() functions.
var rootCmd = &cobra.Command{
	Use:   "bit",
	Short: "Decentralized version control powered by IPFS and blockchain",
	// Errors are printed by Execute in git's lowercase "error: ..." style
	// instead of cobra's default "Error: ..." plus a usage dump.
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// argsWithUsage wraps a positional-argument validator so that a mismatch
// (wrong number of args, etc.) prints the command's usage before the error
// is reported. Usage is otherwise silenced globally for runtime errors via
// rootCmd.SilenceUsage, so this is opt-in per command via its Args field.
func argsWithUsage(validate cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := validate(cmd, args); err != nil {
			cmd.Println(cmd.UsageString())
			return err
		}
		return nil
	}
}

// requiredFlagsUsage is a PreRunE hook that prints the command's usage when
// a flag marked required (via MarkFlagRequired) is missing. It runs cobra's
// own required-flag check early, before cobra would otherwise run it
// silently later in the same command's execute().
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
