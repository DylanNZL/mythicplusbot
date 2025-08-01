package config

import (
	"fmt"
	"os"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type Config struct {
	BlizzardClientID     string `yaml:"blizzardClientId"`
	BlizzardClientSecret string `yaml:"blizzardClientSecret"`
	RaiderIOAccessKey    string `yaml:"raiderIOAccessKey"`
	DiscordToken         string `yaml:"discordToken"`
	DiscordChannelID     string `yaml:"discordChannelId"`
	DatabaseLocation     string `yaml:"databaseLocation"`
	LogLevel             int    `yaml:"logLevel"`         // maps to slog.LogLevels
	UpdaterFrequency     int64  `yaml:"updaterFrequency"` // How frequently to run the updater
}

const (
	defaultConfigPath       = "./config.yml"
	defaultDatabaseLocation = "mythicplusdiscordbot.sqlite"
	defaultUpdaterFrequency = 30
)

// defaultConfig provides some normal defaults for config values that are optional.
var defaultConfig = Config{
	DatabaseLocation: defaultDatabaseLocation,
	UpdaterFrequency: defaultUpdaterFrequency,
}

var config Config

// merge copies values from the passed in Config.
//
// Note 0 is a valid value for Config.LogLevel, so we don't merge that attribute.
func (c *Config) merge(cfg Config) {
	if c.BlizzardClientID == "" {
		c.BlizzardClientID = cfg.BlizzardClientID
	}
	if c.BlizzardClientSecret == "" {
		c.BlizzardClientSecret = cfg.BlizzardClientSecret
	}
	if c.DiscordToken == "" {
		c.DiscordToken = cfg.DiscordToken
	}
	if c.DiscordChannelID == "" {
		c.DiscordChannelID = cfg.DiscordChannelID
	}
	if c.DatabaseLocation == "" {
		c.DatabaseLocation = cfg.DatabaseLocation
	}
	if c.UpdaterFrequency == 0 {
		c.UpdaterFrequency = cfg.UpdaterFrequency
	}
}

func LoadFs(fs afero.Fs) (Config, error) {
	path := os.Getenv("CONFIG_FILE")
	if path == "" {
		path = defaultConfigPath
	}

	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.merge(defaultConfig)
	config = cfg

	return cfg, nil
}

func Load() (Config, error) {
	return LoadFs(afero.NewOsFs())
}

func Get() Config {
	return config
}
