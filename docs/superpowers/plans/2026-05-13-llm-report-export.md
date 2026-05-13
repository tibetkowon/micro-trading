# LLM Report Export 구현 플랜

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 특정 기간의 일별/거래별 리포트와 현재 설정값 목록을 LLM에 전달하기 좋은 구조화된 JSON으로 내보내는 API와 프론트엔드 UI를 구현한다.

**Architecture:** 기존 `daily_reports` / `trade_reports` Firestore 컬렉션 + `settings/config` 를 조합해 단일 JSON 응답을 반환하는 새 엔드포인트를 추가한다. 프론트엔드에서는 DailyReports 페이지에 기간 선택 + Export 버튼을 붙이고, JSON 다운로드 또는 클립보드 복사를 지원한다.

**Tech Stack:** Go 1.26.1 (Gin, Firestore SDK), React 18, Vite

---

## 파일 맵

| 역할 | 파일 경로 | 상태 |
|---|---|---|
| Export 집계 로직 | `backend/internal/report/export.go` | 신규 |
| Export 로직 테스트 | `backend/internal/report/export_test.go` | 신규 |
| DB 날짜 범위 쿼리 | `backend/internal/database/db.go` (기존 파일 수정) | 수정 |
| HTTP 핸들러 | `backend/internal/api/handlers.go` (기존 파일 수정) | 수정 |
| 라우터 등록 | `backend/internal/api/router.go` (기존 파일 수정) | 수정 |
| 프론트엔드 Export UI | `frontend/src/pages/DailyReports.jsx` (기존 파일 수정) | 수정 |

---

## Task 1: DB 날짜 범위 쿼리 메서드 추가

**Files:**
- Modify: `backend/internal/database/db.go` (기존 `ListDailyReports` 아래, ~line 1010 이후)

- [ ] **Step 1: 실패하는 테스트 작성**

`backend/internal/report/export_test.go` 를 신규 생성:

```go
package report_test

import (
	"testing"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/models"
)

// filterDailyReportsByRange 는 날짜 범위로 DailyReport 슬라이스를 필터링한다.
func filterDailyReportsByRange(reports []models.DailyReport, from, to string) []models.DailyReport {
	var out []models.DailyReport
	for _, r := range reports {
		if r.Date >= from && r.Date <= to {
			out = append(out, r)
		}
	}
	return out
}

func TestFilterDailyReportsByRange(t *testing.T) {
	reports := []models.DailyReport{
		{Date: "2026-05-10", TotalTrades: 3},
		{Date: "2026-05-11", TotalTrades: 5},
		{Date: "2026-05-12", TotalTrades: 2},
		{Date: "2026-05-13", TotalTrades: 4},
	}
	got := filterDailyReportsByRange(reports, "2026-05-11", "2026-05-12")
	if len(got) != 2 {
		t.Fatalf("want 2 reports, got %d", len(got))
	}
	if got[0].Date != "2026-05-11" || got[1].Date != "2026-05-12" {
		t.Errorf("unexpected dates: %v %v", got[0].Date, got[1].Date)
	}
}

func TestBuildExportSummary(t *testing.T) {
	_ = time.Now() // time import keep
	trades := []models.TradeReport{
		{ProfitAmount: 10000, ProfitPct: 2.0, SoldAt: ptr(time.Now())},
		{ProfitAmount: -5000, ProfitPct: -1.0, SoldAt: ptr(time.Now())},
		{ProfitAmount: 3000, ProfitPct: 1.5, SoldAt: ptr(time.Now())},
	}
	sum := buildExportSummary(trades, "2026-05-11", "2026-05-13")
	if sum.TotalTrades != 3 {
		t.Errorf("want TotalTrades=3, got %d", sum.TotalTrades)
	}
	if sum.WinningTrades != 2 {
		t.Errorf("want WinningTrades=2, got %d", sum.WinningTrades)
	}
	if sum.TotalProfitAmount != 8000 {
		t.Errorf("want TotalProfitAmount=8000, got %f", sum.TotalProfitAmount)
	}
}

func ptr[T any](v T) *T { return &v }
```

- [ ] **Step 2: 테스트 실행 → FAIL 확인**

```bash
cd backend && go test ./internal/report/... 2>&1 | head -30
```
Expected: `undefined: buildExportSummary`

- [ ] **Step 3: `export.go` 신규 생성**

`backend/internal/report/export.go`:

```go
package report

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/models"
)

// ExportReport 는 LLM에 전달하기 위한 기간별 종합 리포트다.
type ExportReport struct {
	GeneratedAt     string                 `json:"generated_at"`
	Period          ExportPeriod           `json:"period"`
	Summary         ExportSummary          `json:"summary"`
	DailyReports    []ExportDailyEntry     `json:"daily_reports"`
	Trades          []ExportTradeEntry     `json:"trades"`
	CurrentSettings map[string]interface{} `json:"current_settings"`
	SettingsGuide   []SettingsFieldInfo    `json:"settings_guide"`
}

type ExportPeriod struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type ExportSummary struct {
	TotalTrades       int     `json:"total_trades"`
	WinningTrades     int     `json:"winning_trades"`
	LosingTrades      int     `json:"losing_trades"`
	WinRatePct        float64 `json:"win_rate_pct"`
	TotalProfitAmount float64 `json:"total_profit_amount_krw"`
	AvgProfitPct      float64 `json:"avg_profit_pct"`
}

type ExportDailyEntry struct {
	Date          string  `json:"date"`
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	LosingTrades  int     `json:"losing_trades"`
	WinRatePct    float64 `json:"win_rate_pct"`
	TotalPnl      float64 `json:"total_pnl_krw"`
	AvgPnlPct     float64 `json:"avg_pnl_pct"`
}

type ExportTradeEntry struct {
	Date           string  `json:"date"`
	StockCode      string  `json:"stock_code"`
	StockName      string  `json:"stock_name"`
	BuyPrice       float64 `json:"buy_price"`
	SellPrice      float64 `json:"sell_price"`
	Qty            int     `json:"qty"`
	ProfitAmountKRW float64 `json:"profit_amount_krw"`
	ProfitPct      float64 `json:"profit_pct"`
	SellReason     string  `json:"sell_reason"`
	BuyIndicators  map[string]interface{} `json:"buy_indicators"`
	HoldingMinutes float64 `json:"holding_minutes"`
}

// SettingsFieldInfo 는 LLM이 이해할 수 있도록 설정 필드의 설명을 제공한다.
type SettingsFieldInfo struct {
	Key         string      `json:"key"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	CurrentValue interface{} `json:"current_value"`
	MinValue    interface{} `json:"min_value,omitempty"`
	MaxValue    interface{} `json:"max_value,omitempty"`
}

// GenerateExportReport 는 from~to 기간의 거래 데이터와 현재 설정을 집계해 반환한다.
// from, to 는 "YYYY-MM-DD" 형식이다.
func GenerateExportReport(ctx context.Context, db *database.DB, from, to string) (*ExportReport, error) {
	// 날짜 유효성 검사
	if from > to {
		return nil, fmt.Errorf("from(%s) must be <= to(%s)", from, to)
	}

	// 기간 내 daily reports 조회
	dailyReports, err := db.ListDailyReportsByDateRange(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("fetch daily reports: %w", err)
	}

	// 기간 내 trade reports 조회 (날짜별로 조회해서 합침)
	var allTrades []models.TradeReport
	for d := from; d <= to; d = nextDate(d) {
		trades, err := db.GetCompletedTradesBySoldDate(ctx, d)
		if err != nil {
			continue
		}
		allTrades = append(allTrades, trades...)
	}

	// 현재 설정 조회
	settings, err := db.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch settings: %w", err)
	}

	return &ExportReport{
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		Period:          ExportPeriod{From: from, To: to},
		Summary:         buildExportSummary(allTrades, from, to),
		DailyReports:    buildDailyEntries(dailyReports),
		Trades:          buildTradeEntries(allTrades),
		CurrentSettings: buildSettingsMap(settings),
		SettingsGuide:   buildSettingsGuide(settings),
	}, nil
}

func buildExportSummary(trades []models.TradeReport, from, to string) ExportSummary {
	var winning, total int
	var totalPnl, totalPnlPct float64
	for _, t := range trades {
		total++
		if t.ProfitAmount > 0 {
			winning++
		}
		totalPnl += t.ProfitAmount
		totalPnlPct += t.ProfitPct
	}
	var winRate, avgPnlPct float64
	if total > 0 {
		winRate = float64(winning) / float64(total) * 100
		avgPnlPct = totalPnlPct / float64(total)
	}
	return ExportSummary{
		TotalTrades:       total,
		WinningTrades:     winning,
		LosingTrades:      total - winning,
		WinRatePct:        round2(winRate),
		TotalProfitAmount: totalPnl,
		AvgProfitPct:      round2(avgPnlPct),
	}
}

func buildDailyEntries(reports []models.DailyReport) []ExportDailyEntry {
	out := make([]ExportDailyEntry, 0, len(reports))
	for _, r := range reports {
		var wr float64
		if r.TotalTrades > 0 {
			wr = float64(r.WinningTrades) / float64(r.TotalTrades) * 100
		}
		out = append(out, ExportDailyEntry{
			Date:          r.Date,
			TotalTrades:   r.TotalTrades,
			WinningTrades: r.WinningTrades,
			LosingTrades:  r.LosingTrades,
			WinRatePct:    round2(wr),
			TotalPnl:      r.TotalProfitAmount,
			AvgPnlPct:     round2(r.AvgProfitPct),
		})
	}
	return out
}

func buildTradeEntries(trades []models.TradeReport) []ExportTradeEntry {
	out := make([]ExportTradeEntry, 0, len(trades))
	for _, t := range trades {
		var holdMin float64
		if t.SoldAt != nil && !t.CreatedAt.IsZero() {
			holdMin = t.SoldAt.Sub(t.CreatedAt).Minutes()
		}
		var indicators map[string]interface{}
		if t.BuyIndicators != "" {
			_ = json.Unmarshal([]byte(t.BuyIndicators), &indicators)
		}
		out = append(out, ExportTradeEntry{
			Date:           t.Date,
			StockCode:      t.StockCode,
			StockName:      t.StockName,
			BuyPrice:       t.BuyPrice,
			SellPrice:      t.SellPrice,
			Qty:            t.BuyQty,
			ProfitAmountKRW: t.ProfitAmount,
			ProfitPct:      t.ProfitPct,
			SellReason:     t.SellReason,
			BuyIndicators:  indicators,
			HoldingMinutes: round2(holdMin),
		})
	}
	return out
}

func buildSettingsMap(s *database.TradingSettings) map[string]interface{} {
	if s == nil {
		return nil
	}
	return map[string]interface{}{
		"take_profit_pct":            s.TakeProfitPct,
		"stop_loss_pct":              s.StopLossPct,
		"max_positions":              s.MaxPositions,
		"order_amount_pct":           s.OrderAmountPct,
		"trailing_trigger_pct":       s.TrailingTriggerPct,
		"trailing_stop_pct":          s.TrailingStopPct,
		"stagnation_threshold_pct":   s.StagnationThresholdPct,
		"stagnation_duration_min":    s.StagnationDurationMin,
		"min_score_threshold":        s.MinScoreThreshold,
		"hard_rsi_max":               s.HardRSIMax,
		"hard_strength_min":          s.HardStrengthMin,
		"max_consecutive_losses":     s.MaxConsecutiveLosses,
		"daily_max_loss_pct":         s.DailyMaxLossPct,
		"score_weight_strength":      s.ScoreWeightStrength,
		"score_weight_rsi":           s.ScoreWeightRSI,
		"score_weight_macd":          s.ScoreWeightMACD,
		"score_weight_bidask":        s.ScoreWeightBidAsk,
		"score_weight_vwap":          s.ScoreWeightVWAP,
		"score_weight_volume":        s.ScoreWeightVolume,
		"score_weight_program_buy":   s.ScoreWeightProgramBuy,
		"score_weight_micro_bidask":  s.ScoreWeightMicroBidAsk,
		"score_weight_vi_disparity":  s.ScoreWeightVIDisparity,
		"ranking_top_n":              s.RankingTopN,
		"ranking_condition":          s.RankingCondition,
		"sell_on_upper_limit":        s.SellOnUpperLimit,
		"block_reentry_on_loss":      s.BlockReentryOnLoss,
		"reentry_cooldown_min":       s.ReentryCooldownMin,
		"reentry_score_penalty":      s.ReentryScorePenalty,
	}
}

func buildSettingsGuide(s *database.TradingSettings) []SettingsFieldInfo {
	type fieldDef struct {
		key, desc, typ string
		val             interface{}
		min, max        interface{}
	}
	var cur database.TradingSettings
	if s != nil {
		cur = *s
	}
	defs := []fieldDef{
		{"take_profit_pct", "목표 수익률: 이 비율 이상 수익 시 자동 매도", "float", cur.TakeProfitPct, 0.1, 20.0},
		{"stop_loss_pct", "손절 기준: 이 비율 이상 손실 시 자동 매도", "float", cur.StopLossPct, 0.1, 10.0},
		{"trailing_trigger_pct", "트레일링 스탑 활성화 수익률 (0=비활성)", "float", cur.TrailingTriggerPct, 0.0, 5.0},
		{"trailing_stop_pct", "트레일링 스탑 허용 하락폭 (%)", "float", cur.TrailingStopPct, 0.1, 3.0},
		{"stagnation_threshold_pct", "횡보 판단 가격변동 기준 (0=비활성)", "float", cur.StagnationThresholdPct, 0.0, 1.0},
		{"stagnation_duration_min", "횡보로 인정하는 최소 지속시간 (분)", "int", cur.StagnationDurationMin, 1, 30},
		{"min_score_threshold", "매수 진입 최소 종합 점수 (0~100)", "float", cur.MinScoreThreshold, 0.0, 100.0},
		{"hard_rsi_max", "매수 허용 RSI 최댓값 (초과 시 거부)", "float", cur.HardRSIMax, 50.0, 90.0},
		{"hard_strength_min", "매수 허용 체결강도 최솟값", "float", cur.HardStrengthMin, 80.0, 300.0},
		{"max_positions", "동시 보유 최대 포지션 수", "int", cur.MaxPositions, 1, 10},
		{"order_amount_pct", "주문당 사용 가용현금 비율 (%)", "float", cur.OrderAmountPct, 10.0, 100.0},
		{"max_consecutive_losses", "연속 손실 허용 횟수 (0=비활성)", "int", cur.MaxConsecutiveLosses, 0, 20},
		{"daily_max_loss_pct", "일일 최대 손실률 (0=비활성)", "float", cur.DailyMaxLossPct, 0.0, 10.0},
		{"score_weight_strength", "점수: 체결강도 가중치 (%)", "int", cur.ScoreWeightStrength, 0, 100},
		{"score_weight_rsi", "점수: RSI 가중치 (%)", "int", cur.ScoreWeightRSI, 0, 100},
		{"score_weight_macd", "점수: MACD 가중치 (%)", "int", cur.ScoreWeightMACD, 0, 100},
		{"score_weight_bidask", "점수: 매수/매도 잔량비 가중치 (%)", "int", cur.ScoreWeightBidAsk, 0, 100},
		{"score_weight_vwap", "점수: VWAP 괴리율 가중치 (%)", "int", cur.ScoreWeightVWAP, 0, 100},
		{"score_weight_volume", "점수: 거래량 증가율 가중치 (%)", "int", cur.ScoreWeightVolume, 0, 100},
		{"score_weight_program_buy", "점수: 프로그램 매수 가중치 (%)", "int", cur.ScoreWeightProgramBuy, 0, 100},
		{"score_weight_micro_bidask", "점수: 미시 호가 잔량비 가중치 (%)", "int", cur.ScoreWeightMicroBidAsk, 0, 100},
		{"score_weight_vi_disparity", "점수: VI 괴리율 가중치 (%)", "int", cur.ScoreWeightVIDisparity, 0, 100},
	}
	out := make([]SettingsFieldInfo, 0, len(defs))
	for _, d := range defs {
		out = append(out, SettingsFieldInfo{
			Key:         d.key,
			Description: d.desc,
			Type:        d.typ,
			CurrentValue: d.val,
			MinValue:    d.min,
			MaxValue:    d.max,
		})
	}
	return out
}

// nextDate 는 "YYYY-MM-DD" 형식의 날짜를 하루 증가시킨다.
func nextDate(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02")
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
```

- [ ] **Step 4: `db.go` 에 `ListDailyReportsByDateRange` 추가**

`backend/internal/database/db.go` 의 `ListDailyReports` 함수 아래에 추가:

```go
// ListDailyReportsByDateRange returns daily reports within [from, to] date range (inclusive, YYYY-MM-DD).
func (db *DB) ListDailyReportsByDateRange(ctx context.Context, from, to string) ([]models.DailyReport, error) {
	iter := db.client.Collection(colDailyRpt).
		Where("date", ">=", from).
		Where("date", "<=", to).
		OrderBy("date", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()
	var reports []models.DailyReport
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var dr models.DailyReport
		if err := snap.DataTo(&dr); err != nil {
			continue
		}
		reports = append(reports, dr)
	}
	return reports, nil
}
```

- [ ] **Step 5: 테스트 실행 → PASS 확인**

```bash
cd backend && go test ./internal/report/... -v 2>&1
```
Expected: `PASS TestFilterDailyReportsByRange`, `PASS TestBuildExportSummary`

- [ ] **Step 6: 빌드 확인**

```bash
cd backend && go build ./...
```

- [ ] **Step 7: 커밋**

```bash
git add backend/internal/report/export.go backend/internal/report/export_test.go backend/internal/database/db.go
git commit -m "feat(report): add LLM export aggregation logic and date-range DB query"
```

---

## Task 2: HTTP 핸들러 & 라우터 등록

**Files:**
- Modify: `backend/internal/api/handlers.go` (파일 말미에 추가)
- Modify: `backend/internal/api/router.go`

- [ ] **Step 1: 핸들러 추가**

`handlers.go` 파일 말미에 추가:

```go
// HandleExportReport handles GET /api/reports/export?from=YYYY-MM-DD&to=YYYY-MM-DD
func (h *Handler) HandleExportReport(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	if from == "" || to == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to query params required (YYYY-MM-DD)"})
		return
	}
	// 최대 90일 제한 (Firestore 비용 보호)
	fromT, err := time.Parse("2006-01-02", from)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from date"})
		return
	}
	toT, err := time.Parse("2006-01-02", to)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to date"})
		return
	}
	if toT.Sub(fromT) > 90*24*time.Hour {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date range must be 90 days or less"})
		return
	}

	result, err := report.GenerateExportReport(c.Request.Context(), h.db, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
```

참고: `handlers.go` 상단 import 에 `"github.com/micro-trading-for-agent/backend/internal/report"` 가 없으면 추가.

- [ ] **Step 2: 라우터에 엔드포인트 등록**

`router.go` 에서 기존 `reports` 그룹 안에 추가:

```go
// 기존:
// reports.GET("/trades", h.HandleTradeReports)
// reports.GET("/daily", h.HandleDailyReports)
// reports.POST("/daily/generate", h.HandleGenerateDailyReport)
// 추가:
reports.GET("/export", h.HandleExportReport)
```

- [ ] **Step 3: 빌드 확인**

```bash
cd backend && go build ./...
```

- [ ] **Step 4: 수동 스모크 테스트**

백엔드 서버가 실행 중이라면:
```bash
curl "http://localhost:8080/api/reports/export?from=2026-05-01&to=2026-05-13" | jq '.summary'
```
Expected: `{"total_trades": N, "winning_trades": N, ...}` 형식의 JSON

- [ ] **Step 5: 커밋**

```bash
git add backend/internal/api/handlers.go backend/internal/api/router.go
git commit -m "feat(api): add GET /api/reports/export endpoint for LLM data export"
```

---

## Task 3: 프론트엔드 Export UI

**Files:**
- Modify: `frontend/src/pages/DailyReports.jsx`

- [ ] **Step 1: Export 상태와 핸들러 추가**

`DailyReports.jsx` 의 기존 state 선언부 근처(컴포넌트 함수 내부, `useState` 들이 모여있는 곳)에 추가:

```jsx
const [exportModal, setExportModal] = useState(false);
const [exportFrom, setExportFrom] = useState('');
const [exportTo, setExportTo] = useState('');
const [exportLoading, setExportLoading] = useState(false);
const [exportError, setExportError] = useState('');
```

- [ ] **Step 2: Export 핸들러 함수 추가**

컴포넌트 함수 내부, 기존 fetch 함수들 근처에 추가:

```jsx
const handleExport = async (action) => {
  if (!exportFrom || !exportTo) {
    setExportError('시작일과 종료일을 모두 입력해주세요.');
    return;
  }
  setExportLoading(true);
  setExportError('');
  try {
    const res = await fetch(
      `${import.meta.env.VITE_API_BASE_URL}/api/reports/export?from=${exportFrom}&to=${exportTo}`
    );
    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || '내보내기 실패');
    }
    const data = await res.json();
    const json = JSON.stringify(data, null, 2);

    if (action === 'download') {
      const blob = new Blob([json], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `trading-report-${exportFrom}-${exportTo}.json`;
      a.click();
      URL.revokeObjectURL(url);
    } else {
      await navigator.clipboard.writeText(json);
      alert('클립보드에 복사되었습니다.');
    }
    setExportModal(false);
  } catch (e) {
    setExportError(e.message);
  } finally {
    setExportLoading(false);
  }
};
```

- [ ] **Step 3: Export 버튼 추가 (헤더 영역)**

`DailyReports.jsx` 에서 페이지 제목이나 날짜 필터 근처에 Export 버튼 추가:

```jsx
{/* 기존 헤더/필터 UI 끝 부분에 추가 */}
<button
  onClick={() => setExportModal(true)}
  className="px-3 py-1.5 text-sm bg-indigo-600 text-white rounded hover:bg-indigo-700"
>
  LLM Export
</button>
```

- [ ] **Step 4: Export 모달 추가**

컴포넌트 return JSX 의 최상위 div 내부, 다른 모달들과 함께:

```jsx
{exportModal && (
  <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
    <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-96 space-y-4">
      <h2 className="text-lg font-semibold">LLM Export</h2>
      <p className="text-sm text-gray-500">
        선택한 기간의 거래 리포트와 현재 설정을 JSON으로 내보냅니다.
      </p>
      <div className="space-y-2">
        <label className="block text-sm font-medium">시작일</label>
        <input
          type="date"
          value={exportFrom}
          onChange={e => setExportFrom(e.target.value)}
          className="w-full border rounded px-3 py-1.5 text-sm dark:bg-gray-700 dark:border-gray-600"
        />
      </div>
      <div className="space-y-2">
        <label className="block text-sm font-medium">종료일</label>
        <input
          type="date"
          value={exportTo}
          onChange={e => setExportTo(e.target.value)}
          className="w-full border rounded px-3 py-1.5 text-sm dark:bg-gray-700 dark:border-gray-600"
        />
      </div>
      {exportError && (
        <p className="text-sm text-red-500">{exportError}</p>
      )}
      <div className="flex gap-2 pt-2">
        <button
          onClick={() => handleExport('download')}
          disabled={exportLoading}
          className="flex-1 py-2 bg-indigo-600 text-white rounded text-sm hover:bg-indigo-700 disabled:opacity-50"
        >
          {exportLoading ? '처리 중...' : 'JSON 다운로드'}
        </button>
        <button
          onClick={() => handleExport('copy')}
          disabled={exportLoading}
          className="flex-1 py-2 bg-gray-600 text-white rounded text-sm hover:bg-gray-700 disabled:opacity-50"
        >
          클립보드 복사
        </button>
        <button
          onClick={() => { setExportModal(false); setExportError(''); }}
          className="px-3 py-2 border rounded text-sm hover:bg-gray-100 dark:hover:bg-gray-700"
        >
          취소
        </button>
      </div>
    </div>
  </div>
)}
```

- [ ] **Step 5: 프론트엔드 빌드 확인**

```bash
cd frontend && npm run build 2>&1 | tail -10
```
Expected: `✓ built in`

- [ ] **Step 6: 커밋**

```bash
git add frontend/src/pages/DailyReports.jsx
git commit -m "feat(ui): add LLM Export button and modal to DailyReports page"
```

---

## Verification

1. `cd backend && go test ./internal/report/... -v` → 2개 테스트 PASS
2. `cd backend && go build ./...` → 빌드 에러 없음
3. `cd frontend && npm run build` → 빌드 에러 없음
4. 서버 실행 후 `curl "http://localhost:8080/api/reports/export?from=YYYY-MM-DD&to=YYYY-MM-DD"` → JSON 응답 확인
5. 프론트엔드 DailyReports 페이지에서 "LLM Export" 버튼 클릭 → 모달 열림 → 날짜 선택 → 다운로드/복사 동작 확인
