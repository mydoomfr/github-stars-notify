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

// SlackNotifier sends notifications via Slack webhooks
type SlackNotifier struct {
	webhookURL string
	channel    string
	httpClient *http.Client
}

// SlackMessage represents a Slack webhook message
type SlackMessage struct {
	Text        string            `json:"text,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack message attachment
type SlackAttachment struct {
	Color     string       `json:"color,omitempty"`
	Title     string       `json:"title,omitempty"`
	TitleLink string       `json:"title_link,omitempty"`
	Text      string       `json:"text,omitempty"`
	Fields    []SlackField `json:"fields,omitempty"`
	Footer    string       `json:"footer,omitempty"`
	Timestamp int64        `json:"ts,omitempty"`
}

// SlackField represents a field in a Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(webhookURL, channel string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		channel:    channel,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewSlackNotifierWithTimeout creates a new Slack notifier with custom timeout
func NewSlackNotifierWithTimeout(webhookURL, channel string, timeout time.Duration) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		channel:    channel,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetProviderName returns the provider name for Slack
func (s *SlackNotifier) GetProviderName() string {
	return ProviderSlack
}

// NotifyNewStars sends a notification about new stars with context support
func (s *SlackNotifier) NotifyNewStars(ctx context.Context, owner, repo string, newStargazers []github.Stargazer) error {
	if len(newStargazers) == 0 {
		return nil
	}

	message := s.createMessage(owner, repo, newStargazers)
	return s.sendMessage(ctx, message)
}

// createMessage creates a Slack message for new stars
func (s *SlackNotifier) createMessage(owner, repo string, newStargazers []github.Stargazer) SlackMessage {
	repoURL := fmt.Sprintf("https://github.com/%s/%s", owner, repo)

	var title, text string
	if len(newStargazers) == 1 {
		title = fmt.Sprintf("⭐ 1 new star for %s/%s", owner, repo)
		text = fmt.Sprintf("Repository <%s|%s/%s> received a new star!", repoURL, owner, repo)
	} else {
		title = fmt.Sprintf("⭐ %d new stars for %s/%s", len(newStargazers), owner, repo)
		text = fmt.Sprintf("Repository <%s|%s/%s> received %d new stars!", repoURL, owner, repo, len(newStargazers))
	}

	attachment := SlackAttachment{
		Color:     "good",
		Title:     title,
		TitleLink: repoURL,
		Text:      text,
		Footer:    "GitHub Stars Notify",
		Timestamp: time.Now().Unix(),
	}

	// Add fields for stargazers (limit to 10)
	maxStargazers := 10
	for i, sg := range newStargazers {
		if i >= maxStargazers {
			remaining := len(newStargazers) - maxStargazers
			attachment.Fields = append(attachment.Fields, SlackField{
				Title: "And more...",
				Value: fmt.Sprintf("%d more stargazers", remaining),
				Short: false,
			})
			break
		}

		stargazerURL := fmt.Sprintf("https://github.com/%s", sg.Login)
		attachment.Fields = append(attachment.Fields, SlackField{
			Title: sg.Login,
			Value: fmt.Sprintf("<%s|View Profile>", stargazerURL),
			Short: true,
		})
	}

	message := SlackMessage{
		Username:    "GitHub Stars Notify",
		IconEmoji:   ":star:",
		Attachments: []SlackAttachment{attachment},
	}

	if s.channel != "" {
		message.Channel = s.channel
	}

	return message
}

// sendMessage sends a message to the Slack webhook with context support
func (s *SlackNotifier) sendMessage(ctx context.Context, message SlackMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return errors.NewNotificationError(ProviderSlack, "failed to marshal message", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.NewNotificationError(ProviderSlack, "failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "github-stars-notify/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return errors.NewNotificationError(ProviderSlack, "failed to send webhook", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.NewNotificationError(ProviderSlack,
			fmt.Sprintf("webhook request failed with status %d", resp.StatusCode), nil)
	}

	return nil
}

// TestConnection tests the Slack webhook connection with context support
func (s *SlackNotifier) TestConnection(ctx context.Context) error {
	testMessage := SlackMessage{
		Text:      "🔔 GitHub Stars Notify is now active and monitoring your repositories!",
		Username:  "GitHub Stars Notify",
		IconEmoji: ":robot_face:",
	}

	if s.channel != "" {
		testMessage.Channel = s.channel
	}

	return s.sendMessage(ctx, testMessage)
}
