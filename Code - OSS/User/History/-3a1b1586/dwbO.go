package vault

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hashicorp/vault/api"
)

const (
	vaultAddr     = "http://vault:8200"
	vaultToken    = "notify-platform-root"
	secretPath    = "secret/data/notify-platform"
	retryInterval = 5 * time.Second
	maxRetries    = 12
)

type VaultClient struct {
	client *api.Client
}

func NewVaultClient() (*VaultClient, error) {
	config := api.DefaultConfig()
	config.Address = vaultAddr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	client.SetToken(vaultToken)

	return &VaultClient{client: client}, nil
}

func (v *VaultClient) LoadSecrets() error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		secret, err := v.client.Logical().Read(secretPath)
		if err != nil {
			lastErr = err
			log.Printf("Vault read error (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(retryInterval)
			continue
		}

		if secret == nil || secret.Data == nil {
			lastErr = fmt.Errorf("no secrets found at path: %s", secretPath)
			continue
		}

		data, ok := secret.Data["data"].(map[string]interface{})
		if !ok {
			lastErr = fmt.Errorf("invalid secret format at path: %s", secretPath)
			continue
		}

		for key, value := range data {
			strValue, ok := value.(string)
			if !ok {
				log.Printf("Skipping non-string secret for key %s", key)
				continue
			}
			os.Setenv(key, strValue)
			log.Printf("Set env var %s from Vault", key)
		}

		return nil
	}

	return fmt.Errorf("failed to load secrets after %d attempts: %w", maxRetries, lastErr)
}

func InitializeSecretsFromVault() {
	log.Println("Initializing secrets from Vault...")

	client, err := NewVaultClient()
	if err != nil {
		log.Fatalf("Failed to create Vault client: %v", err)
	}

	if err := client.LoadSecrets(); err != nil {
		log.Fatalf("Failed to load secrets from Vault: %v", err)
	}

	log.Println("Secrets initialized from Vault successfully")
}
