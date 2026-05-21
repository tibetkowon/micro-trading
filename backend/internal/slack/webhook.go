package slack

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type Client struct {
	webhooks Webhooks
	http     *http.Client
}

type Webhooks struct {
	Default  string
	Alert    string
	KIS      string
	Trade    string
	Position string
}

func New(webhookURL string) *Client {
	return NewWithWebhooks(Webhooks{Default: webhookURL})
}

func NewWithWebhooks(webhooks Webhooks) *Client {
	return &Client{
		webhooks: webhooks,
		http:     &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && (c.webhooks.Default != "" ||
		c.webhooks.Alert != "" ||
		c.webhooks.KIS != "" ||
		c.webhooks.Trade != "" ||
		c.webhooks.Position != "")
}

func (c *Client) PositionEnabled() bool {
	return c != nil && c.routeURL(c.webhooks.Position) != ""
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
	c.sendAttachmentsTo(c.webhooks.Default, attachments)
}

func (c *Client) sendAlertAttachments(attachments []Attachment) {
	c.sendAttachmentsTo(c.webhooks.Alert, attachments)
}

func (c *Client) sendKISAttachments(attachments []Attachment) {
	c.sendAttachmentsTo(c.webhooks.KIS, attachments)
}

func (c *Client) sendTradeAttachments(attachments []Attachment) {
	c.sendAttachmentsTo(c.webhooks.Trade, attachments)
}

func (c *Client) sendPositionAttachments(attachments []Attachment) {
	c.sendAttachmentsTo(c.webhooks.Position, attachments)
}

func (c *Client) sendAttachmentsTo(webhookURL string, attachments []Attachment) {
	webhookURL = c.routeURL(webhookURL)
	if webhookURL == "" {
		return
	}
	payload := map[string]any{"attachments": attachments}
	b, _ := json.Marshal(payload)
	resp, err := c.http.Post(webhookURL, "application/json", bytes.NewReader(b))
	if err != nil {
		slog.Warn("slack webhook failed", "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		slog.Warn("slack webhook error response", "status", resp.StatusCode)
	}
}

func (c *Client) routeURL(webhookURL string) string {
	if c == nil {
		return ""
	}
	if webhookURL != "" {
		return webhookURL
	}
	return c.webhooks.Default
}
