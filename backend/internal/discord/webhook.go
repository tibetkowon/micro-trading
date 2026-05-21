package discord

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

type Embed struct {
	Title       string  `json:"title,omitempty"`
	Description string  `json:"description,omitempty"`
	Color       int     `json:"color,omitempty"`
	Fields      []Field `json:"fields,omitempty"`
	Timestamp   string  `json:"timestamp,omitempty"`
}

type Field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

func (c *Client) sendEmbed(embeds []Embed) {
	if !c.Enabled() {
		return
	}
	payload := map[string]any{"embeds": embeds}
	b, _ := json.Marshal(payload)
	resp, err := c.http.Post(c.webhookURL, "application/json", bytes.NewReader(b))
	if err != nil {
		slog.Warn("discord webhook failed", "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		slog.Warn("discord webhook error response", "status", resp.StatusCode)
	}
}
