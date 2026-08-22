package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/opendasom/bit/internal/config"
)

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
