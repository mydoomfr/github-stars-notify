package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigBasic(t *testing.T) {
	// Test valid config loading
	configYAML := `
repositories:
  - owner: "facebook"
    repo: "react"
settings:
  check_interval_minutes: 15
notifications:
  discord:
    webhook_url: "https://discord.com/api/webhooks/123/abc"
    enabled: true
  slack:
    webhook_url: "https://hooks.slack.com/services/123/abc/def"
    channel: "#github-stars"
    enabled: false
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test config values
	if len(config.Repositories) != 1 {
		t.Errorf("Expected 1 repository, got %d", len(config.Repositories))
	}
	if config.Repositories[0].Owner != "facebook" {
		t.Errorf("Expected owner 'facebook', got '%s'", config.Repositories[0].Owner)
	}
	if config.Settings.CheckIntervalMinutes != 15 {
		t.Errorf("Expected 15 minutes, got %d", config.Settings.CheckIntervalMinutes)
	}
	if !config.Notifications.Discord.Enabled {
		t.Error("Expected Discord enabled")
	}

	// Test defaults
	defaultConfigYAML := `
repositories:
  - owner: "test"
    repo: "test"
notifications:
  discord:
    enabled: false
`
	err = os.WriteFile(configPath, []byte(defaultConfigYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write default config: %v", err)
	}

	defaultConfig, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	if defaultConfig.Settings.CheckIntervalMinutes != 60 {
		t.Errorf("Expected default 60 minutes, got %d", defaultConfig.Settings.CheckIntervalMinutes)
	}

	// Test check interval
	duration := defaultConfig.GetCheckInterval()
	if duration != 60*time.Minute {
		t.Errorf("Expected 60 minute duration, got %v", duration)
	}
}
