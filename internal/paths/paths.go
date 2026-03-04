package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const appDirName = "whoopy"

// ConfigDir returns the OS-specific configuration directory for whoopy, ensuring it exists.
func ConfigDir() (string, error) {
	var base string
	if env := os.Getenv("WHOOPY_CONFIG_DIR"); env != "" {
		base = env
	} else if env := os.Getenv("XDG_CONFIG_HOME"); env != "" {
		base = env
	} else if runtime.GOOS == "windows" {
		base = os.Getenv("APPDATA")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}

	if base == "" {
		return "", errors.New("unable to determine config directory")
	}

	dir := filepath.Join(base, appDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}
	return dir, nil
}

// TokensFile returns the location for the persisted OAuth tokens file.
func TokensFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tokens.json"), nil
}

// ConfigFile returns the default config TOML path for user overrides.
func ConfigFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}
