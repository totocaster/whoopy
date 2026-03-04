package tokens_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/tokens"
)

func TestStoreSaveLoadClear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "tokens.json")
	store, err := tokens.NewStore(path)
	require.NoError(t, err)

	token := &tokens.Token{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		Scope:        []string{"offline", "sleep.read"},
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
	path := filepath.Join(t.TempDir(), "tokens.json")
	store, err := tokens.NewStore(path)
	require.NoError(t, err)

	err = store.Save(nil)
	require.Error(t, err)
}

func TestStorePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	store, err := tokens.NewStore(path)
	require.NoError(t, err)
	require.Equal(t, path, store.Path())

	info, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	require.True(t, info.IsDir())
}
