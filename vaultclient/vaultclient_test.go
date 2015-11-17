package vaultclient

import (
	"log"
	"os"
	"testing"
)

const (
	testSecretPath = "secret/testing/test_value"
)

var tconfig = VaultConfig{
	Server:     os.Getenv("VAULT_ADDR"),
	AppID:      os.Getenv("VAULT_APP_ID"),
	UserIDPath: os.Getenv("VAULT_USER_ID_PATH"),
}

func TestVaultAppIDAuth(t *testing.T) {
	vc, err := NewClient(&tconfig)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	err = vc.AppIDAuth()
	if err != nil {
		log.Fatalf("Error authenticating: %v", err)
	}
}

func TestVaultGetValue(t *testing.T) {
	vc, err := NewClient(&tconfig)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	err = vc.AppIDAuth()
	if err != nil {
		log.Fatalf("Error authenticating: %v", err)
	}
	d, err := vc.GetValue(testSecretPath)
	if err != nil {
		log.Fatalf("Error getting value: %v", err)
	}
	log.Printf("Got value: %v", d.(string))
}
