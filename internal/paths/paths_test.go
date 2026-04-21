package paths_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/paths"
)

func TestConfigDirUsesExactOverrideEnv(t *testing.T) {
	root := t.TempDir()
	expected := filepath.Join(root, "custom-config")
	t.Setenv("WHOOPY_CONFIG_DIR", expected)
	t.Setenv("XDG_CONFIG_HOME", "")

	dir, err := paths.ConfigDir()
	require.NoError(t, err)
	require.Equal(t, expected, dir)
}

func TestStateDirUsesExactOverrideEnv(t *testing.T) {
	root := t.TempDir()
	expected := filepath.Join(root, "custom-state")
	t.Setenv("WHOOPY_STATE_DIR", expected)
	t.Setenv("XDG_STATE_HOME", "")

	dir, err := paths.StateDir()
	require.NoError(t, err)
	require.Equal(t, expected, dir)
}

func TestStateDirFallsBackToConfigOverride(t *testing.T) {
	root := t.TempDir()
	expected := filepath.Join(root, "single-dir")
	t.Setenv("WHOOPY_CONFIG_DIR", expected)
	t.Setenv("WHOOPY_STATE_DIR", "")
	t.Setenv("XDG_STATE_HOME", "")

	dir, err := paths.StateDir()
	require.NoError(t, err)
	require.Equal(t, expected, dir)
}
