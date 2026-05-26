package slack

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	slackapi "github.com/slack-go/slack"
)

const (
	colorGreen  = "#2ECC71"
	colorRed    = "#E74C3C"
	colorOrange = "#E67E22"
	colorBlue   = "#3498DB"
)

func (c *Client) AlertWarn(message, detail string) {
	c.sendAlertAttachments([]slackapi.Attachment{{
		Color:  colorOrange,
		Title:  "경고",
		Text:   message,
		Fields: []slackapi.AttachmentField{{Title: "상세", Value: truncate(detail, 1000)}},
		Ts:     timestamp(),
	}})
}

func (c *Client) AlertError(message, detail string) {
	c.sendAlertAttachments([]slackapi.Attachment{{
		Color:  colorRed,
		Title:  "에러",
		Text:   message,
		Fields: []slackapi.AttachmentField{{Title: "상세", Value: truncate(detail, 1000)}},
		Ts:     timestamp(),
	}})
}

func (c *Client) KISAPIFailure(endpoint, errorCode string, failCount int) {
	c.sendKISAttachments([]slackapi.Attachment{{
		Title: fmt.Sprintf("KIS API %d회 연속 실패", failCount),
		Color: colorRed,
		Fields: []slackapi.AttachmentField{
			{Title: "Endpoint", Value: endpoint, Short: true},
			{Title: "Error Code", Value: errorCode, Short: true},
		},
		Ts: timestamp(),
	}})
}

func (c *Client) TradeBuy(code, name string, price, qty int, reason string) {
	c.sendTradeAttachments([]slackapi.Attachment{{
		Title: fmt.Sprintf("매수 체결 - %s (%s)", name, code),
		Color: colorGreen,
		Fields: []slackapi.AttachmentField{
			{Title: "체결가", Value: fmt.Sprintf("%d원", price), Short: true},
			{Title: "수량", Value: fmt.Sprintf("%d주", qty), Short: true},
			{Title: "금액", Value: fmt.Sprintf("%d원", price*qty), Short: true},
			{Title: "선정 근거", Value: truncate(reason, 500)},
		},
		Ts: timestamp(),
	}})
}

func (c *Client) TradeSell(code, name string, buyPrice, sellPrice, qty int, reason string, profitPct float64) {
	color := colorGreen
	title := "매도 체결"
	if profitPct < 0 {
		color = colorRed
		title = "손실 매도 체결"
	}
	c.sendTradeAttachments([]slackapi.Attachment{{
		Title: fmt.Sprintf("%s - %s (%s)", title, name, code),
		Color: color,
		Fields: []slackapi.AttachmentField{
			{Title: "매도가", Value: fmt.Sprintf("%d원", sellPrice), Short: true},
			{Title: "수량", Value: fmt.Sprintf("%d주", qty), Short: true},
			{Title: "수익률", Value: fmt.Sprintf("%.2f%%", profitPct), Short: true},
			{Title: "매수가", Value: fmt.Sprintf("%d원", buyPrice), Short: true},
			{Title: "손익", Value: fmt.Sprintf("%d원", (sellPrice-buyPrice)*qty), Short: true},
			{Title: "매도 사유", Value: reason, Short: true},
		},
		Ts: timestamp(),
	}})
}

func (c *Client) PositionSnapshot(positions []PositionInfo) {
	if len(positions) == 0 {
		return
	}
	fields := make([]slackapi.AttachmentField, 0, len(positions))
	for _, p := range positions {
		fields = append(fields, slackapi.AttachmentField{
			Title: fmt.Sprintf("%s (%s)", p.Name, p.Code),
			Value: fmt.Sprintf("현재가: %d | 수익률: %.2f%% | 손절: %d | 목표: %d",
				p.CurrentPrice, p.ProfitPct, p.StopPrice, p.TargetPrice),
		})
	}
	c.sendPositionAttachments([]slackapi.Attachment{{
		Title:  "보유 포지션 현황",
		Color:  colorBlue,
		Fields: fields,
		Ts:     timestamp(),
	}})
}

type PositionInfo struct {
	Code         string
	Name         string
	CurrentPrice int
	BuyPrice     int
	TargetPrice  int
	StopPrice    int
	ProfitPct    float64
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func timestamp() json.Number {
	return json.Number(strconv.FormatInt(time.Now().Unix(), 10))
}
