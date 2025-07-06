package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all the Prometheus metrics for the GitHub Stars Notify service
type Metrics struct {
	// Repository metrics
	TotalStars    *prometheus.GaugeVec
	NewStars      *prometheus.CounterVec
	CheckDuration *prometheus.HistogramVec
	LastCheckTime *prometheus.GaugeVec
	ChecksTotal   *prometheus.CounterVec
	CheckErrors   *prometheus.CounterVec

	// GitHub API metrics
	GitHubAPIRequests        *prometheus.CounterVec
	GitHubAPIErrors          *prometheus.CounterVec
	GitHubRateLimit          *prometheus.GaugeVec
	GitHubRateLimitRemaining *prometheus.GaugeVec

	// Notification metrics (provider-agnostic)
	NotificationsSent   *prometheus.CounterVec
	NotificationErrors  *prometheus.CounterVec
	NotificationLatency *prometheus.HistogramVec

	// Service metrics
	ServiceUptime    prometheus.Gauge
	ServiceStartTime prometheus.Gauge

	// Registry for this metrics instance
	registry *prometheus.Registry
}

// NewMetrics creates and registers all Prometheus metrics using the default registry
func NewMetrics() *Metrics {
	return NewMetricsWithRegistry(nil)
}

// NewMetricsWithRegistry creates and registers all Prometheus metrics using a custom registry
// If registry is nil, uses the default registry
func NewMetricsWithRegistry(registry *prometheus.Registry) *Metrics {
	var factory promauto.Factory
	if registry != nil {
		factory = promauto.With(registry)
	} else {
		// Explicitly use the default registry
		factory = promauto.With(prometheus.DefaultRegisterer)
	}

	return &Metrics{
		registry: registry,
		// Repository metrics
		TotalStars: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "github_stars_total",
				Help: "Total number of stars for each repository",
			},
			[]string{"owner", "repo"},
		),
		NewStars: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "github_stars_new_total",
				Help: "Total number of new stars detected",
			},
			[]string{"owner", "repo"},
		),
		CheckDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "github_stars_check_duration_seconds",
				Help:    "Duration of repository checks in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"owner", "repo"},
		),
		LastCheckTime: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "github_stars_last_check_timestamp",
				Help: "Timestamp of the last successful check",
			},
			[]string{"owner", "repo"},
		),
		ChecksTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "github_stars_checks_total",
				Help: "Total number of repository checks performed",
			},
			[]string{"owner", "repo", "status"},
		),
		CheckErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "github_stars_check_errors_total",
				Help: "Total number of errors during repository checks",
			},
			[]string{"owner", "repo", "error_type"},
		),

		// GitHub API metrics
		GitHubAPIRequests: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "github_api_requests_total",
				Help: "Total number of GitHub API requests",
			},
			[]string{"endpoint", "status"},
		),
		GitHubAPIErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "github_api_errors_total",
				Help: "Total number of GitHub API errors",
			},
			[]string{"endpoint", "error_type"},
		),
		GitHubRateLimit: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "github_api_rate_limit_limit",
				Help: "GitHub API rate limit limit",
			},
			[]string{"resource"},
		),
		GitHubRateLimitRemaining: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "github_api_rate_limit_remaining",
				Help: "GitHub API rate limit remaining",
			},
			[]string{"resource"},
		),

		// Notification metrics (provider-agnostic)
		NotificationsSent: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notifications_sent_total",
				Help: "Total number of notifications sent",
			},
			[]string{"provider", "status"},
		),
		NotificationErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notification_errors_total",
				Help: "Total number of notification errors",
			},
			[]string{"provider", "error_type"},
		),
		NotificationLatency: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "notification_latency_seconds",
				Help:    "Time taken to send notifications",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"provider"},
		),

		// Service metrics
		ServiceUptime: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "github_stars_service_uptime_seconds",
				Help: "Service uptime in seconds",
			},
		),
		ServiceStartTime: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "github_stars_service_start_time_timestamp",
				Help: "Service start time as Unix timestamp",
			},
		),
	}
}

// NewTestMetrics creates metrics for testing using an isolated registry
func NewTestMetrics() *Metrics {
	testRegistry := prometheus.NewRegistry()
	return NewMetricsWithRegistry(testRegistry)
}

// RecordRepositoryStars records the total number of stars for a repository
func (m *Metrics) RecordRepositoryStars(owner, repo string, stars int) {
	m.TotalStars.WithLabelValues(owner, repo).Set(float64(stars))
}

// RecordNewStars records new stars detected for a repository
func (m *Metrics) RecordNewStars(owner, repo string, newStars int) {
	m.NewStars.WithLabelValues(owner, repo).Add(float64(newStars))
}

// RecordCheckDuration records the duration of a repository check
func (m *Metrics) RecordCheckDuration(owner, repo string, duration time.Duration) {
	m.CheckDuration.WithLabelValues(owner, repo).Observe(duration.Seconds())
}

// RecordLastCheckTime records the timestamp of the last successful check
func (m *Metrics) RecordLastCheckTime(owner, repo string) {
	m.LastCheckTime.WithLabelValues(owner, repo).SetToCurrentTime()
}

// RecordCheck records a repository check with its status
func (m *Metrics) RecordCheck(owner, repo, status string) {
	m.ChecksTotal.WithLabelValues(owner, repo, status).Inc()
}

// RecordCheckError records an error during a repository check
func (m *Metrics) RecordCheckError(owner, repo, errorType string) {
	m.CheckErrors.WithLabelValues(owner, repo, errorType).Inc()
}

// RecordGitHubAPIRequest records a GitHub API request
func (m *Metrics) RecordGitHubAPIRequest(endpoint, status string) {
	m.GitHubAPIRequests.WithLabelValues(endpoint, status).Inc()
}

// RecordGitHubAPIError records a GitHub API error
func (m *Metrics) RecordGitHubAPIError(endpoint, errorType string) {
	m.GitHubAPIErrors.WithLabelValues(endpoint, errorType).Inc()
}

// RecordGitHubRateLimit records GitHub API rate limit information
func (m *Metrics) RecordGitHubRateLimit(resource string, limit, remaining int) {
	m.GitHubRateLimit.WithLabelValues(resource).Set(float64(limit))
	m.GitHubRateLimitRemaining.WithLabelValues(resource).Set(float64(remaining))
}

// RecordNotificationSent records a notification attempt
func (m *Metrics) RecordNotificationSent(provider, status string) {
	m.NotificationsSent.WithLabelValues(provider, status).Inc()
}

// RecordNotificationError records a notification error
func (m *Metrics) RecordNotificationError(provider, errorType string) {
	m.NotificationErrors.WithLabelValues(provider, errorType).Inc()
}

// RecordNotificationLatency records the time taken to send a notification
func (m *Metrics) RecordNotificationLatency(provider string, duration time.Duration) {
	m.NotificationLatency.WithLabelValues(provider).Observe(duration.Seconds())
}

// RecordServiceStart records the service start time
func (m *Metrics) RecordServiceStart() {
	m.ServiceStartTime.SetToCurrentTime()
}

// UpdateServiceUptime updates the service uptime metric
func (m *Metrics) UpdateServiceUptime(startTime time.Time) {
	m.ServiceUptime.Set(time.Since(startTime).Seconds())
}

// Helper function to convert HTTP status code to string
func HTTPStatusToString(code int) string {
	return strconv.Itoa(code)
}

// Legacy methods for backward compatibility - these will be deprecated
// TODO: Remove these methods after updating all callers

// RecordDiscordNotification records a Discord notification (deprecated - use RecordNotificationSent)
func (m *Metrics) RecordDiscordNotification(status string) {
	m.RecordNotificationSent("discord", status)
}

// RecordDiscordError records a Discord error (deprecated - use RecordNotificationError)
func (m *Metrics) RecordDiscordError(errorType string) {
	m.RecordNotificationError("discord", errorType)
}
