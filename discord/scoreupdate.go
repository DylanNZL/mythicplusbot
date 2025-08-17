package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"text/template"

	"github.com/DylanNZL/mythicplusbot/db"
	"github.com/DylanNZL/mythicplusbot/raiderio"
	"github.com/bwmarrin/discordgo"
)

type (
	descriptionData struct {
		Scores      []scoreData
		RealmRank   int
		OverallRank int
		Ranks       []rankData
		Dungeon     string
		Level       int
		Result      int
		Points      string
		MoreInfo    string
	}

	scoreData struct {
		Role  string
		Score string
	}

	rankData struct {
		Role        string
		RealmRank   int
		OverallRank int
	}
)

const descriptionTemplate = `{{range $s := .Scores}}**{{$s.Role}} Score** {{$s.Score}}
{{end}}
**--- Ranks ---**
**#{{.RealmRank}} Realm - #{{.OverallRank}} Overall**
{{range $r := .Ranks}}**{{$r.Role}}**: #{{$r.RealmRank}} Realm - #{{$r.OverallRank}} Overall
{{end}}
**--- Last Run ---**
**Dungeon**: {{.Dungeon}}
**Level**: {{.Level}}
**Result**: +{{.Result}}
**Points**: {{.Points}}
[More Info]({{.MoreInfo}}) 
`

func BuildScoreUpdateMessage(ctx context.Context, c db.Character, rc raiderio.Character, oldScore float64) discordgo.MessageSend {
	latestRun := getLatestRun(rc)

	return discordgo.MessageSend{
		Content: fmt.Sprintf("[%s-%s](%s) increased their score from %0.2f to %0.2f",
			c.Name, c.Realm, rc.ProfileUrl, oldScore, c.OverallScore),
		Embeds: []*discordgo.MessageEmbed{
			{
				URL:         rc.ProfileUrl,
				Title:       fmt.Sprintf("%0.2f Overall Mythic+ Score", c.OverallScore),
				Description: buildScoreUpdateMessage(ctx, c, rc, latestRun),
				Color:       getClassColour(c.Class), //nolint:misspell // blizzards fault
				Footer:      nil,
				Image: &discordgo.MessageEmbedImage{
					URL: latestRun.BackgroundImageUrl,
				},
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: rc.ThumbnailUrl,
				},
				Author: &discordgo.MessageEmbedAuthor{
					Name:    fmt.Sprintf("%s-%s (%s)", c.Name, c.Realm, c.Class),
					IconURL: getClassIcon(c.Class),
				},
			},
		},
	}
}

func getLatestRun(rc raiderio.Character) (latestRun raiderio.Run) {
	if len(rc.MythicPlusRecentRuns) > 0 {
		latestRun = rc.MythicPlusRecentRuns[0]

		// This should be the latest run, but we doubl-check the full array in case it is out of order.
		for _, r := range rc.MythicPlusRecentRuns {
			if r.CompletedAt.After(latestRun.CompletedAt) {
				latestRun = r
			}
		}
	}

	return
}

func buildScoreUpdateMessage(ctx context.Context, c db.Character, rc raiderio.Character, latestRun raiderio.Run) string {
	tpl, err := template.New("description").Parse(descriptionTemplate)
	if err != nil {
		slog.ErrorContext(ctx, "failed to parse description template: "+err.Error())
		return buildScoreUpdateMessageFallback(c)
	}

	var s strings.Builder
	if err := tpl.Execute(&s, descriptionData{
		Scores:      buildScoreData(c),
		Ranks:       buildRankData(rc),
		RealmRank:   rc.MythicPlusRanks.Overall.Realm,
		OverallRank: rc.MythicPlusRanks.Overall.World,
		Dungeon:     latestRun.Dungeon,
		Level:       latestRun.MythicLevel,
		Result:      latestRun.NumKeystoneUpgrades,
		Points:      fmt.Sprintf("%0.2f", latestRun.Score),
		MoreInfo:    latestRun.Url,
	}); err != nil {
		slog.ErrorContext(ctx, "failed to execute description template: "+err.Error())
		return buildScoreUpdateMessageFallback(c)
	}

	return s.String()
}

func buildScoreData(c db.Character) (sd []scoreData) {
	if c.TankScore != 0 {
		sd = append(sd, scoreData{
			Role:  "Tank",
			Score: fmt.Sprintf("%0.2f", c.TankScore),
		})
	}
	if c.HealScore != 0 {
		sd = append(sd, scoreData{
			Role:  "Healer",
			Score: fmt.Sprintf("%0.2f", c.HealScore),
		})
	}
	if c.DPSScore != 0 {
		sd = append(sd, scoreData{
			Role:  "DPS",
			Score: fmt.Sprintf("%0.2f", c.DPSScore),
		})
	}
	return
}

func buildRankData(rc raiderio.Character) (rd []rankData) {
	if len(rc.MythicPlusScoresBySeason) == 0 {
		return
	}

	if rc.MythicPlusScoresBySeason[0].Scores.Tank != 0 {
		rd = append(rd, rankData{
			Role:        "Tank",
			RealmRank:   rc.MythicPlusRanks.Tank.Realm,
			OverallRank: rc.MythicPlusRanks.Tank.World,
		})
	}
	if rc.MythicPlusScoresBySeason[0].Scores.Healer != 0 {
		rd = append(rd, rankData{
			Role:        "Healer",
			RealmRank:   rc.MythicPlusRanks.Healer.Realm,
			OverallRank: rc.MythicPlusRanks.Healer.World,
		})
	}
	if rc.MythicPlusScoresBySeason[0].Scores.Dps != 0 {
		rd = append(rd, rankData{
			Role:        "DPS",
			RealmRank:   rc.MythicPlusRanks.Dps.Realm,
			OverallRank: rc.MythicPlusRanks.Dps.World,
		})
	}

	return
}

func buildScoreUpdateMessageFallback(c db.Character) string {
	return fmt.Sprintf("**Tank Score** %02.f\n**Healer Score** %02.f\n**DPS Score** %02.f",
		c.TankScore, c.HealScore, c.DPSScore)
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
		// Blizzard haven't provided an evoker icon?
		return base + "class_2.jpg"

	default:
		return base + "class_2.jpg"
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
