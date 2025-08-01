package config

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFs_Success(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected Config
	}{
		{
			name: "valid config with all fields",
			yaml: `blizzardClientId: test-client-id
blizzardClientSecret: test-client-secret
discordToken: test-discord-token
discordChannelId: test-channel-id
databaseLocation: /path/to/db.sqlite
logLevel: 2
updaterFrequency: 60`,
			expected: Config{
				BlizzardClientID:     "test-client-id",
				BlizzardClientSecret: "test-client-secret",
				DiscordToken:         "test-discord-token",
				DiscordChannelID:     "test-channel-id",
				DatabaseLocation:     "/path/to/db.sqlite",
				LogLevel:             2,
				UpdaterFrequency:     60,
			},
		},
		{
			name: "minimal config with defaults applied",
			yaml: `blizzardClientId: minimal-id
blizzardClientSecret: minimal-secret
discordToken: minimal-token
discordChannelId: minimal-channel`,
			expected: Config{
				BlizzardClientID:     "minimal-id",
				BlizzardClientSecret: "minimal-secret",
				DiscordToken:         "minimal-token",
				DiscordChannelID:     "minimal-channel",
				DatabaseLocation:     "mythicplusdiscordbot.sqlite", // default applied
				LogLevel:             0,
				UpdaterFrequency:     30, // default applied
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			err := afero.WriteFile(fs, "./config.yml", []byte(tt.yaml), 0644)
			require.NoError(t, err)

			cfg, err := LoadFs(fs)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, cfg)
		})
	}
}

func TestLoadFs_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()

	cfg, err := LoadFs(fs)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
	assert.Equal(t, Config{}, cfg)
}

func TestLoadFs_InvalidYAML(t *testing.T) {
	tests := []struct {
		name        string
		invalidYAML string
	}{
		{
			name:        "malformed yaml",
			invalidYAML: "not: valid: yaml: :",
		},
		{
			name:        "invalid structure",
			invalidYAML: "- this is a list\n- not an object",
		},
		{
			name:        "wrong data types",
			invalidYAML: `logLevel: "not a number"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			err := afero.WriteFile(fs, "./config.yml", []byte(tt.invalidYAML), 0644)
			require.NoError(t, err)

			cfg, err := LoadFs(fs)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse config file")
			assert.Equal(t, Config{}, cfg)
		})
	}
}

func TestLoadFs_CustomConfigPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	configYAML := `blizzardClientId: custom-path-test
blizzardClientSecret: custom-secret
discordToken: custom-token
discordChannelId: custom-channel
databaseLocation: custom.db
logLevel: 1`

	// Write config to custom path
	customPath := "./custom-config.yml"
	err := afero.WriteFile(fs, customPath, []byte(configYAML), 0644)
	require.NoError(t, err)

	// Set environment variable
	originalPath := os.Getenv("CONFIG_FILE")
	require.NoError(t, os.Setenv("CONFIG_FILE", customPath))
	defer func() {
		if originalPath == "" {
			_ = os.Unsetenv("CONFIG_FILE")
		} else {
			_ = os.Setenv("CONFIG_FILE", originalPath)
		}
	}()

	cfg, err := LoadFs(fs)
	require.NoError(t, err)

	assert.Equal(t, "custom-path-test", cfg.BlizzardClientID)
	assert.Equal(t, 1, cfg.LogLevel)
}

func TestLoad_UsesOsFilesystem(t *testing.T) {
	// This test verifies that Load() uses the OS filesystem
	// We'll create a temporary config file and verify it's loaded

	configYAML := `blizzardClientId: os-filesystem-test
blizzardClientSecret: os-secret
discordToken: os-token
discordChannelId: os-channel
databaseLocation: os.db
logLevel: 3`

	// Create temporary config file
	tmpFile := "./test-config.yml"
	err := os.WriteFile(tmpFile, []byte(configYAML), 0644)
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile) }()

	// Set environment variable to use our temp file
	originalPath := os.Getenv("CONFIG_FILE")
	require.NoError(t, os.Setenv("CONFIG_FILE", tmpFile))
	defer func() {
		if originalPath == "" {
			_ = os.Unsetenv("CONFIG_FILE")
		} else {
			_ = os.Setenv("CONFIG_FILE", originalPath)
		}
	}()

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "os-filesystem-test", cfg.BlizzardClientID)
	assert.Equal(t, 3, cfg.LogLevel)
}

func TestLoadFs_EmptyFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "./config.yml", []byte(""), 0644)
	require.NoError(t, err)

	cfg, err := LoadFs(fs)

	// Empty file should parse as empty config with defaults applied
	require.NoError(t, err)
	expected := Config{
		DatabaseLocation: "mythicplusdiscordbot.sqlite", // default applied
		UpdaterFrequency: 30,                            // default applied
	}
	assert.Equal(t, expected, cfg)
}

func TestConfig_Merge(t *testing.T) {
	tests := []struct {
		name     string
		base     Config
		merge    Config
		expected Config
	}{
		{
			name: "merge all empty fields",
			base: Config{},
			merge: Config{
				BlizzardClientID:     "test-client-id",
				BlizzardClientSecret: "test-client-secret",
				DiscordToken:         "test-discord-token",
				DiscordChannelID:     "test-channel-id",
				DatabaseLocation:     "test.db",
				UpdaterFrequency:     60,
			},
			expected: Config{
				BlizzardClientID:     "test-client-id",
				BlizzardClientSecret: "test-client-secret",
				DiscordToken:         "test-discord-token",
				DiscordChannelID:     "test-channel-id",
				DatabaseLocation:     "test.db",
				UpdaterFrequency:     60,
			},
		},
		{
			name: "merge only empty string fields",
			base: Config{
				BlizzardClientID: "existing-client-id",
				UpdaterFrequency: 30,
			},
			merge: Config{
				BlizzardClientID:     "merge-client-id",
				BlizzardClientSecret: "merge-client-secret",
				DiscordToken:         "merge-discord-token",
				DiscordChannelID:     "merge-channel-id",
				DatabaseLocation:     "merge.db",
				UpdaterFrequency:     60,
			},
			expected: Config{
				BlizzardClientID:     "existing-client-id", // should not be overridden
				BlizzardClientSecret: "merge-client-secret",
				DiscordToken:         "merge-discord-token",
				DiscordChannelID:     "merge-channel-id",
				DatabaseLocation:     "merge.db",
				UpdaterFrequency:     30, // should not be overridden
			},
		},
		{
			name: "merge partial fields",
			base: Config{
				BlizzardClientID: "existing-id",
				DiscordToken:     "existing-token",
			},
			merge: Config{
				BlizzardClientID:     "merge-id",
				BlizzardClientSecret: "merge-secret",
				DiscordToken:         "merge-token",
				DiscordChannelID:     "merge-channel",
				DatabaseLocation:     "merge.db",
				UpdaterFrequency:     45,
			},
			expected: Config{
				BlizzardClientID:     "existing-id",    // should not be overridden
				BlizzardClientSecret: "merge-secret",   // should be merged
				DiscordToken:         "existing-token", // should not be overridden
				DiscordChannelID:     "merge-channel",  // should be merged
				DatabaseLocation:     "merge.db",       // should be merged
				UpdaterFrequency:     45,               // should be merged (was 0)
			},
		},
		{
			name: "no merge when all fields populated",
			base: Config{
				BlizzardClientID:     "base-client-id",
				BlizzardClientSecret: "base-client-secret",
				DiscordToken:         "base-discord-token",
				DiscordChannelID:     "base-channel-id",
				DatabaseLocation:     "base.db",
				UpdaterFrequency:     15,
			},
			merge: Config{
				BlizzardClientID:     "merge-client-id",
				BlizzardClientSecret: "merge-client-secret",
				DiscordToken:         "merge-discord-token",
				DiscordChannelID:     "merge-channel-id",
				DatabaseLocation:     "merge.db",
				UpdaterFrequency:     60,
			},
			expected: Config{
				BlizzardClientID:     "base-client-id",
				BlizzardClientSecret: "base-client-secret",
				DiscordToken:         "base-discord-token",
				DiscordChannelID:     "base-channel-id",
				DatabaseLocation:     "base.db",
				UpdaterFrequency:     15,
			},
		},
		{
			name: "merge with default config",
			base: Config{
				BlizzardClientID: "test-id",
				DiscordToken:     "test-token",
			},
			merge: defaultConfig,
			expected: Config{
				BlizzardClientID: "test-id",
				DiscordToken:     "test-token",
				DatabaseLocation: "mythicplusdiscordbot.sqlite",
				UpdaterFrequency: 30,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of base to avoid modifying the test case
			result := tt.base
			result.merge(tt.merge)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_MergeLogLevelZeroHandling(t *testing.T) {
	// Special test to verify that LogLevel 0 is not overridden
	// since the comment indicates 0 is a valid value
	base := Config{
		LogLevel: 0, // explicitly set to 0
	}
	merge := Config{
		LogLevel: 2,
	}

	base.merge(merge)

	assert.Equal(t, 0, base.LogLevel, "LogLevel 0 should not be overridden as it's a valid value")
}
