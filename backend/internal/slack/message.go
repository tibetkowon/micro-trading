package slack

import (
	"fmt"
	"time"
)

const (
	colorGreen  = "#2ECC71"
	colorRed    = "#E74C3C"
	colorOrange = "#E67E22"
	colorBlue   = "#3498DB"
)

func (c *Client) AlertWarn(message, detail string) {
	c.sendAlertAttachments([]Attachment{{
		Color:     colorOrange,
		Title:     "경고",
		Text:      message,
		Fields:    []Field{{Title: "상세", Value: truncate(detail, 1000)}},
		Timestamp: time.Now().Unix(),
	}})
}

func (c *Client) AlertError(message, detail string) {
	c.sendAlertAttachments([]Attachment{{
		Color:     colorRed,
		Title:     "에러",
		Text:      message,
		Fields:    []Field{{Title: "상세", Value: truncate(detail, 1000)}},
		Timestamp: time.Now().Unix(),
	}})
}

func (c *Client) KISAPIFailure(endpoint, errorCode string, failCount int) {
	c.sendKISAttachments([]Attachment{{
		Title: fmt.Sprintf("KIS API %d회 연속 실패", failCount),
		Color: colorRed,
		Fields: []Field{
			{Title: "Endpoint", Value: endpoint, Short: true},
			{Title: "Error Code", Value: errorCode, Short: true},
		},
		Timestamp: time.Now().Unix(),
	}})
}

func (c *Client) TradeBuy(code, name string, price, qty int, reason string) {
	c.sendTradeAttachments([]Attachment{{
		Title: fmt.Sprintf("매수 체결 - %s (%s)", name, code),
		Color: colorGreen,
		Fields: []Field{
			{Title: "체결가", Value: fmt.Sprintf("%d원", price), Short: true},
			{Title: "수량", Value: fmt.Sprintf("%d주", qty), Short: true},
			{Title: "금액", Value: fmt.Sprintf("%d원", price*qty), Short: true},
			{Title: "선정 근거", Value: truncate(reason, 500)},
		},
		Timestamp: time.Now().Unix(),
	}})
}

func (c *Client) TradeSell(code, name string, buyPrice, sellPrice, qty int, reason string, profitPct float64) {
	color := colorGreen
	title := "매도 체결"
	if profitPct < 0 {
		color = colorRed
		title = "손실 매도 체결"
	}
	c.sendTradeAttachments([]Attachment{{
		Title: fmt.Sprintf("%s - %s (%s)", title, name, code),
		Color: color,
		Fields: []Field{
			{Title: "매도가", Value: fmt.Sprintf("%d원", sellPrice), Short: true},
			{Title: "수량", Value: fmt.Sprintf("%d주", qty), Short: true},
			{Title: "수익률", Value: fmt.Sprintf("%.2f%%", profitPct), Short: true},
			{Title: "매수가", Value: fmt.Sprintf("%d원", buyPrice), Short: true},
			{Title: "손익", Value: fmt.Sprintf("%d원", (sellPrice-buyPrice)*qty), Short: true},
			{Title: "매도 사유", Value: reason, Short: true},
		},
		Timestamp: time.Now().Unix(),
	}})
}

func (c *Client) PositionSnapshot(positions []PositionInfo) {
	if len(positions) == 0 {
		return
	}
	fields := make([]Field, 0, len(positions))
	for _, p := range positions {
		fields = append(fields, Field{
			Title: fmt.Sprintf("%s (%s)", p.Name, p.Code),
			Value: fmt.Sprintf("현재가: %d | 수익률: %.2f%% | 손절: %d | 목표: %d",
				p.CurrentPrice, p.ProfitPct, p.StopPrice, p.TargetPrice),
		})
	}
	c.sendPositionAttachments([]Attachment{{
		Title:     "보유 포지션 현황",
		Color:     colorBlue,
		Fields:    fields,
		Timestamp: time.Now().Unix(),
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
