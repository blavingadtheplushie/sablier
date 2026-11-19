// sign in with hack club: auth.hackclub.com oauth
package hcauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://auth.hackclub.com"
	Scopes         = "openid profile name slack_id"
)

type Client struct {
	ID, Secret  string
	RedirectURL string
	BaseURL     string
	HTTP        *http.Client
}

func New(id, secret, redirectURL string) *Client {
	return &Client{
		ID: id, Secret: secret, RedirectURL: redirectURL,
		BaseURL: DefaultBaseURL,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

type Identity struct {
	ID        string `json:"id"` // ident!xxxxx
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	SlackID   string `json:"slack_id"` // empty when slack isn't linked
}

func (i Identity) Name() string {
	if n := strings.TrimSpace(i.FirstName + " " + i.LastName); n != "" {
		return n
	}
	return i.ID
}

func (c *Client) AuthorizeURL(state string) string {
	q := url.Values{
		"client_id":     {c.ID},
		"redirect_uri":  {c.RedirectURL},
		"response_type": {"code"},
		"scope":         {Scopes},
		"state":         {state},
	}
	return c.BaseURL + "/oauth/authorize?" + q.Encode()
}

// trades the callback code for an access token
func (c *Client) Exchange(ctx context.Context, code string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"client_id":     c.ID,
		"client_secret": c.Secret,
		"redirect_uri":  c.RedirectURL,
		"code":          code,
		"grant_type":    "authorization_code",
	})
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/oauth/token", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.do(req, &out); err != nil {
		return "", fmt.Errorf("token exchange: %w", err)
	}
	if out.AccessToken == "" {
		return "", errors.New("token exchange: no access token")
	}
	return out.AccessToken, nil
}

func (c *Client) Me(ctx context.Context, token string) (Identity, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/v1/me", nil)
	if err != nil {
		return Identity{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var out struct {
		Identity Identity `json:"identity"`
	}
	if err := c.do(req, &out); err != nil {
		return Identity{}, fmt.Errorf("me: %w", err)
	}
	if out.Identity.ID == "" {
		return Identity{}, errors.New("me: no identity id")
	}
	return out.Identity, nil
}

func (c *Client) do(req *http.Request, out any) error {
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
