package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

const appDirName = "whoopy"

var (
	overrideMu        sync.RWMutex
	configDirOverride string
	stateDirOverride  string
)

// SetConfigDirOverride forces ConfigDir to use the provided directory. Intended for tests.
func SetConfigDirOverride(path string) {
	overrideMu.Lock()
	defer overrideMu.Unlock()
	configDirOverride = path
}

// SetStateDirOverride forces StateDir to use the provided directory. Intended for tests.
func SetStateDirOverride(path string) {
	overrideMu.Lock()
	defer overrideMu.Unlock()
	stateDirOverride = path
}

// ConfigDir returns the OS-specific configuration directory for whoopy, ensuring it exists.
func ConfigDir() (string, error) {
	overrideMu.RLock()
	override := configDirOverride
	overrideMu.RUnlock()
	if override != "" {
		return ensureDir(override, "config")
	}

	var base string
	if env := os.Getenv("WHOOPY_CONFIG_DIR"); env != "" {
		return ensureDir(env, "config")
	} else if env := validAbsoluteEnv("XDG_CONFIG_HOME"); env != "" {
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

	return ensureDir(filepath.Join(base, appDirName), "config")
}

// StateDir returns the OS-specific state directory for whoopy, ensuring it exists.
func StateDir() (string, error) {
	overrideMu.RLock()
	override := stateDirOverride
	overrideMu.RUnlock()
	if override != "" {
		return ensureDir(override, "state")
	}

	var base string
	switch {
	case os.Getenv("WHOOPY_STATE_DIR") != "":
		return ensureDir(os.Getenv("WHOOPY_STATE_DIR"), "state")
	case os.Getenv("WHOOPY_CONFIG_DIR") != "":
		// Preserve a simple single-dir override for callers that intentionally colocate everything.
		return ensureDir(os.Getenv("WHOOPY_CONFIG_DIR"), "state")
	case validAbsoluteEnv("XDG_STATE_HOME") != "":
		base = validAbsoluteEnv("XDG_STATE_HOME")
	case runtime.GOOS == "windows":
		base = os.Getenv("APPDATA")
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		base = filepath.Join(home, ".local", "state")
	}

	if base == "" {
		return "", errors.New("unable to determine state directory")
	}

	return ensureDir(filepath.Join(base, appDirName), "state")
}

// TokensFile returns the location for the persisted OAuth tokens file.
func TokensFile() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tokens.json"), nil
}

// LegacyTokensFile returns the pre-XDG-state token path kept for migration support.
func LegacyTokensFile() (string, error) {
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

// DebugLogFile returns the location for structured debug logs.
func DebugLogFile() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "logs", "debug.log"), nil
}

func ensureDir(path, kind string) (string, error) {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return "", fmt.Errorf("create %s directory: %w", kind, err)
	}
	return path, nil
}

func validAbsoluteEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		return ""
	}
	if !filepath.IsAbs(value) {
		return ""
	}
	return value
}
