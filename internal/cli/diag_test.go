package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/tokens"
)

func TestDiagJSONOutput(t *testing.T) {
	setTestConfigDir(t)
	configPath := filepath.Join(getConfigDir(t), "config.toml")
	require.NoError(t, os.WriteFile(configPath, []byte(`
client_id = "abc"
client_secret = "def"
`), 0o600))

	store, err := tokens.NewStore("")
	require.NoError(t, err)
	require.NoError(t, store.Save(&tokens.Token{
		AccessToken:  "token",
		RefreshToken: "refresh",
		Scope:        []string{"read:profile", "read:body_measurement"},
		ExpiresAt:    time.Now().Add(90 * time.Minute),
	}))

	origHealth := diagHealthCheckFn
	diagHealthCheckFn = func(ctx context.Context) (time.Duration, error) {
		return 120 * time.Millisecond, nil
	}
	defer func() { diagHealthCheckFn = origHealth }()

	output := runCLICommand(t, []string{"diag", "--text=false"}, "")
	require.Contains(t, output, "\"client_id_set\": true")
	require.Contains(t, output, "\"status\": \"ok\"")
	require.Contains(t, output, "\"scope\": [")
}

func TestDiagTextOutputWithErrors(t *testing.T) {
	setTestConfigDir(t)
	origHealth := diagHealthCheckFn
	diagHealthCheckFn = func(ctx context.Context) (time.Duration, error) {
		return 0, errors.New("not authenticated")
	}
	defer func() { diagHealthCheckFn = origHealth }()

	output := runCLICommand(t, []string{"diag", "--text"}, "")
	require.Contains(t, output, "Config")
	require.Contains(t, output, "Tokens")
	require.Contains(t, output, "API")
	require.Contains(t, output, "Status: error")
	require.Contains(t, output, "not authenticated")
}
