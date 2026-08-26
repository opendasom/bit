package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/opendasom/bit/internal/app"
	"github.com/opendasom/bit/internal/chain"
	"github.com/opendasom/bit/internal/config"
	"github.com/opendasom/bit/internal/ipfs"
	"github.com/opendasom/bit/internal/repo"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:     "init",
	Short:   "Initialize a new bit repository",
	Args:    argsWithUsage(cobra.NoArgs),
	PreRunE: requiredFlagsUsage,
	RunE: func(cmd *cobra.Command, args []string) error {
		rpcURL, _ := cmd.Flags().GetString("rpc")
		contractAddress, _ := cmd.Flags().GetString("contract")
		privateKey, _ := cmd.Flags().GetString("key")
		ipfsURL, _ := cmd.Flags().GetString("ipfs")
		repoName, _ := cmd.Flags().GetString("name")
		repoDescription, _ := cmd.Flags().GetString("description")
		defaultBranch, _ := cmd.Flags().GetString("branch")

		if _, err := os.Stat(".git"); os.IsNotExist(err) {
			return fmt.Errorf(".git directory not found; run git init first")
		}
		if repoName == "" {
			repoName = filepath.Base(mustGetwd())
		}
		if defaultBranch == "" {
			defaultBranch = "main"
		}
		privateKey, err := resolveSigningKey(privateKey, nil)
		if err != nil {
			return err
		}
		if err := config.EnsureLocalExclude("."); err != nil {
			return fmt.Errorf("failed to configure .bit exclude: %w", err)
		}

		metadata := &repo.Metadata{
			Version:       repo.MetadataVersion,
			Name:          repoName,
			Description:   repoDescription,
			DefaultBranch: defaultBranch,
		}

		chainClient, err := chain.NewClient(rpcURL, contractAddress, privateKey)
		if err != nil {
			return fmt.Errorf("chain connection failed: %w", err)
		}
		defer chainClient.Close()
		ipfsClient := ipfs.NewClient(ipfsURL)

		repoID, metadataCID, err := app.Init(chainClient, ipfsClient, metadata)
		if err != nil {
			return err
		}
		fmt.Printf("created chain repo (repoId: %s)\n", repoID.String())
		fmt.Printf("registered repo metadata: %s\n", metadataCID)

		cfg := &config.Config{
			RPCURL:          rpcURL,
			ContractAddress: contractAddress,
			IPFSURL:         ipfsURL,
			Remotes:         make(map[string]config.Remote),
			RepoID:          repoID.Uint64(),
		}
		if err := config.Save(".", cfg); err != nil {
			return fmt.Errorf("config save failed: %w", err)
		}
		fmt.Println("config saved (.bit/config.json)")

		return nil
	},
}

func init() {
	initCmd.Flags().String("rpc", "", "Ethereum RPC URL (e.g. https://mainnet.infura.io/v3/...)")
	initCmd.Flags().String("contract", "", "BitRegistry contract address")
	initCmd.Flags().String("key", "", "deprecated: wallet private key; prefer the BIT_PRIVATE_KEY environment variable")
	initCmd.Flags().String("ipfs", "http://localhost:5001", "IPFS node address")
	initCmd.Flags().String("name", "", "repo name to display on the web (default: current directory name)")
	initCmd.Flags().String("description", "", "repo description to display on the web")
	initCmd.Flags().String("branch", "main", "default branch to show on the web")

	initCmd.MarkFlagRequired("rpc")
	initCmd.MarkFlagRequired("contract")

	rootCmd.AddCommand(initCmd)
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
