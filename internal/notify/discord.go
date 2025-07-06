package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github-stars-notify/internal/errors"
	"github-stars-notify/internal/github"
)

// Provider name constants
const (
	ProviderDiscord = "discord"
	ProviderSlack   = "slack"
)

// DiscordNotifier sends notifications via Discord webhooks
type DiscordNotifier struct {
	webhookURL string
	httpClient *http.Client
}

// DiscordMessage represents a Discord webhook message
type DiscordMessage struct {
	Content string         `json:"content,omitempty"`
	Embeds  []DiscordEmbed `json:"embeds,omitempty"`
}

// DiscordEmbed represents a Discord embed
type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

// DiscordEmbedField represents a field in a Discord embed
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// DiscordEmbedFooter represents the footer of a Discord embed
type DiscordEmbedFooter struct {
	Text string `json:"text"`
}

// NewDiscordNotifier creates a new Discord notifier
func NewDiscordNotifier(webhookURL string) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewDiscordNotifierWithTimeout creates a new Discord notifier with custom timeout
func NewDiscordNotifierWithTimeout(webhookURL string, timeout time.Duration) *DiscordNotifier {
	// Create HTTP client with more robust configuration
	transport := &http.Transport{
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &DiscordNotifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

// GetProviderName returns the provider name for Discord
func (d *DiscordNotifier) GetProviderName() string {
	return ProviderDiscord
}

// NotifyNewStars sends a notification about new stars with context support
func (d *DiscordNotifier) NotifyNewStars(ctx context.Context, owner, repo string, newStargazers []github.Stargazer) error {
	if len(newStargazers) == 0 {
		return nil
	}

	message := d.createMessage(owner, repo, newStargazers)
	return d.sendMessage(ctx, message)
}

// createMessage creates a Discord message for new stars
func (d *DiscordNotifier) createMessage(owner, repo string, newStargazers []github.Stargazer) DiscordMessage {
	repoURL := fmt.Sprintf("https://github.com/%s/%s", owner, repo)

	var description string
	if len(newStargazers) == 1 {
		description = fmt.Sprintf("🌟 **1 new star** for [%s/%s](%s)!", owner, repo, repoURL)
	} else {
		description = fmt.Sprintf("🌟 **%d new stars** for [%s/%s](%s)!", len(newStargazers), owner, repo, repoURL)
	}

	embed := DiscordEmbed{
		Title:       "New GitHub Stars",
		Description: description,
		Color:       0x00ff00, // Green color
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &DiscordEmbedFooter{
			Text: "GitHub Stars Notify",
		},
	}

	// Add fields for each new stargazer (limit to 10 to avoid message size limits)
	maxStargazers := 10
	for i, sg := range newStargazers {
		if i >= maxStargazers {
			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:   "And more...",
				Value:  fmt.Sprintf("+ %d more stargazers", len(newStargazers)-maxStargazers),
				Inline: false,
			})
			break
		}

		stargazerURL := fmt.Sprintf("https://github.com/%s", sg.Login)
		embed.Fields = append(embed.Fields, DiscordEmbedField{
			Name:   fmt.Sprintf("⭐ %s", sg.Login),
			Value:  fmt.Sprintf("[View Profile](%s)", stargazerURL),
			Inline: true,
		})
	}

	return DiscordMessage{
		Embeds: []DiscordEmbed{embed},
	}
}

// sendMessage sends a message to the Discord webhook with context support
func (d *DiscordNotifier) sendMessage(ctx context.Context, message DiscordMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return errors.NewNotificationError(ProviderDiscord, "failed to marshal message", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", d.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.NewNotificationError(ProviderDiscord, "failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "curl/8.7.1")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return errors.NewNotificationError(ProviderDiscord,
			fmt.Sprintf("failed to send webhook: %v", err), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body := make([]byte, 512)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])

		return errors.NewNotificationError(ProviderDiscord,
			fmt.Sprintf("webhook request failed with status %d, response: %s", resp.StatusCode, responseBody), nil)
	}

	return nil
}

// TestConnection tests the Discord webhook connection with context support
func (d *DiscordNotifier) TestConnection(ctx context.Context) error {
	testMessage := DiscordMessage{
		Content: "🔔 GitHub Stars Notify is now active and monitoring your repositories!",
	}

	return d.sendMessage(ctx, testMessage)
}
