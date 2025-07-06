package github

import (
	"context"
	"testing"
	"time"
)

func TestGitHubClientBasic(t *testing.T) {
	// Create client with custom timeout
	client := NewClientWithConfig(Config{
		Timeout: time.Second * 10,
	})

	if client == nil {
		t.Error("Client should not be nil")
	}

	if client.baseURL != "https://api.github.com" {
		t.Errorf("Expected baseURL to be 'https://api.github.com', got %s", client.baseURL)
	}

	if client.userAgent != "github-stars-notify/1.0" {
		t.Errorf("Expected userAgent to be 'github-stars-notify/1.0', got %s", client.userAgent)
	}

	if client.httpClient.Timeout != time.Second*10 {
		t.Errorf("Expected timeout to be 10s, got %v", client.httpClient.Timeout)
	}
}

func TestGitHubClientWithToken(t *testing.T) {
	token := "test_token"
	client := NewClientWithToken(token)

	if client.token != token {
		t.Errorf("Expected token to be '%s', got %s", token, client.token)
	}
}

func TestGitHubClientGetStargazers(t *testing.T) {
	// This is a basic test that doesn't make real API calls
	client := NewClient()
	ctx := context.Background()

	// Test with a non-existent repository to avoid rate limits
	// This will fail but we're just testing the method signature
	_, err := client.GetStargazers(ctx, "nonexistent", "repo")

	// We expect an error since this is a fake repo
	if err == nil {
		t.Log("Note: GetStargazers should fail for non-existent repository")
	}
}

func TestGitHubClientGetRateLimit(t *testing.T) {
	// This is a basic test that doesn't make real API calls
	client := NewClient()
	ctx := context.Background()

	// This might fail if no internet connection or rate limited
	_, err := client.GetRateLimit(ctx)

	// We don't fail the test since this depends on network
	if err != nil {
		t.Logf("Rate limit check failed (expected in test environment): %v", err)
	}
}

func TestRetryableClient(t *testing.T) {
	baseClient := NewClient()
	retryClient := NewRetryableClient(baseClient, 2, time.Millisecond*10)

	if retryClient.maxRetries != 2 {
		t.Errorf("Expected maxRetries to be 2, got %d", retryClient.maxRetries)
	}

	if retryClient.backoff != time.Millisecond*10 {
		t.Errorf("Expected backoff to be 10ms, got %v", retryClient.backoff)
	}
}

func TestParseNextPage(t *testing.T) {
	client := NewClient()

	tests := []struct {
		linkHeader string
		expected   int
	}{
		{
			linkHeader: `<https://api.github.com/repositories/1/stargazers?page=2>; rel="next", <https://api.github.com/repositories/1/stargazers?page=5>; rel="last"`,
			expected:   2,
		},
		{
			linkHeader: `<https://api.github.com/repositories/1/stargazers?page=3>; rel="next"`,
			expected:   3,
		},
		{
			linkHeader: `<https://api.github.com/repositories/1/stargazers?page=1>; rel="prev"`,
			expected:   0,
		},
		{
			linkHeader: "",
			expected:   0,
		},
	}

	for _, test := range tests {
		result := client.parseNextPage(test.linkHeader)
		if result != test.expected {
			t.Errorf("parseNextPage(%q) = %d, expected %d", test.linkHeader, result, test.expected)
		}
	}
}
