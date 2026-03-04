package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAuthLoginStatusLogoutManualFlow(t *testing.T) {
	t.Setenv("WHOOPY_CONFIG_DIR", t.TempDir())
	configPath := filepath.Join(getConfigDir(t), "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o700))

	var revokeCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			require.NoError(t, r.ParseForm())
			require.Equal(t, "authorization_code", r.Form.Get("grant_type"))
			require.Equal(t, "code-123", r.Form.Get("code"))
			require.Equal(t, "http://127.0.0.1:8735/oauth/callback", r.Form.Get("redirect_uri"))
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"token-access","refresh_token":"token-refresh","token_type":"Bearer","expires_in":3600,"scope":"offline sleep.read"}`))
		case "/oauth2/revoke":
			revokeCount++
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	configContent := `
client_id = "client-id"
client_secret = "client-secret"
oauth_base_url = "` + server.URL + `"
redirect_uri = "http://127.0.0.1:8735/oauth/callback"
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o600))

	origStateGenerator := stateGenerator
	stateGenerator = func() (string, error) { return "fixed-state", nil }
	t.Cleanup(func() { stateGenerator = origStateGenerator })

	origBrowser := openBrowserFunc
	openBrowserFunc = func(string) error { return nil }
	t.Cleanup(func() { openBrowserFunc = origBrowser })

	loginInput := "http://127.0.0.1:8735/oauth/callback?code=code-123&state=fixed-state\n"
	loginOut := runCLICommand(t, []string{"auth", "login", "--manual", "--no-browser"}, loginInput)
	require.Contains(t, loginOut, "Authorization complete.")

	statusOut := runCLICommand(t, []string{"auth", "status"}, "")
	require.Contains(t, statusOut, "Access token expires")
	require.Contains(t, statusOut, "Scopes:")

	logoutOut := runCLICommand(t, []string{"auth", "logout"}, "")
	require.Contains(t, logoutOut, "Logged out")
	require.Equal(t, 1, revokeCount)

	tokenFile := filepath.Join(getConfigDir(t), "tokens.json")
	_, err := os.Stat(tokenFile)
	require.True(t, os.IsNotExist(err))
}

func runCLICommand(t *testing.T, args []string, input string) string {
	t.Helper()
	var buf bytes.Buffer
	rootCmd.SetArgs(args)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetIn(strings.NewReader(input))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := Execute(ctx)
	rootCmd.SetIn(os.Stdin)
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	rootCmd.SetArgs(nil)
	require.NoError(t, err)
	return buf.String()
}

func getConfigDir(t *testing.T) string {
	t.Helper()
	base := os.Getenv("WHOOPY_CONFIG_DIR")
	require.NotEmpty(t, base, "WHOOPY_CONFIG_DIR must be set in tests")
	return filepath.Join(base, "whoopy")
}
