// Package bot handles processing user commands.
//
// For now these commands are accepted by the bot:
// - !mythicplusbot add <character> <realm>
// - !mythicplusbot remove <character> <realm>
// - !mythicplusbot scores [-n 10]
// - !mythicplusbot list [-n 10]
// - !mythicplusbot update
// - !mythicplusbot help
package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/DylanNZL/mythicplusbot/db"
	"github.com/DylanNZL/mythicplusbot/discord"
)

type (
	Updater interface {
		Update(ctx context.Context, channelID string) error
	}

	CharacterService interface {
		AddCharacter(ctx context.Context, name, realm string) error
		RemoveCharacter(ctx context.Context, name, realm string) error
		ListCharacters(ctx context.Context, limit int) ([]db.Character, error)
	}

	Bot struct {
		messageSender    discord.SenderIface
		updater          Updater
		characterService CharacterService
	}
)

const (
	Command = "!mythicplusbot"

	helpMessage = "This bot tracks characters M+ scores and will post updates to the channel whenever they increase:\n" +
		"\n- To add a character send: `!mythicplusbot character add <character> <realm>`" +
		"\n- To remove a character send: `!mythicplusbot character remove <character> <realm>`" +
		"\n- To list the top `n` scores send: `!mythicplusbot scores [-n 10]`" +
		"\n- To update scores outside the 30 minute window send: `!mythicplusbot update`"

	defaultRows = 10
)

func NewBot(messageSender discord.SenderIface, updater Updater, characterService CharacterService) *Bot {
	return &Bot{
		messageSender:    messageSender,
		updater:          updater,
		characterService: characterService,
	}
}

func (b *Bot) HandleMessage(ctx context.Context, content, channelID string) error {
	if !strings.HasPrefix(content, Command) {
		return nil
	}

	args := strings.Fields(content)
	if len(args) < 2 {
		return b.messageSender.SendMessage(ctx, channelID, "Usage: !mythicplusbot <command> [args]")
	}

	switch args[1] {
	case "add":
		return b.handleAddCharacter(ctx, channelID, args)
	case "remove":
		return b.handleRemoveCharacter(ctx, channelID, args)
	case "scores", "list":
		return b.handleScoresCommand(ctx, channelID, args)
	case "update":
		return b.handleUpdateCommand(ctx, channelID)
	case "help":
		return b.messageSender.SendMessage(ctx, channelID, helpMessage)
	default:
		return b.messageSender.SendMessage(ctx, channelID, "Unknown command. Use "+Command+" help for a list of commands.")
	}
}

// handleAddCharacter handles adding a character
func (b *Bot) handleAddCharacter(ctx context.Context, channelID string, args []string) error {
	if len(args) < 4 {
		return b.messageSender.SendMessage(ctx, channelID, "Usage: !mythicplusbot add <character> <realm>")
	}

	character := formatName(args[2])
	realm := formatRealm(args[3])
	if err := b.characterService.AddCharacter(ctx, character, realm); err != nil {
		slog.ErrorContext(ctx, "failed to add character", "error", err, "character", character, "realm", realm)
		return b.messageSender.SendMessage(ctx, channelID, "Failed to add character.")
	}

	return b.messageSender.SendMessage(ctx, channelID, fmt.Sprintf("Now tracking %s-%s", character, realm))
}

// handleRemoveCharacter handles removing a character
func (b *Bot) handleRemoveCharacter(ctx context.Context, channelID string, args []string) error {
	if len(args) < 4 {
		return b.messageSender.SendMessage(ctx, channelID, "Usage: !mythicplusbot remove <character> <realm>")
	}

	character := formatName(args[2])
	realm := formatRealm(args[3])
	if err := b.characterService.RemoveCharacter(ctx, character, realm); err != nil {
		slog.ErrorContext(ctx, "failed to remove character", "error", err, "character", character, "realm", realm)
		return b.messageSender.SendMessage(ctx, channelID, "Failed to remove character.")
	}

	return b.messageSender.SendMessage(ctx, channelID, fmt.Sprintf("No longer tracking %s-%s.", character, realm))
}

func (b *Bot) handleScoresCommand(ctx context.Context, channelID string, args []string) error {
	n := defaultRows
	for i, arg := range args {
		if arg == "-n" && i+1 < len(args) {
			if _, err := fmt.Sscanf(args[i+1], "%d", &n); err != nil {
				slog.ErrorContext(ctx, "failed to parse scores command", "error", err)
				n = defaultRows // fallback to default
			}
		}
	}

	characters, err := b.characterService.ListCharacters(ctx, n)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get scores", "error", err)
		return b.messageSender.SendMessage(ctx, channelID, "Failed to get scores")
	}

	if args[1] == "list" {
		return b.messageSender.SendMessage(ctx, channelID, "todo :(")
	}

	return b.messageSender.SendComplexMessage(ctx, channelID, discord.BuildScoresMessage(characters))
}

// handleUpdateCommand handles the update command
func (b *Bot) handleUpdateCommand(ctx context.Context, channelID string) error {
	if err := b.messageSender.SendMessage(ctx, channelID, "Checking for updates..."); err != nil {
		return err
	}

	if err := b.updater.Update(ctx, channelID); err != nil {
		slog.ErrorContext(ctx, "failed to update", "error", err)
		return b.messageSender.SendMessage(ctx, channelID, "Failed to update scores")
	}

	return nil
}

// formatName makes sure the character name is in the right format.
//
// We want the names to have a capital letter to start and the rest be lowercase.
func formatName(name string) string {
	return strings.ToUpper(name[:1]) + strings.ToLower(name[1:])
}

// formatRealm makes sure the realm is all lower case.
func formatRealm(realm string) string {
	return strings.ToLower(realm)
}
