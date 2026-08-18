package cmd

import (
	"testing"

	"github.com/opendasom/bit/internal/config"
)

func TestParseRemoteURL(t *testing.T) {
	remote, err := parseRemoteURL("bit://sepolia/0x00000000000000000000000000000000000000aA/42")
	if err != nil {
		t.Fatal(err)
	}
	if remote.Network != "sepolia" || remote.RepoID != 42 {
		t.Fatalf("unexpected remote: %#v", remote)
	}
	if remote.ContractAddress != "0x00000000000000000000000000000000000000AA" {
		t.Fatalf("unexpected checksum address: %s", remote.ContractAddress)
	}
}

func TestParseRemoteURLRejectsAmbiguousValues(t *testing.T) {
	invalid := []string{
		"https://sepolia/0x00000000000000000000000000000000000000AA/1",
		"bit://sepolia/not-an-address/1",
		"bit://sepolia/0x00000000000000000000000000000000000000AA/0",
		"bit://sepolia/0x00000000000000000000000000000000000000AA/1/extra",
		"bit://sepolia/0x00000000000000000000000000000000000000AA/1#main",
		"bit://user@sepolia/0x00000000000000000000000000000000000000AA/1",
		"bit://sepolia:8545/0x00000000000000000000000000000000000000AA/1",
		"bit://sepolia/0x00000000000000000000000000000000000000AA/1?branch=main",
	}
	for _, value := range invalid {
		if _, err := parseRemoteURL(value); err == nil {
			t.Fatalf("expected %q to fail", value)
		}
	}
}

func TestValidateRemoteForConfigRejectsTamperedStoredFields(t *testing.T) {
	cfg := &config.Config{ContractAddress: "0x00000000000000000000000000000000000000AA"}
	remote := config.Remote{
		URL:             "bit://sepolia/0x00000000000000000000000000000000000000AA/42",
		Network:         "mainnet",
		ContractAddress: "0x00000000000000000000000000000000000000bb",
		RepoID:          42,
	}
	if err := validateRemoteForConfig(remote, cfg); err == nil {
		t.Fatal("expected tampered remote fields to be rejected")
	}
}
