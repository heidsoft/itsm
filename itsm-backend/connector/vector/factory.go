package vector

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type VectorStoreConfig struct {
	Backend    string                 `yaml:"backend"`
	Collection string                 `yaml:"collection"`
	Fallback   bool                   `yaml:"fallback"`
	Config     map[string]interface{} `yaml:"config"`
}

// LoadConfig expands ${ENV_VAR} expressions before parsing YAML. source may be
// YAML itself or a file path.
func LoadConfig(source string) (VectorStoreConfig, error) {
	var cfg VectorStoreConfig
	if strings.TrimSpace(source) == "" {
		cfg.Backend = "keyword"
		cfg.Fallback = true
		cfg.Config = map[string]interface{}{}
		return cfg, nil
	}
	body := []byte(source)
	if info, err := os.Stat(source); err == nil && !info.IsDir() {
		body, err = os.ReadFile(source)
		if err != nil {
			return cfg, fmt.Errorf("read vector config: %w", err)
		}
	}
	if err := yaml.Unmarshal([]byte(os.ExpandEnv(string(body))), &cfg); err != nil {
		return cfg, fmt.Errorf("parse vector config: %w", err)
	}
	if cfg.Backend == "" {
		cfg.Backend = "keyword"
	}
	if cfg.Config == nil {
		cfg.Config = map[string]interface{}{}
	}
	return cfg, nil
}

func NewFromConfig(ctx context.Context, cfg VectorStoreConfig) (VectorStore, error) {
	fallback, _ := New("keyword", cfg.Collection, cfg.Config)
	if strings.EqualFold(cfg.Backend, "keyword") {
		return fallback, nil
	}
	primary, err := New(cfg.Backend, cfg.Collection, cfg.Config)
	if err == nil {
		err = primary.Ping(ctx)
	}
	if err != nil {
		if primary != nil {
			_ = primary.Close()
		}
		if cfg.Fallback {
			return fallback, nil
		}
		return nil, err
	}
	if !cfg.Fallback {
		_ = fallback.Close()
		return primary, nil
	}
	return &fallbackStore{primary: primary, fallback: fallback}, nil
}

func NewFromEnvironment(ctx context.Context) (VectorStore, error) {
	cfg, err := LoadConfig(os.Getenv("VECTOR_STORE_CONFIG"))
	if err != nil {
		return nil, err
	}
	return NewFromConfig(ctx, cfg)
}

type fallbackStore struct{ primary, fallback VectorStore }

func (s *fallbackStore) Search(ctx context.Context, r SearchRequest) (SearchResponse, error) {
	out, err := s.primary.Search(ctx, r)
	if err == nil {
		return out, nil
	}
	return s.fallback.Search(ctx, r)
}
func (s *fallbackStore) Insert(ctx context.Context, r InsertRequest) error {
	fallbackErr := s.fallback.Insert(ctx, r)
	if err := s.primary.Insert(ctx, r); err != nil {
		if fallbackErr == nil {
			return nil
		}
		return fmt.Errorf("primary insert: %v; fallback insert: %w", err, fallbackErr)
	}
	return fallbackErr
}
func (s *fallbackStore) Delete(ctx context.Context, ids []string) error {
	if err := s.primary.Delete(ctx, ids); err != nil {
		return err
	}
	return s.fallback.Delete(ctx, ids)
}
func (s *fallbackStore) Ping(ctx context.Context) error { return s.primary.Ping(ctx) }
func (s *fallbackStore) Close() error {
	a := s.primary.Close()
	b := s.fallback.Close()
	if a != nil {
		return a
	}
	return b
}
