package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DylanNZL/mythicplusbot/blizzard"
	"github.com/DylanNZL/mythicplusbot/bot"
	"github.com/DylanNZL/mythicplusbot/config"
	"github.com/DylanNZL/mythicplusbot/db"
	"github.com/DylanNZL/mythicplusbot/discord"
	"github.com/DylanNZL/mythicplusbot/raiderio"
	"github.com/DylanNZL/mythicplusbot/updater"
	"github.com/bwmarrin/discordgo"
)

const defaultHTTPTimeout = 30 * time.Second

func init() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	slog.SetLogLoggerLevel(slog.Level(cfg.LogLevel))
}

func main() {
	ctx := context.Background()
	cfg := config.Get()

	database, err := db.NewSQLiteDB(cfg.DatabaseLocation)
	if err != nil {
		slog.ErrorContext(ctx, "error creating database", "error", err)
		panic(err)
	}
	defer database.Close()

	if err := database.Init(ctx); err != nil {
		slog.ErrorContext(ctx, "error initialising database", "error", err)
		panic(err)
	}

	characterRepo := db.NewCharacterRepo(database)

	httpClient := &http.Client{Timeout: defaultHTTPTimeout}
	timeProvider := &blizzard.RealTimeProvider{}
	blizzardClient := blizzard.NewClient(httpClient, timeProvider)
	blizzardClient.SetCredentials(cfg.BlizzardClientID, cfg.BlizzardClientSecret)
	raiderIOClient := raiderio.NewClient(cfg.RaiderIOAccessKey, httpClient)

	slog.DebugContext(ctx, "setting up discord")
	d, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		slog.ErrorContext(ctx, "error creating Discord session", "error", err)
		panic(err)
	}

	d.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	messageSender := discord.NewDiscordSender(d)

	// Create services with dependency injection
	botService := bot.NewBot(
		messageSender,
		&BotUpdaterService{
			updaterService: createUpdaterService(characterRepo, blizzardClient, raiderIOClient, messageSender),
			channelID:      cfg.DiscordChannelID,
		},
		&BotCharacterService{repo: characterRepo, bClient: blizzardClient, rClient: raiderIOClient},
	)

	// Add Discord message handler
	d.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore bot's own messages
		if m.Author.ID == s.State.User.ID {
			return
		}

		// Lock the bot to the server it is configured for
		if m.ChannelID != cfg.DiscordChannelID {
			return
		}

		if err := botService.HandleMessage(ctx, m.Content, m.ChannelID); err != nil {
			slog.ErrorContext(ctx, "failed to handle message", "error", err)
		}
	})

	slog.DebugContext(ctx, "opening discord session")
	if err := d.Open(); err != nil {
		slog.ErrorContext(ctx, "error opening discord session", "error", err)
		panic(err)
	}
	slog.InfoContext(ctx, "listening for messages")

	updaterService := createUpdaterService(characterRepo, blizzardClient, raiderIOClient, messageSender)
	ticker := time.NewTicker(time.Duration(cfg.UpdaterFrequency) * time.Minute)
	go func() {
		for range ticker.C {
			if err := updaterService.Update(ctx, cfg.DiscordChannelID); err != nil {
				slog.ErrorContext(ctx, "updater failed", "error", err)
			}
		}
	}()

	if err := updaterService.Update(ctx, cfg.DiscordChannelID); err != nil {
		panic(err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	slog.InfoContext(ctx, "closing discord session")
	if err := d.Close(); err != nil {
		slog.ErrorContext(ctx, "error closing Discord session", "error", err)
	}

	ticker.Stop()
}

func createUpdaterService(characterRepo *db.CharacterRepo, blizzardClient *blizzard.Client, raiderIOClient *raiderio.Client, messageSender discord.SenderIface) *updater.Service {
	return updater.NewService(
		&UpdaterCharacterRepository{repo: characterRepo},
		&UpdaterBlizzardClient{client: blizzardClient},
		&UpdaterRaiderIOClient{client: raiderIOClient},
		messageSender,
		&updater.RealSleeper{},
	)
}

// Adapter implementations for bot service

// BotUpdaterService adapts the updater service for the bot
type BotUpdaterService struct {
	updaterService *updater.Service
	channelID      string
}

func (b *BotUpdaterService) Update(ctx context.Context, channelID string) error {
	// Use the configured channel ID for updates
	return b.updaterService.Update(ctx, b.channelID)
}

type BotCharacterService struct {
	repo    *db.CharacterRepo
	bClient *blizzard.Client
	rClient *raiderio.Client
}

func (b *BotCharacterService) AddCharacter(ctx context.Context, name, realm string) error {
	profile, err := b.bClient.GetMythicKeystoneProfile(ctx, realm, name)
	if err != nil {
		return err
	}

	rProfile, err := b.rClient.GetCharacter(ctx, realm, name)
	if err != nil {
		return err
	}

	current := raiderio.Season{}
	if len(rProfile.MythicPlusScoresBySeason) > 0 {
		current = rProfile.MythicPlusScoresBySeason[0]
	}

	character := db.Character{
		ID:           profile.Character.ID,
		Name:         profile.Character.Name,
		Realm:        profile.Character.Realm.Slug,
		Class:        rProfile.Class,
		OverallScore: profile.CurrentMythicRating.Rating,
		TankScore:    current.Scores.Tank,
		DPSScore:     current.Scores.Dps,
		HealScore:    current.Scores.Healer,
		DateCreated:  time.Now().Unix(),
		DateUpdated:  time.Now().Unix(),
	}
	return b.repo.Insert(ctx, &character)
}

func (b *BotCharacterService) RemoveCharacter(ctx context.Context, name, realm string) error {
	character := &db.Character{Name: name, Realm: realm}
	return b.repo.Delete(ctx, character)
}

func (b *BotCharacterService) ListCharacters(ctx context.Context, limit int) ([]db.Character, error) {
	characters, err := b.repo.ListCharacters(ctx, limit)
	if err != nil {
		return nil, err
	}

	return characters, nil
}

type UpdaterCharacterRepository struct {
	repo *db.CharacterRepo
}

func (u *UpdaterCharacterRepository) ListCharacters(ctx context.Context, limit int) ([]db.Character, error) {
	return u.repo.ListCharacters(ctx, limit)
}

func (u *UpdaterCharacterRepository) UpdateCharacter(ctx context.Context, character *db.Character) error {
	return u.repo.Update(ctx, character)
}

type UpdaterBlizzardClient struct {
	client *blizzard.Client
}

func (u *UpdaterBlizzardClient) GetMythicKeystoneProfile(ctx context.Context, realm, character string) (*blizzard.MythicKeystoneProfile, error) {
	return u.client.GetMythicKeystoneProfile(ctx, realm, character)
}

type UpdaterRaiderIOClient struct {
	client *raiderio.Client
}

func (u *UpdaterRaiderIOClient) GetCharacter(ctx context.Context, realm, character string) (*raiderio.Character, error) {
	return u.client.GetCharacter(ctx, realm, character)
}
