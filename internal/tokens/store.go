package tokens

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/toto/whoopy/internal/debuglog"
	"github.com/toto/whoopy/internal/paths"
)

// Token represents persisted OAuth token data.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Scope        []string  `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Store manages secure persistence of tokens on disk.
type Store struct {
	path       string
	legacyPath string
	mu         sync.Mutex
}

// Path returns the absolute file path backing this store.
func (s *Store) Path() string {
	return s.path
}

// NewStore creates a token store using the default token path unless an override is provided.
func NewStore(customPath string) (*Store, error) {
	var (
		path       string
		legacyPath string
	)
	if customPath != "" {
		path = customPath
	} else {
		var err error
		path, err = paths.TokensFile()
		if err != nil {
			return nil, err
		}
		legacyPath, err = paths.LegacyTokensFile()
		if err != nil {
			return nil, err
		}
		if legacyPath == path {
			legacyPath = ""
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("prepare token directory: %w", err)
	}

	store := &Store{path: path, legacyPath: legacyPath}
	if err := store.migrateLegacy(); err != nil {
		return nil, err
	}
	return store, nil
}

// Load returns the stored token or nil if none exists.
func (s *Store) Load() (*Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.migrateLegacyLocked(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read token file: %w", err)
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("parse token file: %w", err)
	}
	return &token, nil
}

// Save persists the provided token atomically.
func (s *Store) Save(token *Token) error {
	if token == nil {
		return errors.New("token is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.migrateLegacyLocked(); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return fmt.Errorf("write temp token file: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename token file: %w", err)
	}
	if err := s.removeLegacyLocked(); err != nil {
		return err
	}
	return nil
}

// Clear removes the stored token from disk.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove token file: %w", err)
	}
	if err := s.removeLegacyLocked(); err != nil {
		return err
	}
	return nil
}

func (s *Store) migrateLegacy() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.migrateLegacyLocked()
}

func (s *Store) migrateLegacyLocked() error {
	if s.legacyPath == "" {
		return nil
	}

	if _, err := os.Stat(s.path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat token file: %w", err)
	}

	data, err := os.ReadFile(s.legacyPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read legacy token file: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write migrated token file: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		// Another process may have completed the same migration between our write and rename.
		if errors.Is(err, os.ErrNotExist) {
			if _, statErr := os.Stat(s.path); statErr == nil {
				if removeErr := os.Remove(s.legacyPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
					return fmt.Errorf("remove legacy token file after concurrent migration: %w", removeErr)
				}
				debuglog.Info("legacy token migration already completed by another process", "from", s.legacyPath, "to", s.path)
				return nil
			}
		}
		return fmt.Errorf("rename migrated token file: %w", err)
	}
	if err := os.Remove(s.legacyPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove legacy token file: %w", err)
	}

	debuglog.Info("migrated legacy token file", "from", s.legacyPath, "to", s.path)
	return nil
}

func (s *Store) removeLegacyLocked() error {
	if s.legacyPath == "" {
		return nil
	}
	if err := os.Remove(s.legacyPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove legacy token file: %w", err)
	}
	return nil
}
