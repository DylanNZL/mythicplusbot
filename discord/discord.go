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

func BuildScoreUpdateMessage(c db.Character, oldScore float64) discordgo.MessageSend {
	raiderIO := fmt.Sprintf("https://raider.io/characters/us/%s/%s", c.Realm, c.Name)
	return discordgo.MessageSend{
		Content: fmt.Sprintf("[%s-%s](%s) increased their score from %0.2f to %0.2f",
			c.Name, c.Realm, raiderIO, oldScore, c.OverallScore),
		Embeds: []*discordgo.MessageEmbed{
			{
				Title: fmt.Sprintf("%0.2f Overall Mythic+ Score]", c.OverallScore),
				URL:   raiderIO,
				Author: &discordgo.MessageEmbedAuthor{
					Name:    fmt.Sprintf("%s-%s (%s)", c.Name, c.Realm, c.Class),
					IconURL: getClassIcon(c.Class),
				},
				Color: getClassColour(c.Class), //nolint:misspell // blizzards fault
				Description: fmt.Sprintf("**Tank Score** %02.f\n**Healer Score** %02.f\n**DPS Score** %02.f",
					c.TankScore, c.HealScore, c.DPSScore),
			},
		},
	}
}

func BuildScoresMessage(characters []db.Character) discordgo.MessageSend {
	return discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:  "Tracked Characters",
				Color:  scoresColour,
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

// getClassIcon returns the URL to an icon hosted by blizzard for that class.
//
//nolint:cyclop
func getClassIcon(class string) string {
	base := "https://render.worldofwarcraft.com/us/icons/18/"
	switch class {
	case "Warrior":
		return base + "class_1.jpg"
	case "Paladin":
		return base + "class_2.jpg"
	case "Hunter":
		return base + "class_3.jpg"
	case "Rogue":
		return base + "class_4.jpg"
	case "Priest":
		return base + "class_5.jpg"
	case "DeathKnight":
		return base + "class_6.jpg"
	case "Shaman":
		return base + "class_7.jpg"
	case "Mage":
		return base + "class_8.jpg"
	case "Warlock":
		return base + "class_9.jpg"
	case "Monk":
		return base + "class_10.jpg"
	case "Druid":
		return base + "class_11.jpg"
	case "DemonHunter":
		return base + "class_12.jpg"
	case "Evoker":
		// Blizzard haven't provided a evoker icon?
		return base + "class_2.jpg"

	default:
		return "https://render-us.worldofwarcraft.com/icons/18/class_2.jpg"
	}
}

// getClassColour returns the class colour.
//
//nolint:mnd,cyclop
func getClassColour(class string) int {
	switch class {
	case "Warrior":
		return 13015917
	case "Paladin":
		return 16026810
	case "Hunter":
		return 11195250
	case "Rogue":
		return 16774248
	case "Priest":
		return 16777215
	case "DeathKnight":
		return 12852794
	case "Shaman":
		return 28893
	case "Mage":
		return 4179947
	case "Warlock":
		return 8882414
	case "Monk":
		return 2326507
	case "Druid":
		return 16743434
	case "DemonHunter":
		return 10694857
	case "Evoker":
		return 3380095

	default:
		return 0
	}
}
