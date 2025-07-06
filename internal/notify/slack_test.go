package notify

import (
	"context"
	"testing"
	"time"

	"github-stars-notify/internal/github"
)

func TestSlackNotifier(t *testing.T) {
	// Create notifier
	notifier := NewSlackNotifier("https://hooks.slack.com/test", "#test")

	// Test provider name
	if notifier.GetProviderName() != "slack" {
		t.Errorf("Expected provider name 'slack', got %s", notifier.GetProviderName())
	}

	// Test timeout configuration
	notifierWithTimeout := NewSlackNotifierWithTimeout("https://hooks.slack.com/test", "#test", time.Second*5)
	if notifierWithTimeout.httpClient.Timeout != time.Second*5 {
		t.Errorf("Expected timeout 5s, got %v", notifierWithTimeout.httpClient.Timeout)
	}

	// Test message creation
	stargazers := []github.Stargazer{
		{Login: "testuser", ID: 123},
	}
	message := notifier.createMessage("facebook", "react", stargazers)

	if message.Username != "GitHub Stars Notify" {
		t.Errorf("Expected username 'GitHub Stars Notify', got %s", message.Username)
	}

	if message.IconEmoji != ":star:" {
		t.Errorf("Expected icon emoji ':star:', got %s", message.IconEmoji)
	}

	if message.Channel != "#test" {
		t.Errorf("Expected channel '#test', got %s", message.Channel)
	}

	// Test notification with context (will fail without real webhook, but tests signature)
	ctx := context.Background()
	err := notifier.NotifyNewStars(ctx, "facebook", "react", stargazers)
	if err == nil {
		t.Log("Note: Notification succeeded (unexpected with fake webhook URL)")
	}

	// Test connection (will fail without real webhook, but tests signature)
	err = notifier.TestConnection(ctx)
	if err == nil {
		t.Log("Note: Connection test succeeded (unexpected with fake webhook URL)")
	}

	// Test with empty stargazers (should not send)
	err = notifier.NotifyNewStars(ctx, "facebook", "react", []github.Stargazer{})
	if err != nil {
		t.Errorf("NotifyNewStars with empty stargazers failed: %v", err)
	}
}
