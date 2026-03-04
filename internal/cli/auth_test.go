package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthManualFlow(t *testing.T) {
	t.Setenv("WHOOPY_CLIENT_ID", "")
	t.Setenv("WHOOPY_CLIENT_SECRET", "")
	t.Setenv("WHOOPY_OAUTH_BASE_URL", "")
	t.Setenv("WHOOPY_REDIRECT_URI", "")

	setTestConfigDir(t)
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
			w.Write([]byte(`{"access_token":"token-access","refresh_token":"token-refresh","token_type":"Bearer","expires_in":3600,"scope":"offline read:sleep"}`))
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

func TestEnsureConfigTemplate(t *testing.T) {
	setTestConfigDir(t)
	path := filepath.Join(getConfigDir(t), "config.toml")

	created, templatePath, err := ensureConfigTemplate()
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, path, templatePath)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), `client_id = ""`)

	createdAgain, _, err := ensureConfigTemplate()
	require.NoError(t, err)
	require.False(t, createdAgain)
}
