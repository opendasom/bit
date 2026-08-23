package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/opendasom/bit/internal/config"
)

// resolveSigningKey picks the private key to sign transactions with, in
// order of preference: the --key flag, the BIT_PRIVATE_KEY environment
// variable, then the legacy privateKey field in .bit/config.json. The flag
// and legacy config paths are discouraged since they can leak the key
// through shell history or a committed file.
func resolveSigningKey(flagValue string, cfg *config.Config) (string, error) {
	if key := strings.TrimSpace(flagValue); key != "" {
		fmt.Fprintln(os.Stderr, "warning: --key can be exposed through shell history; prefer BIT_PRIVATE_KEY")
		return key, nil
	}
	if key := strings.TrimSpace(os.Getenv("BIT_PRIVATE_KEY")); key != "" {
		return key, nil
	}
	if cfg != nil {
		if key := strings.TrimSpace(cfg.PrivateKey); key != "" {
			fmt.Fprintln(os.Stderr, "warning: using legacy privateKey from .bit/config.json; move it to BIT_PRIVATE_KEY")
			return key, nil
		}
	}
	return "", errors.New("signing key is required; set BIT_PRIVATE_KEY")
}
