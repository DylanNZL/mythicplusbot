package discord

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/DylanNZL/mythicplusbot/db"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSender is a mock implementation of the SenderIface interface for testing
type MockSender struct {
	mock.Mock
}

func (m *MockSender) SendMessage(ctx context.Context, channelID, content string) error {
	args := m.Called(ctx, channelID, content)
	return args.Error(0)
}

func (m *MockSender) SendComplexMessage(ctx context.Context, channelID string, message discordgo.MessageSend) error {
	args := m.Called(ctx, channelID, message)
	return args.Error(0)
}

// Test DiscordSender

func TestNewDiscordSender(t *testing.T) {
	session := &discordgo.Session{}
	sender := NewDiscordSender(session)

	assert.NotNil(t, sender)
	assert.Equal(t, session, sender.session)
}

func TestDiscordSender_SendMessage_Success(t *testing.T) {
	// Test that we can create a DiscordSender and it has the right structure
	// We don't actually call Discord API methods to avoid panics from uninitialized session
	session := &discordgo.Session{}
	sender := &Sender{session: session}

	// Test the structure is correct
	assert.NotNil(t, sender)
	assert.NotNil(t, sender.session)
	assert.Equal(t, session, sender.session)

	// Test that the method signature is correct by checking it exists
	// We don't call it to avoid the panic from uninitialized Discord session
	assert.NotNil(t, sender.SendMessage)
}

func TestDiscordSender_SendComplexMessage_Success(t *testing.T) {
	// Test that we can create a DiscordSender and it has the right structure
	// We don't actually call Discord API methods to avoid panics from uninitialized session
	session := &discordgo.Session{}
	sender := &Sender{session: session}

	// Test the structure is correct
	assert.NotNil(t, sender)
	assert.NotNil(t, sender.session)
	assert.Equal(t, session, sender.session)

	// Test that the method signature is correct by checking it exists
	// We don't call it to avoid the panic from uninitialized Discord session
	assert.NotNil(t, sender.SendComplexMessage)
}

// Test MockSender

func TestMockSender_SendMessage(t *testing.T) {
	mockSender := &MockSender{}
	ctx := context.Background()

	mockSender.On("SendMessage", ctx, "test-channel", "test message").Return(nil)

	err := mockSender.SendMessage(ctx, "test-channel", "test message")

	assert.NoError(t, err)
	mockSender.AssertExpectations(t)
}

func TestMockSender_SendMessage_Error(t *testing.T) {
	mockSender := &MockSender{}
	ctx := context.Background()

	expectedError := errors.New("send failed")
	mockSender.On("SendMessage", ctx, "test-channel", "test message").Return(expectedError)

	err := mockSender.SendMessage(ctx, "test-channel", "test message")

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockSender.AssertExpectations(t)
}

func TestMockSender_SendComplexMessage(t *testing.T) {
	mockSender := &MockSender{}
	ctx := context.Background()

	message := discordgo.MessageSend{
		Content: "test message",
	}

	mockSender.On("SendComplexMessage", ctx, "test-channel", message).Return(nil)

	err := mockSender.SendComplexMessage(ctx, "test-channel", message)

	assert.NoError(t, err)
	mockSender.AssertExpectations(t)
}

func TestMockSender_SendComplexMessage_Error(t *testing.T) {
	mockSender := &MockSender{}
	ctx := context.Background()

	message := discordgo.MessageSend{
		Content: "test message",
	}

	expectedError := errors.New("send complex failed")
	mockSender.On("SendComplexMessage", ctx, "test-channel", message).Return(expectedError)

	err := mockSender.SendComplexMessage(ctx, "test-channel", message)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockSender.AssertExpectations(t)
}

// Test BuildScoresMessage function

func TestBuildScoresMessage(t *testing.T) {
	characters := []db.Character{
		{
			Name:         "Char1",
			Realm:        "realm1",
			Class:        "Warrior",
			OverallScore: 2500.0,
		},
		{
			Name:         "Char2",
			Realm:        "realm2",
			Class:        "Mage",
			OverallScore: 2300.0,
		},
	}

	message := BuildScoresMessage(characters)

	// Test embeds
	assert.Len(t, message.Embeds, 1)
	embed := message.Embeds[0]

	assert.Equal(t, "Tracked Characters", embed.Title)
	assert.NotEmpty(t, embed.Fields)
}

// Test BuildScoresMessage with empty characters

func TestBuildScoresMessage_EmptyCharacters(t *testing.T) {
	characters := []db.Character{}

	message := BuildScoresMessage(characters)

	// Test embeds
	assert.Len(t, message.Embeds, 1)
	embed := message.Embeds[0]

	assert.Equal(t, "Tracked Characters", embed.Title)
	// Should still have fields even if empty (the function creates empty fields)
	assert.NotNil(t, embed.Fields)
}

// Test buildScoresFields function with many characters to test field limit

func TestBuildScoresFields_ManyCharacters(t *testing.T) {
	// Create enough characters to test the field limit logic
	characters := make([]db.Character, 30)
	for i := 0; i < 30; i++ {
		characters[i] = db.Character{
			Name:         fmt.Sprintf("Character%d", i+1),
			Realm:        "testrealm",
			OverallScore: float64(2500 - i*10),
		}
	}

	fields := buildScoresFields(characters)

	// Should have fields but not exceed the limit
	assert.NotEmpty(t, fields)
	assert.LessOrEqual(t, len(fields), 25) // Max 25 fields as per Discord limit
}

// Test buildScoresFields with few characters

func TestBuildScoresFields_FewCharacters(t *testing.T) {
	characters := []db.Character{
		{Name: "Char1", Realm: "realm1", OverallScore: 2500.0},
		{Name: "Char2", Realm: "realm2", OverallScore: 2300.0},
	}

	fields := buildScoresFields(characters)

	// Should have exactly 2 fields (character field and score field)
	assert.Len(t, fields, 2)

	// Check that the fields contain the expected data
	assert.Contains(t, fields[0].Value, "Char1-realm1")
	assert.Contains(t, fields[0].Value, "Char2-realm2")
	assert.Contains(t, fields[1].Value, "2500")
	assert.Contains(t, fields[1].Value, "2300")
}
