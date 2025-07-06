package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Repositories  []Repository  `yaml:"repositories"`
	Settings      Settings      `yaml:"settings"`
	GitHub        GitHubConfig  `yaml:"github"`
	Notifications Notifications `yaml:"notifications"`
	Server        ServerConfig  `yaml:"server"`
	Storage       StorageConfig `yaml:"storage"`
	Logging       LoggingConfig `yaml:"logging"`
}

// Repository represents a GitHub repository to monitor
type Repository struct {
	Owner string `yaml:"owner"`
	Repo  string `yaml:"repo"`
}

// Settings contains application settings
type Settings struct {
	CheckIntervalMinutes int `yaml:"check_interval_minutes"`
}

// GitHubConfig contains GitHub API configuration
type GitHubConfig struct {
	Token   string `yaml:"token"`
	Timeout int    `yaml:"timeout_seconds"` // HTTP timeout in seconds
}

// Notifications contains notification configuration
type Notifications struct {
	Discord DiscordConfig `yaml:"discord"`
	Slack   SlackConfig   `yaml:"slack"`
}

// DiscordConfig contains Discord webhook configuration
type DiscordConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	Enabled    bool   `yaml:"enabled"`
}

// SlackConfig contains Slack webhook configuration
type SlackConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel,omitempty"`
	Enabled    bool   `yaml:"enabled"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port         int    `yaml:"port"`
	ReadTimeout  int    `yaml:"read_timeout_seconds"`
	WriteTimeout int    `yaml:"write_timeout_seconds"`
	Host         string `yaml:"host"`
}

// StorageConfig contains storage configuration
type StorageConfig struct {
	Type string `yaml:"type"` // "file" for now, extensible for future storage types
	Path string `yaml:"path"` // Directory path for file storage
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`  // "debug", "info", "warn", "error"
	Format string `yaml:"format"` // "json" or "text"
}

// Load loads and validates configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	cfg.applyEnvOverrides()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults after validation
	cfg.setDefaults()

	return &cfg, nil
}

// applyEnvOverrides applies environment variable overrides
func (c *Config) applyEnvOverrides() {
	// GitHub configuration
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		c.GitHub.Token = token
	}

	// Discord configuration
	if webhookURL := os.Getenv("DISCORD_WEBHOOK_URL"); webhookURL != "" {
		c.Notifications.Discord.WebhookURL = webhookURL
	}
	if enabled := os.Getenv("DISCORD_ENABLED"); enabled != "" {
		c.Notifications.Discord.Enabled = enabled == "true"
	}

	// Slack configuration
	if webhookURL := os.Getenv("SLACK_WEBHOOK_URL"); webhookURL != "" {
		c.Notifications.Slack.WebhookURL = webhookURL
	}
	if channel := os.Getenv("SLACK_CHANNEL"); channel != "" {
		c.Notifications.Slack.Channel = channel
	}
	if enabled := os.Getenv("SLACK_ENABLED"); enabled != "" {
		c.Notifications.Slack.Enabled = enabled == "true"
	}

	// Server configuration
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Server.Port = p
		}
	}
	if host := os.Getenv("SERVER_HOST"); host != "" {
		c.Server.Host = host
	}

	// Storage configuration
	if path := os.Getenv("STORAGE_PATH"); path != "" {
		c.Storage.Path = path
	}

	// Logging configuration
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		c.Logging.Level = level
	}
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		c.Logging.Format = format
	}

	// Settings
	if interval := os.Getenv("CHECK_INTERVAL_MINUTES"); interval != "" {
		if i, err := strconv.Atoi(interval); err == nil {
			c.Settings.CheckIntervalMinutes = i
		}
	}
}

// validate validates the configuration
func (c *Config) validate() error {
	if len(c.Repositories) == 0 {
		return fmt.Errorf("at least one repository must be configured")
	}

	for i, repo := range c.Repositories {
		if repo.Owner == "" {
			return fmt.Errorf("repository[%d]: owner is required", i)
		}
		if repo.Repo == "" {
			return fmt.Errorf("repository[%d]: repo is required", i)
		}
	}

	if c.Notifications.Discord.Enabled && c.Notifications.Discord.WebhookURL == "" {
		return fmt.Errorf("discord webhook URL is required when discord notifications are enabled")
	}

	if c.Notifications.Slack.Enabled && c.Notifications.Slack.WebhookURL == "" {
		return fmt.Errorf("slack webhook URL is required when slack notifications are enabled")
	}

	// Validate logging level
	if c.Logging.Level != "" {
		switch c.Logging.Level {
		case "debug", "info", "warn", "error":
			// Valid levels
		default:
			return fmt.Errorf("invalid log level: %s", c.Logging.Level)
		}
	}

	// Validate logging format
	if c.Logging.Format != "" {
		switch c.Logging.Format {
		case "json", "text":
			// Valid formats
		default:
			return fmt.Errorf("invalid log format: %s", c.Logging.Format)
		}
	}

	return nil
}

// setDefaults sets default values for configuration
func (c *Config) setDefaults() {
	if c.Settings.CheckIntervalMinutes == 0 {
		c.Settings.CheckIntervalMinutes = 60
	}
	if c.GitHub.Timeout == 0 {
		c.GitHub.Timeout = 30
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.Host == "" {
		c.Server.Host = "localhost"
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30
	}
	if c.Storage.Type == "" {
		c.Storage.Type = "file"
	}
	if c.Storage.Path == "" {
		c.Storage.Path = "./data"
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "text"
	}
}

// GetCheckInterval returns the check interval as a time.Duration
func (c *Config) GetCheckInterval() time.Duration {
	return time.Duration(c.Settings.CheckIntervalMinutes) * time.Minute
}

// GetGitHubTimeout returns the GitHub API timeout as a time.Duration
func (c *Config) GetGitHubTimeout() time.Duration {
	return time.Duration(c.GitHub.Timeout) * time.Second
}

// GetServerAddress returns the server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// GetLogLevel returns the log level as slog.Level
func (c *Config) GetLogLevel() slog.Level {
	switch c.Logging.Level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
