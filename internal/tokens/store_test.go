package tokens_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/paths"
	"github.com/toto/whoopy/internal/tokens"
)

func TestStoreSaveLoadClear(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "whoopy")
	paths.SetConfigDirOverride(dir)
	t.Cleanup(func() { paths.SetConfigDirOverride("") })
	path := filepath.Join(dir, "nested", "tokens.json")
	store, err := tokens.NewStore(path)
	require.NoError(t, err)

	token := &tokens.Token{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		Scope:        []string{"offline", "read:sleep"},
		ExpiresAt:    time.Now().Add(2 * time.Hour).UTC().Round(time.Second),
	}

	require.NoError(t, store.Save(token))

	loaded, err := store.Load()
	require.NoError(t, err)
	require.Equal(t, token, loaded)

	require.NoError(t, store.Clear())

	loaded, err = store.Load()
	require.NoError(t, err)
	require.Nil(t, loaded)
}

func TestStoreSaveNilToken(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "whoopy")
	paths.SetConfigDirOverride(dir)
	t.Cleanup(func() { paths.SetConfigDirOverride("") })
	path := filepath.Join(dir, "tokens.json")
	store, err := tokens.NewStore(path)
	require.NoError(t, err)

	err = store.Save(nil)
	require.Error(t, err)
}

func TestStorePath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "whoopy")
	paths.SetConfigDirOverride(dir)
	t.Cleanup(func() { paths.SetConfigDirOverride("") })
	path := filepath.Join(dir, "tokens.json")
	store, err := tokens.NewStore(path)
	require.NoError(t, err)
	require.Equal(t, path, store.Path())

	info, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestStoreMigratesLegacyTokenFile(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config", "whoopy")
	stateDir := filepath.Join(root, "state", "whoopy")
	paths.SetConfigDirOverride(configDir)
	paths.SetStateDirOverride(stateDir)
	t.Cleanup(func() {
		paths.SetConfigDirOverride("")
		paths.SetStateDirOverride("")
	})

	legacyPath := filepath.Join(configDir, "tokens.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(legacyPath), 0o700))

	token := &tokens.Token{
		AccessToken:  "legacy-access",
		RefreshToken: "legacy-refresh",
		TokenType:    "Bearer",
		Scope:        []string{"offline"},
		ExpiresAt:    time.Now().Add(time.Hour).UTC().Round(time.Second),
	}
	data, err := json.Marshal(token)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(legacyPath, data, 0o600))

	store, err := tokens.NewStore("")
	require.NoError(t, err)

	loaded, err := store.Load()
	require.NoError(t, err)
	require.Equal(t, token, loaded)

	_, err = os.Stat(legacyPath)
	require.True(t, os.IsNotExist(err))

	_, err = os.Stat(filepath.Join(stateDir, "tokens.json"))
	require.NoError(t, err)
}
