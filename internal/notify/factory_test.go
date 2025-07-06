package notify

import (
	"testing"

	"github-stars-notify/internal/config"
)

func TestCreateNotifiers(t *testing.T) {
	// Test with Discord enabled
	cfg := &config.Config{
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

	notifiers, err := CreateNotifiers(cfg)
	if err != nil {
		t.Fatalf("Failed to create notifiers: %v", err)
	}

	if len(notifiers) != 1 {
		t.Errorf("Expected 1 notifier, got %d", len(notifiers))
	}

	if notifiers[0].GetProviderName() != "discord" {
		t.Errorf("Expected discord provider, got %s", notifiers[0].GetProviderName())
	}

	// Test with both enabled
	cfg.Notifications.Slack.Enabled = true
	notifiers, err = CreateNotifiers(cfg)
	if err != nil {
		t.Fatalf("Failed to create notifiers: %v", err)
	}

	if len(notifiers) != 2 {
		t.Errorf("Expected 2 notifiers, got %d", len(notifiers))
	}

	// Test with none enabled
	cfg.Notifications.Discord.Enabled = false
	cfg.Notifications.Slack.Enabled = false
	notifiers, err = CreateNotifiers(cfg)
	if err != nil {
		t.Fatalf("Failed to create notifiers: %v", err)
	}

	if len(notifiers) != 0 {
		t.Errorf("Expected 0 notifiers, got %d", len(notifiers))
	}
}

func TestCreateNotifier(t *testing.T) {
	// Test Discord notifier
	notifier, err := CreateNotifier("discord", "https://discord.com/api/webhooks/123/abc")
	if err != nil {
		t.Fatalf("Failed to create discord notifier: %v", err)
	}
	if notifier.GetProviderName() != "discord" {
		t.Errorf("Expected discord provider, got %s", notifier.GetProviderName())
	}

	// Test Slack notifier
	notifier, err = CreateNotifier("slack", "https://hooks.slack.com/services/123/abc/def", "#github-stars")
	if err != nil {
		t.Fatalf("Failed to create slack notifier: %v", err)
	}
	if notifier.GetProviderName() != "slack" {
		t.Errorf("Expected slack provider, got %s", notifier.GetProviderName())
	}

	// Test invalid notifier type
	_, err = CreateNotifier("invalid", "https://example.com")
	if err == nil {
		t.Error("Expected error for invalid notifier type")
	}
}
