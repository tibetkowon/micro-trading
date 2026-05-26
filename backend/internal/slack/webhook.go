package slack

import (
	"log/slog"

	slackapi "github.com/slack-go/slack"
)

type Client struct {
	api      *slackapi.Client
	token    string
	channels Channels
}

type Channels struct {
	Default  string
	Alert    string
	KIS      string
	Trade    string
	Position string
}

func New(token string) *Client {
	return NewWithChannels(token, Channels{Default: ""})
}

func NewWithChannels(token string, channels Channels) *Client {
	return newWithChannels(token, channels)
}

func newWithChannels(token string, channels Channels, options ...slackapi.Option) *Client {
	return &Client{
		api:      slackapi.New(token, options...),
		token:    token,
		channels: channels,
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.token != "" && (c.channels.Default != "" ||
		c.channels.Alert != "" ||
		c.channels.KIS != "" ||
		c.channels.Trade != "" ||
		c.channels.Position != "")
}

func (c *Client) PositionEnabled() bool {
	return c != nil && c.token != "" && c.routeChannel(c.channels.Position) != ""
}

func (c *Client) sendAttachments(attachments []slackapi.Attachment) {
	c.send(c.channels.Default, attachments)
}

func (c *Client) sendAlertAttachments(attachments []slackapi.Attachment) {
	c.send(c.channels.Alert, attachments)
}

func (c *Client) sendKISAttachments(attachments []slackapi.Attachment) {
	c.send(c.channels.KIS, attachments)
}

func (c *Client) sendTradeAttachments(attachments []slackapi.Attachment) {
	c.send(c.channels.Trade, attachments)
}

func (c *Client) sendPositionAttachments(attachments []slackapi.Attachment) {
	c.send(c.channels.Position, attachments)
}

func (c *Client) send(channelID string, attachments []slackapi.Attachment) {
	if c == nil || c.api == nil || c.token == "" {
		return
	}
	channelID = c.routeChannel(channelID)
	if channelID == "" {
		return
	}
	if _, _, err := c.api.PostMessage(channelID, slackapi.MsgOptionAttachments(attachments...)); err != nil {
		slog.Warn("slack webhook failed", "error", err)
	}
}

func (c *Client) routeChannel(channelID string) string {
	if c == nil {
		return ""
	}
	if channelID != "" {
		return channelID
	}
	return c.channels.Default
}
