package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github-stars-notify/internal/errors"
)

// Client represents a GitHub API client
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	userAgent  string
}

// Config holds GitHub client configuration
type Config struct {
	Token     string
	BaseURL   string
	Timeout   time.Duration
	UserAgent string
}

// Stargazer represents a GitHub user who starred a repository
type Stargazer struct {
	Login     string    `json:"login"`
	ID        int64     `json:"id"`
	NodeID    string    `json:"node_id"`
	AvatarURL string    `json:"avatar_url"`
	StarredAt time.Time `json:"starred_at"`
}

// APIResponse represents the API response structure
type APIResponse struct {
	Stargazers []Stargazer
	NextPage   int
	LastPage   int
	RateLimit  RateLimit
}

// RateLimit represents GitHub API rate limit information
type RateLimit struct {
	Limit     int
	Remaining int
	Reset     time.Time
}

// NewClient creates a new GitHub API client with default configuration
func NewClient() *Client {
	return NewClientWithConfig(Config{})
}

// NewClientWithToken creates a new GitHub API client with authentication token
func NewClientWithToken(token string) *Client {
	return NewClientWithConfig(Config{
		Token: token,
	})
}

// NewClientWithConfig creates a new GitHub API client with custom configuration
func NewClientWithConfig(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.github.com"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "github-stars-notify/1.0"
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		baseURL:   cfg.BaseURL,
		token:     cfg.Token,
		userAgent: cfg.UserAgent,
	}
}

// GetStargazers fetches all stargazers for a repository with context support
func (c *Client) GetStargazers(ctx context.Context, owner, repo string) ([]Stargazer, error) {
	var allStargazers []Stargazer
	page := 1

	for {
		stargazers, nextPage, err := c.getStargazersPage(ctx, owner, repo, page)
		if err != nil {
			return nil, err
		}

		allStargazers = append(allStargazers, stargazers...)

		if nextPage == 0 {
			break
		}
		page = nextPage

		// Check if context is cancelled
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	return allStargazers, nil
}

// getStargazersPage fetches a single page of stargazers
func (c *Client) getStargazersPage(ctx context.Context, owner, repo string, page int) ([]Stargazer, int, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/stargazers", owner, repo)
	url := fmt.Sprintf("%s%s?page=%d&per_page=100", c.baseURL, endpoint, page)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, errors.NewGitHubAPIError(endpoint, 0, "failed to create request", err)
	}

	// Request stargazer data with timestamps
	req.Header.Set("Accept", "application/vnd.github.v3.star+json")
	req.Header.Set("User-Agent", c.userAgent)

	// Add authorization header if token is provided
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, errors.NewGitHubAPIError(endpoint, 0, "failed to make request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, errors.NewGitHubAPIError(endpoint, resp.StatusCode,
			fmt.Sprintf("API request failed with status %d", resp.StatusCode), nil)
	}

	var stargazers []struct {
		StarredAt time.Time `json:"starred_at"`
		User      Stargazer `json:"user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&stargazers); err != nil {
		return nil, 0, errors.NewGitHubAPIError(endpoint, resp.StatusCode,
			"failed to decode response", err)
	}

	// Convert to our format
	var result []Stargazer
	for _, sg := range stargazers {
		sg.User.StarredAt = sg.StarredAt
		result = append(result, sg.User)
	}

	// Parse Link header for pagination
	nextPage := c.parseNextPage(resp.Header.Get("Link"))

	return result, nextPage, nil
}

// GetRateLimit fetches the current rate limit status with context support
func (c *Client) GetRateLimit(ctx context.Context) (*RateLimit, error) {
	endpoint := "/rate_limit"
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.NewGitHubAPIError(endpoint, 0, "failed to create request", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	// Add authorization header if token is provided
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.NewGitHubAPIError(endpoint, 0, "failed to make request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewGitHubAPIError(endpoint, resp.StatusCode,
			fmt.Sprintf("API request failed with status %d", resp.StatusCode), nil)
	}

	var rateLimitResp struct {
		Rate struct {
			Limit     int `json:"limit"`
			Remaining int `json:"remaining"`
			Reset     int `json:"reset"`
		} `json:"rate"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rateLimitResp); err != nil {
		return nil, errors.NewGitHubAPIError(endpoint, resp.StatusCode,
			"failed to decode response", err)
	}

	return &RateLimit{
		Limit:     rateLimitResp.Rate.Limit,
		Remaining: rateLimitResp.Rate.Remaining,
		Reset:     time.Unix(int64(rateLimitResp.Rate.Reset), 0),
	}, nil
}

// parseNextPage parses the Link header to extract the next page number
func (c *Client) parseNextPage(linkHeader string) int {
	if linkHeader == "" {
		return 0
	}

	// Parse Link header format: <url>; rel="next", <url>; rel="last"
	links := strings.Split(linkHeader, ",")
	for _, link := range links {
		parts := strings.Split(strings.TrimSpace(link), ";")
		if len(parts) != 2 {
			continue
		}

		// Check if this is the "next" link
		if strings.Contains(parts[1], `rel="next"`) {
			// Extract URL from <url>
			url := strings.Trim(parts[0], "<>")

			// Extract page number from URL
			re := regexp.MustCompile(`page=(\d+)`)
			matches := re.FindStringSubmatch(url)
			if len(matches) > 1 {
				if page, err := strconv.Atoi(matches[1]); err == nil {
					return page
				}
			}
		}
	}

	return 0
}

// RetryableClient wraps the GitHub client with retry logic
type RetryableClient struct {
	*Client
	maxRetries int
	backoff    time.Duration
}

// NewRetryableClient creates a new retryable GitHub client
func NewRetryableClient(client *Client, maxRetries int, backoff time.Duration) *RetryableClient {
	return &RetryableClient{
		Client:     client,
		maxRetries: maxRetries,
		backoff:    backoff,
	}
}

// GetStargazersWithRetry fetches stargazers with retry logic
func (rc *RetryableClient) GetStargazersWithRetry(ctx context.Context, owner, repo string) ([]Stargazer, error) {
	var lastErr error

	for i := 0; i <= rc.maxRetries; i++ {
		stargazers, err := rc.Client.GetStargazers(ctx, owner, repo)
		if err == nil {
			return stargazers, nil
		}

		lastErr = err

		// Check if it's a rate limit error
		if gitHubErr, ok := err.(*errors.GitHubAPIError); ok && gitHubErr.IsRateLimited() {
			// For rate limit errors, don't retry immediately
			return nil, err
		}

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Wait before retrying (except on last attempt)
		if i < rc.maxRetries {
			select {
			case <-time.After(rc.backoff * time.Duration(i+1)):
				// Continue to next retry
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, lastErr
}

// GetRateLimitWithRetry fetches rate limit with retry logic
func (rc *RetryableClient) GetRateLimitWithRetry(ctx context.Context) (*RateLimit, error) {
	var lastErr error

	for i := 0; i <= rc.maxRetries; i++ {
		rateLimit, err := rc.Client.GetRateLimit(ctx)
		if err == nil {
			return rateLimit, nil
		}

		lastErr = err

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Wait before retrying (except on last attempt)
		if i < rc.maxRetries {
			select {
			case <-time.After(rc.backoff * time.Duration(i+1)):
				// Continue to next retry
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, lastErr
}
