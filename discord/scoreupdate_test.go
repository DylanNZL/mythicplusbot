package discord

import (
	"context"
	"testing"
	"time"

	"github.com/DylanNZL/mythicplusbot/db"
	"github.com/DylanNZL/mythicplusbot/raiderio"
	"github.com/stretchr/testify/assert"
)

var (
	testDBCharacter = db.Character{
		Name:         "Paladylan",
		Realm:        "tichondrius",
		Class:        "Paladin",
		OverallScore: 2500.0,
		TankScore:    2400.0,
		HealScore:    2300.0,
		DPSScore:     2200.0,
	}

	testRIOCharacter = raiderio.Character{
		Name:         "Paladylan",
		Race:         "Blood Elf",
		Realm:        "tichondrius",
		Class:        "Paladin",
		ThumbnailUrl: "https://render.worldofwarcraft.com/us/character/tichondrius/206/245077966-avatar.jpg?alt=/wow/static/images/2d/avatar/10-1.jpg",
		ProfileUrl:   "https://raider.io/characters/us/tichondrius/Paladylan",
		MythicPlusScoresBySeason: []raiderio.Season{
			{
				Season: "season-tww-3",
				Scores: raiderio.Scores{
					All:    1234.5,
					Dps:    1234.5,
					Healer: 1234.5,
					Tank:   1234.5,
				},
			},
		},
		MythicPlusRanks: raiderio.Ranks{
			Overall: raiderio.Rank{
				World:  123,
				Region: 123,
				Realm:  123,
			},
			Tank: raiderio.Rank{
				World:  456,
				Region: 456,
				Realm:  456,
			},
			Healer: raiderio.Rank{
				World:  789,
				Region: 789,
				Realm:  789,
			},
			Dps: raiderio.Rank{
				World:  321,
				Region: 321,
				Realm:  321,
			},
		},
		MythicPlusRecentRuns: []raiderio.Run{
			{
				Dungeon:             "Halls of Atonement",
				ShortName:           "HOA",
				MythicLevel:         6,
				CompletedAt:         time.Now(),
				NumKeystoneUpgrades: 1,
				BackgroundImageUrl:  "https://cdn.raiderio.net/images/dungeons/expansion8/base/halls-of-atonement.jpg",
				Score:               232.2,
				Url:                 "https://raider.io/mythic-plus-runs/season-tww-3/2568986-6-halls-of-atonement",
			},
		},
	}
)

func TestGetLatestRun(t *testing.T) {
	now := time.Now()

	t.Run("empty runs", func(t *testing.T) {
		character := raiderio.Character{
			MythicPlusRecentRuns: []raiderio.Run{},
		}

		result := getLatestRun(character)

		// Should return zero value of Run
		assert.Equal(t, raiderio.Run{}, result)
	})

	t.Run("single run", func(t *testing.T) {
		run := raiderio.Run{
			Dungeon:     "Mists of Tirna Scithe",
			MythicLevel: 8,
			CompletedAt: now,
		}

		character := raiderio.Character{
			MythicPlusRecentRuns: []raiderio.Run{run},
		}

		result := getLatestRun(character)

		assert.Equal(t, run, result)
	})

	t.Run("multiple runs in chronological order", func(t *testing.T) {
		oldRun := raiderio.Run{
			Dungeon:     "The Necrotic Wake",
			MythicLevel: 7,
			CompletedAt: now.Add(-2 * time.Hour),
		}

		latestRun := raiderio.Run{
			Dungeon:     "Halls of Atonement",
			MythicLevel: 9,
			CompletedAt: now,
		}

		character := raiderio.Character{
			MythicPlusRecentRuns: []raiderio.Run{latestRun, oldRun},
		}

		result := getLatestRun(character)

		assert.Equal(t, latestRun, result)
	})

	t.Run("multiple runs out of chronological order", func(t *testing.T) {
		oldestRun := raiderio.Run{
			Dungeon:     "The Necrotic Wake",
			MythicLevel: 7,
			CompletedAt: now.Add(-3 * time.Hour),
		}

		middleRun := raiderio.Run{
			Dungeon:     "Plaguefall",
			MythicLevel: 8,
			CompletedAt: now.Add(-1 * time.Hour),
		}

		latestRun := raiderio.Run{
			Dungeon:     "Halls of Atonement",
			MythicLevel: 9,
			CompletedAt: now,
		}

		// Put them in wrong order - oldest first
		character := raiderio.Character{
			MythicPlusRecentRuns: []raiderio.Run{oldestRun, latestRun, middleRun},
		}

		result := getLatestRun(character)

		assert.Equal(t, latestRun, result)
	})
}

// Test BuildScoreData function

func TestBuildScoreData(t *testing.T) {
	t.Run("all scores present", func(t *testing.T) {
		character := db.Character{
			TankScore: 2400.5,
			HealScore: 2300.75,
			DPSScore:  2200.25,
		}

		result := buildScoreData(character)

		expected := []scoreData{
			{Role: "Tank", Score: "2400.50"},
			{Role: "Healer", Score: "2300.75"},
			{Role: "DPS", Score: "2200.25"},
		}

		assert.Equal(t, expected, result)
	})

	t.Run("only tank score", func(t *testing.T) {
		character := db.Character{
			TankScore: 2400.5,
			HealScore: 0,
			DPSScore:  0,
		}

		result := buildScoreData(character)

		expected := []scoreData{
			{Role: "Tank", Score: "2400.50"},
		}

		assert.Equal(t, expected, result)
	})

	t.Run("only healer score", func(t *testing.T) {
		character := db.Character{
			TankScore: 0,
			HealScore: 2300.75,
			DPSScore:  0,
		}

		result := buildScoreData(character)

		expected := []scoreData{
			{Role: "Healer", Score: "2300.75"},
		}

		assert.Equal(t, expected, result)
	})

	t.Run("only dps score", func(t *testing.T) {
		character := db.Character{
			TankScore: 0,
			HealScore: 0,
			DPSScore:  2200.25,
		}

		result := buildScoreData(character)

		expected := []scoreData{
			{Role: "DPS", Score: "2200.25"},
		}

		assert.Equal(t, expected, result)
	})

	t.Run("no scores", func(t *testing.T) {
		character := db.Character{
			TankScore: 0,
			HealScore: 0,
			DPSScore:  0,
		}

		result := buildScoreData(character)

		assert.Empty(t, result)
	})
}

// Test BuildRankData function

func TestBuildRankData(t *testing.T) {
	t.Run("all roles with scores", func(t *testing.T) {
		character := raiderio.Character{
			MythicPlusScoresBySeason: []raiderio.Season{
				{
					Scores: raiderio.Scores{
						Tank:   1500.0,
						Healer: 1400.0,
						Dps:    1300.0,
					},
				},
			},
			MythicPlusRanks: raiderio.Ranks{
				Tank: raiderio.Rank{
					World: 100,
					Realm: 10,
				},
				Healer: raiderio.Rank{
					World: 200,
					Realm: 20,
				},
				Dps: raiderio.Rank{
					World: 300,
					Realm: 30,
				},
			},
		}

		result := buildRankData(character)

		expected := []rankData{
			{Role: "Tank", RealmRank: 10, OverallRank: 100},
			{Role: "Healer", RealmRank: 20, OverallRank: 200},
			{Role: "DPS", RealmRank: 30, OverallRank: 300},
		}

		assert.Equal(t, expected, result)
	})

	t.Run("only tank with score", func(t *testing.T) {
		character := raiderio.Character{
			MythicPlusScoresBySeason: []raiderio.Season{
				{
					Scores: raiderio.Scores{
						Tank:   1500.0,
						Healer: 0,
						Dps:    0,
					},
				},
			},
			MythicPlusRanks: raiderio.Ranks{
				Tank: raiderio.Rank{
					World: 100,
					Realm: 10,
				},
			},
		}

		result := buildRankData(character)

		expected := []rankData{
			{Role: "Tank", RealmRank: 10, OverallRank: 100},
		}

		assert.Equal(t, expected, result)
	})

	t.Run("no scores", func(t *testing.T) {
		character := raiderio.Character{
			MythicPlusScoresBySeason: []raiderio.Season{
				{
					Scores: raiderio.Scores{
						Tank:   0,
						Healer: 0,
						Dps:    0,
					},
				},
			},
		}

		result := buildRankData(character)

		assert.Empty(t, result)
	})
}

// Test BuildScoreUpdateMessage function

func TestBuildScoreUpdateMessage(t *testing.T) {
	t.Run("successful template execution", func(t *testing.T) {
		ctx := context.Background()
		latestRun := raiderio.Run{
			Dungeon:             "Test Dungeon",
			MythicLevel:         15,
			NumKeystoneUpgrades: 2,
			Score:               150.5,
			Url:                 "https://example.com/run",
		}

		result := buildScoreUpdateMessage(ctx, testDBCharacter, testRIOCharacter, latestRun)

		assert.Contains(t, result, "Tank Score")
		assert.Contains(t, result, "2400.00")
		assert.Contains(t, result, "Healer Score")
		assert.Contains(t, result, "2300.00")
		assert.Contains(t, result, "DPS Score")
		assert.Contains(t, result, "2200.00")
		assert.Contains(t, result, "#123 Realm")
		assert.Contains(t, result, "#123 Overall")
		assert.Contains(t, result, "Test Dungeon")
		assert.Contains(t, result, "15")
		assert.Contains(t, result, "+2")
		assert.Contains(t, result, "150.50")
		assert.Contains(t, result, "https://example.com/run")
	})

	t.Run("template parse error fallback", func(t *testing.T) {
		// This test would require mocking the template.New function to return an error
		// For now, we'll test the fallback function directly
		result := buildScoreUpdateMessageFallback(testDBCharacter)

		expected := "**Tank Score** 2400\n**Healer Score** 2300\n**DPS Score** 2200"
		assert.Equal(t, expected, result)
	})
}

func TestBuildScoreUpdateMessageFallback(t *testing.T) {
	t.Run("formats scores correctly", func(t *testing.T) {
		character := db.Character{
			TankScore: 2500.75,
			HealScore: 2400.25,
			DPSScore:  2300.0,
		}

		result := buildScoreUpdateMessageFallback(character)

		expected := "**Tank Score** 2501\n**Healer Score** 2400\n**DPS Score** 2300"
		assert.Equal(t, expected, result)
	})

	t.Run("handles zero scores", func(t *testing.T) {
		character := db.Character{
			TankScore: 0,
			HealScore: 0,
			DPSScore:  0,
		}

		result := buildScoreUpdateMessageFallback(character)

		expected := "**Tank Score** 00\n**Healer Score** 00\n**DPS Score** 00"
		assert.Equal(t, expected, result)
	})
}

// Test BuildScoreUpdateMessage function (main function)

func TestBuildScoreUpdateMessageMain(t *testing.T) {
	t.Run("complete message structure", func(t *testing.T) {
		ctx := context.Background()
		oldScore := 2000.0

		message := BuildScoreUpdateMessage(ctx, testDBCharacter, testRIOCharacter, oldScore)

		// Test the content
		expectedContent := "[Paladylan-tichondrius](https://raider.io/characters/us/tichondrius/Paladylan) increased their score from 2000.00 to 2500.00"
		assert.Equal(t, expectedContent, message.Content)

		// Test embeds
		assert.Len(t, message.Embeds, 1)
		embed := message.Embeds[0]

		assert.Equal(t, "2500.00 Overall Mythic+ Score", embed.Title)
		assert.Equal(t, testRIOCharacter.ProfileUrl, embed.URL)
		assert.Equal(t, "Paladylan-tichondrius (Paladin)", embed.Author.Name)
		assert.Equal(t, getClassIcon("Paladin"), embed.Author.IconURL)
		assert.Equal(t, getClassColour("Paladin"), embed.Color)
		assert.Equal(t, testRIOCharacter.ThumbnailUrl, embed.Thumbnail.URL)
		assert.Equal(t, testRIOCharacter.MythicPlusRecentRuns[0].BackgroundImageUrl, embed.Image.URL)
	})

	t.Run("with empty recent runs", func(t *testing.T) {
		ctx := context.Background()
		oldScore := 2000.0

		emptyRunsCharacter := testRIOCharacter
		emptyRunsCharacter.MythicPlusRecentRuns = []raiderio.Run{}

		message := BuildScoreUpdateMessage(ctx, testDBCharacter, emptyRunsCharacter, oldScore)

		// Should still create a message but with empty run data
		assert.NotEmpty(t, message.Content)
		assert.Len(t, message.Embeds, 1)

		// Image URL should be empty since there's no latest run
		embed := message.Embeds[0]
		assert.Empty(t, embed.Image.URL)
	})
}

// Test edge cases and error scenarios

func TestDescriptionDataStructure(t *testing.T) {
	t.Run("descriptionData with all fields", func(t *testing.T) {
		data := descriptionData{
			Scores: []scoreData{
				{Role: "Tank", Score: "2400.00"},
				{Role: "Healer", Score: "2300.00"},
			},
			RealmRank:   123,
			OverallRank: 456,
			Ranks: []rankData{
				{Role: "Tank", RealmRank: 10, OverallRank: 100},
			},
			Dungeon:  "Test Dungeon",
			Level:    15,
			Result:   2,
			Points:   "150.50",
			MoreInfo: "https://example.com",
		}

		// Verify the structure is properly initialized
		assert.Len(t, data.Scores, 2)
		assert.Equal(t, "Tank", data.Scores[0].Role)
		assert.Equal(t, "2400.00", data.Scores[0].Score)
		assert.Equal(t, 123, data.RealmRank)
		assert.Equal(t, 456, data.OverallRank)
		assert.Len(t, data.Ranks, 1)
		assert.Equal(t, "Test Dungeon", data.Dungeon)
		assert.Equal(t, 15, data.Level)
		assert.Equal(t, 2, data.Result)
		assert.Equal(t, "150.50", data.Points)
		assert.Equal(t, "https://example.com", data.MoreInfo)
	})
}

func TestBuildRankDataEdgeCases(t *testing.T) {
	t.Run("empty seasons array", func(t *testing.T) {
		character := raiderio.Character{
			MythicPlusScoresBySeason: []raiderio.Season{},
		}

		result := buildRankData(character)

		// Should return empty slice without panicking
		assert.Empty(t, result)
	})

	t.Run("mixed scores", func(t *testing.T) {
		character := raiderio.Character{
			MythicPlusScoresBySeason: []raiderio.Season{
				{
					Scores: raiderio.Scores{
						Tank:   1500.0,
						Healer: 0,
						Dps:    1300.0,
					},
				},
			},
			MythicPlusRanks: raiderio.Ranks{
				Tank: raiderio.Rank{
					World: 100,
					Realm: 10,
				},
				Dps: raiderio.Rank{
					World: 300,
					Realm: 30,
				},
			},
		}

		result := buildRankData(character)

		expected := []rankData{
			{Role: "Tank", RealmRank: 10, OverallRank: 100},
			{Role: "DPS", RealmRank: 30, OverallRank: 300},
		}

		assert.Equal(t, expected, result)
		// Should only have Tank and DPS, not Healer
		assert.Len(t, result, 2)
	})
}

func TestScoreDataStructure(t *testing.T) {
	t.Run("scoreData initialization", func(t *testing.T) {
		score := scoreData{
			Role:  "Tank",
			Score: "2500.75",
		}

		assert.Equal(t, "Tank", score.Role)
		assert.Equal(t, "2500.75", score.Score)
	})
}

func TestRankDataStructure(t *testing.T) {
	t.Run("rankData initialization", func(t *testing.T) {
		rank := rankData{
			Role:        "Healer",
			RealmRank:   50,
			OverallRank: 500,
		}

		assert.Equal(t, "Healer", rank.Role)
		assert.Equal(t, 50, rank.RealmRank)
		assert.Equal(t, 500, rank.OverallRank)
	})
}
