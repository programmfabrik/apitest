package util

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

// OAuthClientsConfig is our config for multiple oAuth clients
type OAuthClientsConfig map[string]OAuthClientConfig

// OAuthClientConfig is our config for a single oAuth client
type OAuthClientConfig struct {
	Key      string
	Endpoint OAuthEndpointConfig `mapstructure:"endpoint"`
	Secret   string              `mapstructure:"secret"`
}

// OAuthEndpointConfig is our config for an oAuth endpoint
type OAuthEndpointConfig struct {
	TokenURL string `mapstructure:"token_url"`
}

// GetAuthToken sends request to oAuth token endpoint
// to get a token on behalf of a user
func (c OAuthClientConfig) GetAuthToken(username string, password string) (*oauth2.Token, error) {
	cfg := oauth2.Config{
		ClientID:     c.Key,
		ClientSecret: c.Secret,
		Endpoint: oauth2.Endpoint{
			TokenURL: c.Endpoint.TokenURL,
		},
	}
	httpClient := &http.Client{Timeout: 5 * time.Second}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	return cfg.PasswordCredentialsToken(ctx, username, password)
}
