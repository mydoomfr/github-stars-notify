package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github-stars-notify/internal/github"
)

func TestDiscordNotificationBasic(t *testing.T) {
	// Create a test server that responds to webhook calls
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create Discord notifier with test webhook URL
	notifier := NewDiscordNotifier(server.URL)

	// Test provider name
	if notifier.GetProviderName() != "discord" {
		t.Errorf("Expected provider name 'discord', got %s", notifier.GetProviderName())
	}

	// Test timeout configuration
	notifierWithTimeout := NewDiscordNotifierWithTimeout(server.URL, time.Second*5)
	if notifierWithTimeout.httpClient.Timeout != time.Second*5 {
		t.Errorf("Expected timeout 5s, got %v", notifierWithTimeout.httpClient.Timeout)
	}
}

func TestDiscordNotificationSend(t *testing.T) {
	// Create a test server that responds to webhook calls
	received := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create Discord notifier with test webhook URL
	notifier := NewDiscordNotifier(server.URL)
	ctx := context.Background()

	// Test sending notification with stargazers
	stargazers := []github.Stargazer{
		{
			Login:     "testuser",
			ID:        123,
			StarredAt: time.Now(),
		},
	}

	err := notifier.NotifyNewStars(ctx, "facebook", "react", stargazers)
	if err != nil {
		t.Errorf("NotifyNewStars failed: %v", err)
	}

	if !received {
		t.Error("Expected webhook to be called")
	}

	// Test connection
	err = notifier.TestConnection(ctx)
	if err != nil {
		t.Errorf("TestConnection failed: %v", err)
	}

	// Test with empty stargazers (should not send)
	err = notifier.NotifyNewStars(ctx, "facebook", "react", []github.Stargazer{})
	if err != nil {
		t.Errorf("NotifyNewStars with empty stargazers failed: %v", err)
	}
}

func TestDiscordNotificationError(t *testing.T) {
	// Create a test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	// Create Discord notifier with test webhook URL
	notifier := NewDiscordNotifier(server.URL)
	ctx := context.Background()

	// Test sending notification should fail
	stargazers := []github.Stargazer{
		{
			Login: "testuser",
			ID:    123,
		},
	}

	err := notifier.NotifyNewStars(ctx, "facebook", "react", stargazers)
	if err == nil {
		t.Error("Expected NotifyNewStars to fail with bad webhook response")
	}
}
