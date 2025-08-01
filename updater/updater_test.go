package updater

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DylanNZL/mythicplusbot/blizzard"
	"github.com/DylanNZL/mythicplusbot/db"
	"github.com/DylanNZL/mythicplusbot/raiderio"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing

type MockCharacterRepository struct {
	mock.Mock
}

func (m *MockCharacterRepository) ListCharacters(ctx context.Context, limit int) ([]db.Character, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]db.Character), args.Error(1)
}

func (m *MockCharacterRepository) UpdateCharacter(ctx context.Context, character *db.Character) error {
	args := m.Called(ctx, character)
	return args.Error(0)
}

type MockBlizzardClient struct {
	mock.Mock
}

func (m *MockBlizzardClient) GetMythicKeystoneProfile(ctx context.Context, realm, character string) (*blizzard.MythicKeystoneProfile, error) {
	args := m.Called(ctx, realm, character)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*blizzard.MythicKeystoneProfile), args.Error(1)
}

type MockRaiderIOClient struct {
	mock.Mock
}

func (m *MockRaiderIOClient) GetCharacter(ctx context.Context, realm, character string) (*raiderio.Character, error) {
	args := m.Called(ctx, realm, character)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*raiderio.Character), args.Error(1)
}

type MockMessageSender struct {
	mock.Mock
}

func (m *MockMessageSender) SendMessage(ctx context.Context, channelID string, message string) error {
	args := m.Called(ctx, channelID, message)
	return args.Error(0)
}

func (m *MockMessageSender) SendComplexMessage(ctx context.Context, channelID string, message discordgo.MessageSend) error {
	args := m.Called(ctx, channelID, message)
	return args.Error(0)
}

type MockSleeper struct {
	mock.Mock
}

func (m *MockSleeper) Sleep(duration time.Duration) {
	m.Called(duration)
}

// Test helper functions

func createTestCharacter(name, realm string, score float64) db.Character {
	return db.Character{
		ID:           1,
		Name:         name,
		Realm:        realm,
		OverallScore: score,
	}
}

func createTestProfile(score float64) *blizzard.MythicKeystoneProfile {
	return &blizzard.MythicKeystoneProfile{
		CurrentMythicRating: struct {
			Color  blizzard.Color `json:"color"`
			Rating float64        `json:"rating"`
		}{
			Rating: score,
		},
	}
}

func createTestRaiderIOCharacter(tankScore, healScore, dpsScore float64) *raiderio.Character {
	return &raiderio.Character{
		MythicPlusScoresBySeason: []raiderio.Season{
			{
				Scores: raiderio.Scores{
					Tank:   tankScore,
					Healer: healScore,
					Dps:    dpsScore,
				},
			},
		},
	}
}

func setupService() (*Service, *MockCharacterRepository, *MockBlizzardClient, *MockRaiderIOClient, *MockMessageSender, *MockSleeper) {
	characterRepo := &MockCharacterRepository{}
	blizzardClient := &MockBlizzardClient{}
	raiderIOClient := &MockRaiderIOClient{}
	messageSender := &MockMessageSender{}
	sleeper := &MockSleeper{}

	service := NewService(characterRepo, blizzardClient, raiderIOClient, messageSender, sleeper)
	return service, characterRepo, blizzardClient, raiderIOClient, messageSender, sleeper
}

// Test Service creation

func TestNewService(t *testing.T) {
	characterRepo := &MockCharacterRepository{}
	blizzardClient := &MockBlizzardClient{}
	raiderIOClient := &MockRaiderIOClient{}
	messageSender := &MockMessageSender{}
	sleeper := &MockSleeper{}

	service := NewService(characterRepo, blizzardClient, raiderIOClient, messageSender, sleeper)

	assert.NotNil(t, service)
	assert.Equal(t, characterRepo, service.characterRepo)
	assert.Equal(t, blizzardClient, service.blizzardClient)
	assert.Equal(t, raiderIOClient, service.raiderioClient)
	assert.Equal(t, messageSender, service.messageSender)
	assert.Equal(t, sleeper, service.sleeper)
}

// Test Update method

func TestService_Update_Success_WithScoreChange(t *testing.T) {
	service, characterRepo, blizzardClient, raiderIOClient, messageSender, sleeper := setupService()
	ctx := context.Background()
	channelID := "test-channel"

	// Setup test data
	character := createTestCharacter("testchar", "testrealm", 2500.0)
	characters := []db.Character{character}
	newProfile := createTestProfile(2600.0)                             // Score improved by 100
	raiderIOChar := createTestRaiderIOCharacter(2400.0, 2300.0, 2200.0) // Tank, Heal, DPS scores

	// Mock expectations
	characterRepo.On("ListCharacters", ctx, 0).Return(characters, nil)
	blizzardClient.On("GetMythicKeystoneProfile", ctx, "testrealm", "testchar").Return(newProfile, nil)
	raiderIOClient.On("GetCharacter", ctx, "testrealm", "testchar").Return(raiderIOChar, nil)
	messageSender.On("SendComplexMessage", ctx, channelID, mock.AnythingOfType("discordgo.MessageSend")).Return(nil)
	characterRepo.On("UpdateCharacter", ctx, mock.MatchedBy(func(char *db.Character) bool {
		return char.Name == "testchar" && char.OverallScore == 2600.0
	})).Return(nil)
	sleeper.On("Sleep", cooldownTime).Return()

	err := service.Update(ctx, channelID)

	assert.NoError(t, err)
	characterRepo.AssertExpectations(t)
	blizzardClient.AssertExpectations(t)
	raiderIOClient.AssertExpectations(t)
	messageSender.AssertExpectations(t)
	sleeper.AssertExpectations(t)
}

func TestService_Update_NoScoreChange(t *testing.T) {
	service, characterRepo, blizzardClient, _, messageSender, sleeper := setupService()
	ctx := context.Background()
	channelID := "test-channel"

	// Setup test data - same score
	character := createTestCharacter("testchar", "testrealm", 2500.0)
	characters := []db.Character{character}
	sameProfile := createTestProfile(2500.0) // Same score

	// Mock expectations
	characterRepo.On("ListCharacters", ctx, 0).Return(characters, nil)
	blizzardClient.On("GetMythicKeystoneProfile", ctx, "testrealm", "testchar").Return(sameProfile, nil)
	sleeper.On("Sleep", cooldownTime).Return()

	// Should NOT call messageSender or UpdateCharacter when score is the same
	err := service.Update(ctx, channelID)

	assert.NoError(t, err)
	characterRepo.AssertExpectations(t)
	blizzardClient.AssertExpectations(t)
	messageSender.AssertNotCalled(t, "SendMessage")
	characterRepo.AssertNotCalled(t, "UpdateCharacter")
	sleeper.AssertExpectations(t)
}

func TestService_Update_ListCharactersError(t *testing.T) {
	service, characterRepo, _, _, _, _ := setupService()
	ctx := context.Background()
	channelID := "test-channel"

	characterRepo.On("ListCharacters", ctx, 0).Return([]db.Character{}, errors.New("database error"))

	err := service.Update(ctx, channelID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list characters")
	characterRepo.AssertExpectations(t)
}

func TestService_Update_BlizzardAPIError(t *testing.T) {
	service, characterRepo, blizzardClient, _, messageSender, sleeper := setupService()
	ctx := context.Background()
	channelID := "test-channel"

	character := createTestCharacter("testchar", "testrealm", 2500.0)
	characters := []db.Character{character}

	characterRepo.On("ListCharacters", ctx, 0).Return(characters, nil)
	blizzardClient.On("GetMythicKeystoneProfile", ctx, "testrealm", "testchar").Return((*blizzard.MythicKeystoneProfile)(nil), errors.New("API error"))
	// Sleep is NOT called when updateCharacter fails

	err := service.Update(ctx, channelID)

	// Should not fail completely, just log error and continue
	assert.NoError(t, err)
	characterRepo.AssertExpectations(t)
	blizzardClient.AssertExpectations(t)
	messageSender.AssertNotCalled(t, "SendComplexMessage")
	sleeper.AssertNotCalled(t, "Sleep") // Sleep is not called when updateCharacter fails
}

func TestService_Update_MessageSendError(t *testing.T) {
	service, characterRepo, blizzardClient, raiderIOClient, messageSender, sleeper := setupService()
	ctx := context.Background()
	channelID := "test-channel"

	character := createTestCharacter("testchar", "testrealm", 2500.0)
	characters := []db.Character{character}
	newProfile := createTestProfile(2600.0)
	raiderIOChar := createTestRaiderIOCharacter(2400.0, 2300.0, 2200.0)

	characterRepo.On("ListCharacters", ctx, 0).Return(characters, nil)
	blizzardClient.On("GetMythicKeystoneProfile", ctx, "testrealm", "testchar").Return(newProfile, nil)
	raiderIOClient.On("GetCharacter", ctx, "testrealm", "testchar").Return(raiderIOChar, nil)
	// UpdateCharacter happens BEFORE SendComplexMessage in the implementation
	characterRepo.On("UpdateCharacter", ctx, mock.MatchedBy(func(char *db.Character) bool {
		return char.Name == "testchar" && char.OverallScore == 2600.0
	})).Return(nil)
	messageSender.On("SendComplexMessage", ctx, channelID, mock.AnythingOfType("discordgo.MessageSend")).Return(errors.New("discord error"))
	// Sleep is NOT called when updateCharacter fails (due to message send error)

	err := service.Update(ctx, channelID)

	// Should not fail completely, just log error and continue
	assert.NoError(t, err)
	characterRepo.AssertExpectations(t)
	blizzardClient.AssertExpectations(t)
	raiderIOClient.AssertExpectations(t)
	messageSender.AssertExpectations(t)
	sleeper.AssertNotCalled(t, "Sleep") // Sleep is not called when updateCharacter fails
}

func TestService_Update_MultipleCharacters(t *testing.T) {
	service, characterRepo, blizzardClient, raiderIOClient, messageSender, sleeper := setupService()
	ctx := context.Background()
	channelID := "test-channel"

	// Setup multiple characters
	char1 := createTestCharacter("char1", "realm1", 2500.0)
	char2 := createTestCharacter("char2", "realm2", 2300.0)
	characters := []db.Character{char1, char2}

	profile1 := createTestProfile(2600.0) // Score improved
	profile2 := createTestProfile(2300.0) // No change
	raiderIOChar1 := createTestRaiderIOCharacter(2400.0, 2300.0, 2200.0)

	characterRepo.On("ListCharacters", ctx, 0).Return(characters, nil)
	blizzardClient.On("GetMythicKeystoneProfile", ctx, "realm1", "char1").Return(profile1, nil)
	blizzardClient.On("GetMythicKeystoneProfile", ctx, "realm2", "char2").Return(profile2, nil)

	// Only char1 should trigger RaiderIO call, message and update (char2 has no score change)
	raiderIOClient.On("GetCharacter", ctx, "realm1", "char1").Return(raiderIOChar1, nil)
	messageSender.On("SendComplexMessage", ctx, channelID, mock.AnythingOfType("discordgo.MessageSend")).Return(nil).Once()
	characterRepo.On("UpdateCharacter", ctx, mock.MatchedBy(func(char *db.Character) bool {
		return char.Name == "char1" && char.OverallScore == 2600.0
	})).Return(nil).Once()

	// Should sleep after each character
	sleeper.On("Sleep", cooldownTime).Return().Twice()

	err := service.Update(ctx, channelID)

	assert.NoError(t, err)
	characterRepo.AssertExpectations(t)
	blizzardClient.AssertExpectations(t)
	raiderIOClient.AssertExpectations(t)
	messageSender.AssertExpectations(t)
	sleeper.AssertExpectations(t)
}

// Test real implementations

func TestRealSleeper_Sleep(t *testing.T) {
	sleeper := &RealSleeper{}

	start := time.Now()
	sleeper.Sleep(10 * time.Millisecond)
	elapsed := time.Since(start)

	assert.True(t, elapsed >= 10*time.Millisecond)
}
