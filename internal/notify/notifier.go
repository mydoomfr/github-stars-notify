package notify

import (
	"context"
	"time"

	"github-stars-notify/internal/github"
	"github-stars-notify/internal/logger"
)

// Notifier defines the interface for notification providers
type Notifier interface {
	// NotifyNewStars sends a notification about new stars for a repository
	NotifyNewStars(ctx context.Context, owner, repo string, newStargazers []github.Stargazer) error

	// TestConnection tests the notification provider connection
	TestConnection(ctx context.Context) error

	// GetProviderName returns the name of the notification provider
	GetProviderName() string
}

// RetryableNotifier wraps a notifier with retry logic
type RetryableNotifier struct {
	notifier   Notifier
	maxRetries int
	backoff    time.Duration
	logger     *logger.Logger
}

// NewRetryableNotifier creates a new retryable notifier
func NewRetryableNotifier(notifier Notifier, maxRetries int, backoff time.Duration, logger *logger.Logger) *RetryableNotifier {
	return &RetryableNotifier{
		notifier:   notifier,
		maxRetries: maxRetries,
		backoff:    backoff,
		logger:     logger.WithComponent("retryable_notifier"),
	}
}

// NotifyNewStars sends a notification with retry logic
func (rn *RetryableNotifier) NotifyNewStars(ctx context.Context, owner, repo string, newStargazers []github.Stargazer) error {
	var lastErr error
	provider := rn.notifier.GetProviderName()

	for i := 0; i <= rn.maxRetries; i++ {
		start := time.Now()

		err := rn.notifier.NotifyNewStars(ctx, owner, repo, newStargazers)
		if err == nil {
			rn.logger.Info("notification sent successfully",
				"provider", provider,
				"repo", owner+"/"+repo,
				"stargazers", len(newStargazers),
				"attempt", i+1,
				"duration", time.Since(start))
			return nil
		}

		lastErr = err

		rn.logger.Warn("notification failed",
			"provider", provider,
			"repo", owner+"/"+repo,
			"attempt", i+1,
			"error", err,
			"duration", time.Since(start))

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Wait before retrying (except on last attempt)
		if i < rn.maxRetries {
			backoffDuration := rn.backoff * time.Duration(i+1)
			rn.logger.Debug("waiting before retry",
				"provider", provider,
				"backoff", backoffDuration,
				"next_attempt", i+2)

			select {
			case <-time.After(backoffDuration):
				// Continue to next retry
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	rn.logger.Error("notification failed after all retries",
		"provider", provider,
		"repo", owner+"/"+repo,
		"max_retries", rn.maxRetries,
		"error", lastErr)

	return lastErr
}

// TestConnection tests the connection with retry logic
func (rn *RetryableNotifier) TestConnection(ctx context.Context) error {
	var lastErr error
	provider := rn.notifier.GetProviderName()

	for i := 0; i <= rn.maxRetries; i++ {
		start := time.Now()

		err := rn.notifier.TestConnection(ctx)
		if err == nil {
			rn.logger.Info("connection test successful",
				"provider", provider,
				"attempt", i+1,
				"duration", time.Since(start))
			return nil
		}

		lastErr = err

		rn.logger.Warn("connection test failed",
			"provider", provider,
			"attempt", i+1,
			"error", err,
			"duration", time.Since(start))

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Wait before retrying (except on last attempt)
		if i < rn.maxRetries {
			backoffDuration := rn.backoff * time.Duration(i+1)

			select {
			case <-time.After(backoffDuration):
				// Continue to next retry
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return lastErr
}

// GetProviderName returns the underlying provider name
func (rn *RetryableNotifier) GetProviderName() string {
	return rn.notifier.GetProviderName()
}

// NotificationConfig represents configuration for a notification provider
type NotificationConfig struct {
	Enabled bool   `yaml:"enabled"`
	Type    string `yaml:"type"` // "discord", "slack", etc.
}

// DiscordConfig contains Discord-specific configuration
type DiscordConfig struct {
	NotificationConfig `yaml:",inline"`
	WebhookURL         string `yaml:"webhook_url"`
}

// SlackConfig contains Slack-specific configuration
type SlackConfig struct {
	NotificationConfig `yaml:",inline"`
	WebhookURL         string `yaml:"webhook_url"`
	Channel            string `yaml:"channel,omitempty"`
}

// RateLimiter provides rate limiting for notifications
type RateLimiter struct {
	lastNotification time.Time
	interval         time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(interval time.Duration) *RateLimiter {
	return &RateLimiter{
		interval: interval,
	}
}

// Allow checks if a notification is allowed (not rate limited)
func (rl *RateLimiter) Allow() bool {
	now := time.Now()
	if now.Sub(rl.lastNotification) >= rl.interval {
		rl.lastNotification = now
		return true
	}
	return false
}

// Wait waits until the next notification is allowed
func (rl *RateLimiter) Wait(ctx context.Context) error {
	now := time.Now()
	nextAllowed := rl.lastNotification.Add(rl.interval)

	if now.Before(nextAllowed) {
		waitTime := nextAllowed.Sub(now)
		select {
		case <-time.After(waitTime):
			rl.lastNotification = time.Now()
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	rl.lastNotification = now
	return nil
}

// RateLimitedNotifier wraps a notifier with rate limiting
type RateLimitedNotifier struct {
	notifier    Notifier
	rateLimiter *RateLimiter
	logger      *logger.Logger
}

// NewRateLimitedNotifier creates a new rate-limited notifier
func NewRateLimitedNotifier(notifier Notifier, interval time.Duration, logger *logger.Logger) *RateLimitedNotifier {
	return &RateLimitedNotifier{
		notifier:    notifier,
		rateLimiter: NewRateLimiter(interval),
		logger:      logger.WithComponent("rate_limited_notifier"),
	}
}

// NotifyNewStars sends a notification with rate limiting
func (rln *RateLimitedNotifier) NotifyNewStars(ctx context.Context, owner, repo string, newStargazers []github.Stargazer) error {
	provider := rln.notifier.GetProviderName()

	if !rln.rateLimiter.Allow() {
		rln.logger.Debug("rate limit hit, waiting",
			"provider", provider,
			"repo", owner+"/"+repo)

		if err := rln.rateLimiter.Wait(ctx); err != nil {
			return err
		}
	}

	return rln.notifier.NotifyNewStars(ctx, owner, repo, newStargazers)
}

// TestConnection tests the connection (not rate limited)
func (rln *RateLimitedNotifier) TestConnection(ctx context.Context) error {
	return rln.notifier.TestConnection(ctx)
}

// GetProviderName returns the underlying provider name
func (rln *RateLimitedNotifier) GetProviderName() string {
	return rln.notifier.GetProviderName()
}
