// Package updater handles reading all the characters and checking with blizzard for score updates.
//
// We also track the updates we make to send to the discord channel.
package updater

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/DylanNZL/mythicplusbot/blizzard"
	"github.com/DylanNZL/mythicplusbot/db"
	"github.com/DylanNZL/mythicplusbot/discord"
	"github.com/DylanNZL/mythicplusbot/raiderio"
)

const cooldownTime = time.Millisecond * 250

type (
	CharacterRepository interface {
		ListCharacters(ctx context.Context, limit int) ([]db.Character, error)
		UpdateCharacter(ctx context.Context, character *db.Character) error
	}

	BlizzardClient interface {
		GetMythicKeystoneProfile(ctx context.Context, realm string, character string) (*blizzard.MythicKeystoneProfile, error)
	}

	RaiderIOClient interface {
		GetCharacter(ctx context.Context, realm string, character string) (*raiderio.Character, error)
	}

	Sleeper interface {
		Sleep(duration time.Duration)
	}
)

type RealSleeper struct{}

func (r *RealSleeper) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

// Service handles score updates with injected dependencies
type Service struct {
	characterRepo  CharacterRepository
	blizzardClient BlizzardClient
	raiderioClient RaiderIOClient
	messageSender  discord.SenderIface
	sleeper        Sleeper
}

// NewService creates a new updater service with dependencies
func NewService(
	characterRepo CharacterRepository,
	blizzardClient BlizzardClient,
	raiderIOClient RaiderIOClient,
	messageSender discord.SenderIface,
	sleeper Sleeper,
) *Service {
	return &Service{
		characterRepo:  characterRepo,
		blizzardClient: blizzardClient,
		raiderioClient: raiderIOClient,
		messageSender:  messageSender,
		sleeper:        sleeper,
	}
}

// Update lists all characters in the db and checks with Blizzard on if their score has changed.
//
// It will also send messages to discord showing the change.
// Note it will also be triggered when seasons change (score goes from 1234 to 0).
func (s *Service) Update(ctx context.Context, discordChannelID string) error {
	slog.InfoContext(ctx, "running updater")
	characters, err := s.characterRepo.ListCharacters(ctx, 0)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list characters", "error", err)
		return fmt.Errorf("failed to list characters: %w", err)
	}

	for _, character := range characters {
		if err := s.updateCharacter(ctx, discordChannelID, character); err != nil {
			slog.ErrorContext(ctx, "failed to update character", "error", err,
				"character", character.Name, "realm", character.Realm)
			// Continue with other characters even if one fails
			continue
		}

		// We don't want to spam blizzard/discord apis so add an artificial delay in between character updates
		s.sleeper.Sleep(cooldownTime)
	}

	return nil
}

func (s *Service) updateCharacter(ctx context.Context, discordChannelID string, character db.Character) error {
	profile, err := s.blizzardClient.GetMythicKeystoneProfile(ctx, character.Realm, character.Name)
	if err != nil {
		return fmt.Errorf("failed to get mythic profile for %s-%s: %w", character.Name, character.Realm, err)
	}

	if profile.CurrentMythicRating.Rating == character.OverallScore {
		return nil
	}

	rCharacter, err := s.raiderioClient.GetCharacter(ctx, character.Realm, character.Name)
	if err != nil {
		return fmt.Errorf("failed to get character %s-%s: %w", character.Name, character.Realm, err)
	}

	season := raiderio.Season{}
	if len(rCharacter.MythicPlusScoresBySeason) > 0 {
		season = rCharacter.MythicPlusScoresBySeason[0]
	}

	oldScore := character.OverallScore
	character.OverallScore = profile.CurrentMythicRating.Rating
	character.TankScore = season.Scores.Tank
	character.HealScore = season.Scores.Healer
	character.DPSScore = season.Scores.Dps
	if err := s.characterRepo.UpdateCharacter(ctx, &character); err != nil {
		return fmt.Errorf("failed to update character score: %w", err)
	}

	if err := s.messageSender.SendComplexMessage(ctx, discordChannelID, discord.BuildScoreUpdateMessage(ctx, character, *rCharacter, oldScore)); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
