package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github-stars-notify/internal/config"
	"github-stars-notify/internal/errors"
	"github-stars-notify/internal/github"
	"github-stars-notify/internal/logger"
	"github-stars-notify/internal/metrics"
	"github-stars-notify/internal/notify"
	"github-stars-notify/internal/storage"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Service represents the main application service
type Service struct {
	configReloader *config.Reloader
	github         *github.RetryableClient
	storage        storage.Storage
	notifiers      []notify.Notifier
	metrics        *metrics.Metrics
	metricsServer  *http.Server
	logger         *logger.Logger
	cancel         context.CancelFunc
	running        bool
	startTime      time.Time
	configPath     string
	tickerUpdate   chan struct{} // Channel to signal ticker updates
}

// Dependencies holds all service dependencies
type Dependencies struct {
	ConfigPath string
	Config     *config.Config
	Storage    storage.Storage
	Logger     *logger.Logger
	Metrics    *metrics.Metrics
	Notifiers  []notify.Notifier
	GitHub     *github.RetryableClient
}

// New creates a new service instance with automatic dependency setup
func New(configPath string) (*Service, error) {
	// Load initial config
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create logger from config
	log := logger.NewLogger(logger.Config{
		Level:   cfg.GetLogLevel(),
		Format:  cfg.Logging.Format,
		Service: "github-stars-notify",
	})

	// Create storage from config
	stor, err := storage.NewStorageFromConfig(storage.StorageConfig{
		Type: cfg.Storage.Type,
		Path: cfg.Storage.Path,
	})
	if err != nil {
		return nil, errors.NewServiceError("storage", "failed to create storage", err)
	}

	// Create GitHub client with retry logic
	baseClient := github.NewClientWithConfig(github.Config{
		Token:   cfg.GitHub.Token,
		Timeout: cfg.GetGitHubTimeout(),
	})
	githubClient := github.NewRetryableClient(baseClient, 3, time.Second*2)

	// Create metrics
	met := metrics.NewMetrics()

	// Create notifiers
	notifiers, err := notify.CreateNotifiersWithLogger(cfg, log)
	if err != nil {
		log.Warn("failed to create notifiers", "error", err)
		notifiers = []notify.Notifier{} // Continue without notifiers
	}

	deps := Dependencies{
		ConfigPath: configPath,
		Config:     cfg,
		Storage:    stor,
		Logger:     log,
		Metrics:    met,
		Notifiers:  notifiers,
		GitHub:     githubClient,
	}

	return NewWithDependencies(deps)
}

// NewWithDependencies creates a new service instance with provided dependencies
func NewWithDependencies(deps Dependencies) (*Service, error) {
	// Create config reloader
	reloader, err := config.NewReloader(deps.ConfigPath, deps.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create config reloader: %w", err)
	}

	service := &Service{
		configReloader: reloader,
		github:         deps.GitHub,
		storage:        deps.Storage,
		notifiers:      deps.Notifiers,
		metrics:        deps.Metrics,
		logger:         deps.Logger.WithComponent("service"),
		startTime:      time.Now(),
		configPath:     deps.ConfigPath,
		tickerUpdate:   make(chan struct{}),
	}

	// Register config reload callback
	reloader.AddCallback(service.handleConfigReload)

	return service, nil
}

// NewForTest creates a new service instance for testing
func NewForTest(cfg *config.Config) (*Service, error) {
	// Create a temporary config file for testing
	tmpDir, err := os.MkdirTemp("", "github-stars-notify-test")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Create a minimal config YAML for testing
	configYAML := fmt.Sprintf(`
repositories:
  - owner: "test"
    repo: "test-repo"

settings:
  check_interval_minutes: %d

github:
  token: "%s"
  timeout_seconds: %d

server:
  port: %d
  host: "%s"
  read_timeout_seconds: %d
  write_timeout_seconds: %d

storage:
  type: "%s"
  path: "%s"

logging:
  level: "%s"
  format: "%s"

notifications:
  discord:
    webhook_url: "%s"
    enabled: %t
  slack:
    webhook_url: "%s"
    channel: "%s"
    enabled: %t
`,
		cfg.Settings.CheckIntervalMinutes,
		cfg.GitHub.Token,
		cfg.GitHub.Timeout,
		cfg.Server.Port,
		cfg.Server.Host,
		cfg.Server.ReadTimeout,
		cfg.Server.WriteTimeout,
		cfg.Storage.Type,
		cfg.Storage.Path,
		cfg.Logging.Level,
		cfg.Logging.Format,
		cfg.Notifications.Discord.WebhookURL,
		cfg.Notifications.Discord.Enabled,
		cfg.Notifications.Slack.WebhookURL,
		cfg.Notifications.Slack.Channel,
		cfg.Notifications.Slack.Enabled,
	)

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		return nil, fmt.Errorf("failed to write test config: %w", err)
	}

	// Create test logger
	log := logger.NewLogger(logger.Config{
		Level:   cfg.GetLogLevel(),
		Format:  "text",
		Service: "github-stars-notify-test",
	})

	// Create storage
	stor := storage.NewFileStorage("./test_data")

	// Create GitHub client
	baseClient := github.NewClient()
	githubClient := github.NewRetryableClient(baseClient, 1, time.Millisecond*100)

	// Create test metrics
	met := metrics.NewTestMetrics()

	// Create basic notifiers for testing
	var notifiers []notify.Notifier
	if cfg.Notifications.Discord.Enabled {
		if notifier, err := notify.CreateBasicNotifier(notify.ProviderDiscord, cfg.Notifications.Discord.WebhookURL); err == nil {
			notifiers = append(notifiers, notifier)
		}
	}
	if cfg.Notifications.Slack.Enabled {
		if notifier, err := notify.CreateBasicNotifier(notify.ProviderSlack, cfg.Notifications.Slack.WebhookURL, cfg.Notifications.Slack.Channel); err == nil {
			notifiers = append(notifiers, notifier)
		}
	}

	deps := Dependencies{
		ConfigPath: configPath,
		Config:     cfg,
		Storage:    stor,
		Logger:     log,
		Metrics:    met,
		Notifiers:  notifiers,
		GitHub:     githubClient,
	}

	return NewWithDependencies(deps)
}

// Start starts the service
func (s *Service) Start(ctx context.Context) error {
	s.logger.Info("initializing GitHub Stars Notify service")

	// Create cancellable context
	serviceCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	// Start config reloader
	if err := s.configReloader.Start(serviceCtx); err != nil {
		return errors.NewServiceError("config-reloader", "failed to start config reloader", err)
	}

	// Record service start
	s.metrics.RecordServiceStart()

	// Initialize storage
	if err := s.storage.Initialize(serviceCtx); err != nil {
		return errors.NewServiceError("storage", "failed to initialize storage", err)
	}

	// Start metrics server
	if err := s.startMetricsServer(); err != nil {
		return errors.NewServiceError("metrics", "failed to start metrics server", err)
	}

	// Test notification connections if enabled
	for _, notifier := range s.notifiers {
		provider := notifier.GetProviderName()
		s.logger.Info("testing notification connection", "provider", provider)

		if err := notifier.TestConnection(serviceCtx); err != nil {
			s.metrics.RecordNotificationError(provider, "connection_test_failed")
			s.logger.Error("notification connection test failed", "provider", provider, "error", err)
			return errors.NewServiceError("notification",
				fmt.Sprintf("failed to test %s connection", provider), err)
		}

		s.logger.Info("notification connection test successful", "provider", provider)
		s.metrics.RecordNotificationSent(provider, "connection_test_success")
	}

	// Check rate limits
	if err := s.checkRateLimits(serviceCtx); err != nil {
		s.logger.Warn("rate limit check failed", "error", err)
	}

	s.running = true
	config := s.configReloader.GetConfig()
	s.logger.Info("service started successfully",
		"repositories", len(config.Repositories),
		"check_interval", config.GetCheckInterval(),
		"notifiers", len(s.notifiers))

	// Start the monitoring loop
	currentInterval := config.GetCheckInterval()
	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()

	// Start uptime updater
	uptimeTicker := time.NewTicker(30 * time.Second)
	defer uptimeTicker.Stop()

	for {
		select {
		case <-serviceCtx.Done():
			return nil
		case <-ticker.C:
			s.runCheck(serviceCtx)
		case <-s.tickerUpdate:
			// Handle immediate ticker interval updates from config changes
			newInterval := s.configReloader.GetConfig().GetCheckInterval()
			s.logger.Debug("received ticker update signal",
				"current_interval", currentInterval,
				"new_interval", newInterval,
				"needs_update", newInterval != currentInterval)
			if newInterval != currentInterval {
				oldInterval := currentInterval
				ticker.Reset(newInterval)
				currentInterval = newInterval
				s.logger.Info("check interval updated immediately",
					"old_interval", oldInterval,
					"new_interval", newInterval)
			}
		case <-uptimeTicker.C:
			s.metrics.UpdateServiceUptime(s.startTime)
		}
	}
}

// Stop stops the service
func (s *Service) Stop() {
	if !s.running {
		return
	}

	s.logger.Info("stopping service")
	s.running = false

	// Cancel context to stop all background operations
	if s.cancel != nil {
		s.cancel()
	}

	// Close ticker update channel
	close(s.tickerUpdate)

	// Stop config reloader
	s.configReloader.Stop()

	// Stop metrics server
	if s.metricsServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.metricsServer.Shutdown(ctx); err != nil {
			s.logger.Error("failed to shutdown metrics server", "error", err)
		}
	}

	// Close storage
	if err := s.storage.Close(); err != nil {
		s.logger.Error("failed to close storage", "error", err)
	}

	s.logger.Info("service stopped successfully")
}

// startMetricsServer starts the HTTP server for Prometheus metrics
func (s *Service) startMetricsServer() error {
	config := s.configReloader.GetConfig()
	addr := config.GetServerAddress()
	if addr == "" {
		return fmt.Errorf("server address is empty")
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			s.logger.Error("failed to write health check response", "error", err)
		}
	})

	s.metricsServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  time.Duration(config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.Server.WriteTimeout) * time.Second,
	}

	go func() {
		if err := s.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("metrics server error", "error", err)
		}
	}()

	s.logger.Info("metrics server started", "address", addr)
	return nil
}

// runCheck performs a single check cycle for all repositories
func (s *Service) runCheck(ctx context.Context) {
	s.logger.Info("starting repository check cycle")

	config := s.configReloader.GetConfig()
	s.logger.Info("current configuration for check cycle",
		"repository_count", len(config.Repositories),
		"check_interval", config.GetCheckInterval())

	for i, repo := range config.Repositories {
		s.logger.Info("processing repository",
			"index", i,
			"owner", repo.Owner,
			"repo", repo.Repo)

		if err := s.checkRepository(ctx, repo.Owner, repo.Repo); err != nil {
			s.logger.Error("repository check failed",
				"repo", repo.Owner+"/"+repo.Repo,
				"error", err)
			s.metrics.RecordCheck(repo.Owner, repo.Repo, "error")
			s.metrics.RecordCheckError(repo.Owner, repo.Repo, "general_error")
			continue
		}
	}

	// Update rate limit metrics after each check cycle
	if err := s.checkRateLimits(ctx); err != nil {
		s.logger.Warn("rate limit check failed after repository cycle", "error", err)
	}

	s.logger.Info("repository check cycle completed")
}

// checkRepository checks a single repository for new stars
func (s *Service) checkRepository(ctx context.Context, owner, repo string) error {
	start := time.Now()
	repoLogger := s.logger.WithRepository(owner, repo)

	repoLogger.Debug("checking repository")

	// Fetch current stargazers
	stargazers, err := s.github.GetStargazersWithRetry(ctx, owner, repo)
	if err != nil {
		s.metrics.RecordCheckError(owner, repo, "github_api_error")
		s.metrics.RecordGitHubAPIRequest("stargazers", "error")
		return errors.NewServiceError("github", "failed to fetch stargazers", err)
	}
	s.metrics.RecordGitHubAPIRequest("stargazers", "success")

	// Record metrics
	s.metrics.RecordRepositoryStars(owner, repo, len(stargazers))
	s.metrics.RecordCheckDuration(owner, repo, time.Since(start))

	repoLogger.Info("repository check completed",
		"total_stars", len(stargazers),
		"duration", time.Since(start))

	// Compare with previous data to find new stars
	newStargazers, err := s.storage.GetNewStargazers(ctx, owner, repo, stargazers)
	if err != nil {
		s.metrics.RecordCheckError(owner, repo, "storage_error")
		return errors.NewServiceError("storage", "failed to get new stargazers", err)
	}

	if len(newStargazers) > 0 {
		repoLogger.Info("new stargazers detected", "count", len(newStargazers))
		s.metrics.RecordNewStars(owner, repo, len(newStargazers))

		// Send notifications
		for _, notifier := range s.notifiers {
			provider := notifier.GetProviderName()
			notificationStart := time.Now()

			if err := notifier.NotifyNewStars(ctx, owner, repo, newStargazers); err != nil {
				repoLogger.Error("notification failed",
					"provider", provider,
					"error", err)
				s.metrics.RecordNotificationError(provider, "notification_failed")
			} else {
				repoLogger.Info("notification sent successfully",
					"provider", provider,
					"stargazers", len(newStargazers))
				s.metrics.RecordNotificationSent(provider, "success")
			}

			s.metrics.RecordNotificationLatency(provider, time.Since(notificationStart))
		}
	} else {
		repoLogger.Debug("no new stargazers found")
	}

	// Save current stargazers data
	if err := s.storage.Save(ctx, owner, repo, stargazers); err != nil {
		s.metrics.RecordCheckError(owner, repo, "storage_save_error")
		return errors.NewServiceError("storage", "failed to save stargazers data", err)
	}

	// Record successful check
	s.metrics.RecordCheck(owner, repo, "success")
	s.metrics.RecordLastCheckTime(owner, repo)

	return nil
}

// checkRateLimits checks the GitHub API rate limits
func (s *Service) checkRateLimits(ctx context.Context) error {
	rateLimit, err := s.github.GetRateLimitWithRetry(ctx)
	if err != nil {
		s.metrics.RecordGitHubAPIError("rate_limit", "request_failed")
		s.metrics.RecordGitHubAPIRequest("rate_limit", "error")
		return errors.NewServiceError("github", "failed to check rate limits", err)
	}
	s.metrics.RecordGitHubAPIRequest("rate_limit", "success")

	// Record rate limit metrics
	s.metrics.RecordGitHubRateLimit("core", rateLimit.Limit, rateLimit.Remaining)

	s.logger.Info("rate limit status",
		"remaining", rateLimit.Remaining,
		"limit", rateLimit.Limit,
		"reset", rateLimit.Reset)

	if rateLimit.Remaining < 10 {
		s.logger.Warn("low API rate limit remaining", "remaining", rateLimit.Remaining)
		return errors.NewServiceError("github",
			fmt.Sprintf("low API rate limit remaining: %d", rateLimit.Remaining), nil)
	}

	return nil
}

// GetStatus returns the current service status
func (s *Service) GetStatus() map[string]interface{} {
	config := s.configReloader.GetConfig()
	status := map[string]interface{}{
		"running":        s.running,
		"repositories":   len(config.Repositories),
		"notifiers":      len(s.notifiers),
		"check_interval": config.GetCheckInterval().String(),
		"uptime":         time.Since(s.startTime).String(),
	}

	// Add rate limit info if available
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if rateLimit, err := s.github.GetRateLimit(ctx); err == nil {
		status["rate_limit"] = map[string]interface{}{
			"remaining": rateLimit.Remaining,
			"limit":     rateLimit.Limit,
			"reset":     rateLimit.Reset,
		}
	}

	return status
}

// handleConfigReload handles config reload events
func (s *Service) handleConfigReload(oldConfig, newConfig *config.Config) error {
	s.logger.Info("handling configuration reload",
		"old_check_interval", oldConfig.GetCheckInterval(),
		"new_check_interval", newConfig.GetCheckInterval(),
		"old_repo_count", len(oldConfig.Repositories),
		"new_repo_count", len(newConfig.Repositories))

	// Log repository changes
	if len(oldConfig.Repositories) != len(newConfig.Repositories) {
		s.logger.Info("repository count changed",
			"old_count", len(oldConfig.Repositories),
			"new_count", len(newConfig.Repositories))

		for i, repo := range newConfig.Repositories {
			s.logger.Info("new repository list entry",
				"index", i,
				"owner", repo.Owner,
				"repo", repo.Repo)
		}
	}

	// Recreate GitHub client if token or timeout changed
	if oldConfig.GitHub.Token != newConfig.GitHub.Token ||
		oldConfig.GetGitHubTimeout() != newConfig.GetGitHubTimeout() {
		baseClient := github.NewClientWithConfig(github.Config{
			Token:   newConfig.GitHub.Token,
			Timeout: newConfig.GetGitHubTimeout(),
		})
		s.github = github.NewRetryableClient(baseClient, 3, time.Second*2)
		s.logger.Info("recreated GitHub client")
	}

	// Recreate notifiers if notification config changed
	if !equalNotifications(oldConfig.Notifications, newConfig.Notifications) {
		notifiers, err := notify.CreateNotifiersWithLogger(newConfig, s.logger)
		if err != nil {
			s.logger.Warn("failed to recreate notifiers", "error", err)
			s.notifiers = []notify.Notifier{} // Continue without notifiers
		} else {
			s.notifiers = notifiers
			s.logger.Info("recreated notifiers")
		}

		// Test new notification connections
		for _, notifier := range s.notifiers {
			provider := notifier.GetProviderName()
			if err := notifier.TestConnection(context.Background()); err != nil {
				s.metrics.RecordNotificationError(provider, "connection_test_failed")
				s.logger.Error("new notification connection test failed", "provider", provider, "error", err)
			} else {
				s.logger.Info("new notification connection test successful", "provider", provider)
				s.metrics.RecordNotificationSent(provider, "connection_test_success")
			}
		}
	}

	// Update logger level if changed
	if oldConfig.GetLogLevel() != newConfig.GetLogLevel() {
		// Note: Logger level updating would need to be implemented in the logger package
		s.logger.Info("log level changed",
			"old_level", oldConfig.GetLogLevel(),
			"new_level", newConfig.GetLogLevel())
	}

	// Signal ticker update if check interval changed
	if oldConfig.GetCheckInterval() != newConfig.GetCheckInterval() {
		s.logger.Info("check interval changed, signaling ticker update",
			"old_interval", oldConfig.GetCheckInterval(),
			"new_interval", newConfig.GetCheckInterval())

		// Non-blocking send to ticker update channel
		select {
		case s.tickerUpdate <- struct{}{}:
			s.logger.Debug("ticker update signal sent successfully")
		default:
			s.logger.Debug("ticker update channel full, signal skipped")
		}
	}

	s.logger.Info("configuration reload completed successfully")
	return nil
}

// equalNotifications compares notification configurations (helper function)
func equalNotifications(a, b config.Notifications) bool {
	return a.Discord.Enabled == b.Discord.Enabled &&
		a.Discord.WebhookURL == b.Discord.WebhookURL &&
		a.Slack.Enabled == b.Slack.Enabled &&
		a.Slack.WebhookURL == b.Slack.WebhookURL &&
		a.Slack.Channel == b.Slack.Channel
}
