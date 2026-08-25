package connector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"itsm-backend/ent"
	"itsm-backend/ent/connectorconfig"
	"itsm-backend/middleware"
)

type PersistentConfigStore struct {
	client     *ent.Client
	encryption *middleware.EncryptionService
}

func NewPersistentConfigStore(client *ent.Client, encryptionSecret string) (*PersistentConfigStore, error) {
	if client == nil {
		return nil, errors.New("connector config store requires database client")
	}
	if len(strings.TrimSpace(encryptionSecret)) < 16 {
		return nil, errors.New("CONNECTOR_CONFIG_ENCRYPTION_KEY must contain at least 16 characters")
	}
	return &PersistentConfigStore{client: client, encryption: middleware.NewEncryptionService(encryptionSecret)}, nil
}

func (s *PersistentConfigStore) Save(ctx context.Context, cfg Config) error {
	if cfg.TenantID <= 0 || cfg.Name == "" || cfg.Provider == "" {
		return errors.New("invalid connector config")
	}
	if cfg.Name == "email" && cfg.Enabled {
		count, err := s.client.ConnectorConfig.Query().Where(connectorconfig.TenantIDEQ(cfg.TenantID), connectorconfig.NameEQ("email"), connectorconfig.EnabledEQ(true), connectorconfig.ProviderNEQ(cfg.Provider)).Count(ctx)
		if err != nil {
			return err
		}
		if count > 0 {
			return errors.New("only one enabled email connector is allowed per tenant")
		}
	}
	credentials, err := json.Marshal(cfg.Credentials)
	if err != nil {
		return err
	}
	encrypted, err := s.encryption.Encrypt(string(credentials))
	if err != nil {
		return err
	}
	existing, err := s.client.ConnectorConfig.Query().Where(connectorconfig.TenantIDEQ(cfg.TenantID), connectorconfig.NameEQ(cfg.Name), connectorconfig.ProviderEQ(cfg.Provider)).Only(ctx)
	if ent.IsNotFound(err) {
		_, err = s.client.ConnectorConfig.Create().SetTenantID(cfg.TenantID).SetName(cfg.Name).SetProvider(cfg.Provider).SetConnectorType(string(cfg.Type)).SetEnabled(cfg.Enabled).SetEncryptedCredentials(encrypted).SetSettings(cfg.Settings).SetLabels(cfg.Labels).Save(ctx)
		return err
	}
	if err != nil {
		return err
	}
	update := s.client.ConnectorConfig.UpdateOneID(existing.ID).SetConnectorType(string(cfg.Type)).SetEnabled(cfg.Enabled).SetSettings(cfg.Settings).SetLabels(cfg.Labels)
	if len(cfg.Credentials) > 0 {
		update.SetEncryptedCredentials(encrypted)
	}
	_, err = update.Save(ctx)
	return err
}

func (s *PersistentConfigStore) Delete(ctx context.Context, tenantID int, name, provider string) error {
	_, err := s.client.ConnectorConfig.Delete().Where(connectorconfig.TenantIDEQ(tenantID), connectorconfig.NameEQ(name), connectorconfig.ProviderEQ(provider)).Exec(ctx)
	return err
}

func (s *PersistentConfigStore) LoadAll(ctx context.Context) ([]Config, error) {
	entities, err := s.client.ConnectorConfig.Query().Where(connectorconfig.EnabledEQ(true)).All(ctx)
	if err != nil {
		return nil, err
	}
	configs := make([]Config, 0, len(entities))
	for _, entity := range entities {
		plain, decryptErr := s.encryption.Decrypt(entity.EncryptedCredentials)
		if decryptErr != nil {
			return nil, fmt.Errorf("decrypt connector config %d: %w", entity.ID, decryptErr)
		}
		credentials := map[string]string{}
		if plain != "" {
			if decodeErr := json.Unmarshal([]byte(plain), &credentials); decodeErr != nil {
				return nil, fmt.Errorf("decode connector credentials %d: %w", entity.ID, decodeErr)
			}
		}
		configs = append(configs, Config{TenantID: entity.TenantID, Name: entity.Name, Provider: entity.Provider, Type: ConnectorType(entity.ConnectorType), Enabled: entity.Enabled, Credentials: credentials, Settings: entity.Settings, Labels: entity.Labels, CreatedAt: entity.CreatedAt, UpdatedAt: entity.UpdatedAt})
	}
	return configs, nil
}
