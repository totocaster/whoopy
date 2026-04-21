package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/config"
	"github.com/toto/whoopy/internal/tokens"
)

func TestNewPKCE(t *testing.T) {
	pkce, err := NewPKCE()
	require.NoError(t, err)
	require.Equal(t, "S256", pkce.Method)
	require.True(t, len(pkce.Verifier) >= 43) // min length for PKCE base64
	require.True(t, len(pkce.Challenge) > 10)
}

func TestBuildAuthURLIncludesScopesAndState(t *testing.T) {
	cfg := &config.Config{
		ClientID:     "client",
		ClientSecret: "secret",
		OAuthBaseURL: "https://example.com/oauth",
	}
	store, err := tokens.NewStore(filepath.Join(t.TempDir(), "tokens.json"))
	require.NoError(t, err)

	flow := NewFlow(cfg, store)
	pkce := &PKCE{Challenge: "challenge", Method: "S256"}
	urlStr, err := flow.BuildAuthURL("http://localhost:1234/callback", "state123", pkce)
	require.NoError(t, err)

	parsed, err := url.Parse(urlStr)
	require.NoError(t, err)
	q := parsed.Query()
	require.Equal(t, "code", q.Get("response_type"))
	require.Equal(t, "client", q.Get("client_id"))
	require.Equal(t, "http://localhost:1234/callback", q.Get("redirect_uri"))
	require.Equal(t, "state123", q.Get("state"))
	require.Equal(t, "challenge", q.Get("code_challenge"))
	require.Equal(t, "S256", q.Get("code_challenge_method"))
	require.Contains(t, q.Get("scope"), "offline")
	require.Contains(t, q.Get("scope"), "read:sleep")
}

func TestExchangeCodeSuccess(t *testing.T) {
	tokenResp := `{"access_token":"new-access","refresh_token":"new-refresh","token_type":"Bearer","expires_in":3600,"scope":"offline read:sleep"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/oauth2/token", r.URL.Path)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "authorization_code", r.Form.Get("grant_type"))
		require.Equal(t, "auth-code", r.Form.Get("code"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(tokenResp))
	}))
	t.Cleanup(server.Close)

	cfg := &config.Config{
		ClientID:     "client",
		ClientSecret: "secret",
		OAuthBaseURL: server.URL,
	}
	store, err := tokens.NewStore(filepath.Join(t.TempDir(), "tokens.json"))
	require.NoError(t, err)
	flow := NewFlow(cfg, store)
	flow.httpClient = server.Client()
	frozen := time.Unix(1_700_000_000, 0).UTC()
	flow.now = func() time.Time { return frozen }

	token, err := flow.ExchangeCode(context.Background(), "auth-code", "http://localhost/callback", &PKCE{Verifier: "v"})
	require.NoError(t, err)
	require.Equal(t, "new-access", token.AccessToken)
	require.Equal(t, "new-refresh", token.RefreshToken)
	require.Equal(t, frozen.Add(3600*time.Second), token.ExpiresAt)

	stored, err := store.Load()
	require.NoError(t, err)
	require.Equal(t, token, stored)
}

func TestRefreshUsesStoredToken(t *testing.T) {
	var wasRefresh bool
	tokenResp := `{"access_token":"refreshed","refresh_token":"refreshed-refresh","token_type":"Bearer","expires_in":1200,"scope":"offline"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		require.Equal(t, "/oauth2/token", r.URL.Path)
		require.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		require.Equal(t, "old-refresh", r.Form.Get("refresh_token"))
		require.Equal(t, "offline", r.Form.Get("scope"))
		wasRefresh = true
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(tokenResp))
	}))
	t.Cleanup(server.Close)

	cfg := &config.Config{
		ClientID:     "client",
		ClientSecret: "secret",
		OAuthBaseURL: server.URL,
	}
	store, err := tokens.NewStore(filepath.Join(t.TempDir(), "tokens.json"))
	require.NoError(t, err)
	require.NoError(t, store.Save(&tokens.Token{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		TokenType:    "Bearer",
		Scope:        []string{"offline"},
		ExpiresAt:    time.Now(),
	}))

	flow := NewFlow(cfg, store)
	flow.httpClient = server.Client()
	flow.now = func() time.Time { return time.Unix(0, 0) }

	token, err := flow.Refresh(context.Background())
	require.NoError(t, err)
	require.True(t, wasRefresh)
	require.Equal(t, "refreshed", token.AccessToken)
}

func TestExchangeCodeErrorIncludesBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad stuff"))
	}))
	t.Cleanup(server.Close)

	cfg := &config.Config{
		ClientID:     "client",
		ClientSecret: "secret",
		OAuthBaseURL: server.URL,
	}
	store, err := tokens.NewStore(filepath.Join(t.TempDir(), "tokens.json"))
	require.NoError(t, err)

	flow := NewFlow(cfg, store)
	flow.httpClient = server.Client()

	_, err = flow.ExchangeCode(context.Background(), "bad", "http://cb", nil)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "bad stuff"))
}

func TestLogoutClearsStoreEvenIfRevokeFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method)
		require.Equal(t, "/user/access", r.URL.Path)
		require.Equal(t, "Bearer access", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	cfg := &config.Config{
		ClientID:     "client",
		ClientSecret: "secret",
		OAuthBaseURL: server.URL,
		APIBaseURL:   server.URL,
	}
	store, err := tokens.NewStore(filepath.Join(t.TempDir(), "tokens.json"))
	require.NoError(t, err)
	require.NoError(t, store.Save(&tokens.Token{
		AccessToken:  "access",
		RefreshToken: "refresh",
	}))

	flow := NewFlow(cfg, store)
	flow.httpClient = server.Client()
	require.NoError(t, flow.Logout(context.Background()))

	token, err := store.Load()
	require.NoError(t, err)
	require.Nil(t, token)
}
