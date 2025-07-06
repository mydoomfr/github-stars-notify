package notify

import (
	"fmt"
	"time"

	"github-stars-notify/internal/config"
	"github-stars-notify/internal/logger"
)

// NotifierConfig holds configuration for creating notifiers
type NotifierConfig struct {
	MaxRetries      int
	RetryBackoff    time.Duration
	RateLimitWindow time.Duration
	Timeout         time.Duration
}

// DefaultNotifierConfig returns default configuration for notifiers
func DefaultNotifierConfig() NotifierConfig {
	return NotifierConfig{
		MaxRetries:      3,
		RetryBackoff:    time.Second * 2,
		RateLimitWindow: time.Minute * 1, // 1 notification per minute per provider
		Timeout:         time.Second * 30,
	}
}

// CreateNotifiers creates all enabled notifiers based on configuration
func CreateNotifiers(cfg *config.Config) ([]Notifier, error) {
	return CreateNotifiersWithLogger(cfg, logger.Default())
}

// CreateNotifiersWithLogger creates all enabled notifiers with custom logger
func CreateNotifiersWithLogger(cfg *config.Config, log *logger.Logger) ([]Notifier, error) {
	return CreateNotifiersWithConfig(cfg, DefaultNotifierConfig(), log)
}

// CreateNotifiersWithConfig creates all enabled notifiers with custom configuration
func CreateNotifiersWithConfig(cfg *config.Config, notifierCfg NotifierConfig, log *logger.Logger) ([]Notifier, error) {
	var notifiers []Notifier

	// Create Discord notifier if enabled
	if cfg.Notifications.Discord.Enabled {
		baseNotifier := NewDiscordNotifierWithTimeout(cfg.Notifications.Discord.WebhookURL, notifierCfg.Timeout)

		// Wrap with rate limiting
		rateLimitedNotifier := NewRateLimitedNotifier(baseNotifier, notifierCfg.RateLimitWindow, log)

		// Wrap with retry logic
		retryableNotifier := NewRetryableNotifier(rateLimitedNotifier, notifierCfg.MaxRetries, notifierCfg.RetryBackoff, log)

		notifiers = append(notifiers, retryableNotifier)
	}

	// Create Slack notifier if enabled
	if cfg.Notifications.Slack.Enabled {
		baseNotifier := NewSlackNotifierWithTimeout(cfg.Notifications.Slack.WebhookURL, cfg.Notifications.Slack.Channel, notifierCfg.Timeout)

		// Wrap with rate limiting
		rateLimitedNotifier := NewRateLimitedNotifier(baseNotifier, notifierCfg.RateLimitWindow, log)

		// Wrap with retry logic
		retryableNotifier := NewRetryableNotifier(rateLimitedNotifier, notifierCfg.MaxRetries, notifierCfg.RetryBackoff, log)

		notifiers = append(notifiers, retryableNotifier)
	}

	return notifiers, nil
}

// CreateNotifier creates a single notifier by type (for testing/specific use)
func CreateNotifier(notifierType string, webhookURL string, options ...string) (Notifier, error) {
	return CreateNotifierWithConfig(notifierType, webhookURL, DefaultNotifierConfig(), logger.Default(), options...)
}

// CreateNotifierWithConfig creates a single notifier with custom configuration
func CreateNotifierWithConfig(notifierType string, webhookURL string, cfg NotifierConfig, log *logger.Logger, options ...string) (Notifier, error) {
	var baseNotifier Notifier

	switch notifierType {
	case ProviderDiscord:
		baseNotifier = NewDiscordNotifierWithTimeout(webhookURL, cfg.Timeout)
	case ProviderSlack:
		channel := ""
		if len(options) > 0 {
			channel = options[0]
		}
		baseNotifier = NewSlackNotifierWithTimeout(webhookURL, channel, cfg.Timeout)
	default:
		return nil, fmt.Errorf("unsupported notifier type: %s", notifierType)
	}

	// Wrap with rate limiting
	rateLimitedNotifier := NewRateLimitedNotifier(baseNotifier, cfg.RateLimitWindow, log)

	// Wrap with retry logic
	retryableNotifier := NewRetryableNotifier(rateLimitedNotifier, cfg.MaxRetries, cfg.RetryBackoff, log)

	return retryableNotifier, nil
}

// CreateBasicNotifier creates a basic notifier without enhancements (for testing)
func CreateBasicNotifier(notifierType string, webhookURL string, options ...string) (Notifier, error) {
	switch notifierType {
	case ProviderDiscord:
		return NewDiscordNotifier(webhookURL), nil
	case ProviderSlack:
		channel := ""
		if len(options) > 0 {
			channel = options[0]
		}
		return NewSlackNotifier(webhookURL, channel), nil
	default:
		return nil, fmt.Errorf("unsupported notifier type: %s", notifierType)
	}
}
