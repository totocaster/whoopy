package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/toto/whoopy/internal/paths"
)

const defaultAPIBaseURL = "https://api.prod.whoop.com/developer/v2"

// Config captures user/tenant level configuration.
type Config struct {
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
	APIBaseURL   string `toml:"api_base_url"`
}

// Load reads configuration from environment variables with optional overrides from config.toml.
func Load() (*Config, error) {
	cfg := &Config{
		APIBaseURL: defaultAPIBaseURL,
	}

	if err := mergeFile(cfg); err != nil {
		return nil, err
	}
	mergeEnv(cfg)

	if cfg.ClientID == "" {
		return nil, errors.New("missing client_id (set WHOOPY_CLIENT_ID or config file)")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("missing client_secret (set WHOOPY_CLIENT_SECRET or config file)")
	}

	return cfg, nil
}

func mergeFile(cfg *Config) error {
	path, err := paths.ConfigFile()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read config file: %w", err)
	}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}
	return nil
}

func mergeEnv(cfg *Config) {
	if v := strings.TrimSpace(os.Getenv("WHOOPY_CLIENT_ID")); v != "" {
		cfg.ClientID = v
	}
	if v := strings.TrimSpace(os.Getenv("WHOOPY_CLIENT_SECRET")); v != "" {
		cfg.ClientSecret = v
	}
	if v := strings.TrimSpace(os.Getenv("WHOOPY_API_BASE_URL")); v != "" {
		cfg.APIBaseURL = v
	}
}
