// Command bit is the entry point for the Bit CLI, a decentralized version
// control tool that records commit history on-chain and stores content on
// IPFS. See internal/cli for the available subcommands.
package main

import (
	"github.com/opendasom/bit/internal/cli"
)

func main() {
	cli.Execute()
}
