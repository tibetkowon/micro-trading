package discord

import (
	"fmt"
	"time"
)

const (
	colorGreen  = 0x2ECC71
	colorRed    = 0xE74C3C
	colorOrange = 0xE67E22
	colorBlue   = 0x3498DB
)

func (c *Client) AlertWarn(message, detail string) {
	c.sendEmbed([]Embed{{
		Title:       "경고",
		Description: message,
		Color:       colorOrange,
		Fields:      []Field{{Name: "상세", Value: truncate(detail, 1000)}},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}})
}

func (c *Client) AlertError(message, detail string) {
	c.sendEmbed([]Embed{{
		Title:       "에러",
		Description: message,
		Color:       colorRed,
		Fields:      []Field{{Name: "상세", Value: truncate(detail, 1000)}},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}})
}

func (c *Client) KISAPIFailure(endpoint, errorCode string, failCount int) {
	c.sendEmbed([]Embed{{
		Title: fmt.Sprintf("KIS API %d회 연속 실패", failCount),
		Color: colorRed,
		Fields: []Field{
			{Name: "Endpoint", Value: endpoint, Inline: true},
			{Name: "Error Code", Value: errorCode, Inline: true},
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}})
}

func (c *Client) TradeBuy(code, name string, price, qty int, reason string) {
	c.sendEmbed([]Embed{{
		Title: fmt.Sprintf("매수 체결 - %s (%s)", name, code),
		Color: colorGreen,
		Fields: []Field{
			{Name: "체결가", Value: fmt.Sprintf("%d원", price), Inline: true},
			{Name: "수량", Value: fmt.Sprintf("%d주", qty), Inline: true},
			{Name: "금액", Value: fmt.Sprintf("%d원", price*qty), Inline: true},
			{Name: "선정 근거", Value: truncate(reason, 500)},
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}})
}

func (c *Client) TradeSell(code, name string, buyPrice, sellPrice, qty int, reason string, profitPct float64) {
	color := colorGreen
	title := "매도 체결"
	if profitPct < 0 {
		color = colorRed
		title = "손실 매도 체결"
	}
	c.sendEmbed([]Embed{{
		Title: fmt.Sprintf("%s - %s (%s)", title, name, code),
		Color: color,
		Fields: []Field{
			{Name: "매도가", Value: fmt.Sprintf("%d원", sellPrice), Inline: true},
			{Name: "수량", Value: fmt.Sprintf("%d주", qty), Inline: true},
			{Name: "수익률", Value: fmt.Sprintf("%.2f%%", profitPct), Inline: true},
			{Name: "매수가", Value: fmt.Sprintf("%d원", buyPrice), Inline: true},
			{Name: "손익", Value: fmt.Sprintf("%d원", (sellPrice-buyPrice)*qty), Inline: true},
			{Name: "매도 사유", Value: reason, Inline: true},
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}})
}

func (c *Client) PositionSnapshot(positions []PositionInfo) {
	if len(positions) == 0 {
		return
	}
	fields := make([]Field, 0, len(positions))
	for _, p := range positions {
		fields = append(fields, Field{
			Name: fmt.Sprintf("%s (%s)", p.Name, p.Code),
			Value: fmt.Sprintf("현재가: %d | 수익률: %.2f%% | 손절: %d | 목표: %d",
				p.CurrentPrice, p.ProfitPct, p.StopPrice, p.TargetPrice),
		})
	}
	c.sendEmbed([]Embed{{
		Title:     "보유 포지션 현황",
		Color:     colorBlue,
		Fields:    fields,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
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
