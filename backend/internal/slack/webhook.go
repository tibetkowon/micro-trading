package slack

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type Client struct {
	webhookURL string
	http       *http.Client
}

func New(webhookURL string) *Client {
	return &Client{
		webhookURL: webhookURL,
		http:       &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.webhookURL != ""
}

type Attachment struct {
	Color     string  `json:"color,omitempty"`
	Title     string  `json:"title,omitempty"`
	Text      string  `json:"text,omitempty"`
	Fields    []Field `json:"fields,omitempty"`
	Timestamp int64   `json:"ts,omitempty"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

func (c *Client) sendAttachments(attachments []Attachment) {
	if !c.Enabled() {
		return
	}
	payload := map[string]any{"attachments": attachments}
	b, _ := json.Marshal(payload)
	resp, err := c.http.Post(c.webhookURL, "application/json", bytes.NewReader(b))
	if err != nil {
		slog.Warn("slack webhook failed", "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		slog.Warn("slack webhook error response", "status", resp.StatusCode)
	}
}
