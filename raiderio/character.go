package raiderio

import (
	"encoding/json"
	"time"
)

// Character is the partial response from Raider.io.
//
// Note some attributes have been removed.
//
//nolint:all
type (
	Character struct {
		Name                     string            `json:"name"`
		Race                     string            `json:"race"`
		Class                    string            `json:"class"`
		ThumbnailUrl             string            `json:"thumbnail_url"`
		Realm                    string            `json:"realm"`
		LastCrawledAt            time.Time         `json:"last_crawled_at"`
		ProfileUrl               string            `json:"profile_url"`
		ProfileBanner            string            `json:"profile_banner"`
		Gear                     json.RawMessage   `json:"gear"`
		RaidProgression          json.RawMessage   `json:"raid_progression"` // We don't care about this so don't unmarshal it
		MythicPlusScoresBySeason []Season          `json:"mythic_plus_scores_by_season"`
		MythicPlusRanks          Ranks             `json:"mythic_plus_ranks"`
		PreviousMythicPlusRanks  []json.RawMessage `json:"previous_mythic_plus_ranks"`
		MythicPlusRecentRuns     []Run             `json:"mythic_plus_recent_runs"`
		MythicPlusBestRuns       []json.RawMessage `json:"mythic_plus_best_runs"`
		MythicPlusAlternateRuns  []json.RawMessage `json:"mythic_plus_alternate_runs"`
	}

	Season struct {
		Season   string `json:"season"`
		Scores   Scores `json:"scores"`
		Segments struct {
			All    ScoreSegment `json:"all"`
			Dps    ScoreSegment `json:"dps"`
			Healer ScoreSegment `json:"healer"`
			Tank   ScoreSegment `json:"tank"`
			// Note we may want to add some special unwrapping here as we each spec will have its own spec# attribute unique to the class
		} `json:"segments"`
	}

	Scores struct {
		All    float64 `json:"all"`
		Dps    float64 `json:"dps"`
		Healer float64 `json:"healer"`
		Tank   float64 `json:"tank"`
		// Note we may want to add some special unwrapping here as we each spec will have its own spec# attribute unique to the class
	}

	Run struct {
		Dungeon             string    `json:"dungeon"`
		ShortName           string    `json:"short_name"`
		MythicLevel         int       `json:"mythic_level"`
		KeystoneRunId       int       `json:"keystone_run_id"`
		CompletedAt         time.Time `json:"completed_at"`
		ClearTimeMs         int       `json:"clear_time_ms"`
		ParTimeMs           int       `json:"par_time_ms"`
		NumKeystoneUpgrades int       `json:"num_keystone_upgrades"`
		MapChallengeModeId  int       `json:"map_challenge_mode_id"`
		ZoneId              int       `json:"zone_id"`
		ZoneExpansionId     int       `json:"zone_expansion_id"`
		IconUrl             string    `json:"icon_url"`
		BackgroundImageUrl  string    `json:"background_image_url"`
		Score               float64   `json:"score"`
		Url                 string    `json:"url"`
		Affixes             []Affix   `json:"affixes"`
	}

	Affix struct {
		Id          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		IconUrl     string `json:"icon_url"`
		WowheadUrl  string `json:"wowhead_url"`
	}

	ScoreSegment struct {
		Score float64 `json:"score"`
		Color string  `json:"color"`
	}

	Ranks struct {
		Overall     Rank `json:"overall"`
		Tank        Rank `json:"tank"`
		Healer      Rank `json:"healer"`
		Dps         Rank `json:"dps"`
		Class       Rank `json:"class"`
		ClassTank   Rank `json:"class_tank"`
		ClassHealer Rank `json:"class_healer"`
		ClassDps    Rank `json:"class_dps"`
	}

	Rank struct {
		World  int `json:"world"`
		Region int `json:"region"`
		Realm  int `json:"realm"`
	}
)
