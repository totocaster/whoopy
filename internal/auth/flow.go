package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/toto/whoopy/internal/config"
	"github.com/toto/whoopy/internal/debuglog"
	"github.com/toto/whoopy/internal/tokens"
)

const (
	authPath         = "/oauth2/auth"
	tokenPath        = "/oauth2/token"
	revokeAccessPath = "/user/access"
)

var defaultScopes = []string{
	"offline",
	"read:profile",
	"read:body_measurement",
	"read:cycles",
	"read:recovery",
	"read:sleep",
	"read:workout",
}

// Flow orchestrates OAuth interactions with WHOOP.
type Flow struct {
	cfg        *config.Config
	store      *tokens.Store
	httpClient *http.Client
	now        func() time.Time
}

// NewFlow returns a Flow with sane defaults.
func NewFlow(cfg *config.Config, store *tokens.Store) *Flow {
	return &Flow{
		cfg:   cfg,
		store: store,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		now: time.Now,
	}
}

func (f *Flow) authURLBase() string {
	return strings.TrimRight(f.cfg.OAuthBaseURL, "/")
}

func (f *Flow) tokenEndpoint() string {
	return f.authURLBase() + tokenPath
}

func (f *Flow) revokeAccessEndpoint() string {
	return strings.TrimRight(f.cfg.APIBaseURL, "/") + revokeAccessPath
}

// BuildAuthURL creates the URL users should open to authorize the CLI.
func (f *Flow) BuildAuthURL(redirectURI, state string, pkce *PKCE) (string, error) {
	base := f.authURLBase() + authPath
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse auth url: %w", err)
	}

	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", f.cfg.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", strings.Join(defaultScopes, " "))
	q.Set("state", state)
	if pkce != nil {
		q.Set("code_challenge", pkce.Challenge)
		q.Set("code_challenge_method", pkce.Method)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// ExchangeCode swaps an authorization code for tokens and persists them.
func (f *Flow) ExchangeCode(ctx context.Context, code, redirectURI string, pkce *PKCE) (*tokens.Token, error) {
	if code == "" {
		return nil, errors.New("authorization code is empty")
	}
	debuglog.Info("exchanging authorization code", "redirect_uri", redirectURI, "pkce", pkce != nil)
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", f.cfg.ClientID)
	if pkce != nil {
		form.Set("code_verifier", pkce.Verifier)
	}
	if f.cfg.ClientSecret != "" {
		form.Set("client_secret", f.cfg.ClientSecret)
	}

	token, err := f.postToken(ctx, form)
	if err != nil {
		return nil, err
	}

	if err := f.store.Save(token); err != nil {
		return nil, err
	}
	debuglog.Info("authorization code exchange completed", "expires_at", token.ExpiresAt.UTC().Format(time.RFC3339), "scopes", strings.Join(token.Scope, " "))
	return token, nil
}

// Refresh looks up the stored refresh token and obtains fresh access credentials.
func (f *Flow) Refresh(ctx context.Context) (*tokens.Token, error) {
	current, err := f.store.Load()
	if err != nil {
		return nil, err
	}
	if current == nil || current.RefreshToken == "" {
		debuglog.Warn("refresh requested without a stored refresh token")
		return nil, errors.New("no refresh token available")
	}

	debuglog.Info("refreshing access token", "expires_at", current.ExpiresAt.UTC().Format(time.RFC3339), "scopes", strings.Join(current.Scope, " "), "store_path", f.store.Path())
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", current.RefreshToken)
	form.Set("client_id", f.cfg.ClientID)
	if f.cfg.ClientSecret != "" {
		form.Set("client_secret", f.cfg.ClientSecret)
	}
	form.Set("scope", "offline")
	token, err := f.postToken(ctx, form)
	if err != nil {
		debuglog.Error("refresh token request failed", "error", err)
		return nil, err
	}
	if err := f.store.Save(token); err != nil {
		return nil, err
	}
	debuglog.Info("access token refresh completed", "expires_at", token.ExpiresAt.UTC().Format(time.RFC3339), "scopes", strings.Join(token.Scope, " "))
	return token, nil
}

// Logout revokes the refresh token remotely (best-effort) and clears local cache.
func (f *Flow) Logout(ctx context.Context) error {
	current, err := f.store.Load()
	if err != nil {
		return err
	}
	if current != nil && current.AccessToken != "" && strings.TrimSpace(f.cfg.APIBaseURL) != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, f.revokeAccessEndpoint(), nil)
		if err != nil {
			debuglog.Warn("failed to build revoke request", "error", err)
		} else {
			req.Header.Set("Authorization", "Bearer "+current.AccessToken)
			req.Header.Set("Accept", "application/json")
			resp, httpErr := f.httpClient.Do(req)
			if httpErr != nil {
				debuglog.Warn("revoke request failed", "error", httpErr)
			} else {
				body := ioReadLimited(resp.Body)
				_ = resp.Body.Close()
				if resp.StatusCode/100 == 2 {
					debuglog.Info("remote access revoked", "status", resp.Status)
				} else {
					debuglog.Warn("remote revoke returned non-success", "status", resp.Status, "body", strings.TrimSpace(body))
				}
			}
		}
	}
	debuglog.Info("clearing local tokens", "store_path", f.store.Path())
	return f.store.Clear()
}

func (f *Flow) postToken(ctx context.Context, form url.Values) (*tokens.Token, error) {
	grantType := form.Get("grant_type")
	debuglog.Debug("sending token request", "grant_type", grantType, "endpoint", f.tokenEndpoint())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.tokenEndpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		debuglog.Error("failed to build token request", "grant_type", grantType, "error", err)
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := f.httpClient.Do(req)
	if err != nil {
		debuglog.Error("token request transport error", "grant_type", grantType, "error", err)
		return nil, fmt.Errorf("execute token request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body := ioReadLimited(resp.Body)
		debuglog.Error("token request returned non-success", "grant_type", grantType, "status", resp.Status, "body", strings.TrimSpace(body))
		return nil, fmt.Errorf("token request failed: %d %s: %s", resp.StatusCode, resp.Status, body)
	}
	var payload tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		debuglog.Error("failed to decode token response", "grant_type", grantType, "error", err)
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if payload.AccessToken == "" || payload.RefreshToken == "" {
		debuglog.Error("token response missing required fields", "grant_type", grantType, "has_access_token", payload.AccessToken != "", "has_refresh_token", payload.RefreshToken != "")
		return nil, errors.New("token response missing access or refresh token")
	}
	expiresIn := payload.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	expiresAt := f.now().Add(time.Duration(expiresIn) * time.Second)
	scope := strings.Fields(payload.Scope)
	token := &tokens.Token{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		TokenType:    payload.TokenType,
		Scope:        scope,
		ExpiresAt:    expiresAt,
	}
	debuglog.Info("token request succeeded", "grant_type", grantType, "expires_in", expiresIn, "expires_at", expiresAt.UTC().Format(time.RFC3339), "scope", payload.Scope, "token_type", payload.TokenType)
	return token, nil
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

func ioReadLimited(body io.Reader) string {
	const limit = 4 * 1024
	data, _ := io.ReadAll(io.LimitReader(body, limit))
	return string(data)
}
