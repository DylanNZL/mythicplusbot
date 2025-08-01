package blizzard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPClient defines the interface for making HTTP requests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// TimeProvider defines the interface for getting current time (for testing).
type TimeProvider interface {
	Now() time.Time
}

// RealTimeProvider implements TimeProvider with real time.
type RealTimeProvider struct{}

func (r *RealTimeProvider) Now() time.Time {
	return time.Now()
}

// APIClient defines the interface for Blizzard API operations.
type APIClient interface {
	GetMythicKeystoneProfile(ctx context.Context, realm, character string) (*MythicKeystoneProfile, error)
	SetCredentials(clientID, clientSecret string)
}

type auth struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

type Client struct {
	ID           string
	Secret       string
	Bearer       string
	Expires      time.Time
	httpClient   HTTPClient
	timeProvider TimeProvider
	oauthURL     string
	baseURL      string
}

const expiryBuffer = time.Minute * 5

func NewClient(httpClient HTTPClient, timeProvider TimeProvider) *Client {
	return &Client{
		httpClient:   httpClient,
		timeProvider: timeProvider,
		oauthURL:     "https://oauth.battle.net/token",
		baseURL:      "https://us.api.blizzard.com",
	}
}

func (c *Client) SetCredentials(clientID, clientSecret string) {
	c.ID = clientID
	c.Secret = clientSecret
}

func (c *Client) checkClient(ctx context.Context) error {
	if c.ID == "" || c.Secret == "" {
		return fmt.Errorf("client is not initialised")
	}

	// Check the bearer is set and won't expire in the next 5 minutes
	if c.Bearer == "" || c.timeProvider.Now().Add(expiryBuffer).After(c.Expires) {
		if err := c.getBearerToken(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) getBearerToken(ctx context.Context) error {
	slog.DebugContext(ctx, "getting bearer token")

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.oauthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.ID, c.Secret)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get bearer token: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var authResp auth
	if err := json.Unmarshal(body, &authResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	c.Bearer = authResp.AccessToken
	c.Expires = c.timeProvider.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)

	slog.DebugContext(ctx, "bearer token acquired", "expires", c.Expires)
	return nil
}

func (c *Client) sendRequest(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.Bearer)

	return c.httpClient.Do(req)
}

func (c *Client) GetMythicKeystoneProfile(ctx context.Context, realm string, character string) (*MythicKeystoneProfile, error) {
	if err := c.checkClient(ctx); err != nil {
		return nil, err
	}

	realm = strings.ToLower(realm)
	character = strings.ToLower(character)

	slog.DebugContext(ctx, "getting mythic profile", "character", character, "realm", realm)
	apiURL := fmt.Sprintf("%s/profile/wow/character/%s/%s/mythic-keystone-profile?namespace=profile-us&locale=en_US",
		c.baseURL, realm, character)

	resp, err := c.sendRequest(ctx, apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get mythic keystone profile: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var profile MythicKeystoneProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}
