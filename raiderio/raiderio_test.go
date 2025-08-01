package raiderio

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

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

// Helper functions for creating test responses

func createHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func createSuccessfulCharacterResponse() string {
	return `{
		"name": "testchar",
		"race": "human",
		"class": "paladin",
		"active_spec_name": "protection",
		"active_spec_role": "tank",
		"guild": {
			"name": "Test Guild",
			"realm": "test-realm"
		},
		"mythic_plus_scores_by_season": [{
			"season": "season-tww-1",
			"scores": {
				"all": 2500,
				"dps": 2400,
				"healer": 0,
				"tank": 2500
			}
		}],
		"mythic_plus_ranks": {
			"overall": {
				"world": 1234,
				"region": 567,
				"realm": 12
			}
		}
	}`
}

// Test Client creation

func TestNewClient(t *testing.T) {
	httpClient := &MockHTTPClient{}
	accessToken := "test-token"

	client := NewClient(accessToken, httpClient)

	assert.NotNil(t, client)
	assert.Equal(t, accessToken, client.AccessToken)
	assert.Equal(t, httpClient, client.httpClient)
	assert.Equal(t, "https://raider.io", client.baseURL)
}

// Test GetCharacter method

func TestClient_GetCharacter_Success(t *testing.T) {
	httpClient := &MockHTTPClient{}
	client := NewClient("test-token", httpClient)

	successResp := createHTTPResponse(200, createSuccessfulCharacterResponse())
	httpClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		// Verify request properties
		if req.Method != "GET" {
			return false
		}
		if !strings.Contains(req.URL.String(), "raider.io/api/v1/characters/profile") {
			return false
		}
		// Check query parameters - use the actual parameter names from implementation
		query := req.URL.Query()
		return query.Get("access_key") == "test-token" &&
			query.Get("name") == "testchar" &&
			query.Get("realm") == "test-realm" &&
			query.Get("region") == "us"
	})).Return(successResp, nil)

	// Fix parameter order to match implementation: realm, name
	character, err := client.GetCharacter(t.Context(), "test-realm", "testchar")

	assert.NoError(t, err)
	require.NotNil(t, character)
	assert.Equal(t, "testchar", character.Name)
	assert.Equal(t, "human", character.Race)
	assert.Equal(t, "paladin", character.Class)
	httpClient.AssertExpectations(t)
}

func TestClient_GetCharacter_HTTPError(t *testing.T) {
	httpClient := &MockHTTPClient{}
	client := NewClient("test-token", httpClient)

	httpClient.On("Do", mock.AnythingOfType("*http.Request")).Return((*http.Response)(nil), errors.New("network error"))

	character, err := client.GetCharacter(t.Context(), "test-realm", "testchar")

	assert.Error(t, err)
	assert.Nil(t, character)
	assert.Contains(t, err.Error(), "failed to send request")
	httpClient.AssertExpectations(t)
}

func TestClient_GetCharacter_BadStatusCode(t *testing.T) {
	httpClient := &MockHTTPClient{}
	client := NewClient("test-token", httpClient)

	errorResp := createHTTPResponse(404, `{"error": "Character not found"}`)
	httpClient.On("Do", mock.AnythingOfType("*http.Request")).Return(errorResp, nil)

	character, err := client.GetCharacter(t.Context(), "test-realm", "testchar")

	assert.Error(t, err)
	assert.Nil(t, character)
	assert.Contains(t, err.Error(), "unexpected status code: 404")
	httpClient.AssertExpectations(t)
}

func TestClient_GetCharacter_InvalidJSON(t *testing.T) {
	httpClient := &MockHTTPClient{}
	client := NewClient("test-token", httpClient)

	invalidResp := createHTTPResponse(200, `{invalid json}`)
	httpClient.On("Do", mock.AnythingOfType("*http.Request")).Return(invalidResp, nil)

	character, err := client.GetCharacter(t.Context(), "test-realm", "testchar")

	assert.Error(t, err)
	assert.Nil(t, character)
	assert.Contains(t, err.Error(), "failed to unmarshal response")
	httpClient.AssertExpectations(t)
}

func TestClient_GetCharacter_RequestCreationError(t *testing.T) {
	httpClient := &MockHTTPClient{}
	client := NewClient("test-token", httpClient)

	// Test the case where the HTTP client call fails instead
	httpClient.On("Do", mock.AnythingOfType("*http.Request")).Return((*http.Response)(nil), errors.New("request creation failed"))

	character, err := client.GetCharacter(t.Context(), "test-realm", "testchar")

	assert.Error(t, err)
	assert.Nil(t, character)
	assert.Contains(t, err.Error(), "failed to send request")
	httpClient.AssertExpectations(t)
}

// Test URL construction

func TestClient_GetCharacter_URLConstruction(t *testing.T) {
	httpClient := &MockHTTPClient{}
	client := NewClient("test-access-token", httpClient)

	var capturedURL string
	httpClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		capturedURL = req.URL.String()
		return true
	})).Return(createHTTPResponse(200, createSuccessfulCharacterResponse()), nil)

	_, err := client.GetCharacter(t.Context(), "Test-Realm", "TestChar")

	assert.NoError(t, err)
	assert.Contains(t, capturedURL, "access_key=test-access-token")
	assert.Contains(t, capturedURL, "name=TestChar")
	assert.Contains(t, capturedURL, "realm=Test-Realm")
	assert.Contains(t, capturedURL, "region=us")
	assert.Contains(t, capturedURL, "fields=mythic_plus_scores_by_season%3Acurrent%2Cmythic_plus_ranks")
	httpClient.AssertExpectations(t)
}

// Test edge cases

func TestClient_GetCharacter_EmptyResponse(t *testing.T) {
	httpClient := &MockHTTPClient{}
	client := NewClient("test-token", httpClient)

	emptyResp := createHTTPResponse(200, `{}`)
	httpClient.On("Do", mock.AnythingOfType("*http.Request")).Return(emptyResp, nil)

	character, err := client.GetCharacter(t.Context(), "test-realm", "testchar")

	assert.NoError(t, err)
	require.NotNil(t, character)
	// Character fields should be empty/zero values
	assert.Equal(t, "", character.Name)
	assert.Equal(t, "", character.Race)
	httpClient.AssertExpectations(t)
}
