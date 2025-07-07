package config

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github-stars-notify/internal/logger"
)

func TestReloader(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	initialConfig := `
repositories:
  - owner: "test"
    repo: "repo1"

github:
  token: "test-token"

notifications:
  discord:
    enabled: true
    webhook_url: "https://discord.com/api/webhooks/test"
`

	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create logger for testing
	log := logger.NewLogger(logger.Config{
		Level:   slog.LevelDebug,
		Format:  "text",
		Service: "test",
	})

	// Create reloader
	reloader, err := NewReloader(configPath, log)
	if err != nil {
		t.Fatalf("Failed to create reloader: %v", err)
	}
	defer reloader.Stop()

	// Verify initial config
	config := reloader.GetConfig()
	if len(config.Repositories) != 1 {
		t.Errorf("Expected 1 repository, got %d", len(config.Repositories))
	}
	if config.Repositories[0].Owner != "test" {
		t.Errorf("Expected owner 'test', got '%s'", config.Repositories[0].Owner)
	}

	// Set up callback to track config changes using atomic operations
	var reloadCount int64
	reloader.AddCallback(func(oldConfig, newConfig *Config) error {
		count := atomic.AddInt64(&reloadCount, 1)
		t.Logf("Config reloaded (count: %d)", count)
		return nil
	})

	// Start the reloader
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := reloader.Start(ctx); err != nil {
		t.Fatalf("Failed to start reloader: %v", err)
	}

	// Wait a bit for the watcher to be ready
	time.Sleep(100 * time.Millisecond)

	// Update the config file
	updatedConfig := `
repositories:
  - owner: "test"
    repo: "repo1"
  - owner: "test"
    repo: "repo2"

github:
  token: "updated-token"

notifications:
  discord:
    enabled: true
    webhook_url: "https://discord.com/api/webhooks/updated"
`

	if err := os.WriteFile(configPath, []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("Failed to update test config: %v", err)
	}

	// Wait for the reload to happen
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && atomic.LoadInt64(&reloadCount) == 0 {
		time.Sleep(50 * time.Millisecond)
	}

	if atomic.LoadInt64(&reloadCount) == 0 {
		t.Error("Config reload callback was not called")
	}

	// Verify updated config
	updatedConf := reloader.GetConfig()
	if len(updatedConf.Repositories) != 2 {
		t.Errorf("Expected 2 repositories after reload, got %d", len(updatedConf.Repositories))
	}
	if updatedConf.GitHub.Token != "updated-token" {
		t.Errorf("Expected token 'updated-token', got '%s'", updatedConf.GitHub.Token)
	}
}

func TestReloaderInvalidConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	initialConfig := `
repositories:
  - owner: "test"
    repo: "repo1"

github:
  token: "test-token"
`

	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create logger for testing
	log := logger.NewLogger(logger.Config{
		Level:   slog.LevelDebug,
		Format:  "text",
		Service: "test",
	})

	// Create reloader
	reloader, err := NewReloader(configPath, log)
	if err != nil {
		t.Fatalf("Failed to create reloader: %v", err)
	}
	defer reloader.Stop()

	// Set up callback to track config changes using atomic operations
	var reloadCount int64
	reloader.AddCallback(func(oldConfig, newConfig *Config) error {
		atomic.AddInt64(&reloadCount, 1)
		return nil
	})

	// Start the reloader
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := reloader.Start(ctx); err != nil {
		t.Fatalf("Failed to start reloader: %v", err)
	}

	// Wait a bit for the watcher to be ready
	time.Sleep(100 * time.Millisecond)

	// Write invalid config (missing github token)
	invalidConfig := `
repositories:
  - owner: "test"
    repo: "repo1"

github:
  token: ""
`

	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	// Wait a bit and verify no reload happened
	time.Sleep(500 * time.Millisecond)

	if atomic.LoadInt64(&reloadCount) != 0 {
		t.Error("Config reload should not have happened for invalid config")
	}

	// Verify original config is still active
	config := reloader.GetConfig()
	if config.GitHub.Token != "test-token" {
		t.Errorf("Original config should still be active, got token: %s", config.GitHub.Token)
	}
}

func TestDetectChanges(t *testing.T) {
	log := logger.NewLogger(logger.Config{Level: slog.LevelDebug, Format: "text", Service: "test"})

	// Create a minimal reloader for testing
	reloader := &Reloader{logger: log}

	oldConfig := &Config{
		Repositories: []Repository{
			{Owner: "test", Repo: "repo1"},
		},
		GitHub: GitHubConfig{
			Token: "old-token",
		},
		Notifications: Notifications{
			Discord: DiscordConfig{
				Enabled:    true,
				WebhookURL: "old-webhook",
			},
		},
		Settings: Settings{
			CheckIntervalMinutes: 60,
		},
	}

	newConfig := &Config{
		Repositories: []Repository{
			{Owner: "test", Repo: "repo1"},
			{Owner: "test", Repo: "repo2"},
		},
		GitHub: GitHubConfig{
			Token: "new-token",
		},
		Notifications: Notifications{
			Discord: DiscordConfig{
				Enabled:    true,
				WebhookURL: "new-webhook",
			},
		},
		Settings: Settings{
			CheckIntervalMinutes: 30,
		},
	}

	changes := reloader.detectChanges(oldConfig, newConfig)

	expectedChanges := []string{"repositories", "check_interval", "github_token", "notifications"}
	if len(changes) != len(expectedChanges) {
		t.Errorf("Expected %d changes, got %d: %v", len(expectedChanges), len(changes), changes)
	}

	for _, expected := range expectedChanges {
		found := false
		for _, change := range changes {
			if change == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected change '%s' not found in: %v", expected, changes)
		}
	}
}
