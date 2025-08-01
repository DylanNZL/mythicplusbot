// Package raiderio provides access to Raider.IO API for character data
package raiderio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

// HTTPClient defines the interface for making HTTP requests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// APIClient defines the interface for Raider.IO API operations
type APIClient interface {
	GetCharacter(ctx context.Context, name, realm string) (*Character, error)
}

// Client handles Raider.IO API requests with injected dependencies
type Client struct {
	AccessToken string
	httpClient  HTTPClient
	baseURL     string
}

// NewClient creates a new Raider.IO API client
func NewClient(accessToken string, httpClient HTTPClient) *Client {
	return &Client{
		AccessToken: accessToken,
		httpClient:  httpClient,
		baseURL:     "https://raider.io",
	}
}

// GetCharacter returns the raider.io profile of a character.
//
// docs: https://raider.io/api#/character/getApiV1CharactersProfile.
func (c *Client) GetCharacter(ctx context.Context, realm string, name string) (*Character, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "raider.io",
		Path:   "/api/v1/characters/profile",
	}
	query := url.Values{
		"access_key": []string{c.AccessToken},
		"region":     []string{"us"},
		"realm":      []string{realm},
		"name":       []string{name},
		"fields":     []string{"mythic_plus_scores_by_season:current,mythic_plus_ranks"},
	}

	u.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	slog.DebugContext(ctx, "fetching character from raider.io", slog.String("character", name), slog.String("realm", realm))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var char Character
	if err := json.Unmarshal(body, &char); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &char, nil
}
