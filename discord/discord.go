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
	maxEmbedFields     = 24
	scoresColour       = 2326507
)

func NewDiscordSender(session *discordgo.Session) *Sender {
	return &Sender{session: session}
}

func (d *Sender) SendMessage(_ context.Context, channelID, content string) error {
	_, err := d.session.ChannelMessageSend(channelID, content)
	return err
}

func (d *Sender) SendComplexMessage(_ context.Context, channelID string, message discordgo.MessageSend) error {
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
	fields := getBasicScoresFields()
	charField := 0
	scoreField := 1
	for i, c := range characters {
		msg := fmt.Sprintf("%d) [%s-%s](https://raider.io/characters/us/%s/%s)\n", i+1, c.Name, c.Realm, c.Realm, c.Name)
		score := fmt.Sprintf("%0.0f\n", c.OverallScore)
		if len(msg)+len(fields[charField].Value) >= maxEmbedFieldChars {
			// there is a max of 25 fields
			if charField >= maxEmbedFields {
				fields[charField].Value += "\nToo many characters tracked to list them all."
				break
			}
			charField += 3
			scoreField += 3

			fields[charField].Value = msg
			fields[scoreField].Value = score

			continue
		}
		fields[charField].Value += msg
		fields[scoreField].Value += score
	}

	return fields[0 : scoreField+1]
}

func getBasicScoresFields() []*discordgo.MessageEmbedField {
	fields := make([]*discordgo.MessageEmbedField, maxEmbedFields+1)
	for i, _ := range fields {
		fields[i] = &discordgo.MessageEmbedField{
			Name:   " ",
			Inline: true,
		}
	}

	return fields
}
