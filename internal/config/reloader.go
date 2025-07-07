package config

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github-stars-notify/internal/logger"

	"github.com/fsnotify/fsnotify"
)

// ReloadCallback is called when configuration is successfully reloaded
type ReloadCallback func(oldConfig, newConfig *Config) error

// Reloader handles hot reloading of configuration files
type Reloader struct {
	configPath string
	logger     *logger.Logger
	watcher    *fsnotify.Watcher
	callbacks  []ReloadCallback
	mu         sync.RWMutex
	config     *Config
	running    bool
	cancel     context.CancelFunc
}

// NewReloader creates a new configuration reloader
func NewReloader(configPath string, logger *logger.Logger) (*Reloader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Load initial config
	config, err := Load(configPath)
	if err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}

	return &Reloader{
		configPath: configPath,
		logger:     logger.WithComponent("config-reloader"),
		watcher:    watcher,
		config:     config,
		callbacks:  make([]ReloadCallback, 0),
	}, nil
}

// GetConfig returns the current configuration (thread-safe)
func (r *Reloader) GetConfig() *Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

// AddCallback adds a callback to be called when config is reloaded
func (r *Reloader) AddCallback(callback ReloadCallback) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.callbacks = append(r.callbacks, callback)
}

// Start starts the configuration file watcher
func (r *Reloader) Start(ctx context.Context) error {
	if r.running {
		return fmt.Errorf("reloader is already running")
	}

	// Watch the config file
	if err := r.watcher.Add(r.configPath); err != nil {
		return fmt.Errorf("failed to watch config file: %w", err)
	}

	// Also watch the directory for atomic writes (like editors do)
	configDir := filepath.Dir(r.configPath)
	if err := r.watcher.Add(configDir); err != nil {
		r.logger.Warn("failed to watch config directory", "dir", configDir, "error", err)
	}

	watchCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.running = true

	go r.watchLoop(watchCtx)

	r.logger.Info("configuration reloader started", "config_path", r.configPath)
	return nil
}

// Stop stops the configuration file watcher
func (r *Reloader) Stop() {
	if !r.running {
		return
	}

	r.running = false
	if r.cancel != nil {
		r.cancel()
	}

	if r.watcher != nil {
		r.watcher.Close()
	}

	r.logger.Info("configuration reloader stopped")
}

// watchLoop runs the file watching loop
func (r *Reloader) watchLoop(ctx context.Context) {
	debounceTimer := time.NewTimer(0)
	if !debounceTimer.Stop() {
		<-debounceTimer.C
	}

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-r.watcher.Events:
			if !ok {
				return
			}

			// Only process events for our config file
			if !r.isConfigFile(event.Name) {
				continue
			}

			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				r.logger.Debug("config file changed", "event", event)

				// Debounce multiple rapid changes
				debounceTimer.Reset(200 * time.Millisecond)
			}

		case err, ok := <-r.watcher.Errors:
			if !ok {
				return
			}
			r.logger.Error("config file watcher error", "error", err)

		case <-debounceTimer.C:
			r.handleConfigChange()
		}
	}
}

// isConfigFile checks if the event is for our config file
func (r *Reloader) isConfigFile(eventPath string) bool {
	// Handle absolute paths and relative paths
	absConfigPath, _ := filepath.Abs(r.configPath)
	absEventPath, _ := filepath.Abs(eventPath)

	return absEventPath == absConfigPath ||
		filepath.Base(eventPath) == filepath.Base(r.configPath)
}

// handleConfigChange processes a configuration file change
func (r *Reloader) handleConfigChange() {
	r.logger.Info("reloading configuration file")

	// Load new config
	newConfig, err := Load(r.configPath)
	if err != nil {
		r.logger.Error("failed to load new configuration", "error", err)
		return
	}

	// Validate new config
	if err := r.validateConfig(newConfig); err != nil {
		r.logger.Error("new configuration is invalid", "error", err)
		return
	}

	// Get current config for comparison and make a copy of callbacks
	r.mu.RLock()
	oldConfig := r.config
	callbacks := make([]ReloadCallback, len(r.callbacks))
	copy(callbacks, r.callbacks)
	r.mu.RUnlock()

	r.logger.Info("loaded new configuration",
		"old_check_interval", oldConfig.GetCheckInterval(),
		"new_check_interval", newConfig.GetCheckInterval(),
		"old_repo_count", len(oldConfig.Repositories),
		"new_repo_count", len(newConfig.Repositories))

	// Check what changed
	changes := r.detectChanges(oldConfig, newConfig)
	if len(changes) == 0 {
		r.logger.Debug("no meaningful configuration changes detected")
		return
	}

	r.logger.Info("configuration changes detected", "changes", changes)

	// Execute callbacks (using the copy to avoid race conditions)
	for i, callback := range callbacks {
		r.logger.Debug("executing config reload callback", "callback_index", i)
		if err := callback(oldConfig, newConfig); err != nil {
			r.logger.Error("config reload callback failed", "error", err, "callback_index", i)
			return
		}
	}

	// Update current config
	r.mu.Lock()
	r.config = newConfig
	r.mu.Unlock()

	r.logger.Info("configuration reloaded successfully",
		"final_check_interval", r.config.GetCheckInterval(),
		"final_repo_count", len(r.config.Repositories))
}

// validateConfig validates the new configuration
func (r *Reloader) validateConfig(config *Config) error {
	// Basic validation
	if len(config.Repositories) == 0 {
		return fmt.Errorf("no repositories configured")
	}

	if config.GitHub.Token == "" {
		return fmt.Errorf("github token is required")
	}

	if config.GetCheckInterval() < time.Minute {
		return fmt.Errorf("check interval must be at least 1 minute")
	}

	return nil
}

// detectChanges detects what changed between old and new config
func (r *Reloader) detectChanges(oldConfig, newConfig *Config) []string {
	var changes []string

	// Repository changes
	if !equalRepositories(oldConfig.Repositories, newConfig.Repositories) {
		changes = append(changes, "repositories")
	}

	// Check interval changes
	if oldConfig.GetCheckInterval() != newConfig.GetCheckInterval() {
		changes = append(changes, "check_interval")
	}

	// GitHub token changes
	if oldConfig.GitHub.Token != newConfig.GitHub.Token {
		changes = append(changes, "github_token")
	}

	// GitHub timeout changes
	if oldConfig.GetGitHubTimeout() != newConfig.GetGitHubTimeout() {
		changes = append(changes, "github_timeout")
	}

	// Notification changes
	if !equalNotifications(oldConfig.Notifications, newConfig.Notifications) {
		changes = append(changes, "notifications")
	}

	// Log level changes
	if oldConfig.GetLogLevel() != newConfig.GetLogLevel() {
		changes = append(changes, "log_level")
	}

	return changes
}

// equalRepositories compares two repository slices
func equalRepositories(a, b []Repository) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].Owner != b[i].Owner || a[i].Repo != b[i].Repo {
			return false
		}
	}

	return true
}

// equalNotifications compares two notification configurations
func equalNotifications(a, b Notifications) bool {
	return a.Discord.Enabled == b.Discord.Enabled &&
		a.Discord.WebhookURL == b.Discord.WebhookURL &&
		a.Slack.Enabled == b.Slack.Enabled &&
		a.Slack.WebhookURL == b.Slack.WebhookURL &&
		a.Slack.Channel == b.Slack.Channel
}
