package cli

import (
	"testing"

	"github.com/opendasom/bit/internal/config"
)

func TestResolveSigningKeyPrefersFlagThenEnvironment(t *testing.T) {
	t.Setenv("BIT_PRIVATE_KEY", "environment-key")
	key, err := resolveSigningKey("flag-key", &config.Config{PrivateKey: "legacy-key"})
	if err != nil {
		t.Fatal(err)
	}
	if key != "flag-key" {
		t.Fatalf("key = %q, want flag-key", key)
	}

	key, err = resolveSigningKey("", &config.Config{PrivateKey: "legacy-key"})
	if err != nil {
		t.Fatal(err)
	}
	if key != "environment-key" {
		t.Fatalf("key = %q, want environment-key", key)
	}
}

func TestResolveSigningKeySupportsLegacyReadWithoutPersisting(t *testing.T) {
	t.Setenv("BIT_PRIVATE_KEY", "")
	key, err := resolveSigningKey("", &config.Config{PrivateKey: "legacy-key"})
	if err != nil {
		t.Fatal(err)
	}
	if key != "legacy-key" {
		t.Fatalf("key = %q, want legacy-key", key)
	}
}

func TestResolveSigningKeyRequiresExplicitSecret(t *testing.T) {
	t.Setenv("BIT_PRIVATE_KEY", "")
	if _, err := resolveSigningKey("", &config.Config{}); err == nil {
		t.Fatal("expected missing signing key error")
	}
}
