package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/toto/whoopy/internal/auth"
	"github.com/toto/whoopy/internal/config"
	"github.com/toto/whoopy/internal/debuglog"
	"github.com/toto/whoopy/internal/tokens"
)

const (
	tokenRefreshLeeway    = time.Minute
	defaultRequestTimeout = 30 * time.Second
	defaultMax429Retry    = 3
	defaultUserAgent      = "whoopy/0.1"
)

// ErrNotAuthenticated is returned when no OAuth token is available locally.
var ErrNotAuthenticated = errors.New("whoopy: not authenticated; run 'whoopy auth login'")

type tokenStore interface {
	Load() (*tokens.Token, error)
}

type tokenRefresher interface {
	Refresh(ctx context.Context) (*tokens.Token, error)
}

// Client handles authenticated requests against WHOOP's developer API.
type Client struct {
	cfg        *config.Config
	store      tokenStore
	refresher  tokenRefresher
	httpClient *http.Client

	tokenMu sync.Mutex
	token   *tokens.Token

	now       func() time.Time
	sleepFn   func(context.Context, time.Duration) error
	backoff   func(int) time.Duration
	max429    int
	baseURL   string
	userAgent string
}

// ClientOption customizes a Client during construction.
type ClientOption func(*Client)

// WithUserAgent overrides the default HTTP User-Agent header.
func WithUserAgent(agent string) ClientOption {
	return func(c *Client) {
		if trimmed := strings.TrimSpace(agent); trimmed != "" {
			c.userAgent = trimmed
		}
	}
}

// NewClient constructs a Client using the supplied config and token store.
func NewClient(cfg *config.Config, store *tokens.Store, opts ...ClientOption) *Client {
	client := &Client{
		cfg:        cfg,
		store:      store,
		refresher:  auth.NewFlow(cfg, store),
		httpClient: &http.Client{Timeout: defaultRequestTimeout},
		now:        time.Now,
		sleepFn:    sleepWithContext,
		backoff:    defaultBackoff,
		max429:     defaultMax429Retry,
		baseURL:    strings.TrimRight(cfg.APIBaseURL, "/"),
		userAgent:  defaultUserAgent,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	return client
}

// GetJSON performs a GET request against the given API path and unmarshals the JSON response.
func (c *Client) GetJSON(ctx context.Context, path string, query url.Values, dest any) error {
	resp, err := c.do(ctx, http.MethodGet, path, query, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if dest == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c *Client) do(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	body io.Reader,
	extraHeaders map[string]string,
) (*http.Response, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return nil, err
	}

	fullURL, err := c.buildURL(path, query)
	if err != nil {
		return nil, err
	}

	refreshed := false
	for attempt := 0; attempt <= c.max429; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
		if err != nil {
			return nil, err
		}
		c.setHeaders(req, token, extraHeaders)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusUnauthorized && !refreshed {
			resp.Body.Close()
			debuglog.Warn("api request returned unauthorized; attempting token refresh", "path", path)
			token, err = c.forceRefresh(ctx, token.AccessToken)
			if err != nil {
				return nil, err
			}
			refreshed = true
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < c.max429 {
			resp.Body.Close()
			delay := c.backoff(attempt)
			debuglog.Warn("api request hit rate limit; backing off", "path", path, "attempt", attempt+1, "delay", delay.String())
			if err := c.sleepFn(ctx, delay); err != nil {
				return nil, err
			}
			continue
		}

		if resp.StatusCode >= 400 {
			bodyText := readLimited(resp.Body)
			resp.Body.Close()
			debuglog.Error("api request failed", "path", path, "status", resp.Status, "body", bodyText)
			return nil, fmt.Errorf("whoopy api error %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), bodyText)
		}

		return resp, nil
	}

	return nil, errors.New("whoopy api error: exhausted retries after 429 responses")
}

func (c *Client) setHeaders(req *http.Request, token *tokens.Token, extra map[string]string) {
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if extra != nil {
		for k, v := range extra {
			req.Header.Set(k, v)
		}
	}
}

func (c *Client) ensureValidToken(ctx context.Context) (*tokens.Token, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.token == nil {
		token, err := c.store.Load()
		if err != nil {
			return nil, err
		}
		if token == nil {
			return nil, ErrNotAuthenticated
		}
		c.token = token
	}

	if !c.token.ExpiresAt.IsZero() && c.now().Add(tokenRefreshLeeway).After(c.token.ExpiresAt) {
		debuglog.Info("refreshing token before request due to expiry window", "expires_at", c.token.ExpiresAt.UTC().Format(time.RFC3339))
		token, err := c.refresher.Refresh(ctx)
		if err != nil {
			return nil, err
		}
		c.token = token
	}

	return c.token, nil
}

func (c *Client) forceRefresh(ctx context.Context, usedAccessToken string) (*tokens.Token, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.token != nil && c.token.AccessToken != "" && c.token.AccessToken != usedAccessToken {
		debuglog.Info("skipping duplicate refresh; another request already rotated the token", "expires_at", c.token.ExpiresAt.UTC().Format(time.RFC3339))
		return c.token, nil
	}

	token, err := c.refresher.Refresh(ctx)
	if err != nil {
		return nil, err
	}
	c.token = token
	return token, nil
}

func (c *Client) buildURL(path string, query url.Values) (string, error) {
	trimmed := strings.TrimLeft(path, "/")
	full := c.baseURL + "/" + trimmed
	u, err := url.Parse(full)
	if err != nil {
		return "", err
	}
	if query != nil {
		q := u.Query()
		for key, values := range query {
			for _, value := range values {
				q.Add(key, value)
			}
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func defaultBackoff(attempt int) time.Duration {
	base := 200 * time.Millisecond
	return base * time.Duration(1<<attempt)
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func readLimited(r io.Reader) string {
	const limit = 4 * 1024
	buf := make([]byte, limit)
	n, _ := r.Read(buf)
	return strings.TrimSpace(string(buf[:n]))
}
