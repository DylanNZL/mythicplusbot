package bot

import (
	"context"
	"errors"
	"testing"

	"github.com/DylanNZL/mythicplusbot/db"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing

type MockMessageSender struct {
	mock.Mock
}

func (m *MockMessageSender) SendMessage(ctx context.Context, channelID, content string) error {
	args := m.Called(ctx, channelID, content)
	return args.Error(0)
}

func (m *MockMessageSender) SendComplexMessage(ctx context.Context, channelID string, content discordgo.MessageSend) error {
	args := m.Called(ctx, channelID, content)
	return args.Error(0)
}

type MockUpdater struct {
	mock.Mock
}

func (m *MockUpdater) Update(ctx context.Context, channelID string) error {
	args := m.Called(ctx, channelID)
	return args.Error(0)
}

type MockCharacterService struct {
	mock.Mock
}

func (m *MockCharacterService) AddCharacter(ctx context.Context, name, realm string) error {
	args := m.Called(ctx, name, realm)
	return args.Error(0)
}

func (m *MockCharacterService) RemoveCharacter(ctx context.Context, name, realm string) error {
	args := m.Called(ctx, name, realm)
	return args.Error(0)
}

func (m *MockCharacterService) ListCharacters(ctx context.Context, n int) ([]db.Character, error) {
	args := m.Called(ctx, n)
	return args.Get(0).([]db.Character), args.Error(1)
}

// Test setup helper
func setupBot() (*Bot, *MockMessageSender, *MockUpdater, *MockCharacterService) {
	messageSender := &MockMessageSender{}
	updater := &MockUpdater{}
	characterService := &MockCharacterService{}

	bot := NewBot(messageSender, updater, characterService)
	return bot, messageSender, updater, characterService
}

func TestBot_HandleMessage_InvalidCommand(t *testing.T) {
	bot, messageSender, _, _ := setupBot()

	// Test non-bot message
	err := bot.HandleMessage(t.Context(), "regular message", "channel1")
	assert.NoError(t, err)
	messageSender.AssertNotCalled(t, "SendMessage")

	// Test message without subcommand
	messageSender.On("SendMessage", t.Context(), "channel1", "Usage: !mythicplusbot <command> [args]").Return(nil)
	err = bot.HandleMessage(t.Context(), "!mythicplusbot", "channel1")
	assert.NoError(t, err)
	messageSender.AssertCalled(t, "SendMessage", t.Context(), "channel1", "Usage: !mythicplusbot <command> [args]")
}

func TestBot_HandleMessage_Help(t *testing.T) {
	bot, messageSender, _, _ := setupBot()

	messageSender.On("SendMessage", t.Context(), "channel1", helpMessage).Return(nil)

	err := bot.HandleMessage(t.Context(), "!mythicplusbot help", "channel1")
	assert.NoError(t, err)
	messageSender.AssertCalled(t, "SendMessage", t.Context(), "channel1", helpMessage)
}

func TestBot_HandleMessage_UnknownCommand(t *testing.T) {
	bot, messageSender, _, _ := setupBot()

	expectedMessage := "Unknown command. Use " + Command + " help for a list of commands."
	messageSender.On("SendMessage", t.Context(), "channel1", expectedMessage).Return(nil)

	err := bot.HandleMessage(t.Context(), "!mythicplusbot unknown", "channel1")
	assert.NoError(t, err)
	messageSender.AssertCalled(t, "SendMessage", t.Context(), "channel1", expectedMessage)
}

func TestBot_HandleAddCharacter_Success(t *testing.T) {
	bot, messageSender, _, characterService := setupBot()

	characterService.On("AddCharacter", t.Context(), "Testchar", "testrealm").Return(nil)
	messageSender.On("SendMessage", t.Context(), "channel1", "Now tracking Testchar-testrealm").Return(nil)

	err := bot.HandleMessage(t.Context(), "!mythicplusbot add testchar testrealm", "channel1")
	assert.NoError(t, err)

	characterService.AssertCalled(t, "AddCharacter", t.Context(), "Testchar", "testrealm")
	messageSender.AssertCalled(t, "SendMessage", t.Context(), "channel1", "Now tracking Testchar-testrealm")
}

func TestBot_HandleAddCharacter_ServiceError(t *testing.T) {
	bot, messageSender, _, characterService := setupBot()

	characterService.On("AddCharacter", t.Context(), "Testchar", "testrealm").Return(errors.New("service error"))
	messageSender.On("SendMessage", t.Context(), "channel1", "Failed to add character.").Return(nil)

	err := bot.HandleMessage(t.Context(), "!mythicplusbot add testchar testrealm", "channel1")
	assert.NoError(t, err)

	characterService.AssertCalled(t, "AddCharacter", t.Context(), "Testchar", "testrealm")
	messageSender.AssertCalled(t, "SendMessage", t.Context(), "channel1", "Failed to add character.")
}

func TestBot_HandleAddCharacter_InvalidArgs(t *testing.T) {
	bot, messageSender, _, _ := setupBot()

	messageSender.On("SendMessage", t.Context(), "channel1", "Usage: !mythicplusbot add <character> <realm>").Return(nil)

	err := bot.HandleMessage(t.Context(), "!mythicplusbot add", "channel1")
	assert.NoError(t, err)
	messageSender.AssertCalled(t, "SendMessage", t.Context(), "channel1", "Usage: !mythicplusbot add <character> <realm>")
}

func TestBot_HandleRemoveCharacter_Success(t *testing.T) {
	bot, messageSender, _, characterService := setupBot()

	characterService.On("RemoveCharacter", t.Context(), "Testchar", "testrealm").Return(nil)
	messageSender.On("SendMessage", t.Context(), "channel1", "No longer tracking Testchar-testrealm.").Return(nil)

	err := bot.HandleMessage(t.Context(), "!mythicplusbot remove testchar testrealm", "channel1")
	assert.NoError(t, err)

	characterService.AssertCalled(t, "RemoveCharacter", t.Context(), "Testchar", "testrealm")
	messageSender.AssertCalled(t, "SendMessage", t.Context(), "channel1", "No longer tracking Testchar-testrealm.")
}

func TestBot_HandleRemoveCharacter_ServiceError(t *testing.T) {
	bot, messageSender, _, characterService := setupBot()

	characterService.On("RemoveCharacter", t.Context(), "Testchar", "testrealm").Return(errors.New("service error"))
	messageSender.On("SendMessage", t.Context(), "channel1", "Failed to remove character.").Return(nil)

	err := bot.HandleMessage(t.Context(), "!mythicplusbot remove testchar testrealm", "channel1")
	assert.NoError(t, err)

	characterService.AssertCalled(t, "RemoveCharacter", t.Context(), "Testchar", "testrealm")
	messageSender.AssertCalled(t, "SendMessage", t.Context(), "channel1", "Failed to remove character.")
}

func TestBot_HandleRemoveCharacter_InvalidArgs(t *testing.T) {
	bot, messageSender, _, _ := setupBot()

	messageSender.On("SendMessage", t.Context(), "channel1", "Usage: !mythicplusbot remove <character> <realm>").Return(nil)

	err := bot.HandleMessage(t.Context(), "!mythicplusbot remove", "channel1")
	assert.NoError(t, err)
	messageSender.AssertCalled(t, "SendMessage", t.Context(), "channel1", "Usage: !mythicplusbot remove <character> <realm>")
}

func TestBot_HandleScores_Success(t *testing.T) {
	bot, messageSender, _, characterService := setupBot()

	characters := []db.Character{
		{Name: "char1", Realm: "realm1", OverallScore: 2500.5},
		{Name: "char2", Realm: "realm1", OverallScore: 2300.0},
	}

	characterService.On("ListCharacters", t.Context(), 10).Return(characters, nil)
	messageSender.On("SendComplexMessage", t.Context(), "channel1", mock.Anything).Return(nil)

	err := bot.HandleMessage(t.Context(), "!mythicplusbot scores", "channel1")
	assert.NoError(t, err)

	characterService.AssertCalled(t, "ListCharacters", t.Context(), 10)
	messageSender.AssertCalled(t, "SendComplexMessage", t.Context(), "channel1", mock.Anything)
}

func TestBot_HandleList_Success(t *testing.T) {
	bot, messageSender, _, characterService := setupBot()

	characters := []db.Character{
		{Name: "char1", Realm: "realm1", OverallScore: 2500.5},
		{Name: "char2", Realm: "realm1", OverallScore: 2300.0},
	}

	characterService.On("ListCharacters", t.Context(), 10).Return(characters, nil)
	messageSender.On("SendMessage", t.Context(), "channel1", "todo :(").Return(nil)

	err := bot.HandleMessage(t.Context(), "!mythicplusbot list", "channel1")
	assert.NoError(t, err)

	characterService.AssertCalled(t, "ListCharacters", t.Context(), 10)
	messageSender.AssertCalled(t, "SendMessage", t.Context(), "channel1", "todo :(")
}
