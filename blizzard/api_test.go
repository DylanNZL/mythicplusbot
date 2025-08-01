package blizzard

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

type MockTimeProvider struct {
	mock.Mock
}

func (m *MockTimeProvider) Now() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

// Helper functions for creating test responses

func createHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func createSuccessfulOAuthResponse() string {
	return `{
		"access_token": "test-bearer-token",
		"token_type": "Bearer",
		"expires_in": 3600,
		"scope": "wow.profile"
	}`
}

func createMythicKeystoneProfileResponse() string {
	return `{
		"current_mythic_rating": {
			"color": {"r": 1.0, "g": 0.5, "b": 0.0, "a": 1.0},
			"rating": 2500.5
		},
		"character": {
			"name": "testchar",
			"id": 123,
			"realm": {
				"id": 456,
				"slug": "test-realm"
			}
		}
	}`
}

// Test Client creation and credential setting

func TestNewClient(t *testing.T) {
	httpClient := &MockHTTPClient{}
	timeProvider := &MockTimeProvider{}

	client := NewClient(httpClient, timeProvider)

	assert.NotNil(t, client)
	assert.Equal(t, httpClient, client.httpClient)
	assert.Equal(t, timeProvider, client.timeProvider)
	assert.Equal(t, "https://oauth.battle.net/token", client.oauthURL)
	assert.Equal(t, "https://us.api.blizzard.com", client.baseURL)
}

func TestClient_SetCredentials(t *testing.T) {
	client := NewClient(&MockHTTPClient{}, &MockTimeProvider{})

	client.SetCredentials("test-id", "test-secret")

	assert.Equal(t, "test-id", client.ID)
	assert.Equal(t, "test-secret", client.Secret)
}

// Test authentication and token management

func TestClient_CheckClient_NotInitialized(t *testing.T) {
	client := NewClient(&MockHTTPClient{}, &MockTimeProvider{})
	ctx := context.Background()

	err := client.checkClient(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client is not initialised")
}

func TestClient_GetBearerToken_Success(t *testing.T) {
	httpClient := &MockHTTPClient{}
	timeProvider := &MockTimeProvider{}
	client := NewClient(httpClient, timeProvider)

	client.SetCredentials("test-id", "test-secret")

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	timeProvider.On("Now").Return(now)

	oauthResp := createHTTPResponse(200, createSuccessfulOAuthResponse())
	httpClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		if req.URL.String() != client.oauthURL {
			return false
		}
		if req.Method != "POST" {
			return false
		}

		username, password, ok := req.BasicAuth()
		return ok && username == "test-id" && password == "test-secret"
	})).Return(oauthResp, nil)

	ctx := context.Background()
	err := client.getBearerToken(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "test-bearer-token", client.Bearer)
	assert.Equal(t, now.Add(3600*time.Second), client.Expires)
	httpClient.AssertExpectations(t)
	timeProvider.AssertExpectations(t)
}

func TestClient_GetMythicKeystoneProfile_Success(t *testing.T) {
	httpClient := &MockHTTPClient{}
	timeProvider := &MockTimeProvider{}
	client := NewClient(httpClient, timeProvider)

	client.SetCredentials("test-id", "test-secret")
	client.Bearer = "test-token"
	client.Expires = time.Now().Add(time.Hour)

	now := time.Now()
	timeProvider.On("Now").Return(now)

	profileResp := createHTTPResponse(200, createMythicKeystoneProfileResponse())
	httpClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		expectedURL := "https://us.api.blizzard.com/profile/wow/character/test-realm/testchar/mythic-keystone-profile?namespace=profile-us&locale=en_US"
		return req.URL.String() == expectedURL &&
			req.Header.Get("Authorization") == "Bearer test-token"
	})).Return(profileResp, nil)

	ctx := context.Background()
	profile, err := client.GetMythicKeystoneProfile(ctx, "Test-Realm", "TestChar")

	assert.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, 2500.5, profile.CurrentMythicRating.Rating)
	assert.Equal(t, "testchar", profile.Character.Name)
	httpClient.AssertExpectations(t)
	timeProvider.AssertExpectations(t)
}

func TestClient_GetMythicKeystoneProfile_ClientNotInitialized(t *testing.T) {
	client := NewClient(&MockHTTPClient{}, &MockTimeProvider{})

	ctx := context.Background()
	profile, err := client.GetMythicKeystoneProfile(ctx, "test-realm", "testchar")

	assert.Error(t, err)
	assert.Nil(t, profile)
	assert.Contains(t, err.Error(), "client is not initialised")
}

func TestRealTimeProvider_Now(t *testing.T) {
	provider := &RealTimeProvider{}

	before := time.Now()
	result := provider.Now()
	after := time.Now()

	assert.True(t, result.After(before) || result.Equal(before))
	assert.True(t, result.Before(after) || result.Equal(after))
}
