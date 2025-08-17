package discord

import (
	"context"
	"fmt"

	"github.com/DylanNZL/mythicplusbot/db"
	"github.com/bwmarrin/discordgo"
)

type SenderIface interface {
	SendMessage(ctx context.Context, channelID, content string) error
	SendComplexMessage(ctx context.Context, channelID string, message discordgo.MessageSend) error
}

type Sender struct {
	session *discordgo.Session
}

const (
	maxEmbedFieldChars = 1024
	maxEmbedFields     = 22
	scoresColour       = 2326507
)

func NewDiscordSender(session *discordgo.Session) *Sender {
	return &Sender{session: session}
}

func (d *Sender) SendMessage(ctx context.Context, channelID, content string) error {
	_, err := d.session.ChannelMessageSend(channelID, content)
	return err
}

func (d *Sender) SendComplexMessage(ctx context.Context, channelID string, message discordgo.MessageSend) error {
	_, err := d.session.ChannelMessageSendComplex(channelID, &message)
	return err
}

func BuildScoresMessage(characters []db.Character) discordgo.MessageSend {
	return discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:  "Tracked Characters",
				Color:  scoresColour, //nolint:misspell // Discord not using the right language
				Fields: buildScoresFields(characters),
			},
		},
	}
}

func buildScoresFields(characters []db.Character) []*discordgo.MessageEmbedField {
	fields := make([]*discordgo.MessageEmbedField, 0)
	emptyField := discordgo.MessageEmbedField{
		Name:   " ",
		Inline: true,
	}

	charField := emptyField
	scoreField := emptyField
	for i, c := range characters {
		msg := fmt.Sprintf("%d) [%s-%s](https://raider.io/characters/us/%s/%s)\n", i+1, c.Name, c.Realm, c.Realm, c.Name)
		score := fmt.Sprintf("%0.2f\n", c.OverallScore)
		if len(msg)+len(charField.Value) >= maxEmbedFieldChars {
			// there is a max of 25 fields
			if len(fields) > maxEmbedFields {
				charField.Value += "\nToo many characters tracked - use `list` instead."
				break
			}
			fields = append(fields, &charField, &scoreField)
			charField = emptyField
			scoreField = emptyField

			charField.Value = msg
			scoreField.Value = score

			continue
		}
		charField.Value += msg
		scoreField.Value += score
	}

	fields = append(fields, &charField, &scoreField)
	return fields
}
