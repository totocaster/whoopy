package cli

import (
	"github.com/toto/whoopy/internal/api"
	"github.com/toto/whoopy/internal/config"
	"github.com/toto/whoopy/internal/tokens"
)

var apiClientFactory = func() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	store, err := tokens.NewStore("")
	if err != nil {
		return nil, err
	}
	return api.NewClient(cfg, store), nil
}
