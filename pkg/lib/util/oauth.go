package util

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"log"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// OAuthClientsConfig is our config for multiple oAuth clients
type OAuthClientsConfig map[string]OAuthClientConfig

// OAuthClientConfig is our config for a single oAuth client
type OAuthClientConfig struct {
	Client      string              `json:"client"`
	Endpoint    oAuthEndpointConfig `mapstructure:"endpoint" json:"endpoint"`
	Secret      string              `mapstructure:"secret" json:"secret"`
	RedirectURL string              `mapstructure:"redirect_url" json:"redirect_url"`
	Scopes      []string            `mapstructure:"scopes" json:"scopes"`
}

// oAuthEndpointConfig is our config for an oAuth endpoint
type oAuthEndpointConfig struct {
	TokenURL string `mapstructure:"token_url" json:"token_url"`
	AuthURL  string `mapstructure:"auth_url" json:"auth_url"`
}

func getOAuthClientConfig(c OAuthClientConfig) (cfg oauth2.Config) {
	return oauth2.Config{
		ClientID:     c.Client,
		ClientSecret: c.Secret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  c.Endpoint.AuthURL,
			TokenURL: c.Endpoint.TokenURL,
		},
		RedirectURL: c.RedirectURL,
		Scopes:      c.Scopes,
	}
}

func getOAuthClientCredentialsConfig(c OAuthClientConfig) (cfg clientcredentials.Config) {
	return clientcredentials.Config{
		ClientID:     c.Client,
		ClientSecret: c.Secret,
		TokenURL:     c.Endpoint.TokenURL,
	}
}

// GetPasswordCredentialsAuthToken sends request to oAuth token endpoint
// to get a token on behalf of a user
func (c OAuthClientConfig) GetPasswordCredentialsAuthToken(username string, password string) (token *oauth2.Token, err error) {
	var (
		cfg        oauth2.Config
		httpClient *http.Client
		ctx        context.Context
	)

	defer func() {
		if err != nil {
			log.Printf("oauth2 password error: %s client: %s secret: %s", err.Error(), cfg.ClientID, cfg.ClientSecret)
		}
	}()

	cfg = getOAuthClientConfig(c)
	httpClient = &http.Client{Timeout: 60 * time.Second}
	ctx = context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	return cfg.PasswordCredentialsToken(ctx, username, password)
}

// GetClientCredentialsAuthToken sends request to oAuth token endpoint
// to get a token on behalf of a user
func (c OAuthClientConfig) GetClientCredentialsAuthToken() (token *oauth2.Token, err error) {
	var (
		cfg        clientcredentials.Config
		httpClient *http.Client
		ctx        context.Context
	)

	cfg = getOAuthClientCredentialsConfig(c)
	httpClient = &http.Client{Timeout: 60 * time.Second}
	ctx = context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	return cfg.Token(ctx)
}

// GetCodeAuthURL sends request to oAuth code endpoint
// to get the auth url back
func (c OAuthClientConfig) getCodeAuthURL(params ...string) (authURLStr string) {
	var (
		cfg   oauth2.Config
		state string
		opts  []oauth2.AuthCodeOption
	)

	cfg = getOAuthClientConfig(c)
	state = ""
	opts = []oauth2.AuthCodeOption{}
	for i := 0; i < len(params); i += 2 {
		if params[i] == "state" {
			state = params[i+1]
			continue
		}
		opts = append(opts, oauth2.SetAuthURLParam(params[i], params[i+1]))
	}
	return cfg.AuthCodeURL(state, opts...)
}

// GetRedirectURL sends request to oAuth auth endpoint
// to get the redirect URL, optionally bypassing login form
// username, password have to be provided in the params list if needed
func (c OAuthClientConfig) getRedirectURL(params ...string) (redirectURL *url.URL, err error) {
	var (
		authURLStr string
		authURL    *url.URL
		qv, cqv    url.Values
		httpClient *http.Client
		ctx        context.Context
		res        *http.Response
	)

	authURLStr = c.getCodeAuthURL(params...)
	authURL, err = url.Parse(authURLStr)
	if err != nil {
		return nil, err
	}

	qv = url.Values{}
	for i := 0; i < len(params); i += 2 {
		qv.Set(params[i], params[i+1])
	}

	cqv = authURL.Query()
	for k, v := range qv {
		for _, vv := range v {
			cqv.Set(k, vv)
		}
	}

	authURL.RawQuery = cqv.Encode()
	httpClient = &http.Client{Timeout: 60 * time.Second}
	ctx = context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	req, err := http.NewRequestWithContext(ctx, "GET", authURL.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err = httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("No proper status after redirect returned: %s (%d)", res.Status, res.StatusCode)
	}

	return res.Request.URL, nil
}

// GetCodeAuthToken sends request to oAuth auth endpoint
// to get a token, optionally bypassing login form
// username, password have to be provided in the params list if needed
func (c OAuthClientConfig) GetCodeAuthToken(params ...string) (token *oauth2.Token, err error) {
	var (
		redirectURL *url.URL
		code        string
		cfg         oauth2.Config
		httpClient  *http.Client
		ctx         context.Context
		opts        []oauth2.AuthCodeOption
	)

	redirectURL, err = c.getRedirectURL(params...)
	if err != nil {
		return nil, err
	}

	code = redirectURL.Query().Get("code")
	cfg = getOAuthClientConfig(c)
	httpClient = &http.Client{Timeout: 60 * time.Second}
	ctx = context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	opts = []oauth2.AuthCodeOption{}
	for i := 0; i < len(params); i += 2 {
		opts = append(opts, oauth2.SetAuthURLParam(params[i], params[i+1]))
	}

	return cfg.Exchange(ctx, code, opts...)
}

// GetCodeAuthToken sends request to oAuth auth endpoint
// to get a token, optionally bypassing login form
// username, password have to be provided in the params list if needed
func (c OAuthClientConfig) GetAuthToken(params ...string) (token *oauth2.Token, err error) {
	var (
		redirectURL *url.URL
		tokensF     string
		qv          url.Values
	)

	redirectURL, err = c.getRedirectURL(params...)
	if err != nil {
		return nil, err
	}

	tokensF = redirectURL.EscapedFragment()
	qv, err = url.ParseQuery(tokensF)
	if err != nil {
		return nil, err
	}
	token = &oauth2.Token{
		AccessToken:  qv.Get("access_token"),
		RefreshToken: qv.Get("refresh_token"),
	}
	return token, nil
}
