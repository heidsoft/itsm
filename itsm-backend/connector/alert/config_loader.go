package alert

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfigFile loads one alert source definition and expands environment
// variables so secrets do not need to be committed to YAML.
func LoadConfigFile(path string) (*AlertSourceConfig, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read alert source config: %w", err)
	}
	var cfg AlertSourceConfig
	if err := yaml.Unmarshal([]byte(os.ExpandEnv(string(body))), &cfg); err != nil {
		return nil, fmt.Errorf("decode alert source config: %w", err)
	}
	if cfg.Source == "" || cfg.Type != "webhook" {
		return nil, fmt.Errorf("alert source config requires source and type=webhook")
	}
	return &cfg, nil
}
