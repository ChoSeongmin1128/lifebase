package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	portout "lifebase/internal/auth/port/out"
)

type oauthClient struct {
	config *oauth2.Config
}

func NewOAuthClient(clientID, clientSecret, redirectURL string) *oauthClient {
	return &oauthClient{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes: []string{
				"openid",
				"email",
				"profile",
				"https://www.googleapis.com/auth/calendar",
				"https://www.googleapis.com/auth/tasks",
			},
			Endpoint:    google.Endpoint,
			RedirectURL: redirectURL,
		},
	}
}

func (c *oauthClient) AuthURL(state string) string {
	return c.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
}

func (c *oauthClient) ExchangeCode(ctx context.Context, code string) (*portout.OAuthToken, error) {
	token, err := c.config.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	return &portout.OAuthToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}, nil
}

func (c *oauthClient) FetchUserInfo(ctx context.Context, token portout.OAuthToken) (*portout.OAuthUserInfo, error) {
	ot := &oauth2.Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}

	client := c.config.Client(ctx, ot)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo returned %d", resp.StatusCode)
	}

	var info struct {
		Sub     string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return &portout.OAuthUserInfo{
		GoogleID: info.Sub,
		Email:    info.Email,
		Name:     info.Name,
		Picture:  info.Picture,
	}, nil
}
