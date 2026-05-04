package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Notifier sends notifications via Discord webhook.
type Notifier struct {
	webhookURL string
	client     *http.Client
}

// New creates a Notifier. If webhookURL is empty, all Send calls are no-ops.
func New(webhookURL string) *Notifier {
	return &Notifier{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 5 * time.Second},
	}
}

// Send posts a plain-text message to the Discord channel.
func (n *Notifier) Send(message string) error {
	if n.webhookURL == "" {
		return nil
	}
	payload := map[string]string{"content": message}
	body, _ := json.Marshal(payload)
	resp, err := n.client.Post(n.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord webhook post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook status: %d", resp.StatusCode)
	}
	return nil
}

// SendEmbed posts a rich embed message (title + description).
func (n *Notifier) SendEmbed(title, description string, color int) error {
	if n.webhookURL == "" {
		return nil
	}
	payload := map[string]any{
		"embeds": []map[string]any{
			{
				"title":       title,
				"description": description,
				"color":       color,
			},
		},
	}
	body, _ := json.Marshal(payload)
	resp, err := n.client.Post(n.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord embed post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord embed status: %d", resp.StatusCode)
	}
	return nil
}

// Colors for common notification types.
const (
	ColorGreen  = 0x2ECC71
	ColorRed    = 0xE74C3C
	ColorYellow = 0xF39C12
	ColorBlue   = 0x3498DB
)
