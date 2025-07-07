package service

import (
	"context"
	"testing"
	"time"

	"github-stars-notify/internal/config"
)

func TestServiceBasic(t *testing.T) {
	cfg := &config.Config{
		Repositories: []config.Repository{
			{Owner: "facebook", Repo: "react"},
		},
		Settings: config.Settings{
			CheckIntervalMinutes: 10,
		},
		GitHub: config.GitHubConfig{
			Token:   "test-token",
			Timeout: 30,
		},
		Server: config.ServerConfig{
			Port:         9090,
			Host:         "localhost",
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		Storage: config.StorageConfig{
			Type: "file",
			Path: "./test_data",
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		Notifications: config.Notifications{
			Discord: config.DiscordConfig{
				WebhookURL: "https://discord.com/api/webhooks/123/abc",
				Enabled:    true,
			},
			Slack: config.SlackConfig{
				WebhookURL: "https://hooks.slack.com/services/123/abc/def",
				Channel:    "#github-stars",
				Enabled:    false,
			},
		},
	}

	service, err := NewForTest(cfg)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Test service creation
	loadedCfg := service.configReloader.GetConfig()
	if loadedCfg == nil {
		t.Error("Config not loaded correctly")
	}
	if service.github == nil {
		t.Error("GitHub client not initialized")
	}
	if service.storage == nil {
		t.Error("Storage not initialized")
	}
	if service.notifiers == nil {
		t.Error("Notifiers not initialized")
	}
	if service.logger == nil {
		t.Error("Logger not initialized")
	}

	// Test status
	status := service.GetStatus()
	if status["running"] != false {
		t.Error("Expected running=false initially")
	}
	if status["repositories"] != 1 {
		t.Error("Expected 1 repository")
	}
	if status["notifiers"] != 1 { // Only Discord is enabled
		t.Errorf("Expected 1 notifier, got %v", status["notifiers"])
	}

	// Test stop functionality
	service.running = true

	// Create a context with timeout for testing
	_, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Set up the cancel function
	service.cancel = cancel

	// Test that stop works
	service.Stop()

	if service.running {
		t.Error("Service should not be running after stop")
	}
}
