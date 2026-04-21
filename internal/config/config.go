package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/toto/whoopy/internal/paths"
)

const (
	defaultAPIBaseURL  = "https://api.prod.whoop.com/developer/v2"
	defaultOAuthBase   = "https://api.prod.whoop.com/oauth"
	defaultRedirectURI = "http://127.0.0.1:8735/oauth/callback"
)

// Config captures user/tenant level configuration.
type Config struct {
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
	APIBaseURL   string `toml:"api_base_url"`
	OAuthBaseURL string `toml:"oauth_base_url"`
	RedirectURI  string `toml:"redirect_uri"`
}

// Load reads configuration from environment variables with optional overrides from config.toml.
func Load() (*Config, error) {
	cfg, err := LoadAllowEmpty()
	if err != nil {
		return nil, err
	}

	if cfg.ClientID == "" {
		return nil, errors.New("missing client_id (set WHOOPY_CLIENT_ID or config file)")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("missing client_secret (set WHOOPY_CLIENT_SECRET or config file)")
	}

	return cfg, nil
}

// LoadAllowEmpty mirrors Load but does not enforce client credentials, making it suitable for diagnostics.
func LoadAllowEmpty() (*Config, error) {
	cfg := &Config{
		APIBaseURL:   defaultAPIBaseURL,
		OAuthBaseURL: defaultOAuthBase,
		RedirectURI:  defaultRedirectURI,
	}

	if err := mergeFile(cfg); err != nil {
		return nil, err
	}
	mergeEnv(cfg)
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
	if v := firstEnv("WHOOPY_CLIENT_ID", "WHOOP_CLIENT_ID"); v != "" {
		cfg.ClientID = v
	}
	if v := firstEnv("WHOOPY_CLIENT_SECRET", "WHOOP_CLIENT_SECRET"); v != "" {
		cfg.ClientSecret = v
	}
	if v := strings.TrimSpace(os.Getenv("WHOOPY_API_BASE_URL")); v != "" {
		cfg.APIBaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("WHOOPY_OAUTH_BASE_URL")); v != "" {
		cfg.OAuthBaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("WHOOPY_REDIRECT_URI")); v != "" {
		cfg.RedirectURI = v
	}
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}
