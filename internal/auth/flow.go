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
	"github.com/toto/whoopy/internal/tokens"
)

const (
	authPath   = "/oauth2/auth"
	tokenPath  = "/oauth2/token"
	revokePath = "/oauth2/revoke"
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

func (f *Flow) revokeEndpoint() string {
	return f.authURLBase() + revokePath
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
	return token, nil
}

// Refresh looks up the stored refresh token and obtains fresh access credentials.
func (f *Flow) Refresh(ctx context.Context) (*tokens.Token, error) {
	current, err := f.store.Load()
	if err != nil {
		return nil, err
	}
	if current == nil || current.RefreshToken == "" {
		return nil, errors.New("no refresh token available")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", current.RefreshToken)
	form.Set("client_id", f.cfg.ClientID)
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
	return token, nil
}

// Logout revokes the refresh token remotely (best-effort) and clears local cache.
func (f *Flow) Logout(ctx context.Context) error {
	current, err := f.store.Load()
	if err != nil {
		return err
	}
	if current != nil && current.RefreshToken != "" {
		form := url.Values{}
		form.Set("token", current.RefreshToken)
		form.Set("token_type_hint", "refresh_token")
		form.Set("client_id", f.cfg.ClientID)
		if f.cfg.ClientSecret != "" {
			form.Set("client_secret", f.cfg.ClientSecret)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.revokeEndpoint(), strings.NewReader(form.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, httpErr := f.httpClient.Do(req)
			if httpErr == nil {
				resp.Body.Close()
			}
		}
	}
	return f.store.Clear()
}

func (f *Flow) postToken(ctx context.Context, form url.Values) (*tokens.Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.tokenEndpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute token request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body := ioReadLimited(resp.Body)
		return nil, fmt.Errorf("token request failed: %d %s: %s", resp.StatusCode, resp.Status, body)
	}
	var payload tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if payload.AccessToken == "" || payload.RefreshToken == "" {
		return nil, errors.New("token response missing access or refresh token")
	}
	expiresIn := payload.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	expiresAt := f.now().Add(time.Duration(expiresIn) * time.Second)
	scope := strings.Fields(payload.Scope)
	return &tokens.Token{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		TokenType:    payload.TokenType,
		Scope:        scope,
		ExpiresAt:    expiresAt,
	}, nil
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
	buf := make([]byte, limit)
	n, _ := body.Read(buf)
	return string(buf[:n])
}
