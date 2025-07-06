package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsBasic(t *testing.T) {
	m := NewTestMetrics()
	if m == nil {
		t.Fatal("NewTestMetrics returned nil")
	}

	// Test repository metrics
	m.RecordRepositoryStars("facebook", "react", 100)
	m.RecordNewStars("facebook", "react", 5)
	m.RecordCheckDuration("facebook", "react", time.Second*2)
	m.RecordLastCheckTime("facebook", "react")
	m.RecordCheck("facebook", "react", "success")
	m.RecordCheckError("facebook", "react", "api_error")

	// Test GitHub API metrics
	m.RecordGitHubAPIRequest("stargazers", "success")
	m.RecordGitHubAPIError("stargazers", "timeout")
	m.RecordGitHubRateLimit("core", 5000, 4900)

	// Test notification metrics (provider-agnostic)
	m.RecordNotificationSent("discord", "success")
	m.RecordNotificationError("discord", "webhook_failed")
	m.RecordNotificationLatency("discord", time.Millisecond*500)

	m.RecordNotificationSent("slack", "success")
	m.RecordNotificationError("slack", "timeout")
	m.RecordNotificationLatency("slack", time.Millisecond*750)

	// Test service metrics
	m.RecordServiceStart()
	m.UpdateServiceUptime(time.Now().Add(-time.Hour))

	// Test backward compatibility methods
	m.RecordDiscordNotification("success")
	m.RecordDiscordError("connection_failed")

	// Verify metrics were recorded
	if testutil.ToFloat64(m.TotalStars.WithLabelValues("facebook", "react")) != 100 {
		t.Error("Repository stars not recorded correctly")
	}
	if testutil.ToFloat64(m.NewStars.WithLabelValues("facebook", "react")) != 5 {
		t.Error("New stars not recorded correctly")
	}
	if testutil.ToFloat64(m.ChecksTotal.WithLabelValues("facebook", "react", "success")) != 1 {
		t.Error("Check not recorded correctly")
	}
	if testutil.ToFloat64(m.GitHubRateLimit.WithLabelValues("core")) != 5000 {
		t.Error("Rate limit not recorded correctly")
	}
	if testutil.ToFloat64(m.NotificationsSent.WithLabelValues("discord", "success")) != 2 {
		t.Error("Discord notification not recorded correctly")
	}
}

func TestMetricsRegistry(t *testing.T) {
	// Test default registry
	m1 := NewMetrics()
	if m1 == nil {
		t.Error("NewMetrics returned nil")
	}

	// Test test registry (isolated)
	m2 := NewTestMetrics()
	if m2 == nil {
		t.Error("NewTestMetrics returned nil")
	}

	// Test that they use different registries
	if m1.registry == m2.registry {
		t.Error("Test metrics should use isolated registry")
	}
}

func TestHTTPStatusToString(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{200, "200"},
		{404, "404"},
		{500, "500"},
	}

	for _, test := range tests {
		result := HTTPStatusToString(test.code)
		if result != test.expected {
			t.Errorf("HTTPStatusToString(%d) = %s, expected %s", test.code, result, test.expected)
		}
	}
}
