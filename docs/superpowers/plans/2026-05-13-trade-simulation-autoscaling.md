# 당일 거래 시뮬레이션 & 설정 오토스케일링 구현 플랜

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 장 종료 후 당일 거래 종목의 1분봉을 조회해 여러 파라미터 시나리오(목표가/손절/트레일링/횡보 기준 변동)로 가상 재시뮬레이션하고, 가장 높은 수익을 냈을 설정을 추천/적용하는 기능을 구현한다.

**Architecture:**  
장 종료 스케줄러(15:25 KST)에서 트리거 → 당일 체결 거래별 1분봉 조회(KIS API `GetDayMinuteChart`) → 감도 분석(파라미터 하나씩 변화, 나머지 현재값 유지) → 결과를 `simulation_results` Firestore 컬렉션에 저장 → 프론트엔드 DailyReports 새 탭에서 시나리오 비교 표시 + "이 설정 적용" 버튼.  
단일 파라미터 감도 분석(1D sweep)을 선택해 조합 폭발을 방지하고, 저사양 서버에서도 수십 초 내 완료되도록 한다.

**Tech Stack:** Go 1.26.1 (Gin, Firestore SDK), React 18, Vite  
**재사용 함수:**  
- `kis.Client.GetDayMinuteChart(ctx, code, date, inputHour)` — `backend/internal/kis/chart.go:81`  
- `db.GetCompletedTradesBySoldDate(ctx, date)` — `backend/internal/database/db.go:899`  
- `PATCH /api/settings` — 기존 설정 저장 엔드포인트

---

## 파일 맵

| 역할 | 파일 경로 | 상태 |
|---|---|---|
| 시뮬레이션 엔진 (순수 계산) | `backend/internal/simulation/simulator.go` | 신규 |
| 시나리오 정의 & 감도 분석 | `backend/internal/simulation/scenarios.go` | 신규 |
| 시뮬레이션 결과 Firestore CRUD | `backend/internal/database/db.go` (기존 파일 수정) | 수정 |
| 시뮬레이션 모델 | `backend/internal/models/models.go` (기존 파일 수정) | 수정 |
| 시뮬레이터 테스트 | `backend/internal/simulation/simulator_test.go` | 신규 |
| 시나리오 테스트 | `backend/internal/simulation/scenarios_test.go` | 신규 |
| 장 종료 후 시뮬레이션 실행 | `backend/cmd/server/main.go` (기존 파일 수정) | 수정 |
| HTTP 핸들러 | `backend/internal/api/handlers.go` (기존 파일 수정) | 수정 |
| 라우터 등록 | `backend/internal/api/router.go` (기존 파일 수정) | 수정 |
| 프론트엔드 시뮬레이션 탭 | `frontend/src/pages/DailyReports.jsx` (기존 파일 수정) | 수정 |

---

## Task 1: 시뮬레이션 엔진 (순수 계산 함수)

**Files:**
- Create: `backend/internal/simulation/simulator.go`
- Create: `backend/internal/simulation/simulator_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

`backend/internal/simulation/simulator_test.go`:

```go
package simulation_test

import (
	"testing"

	"github.com/micro-trading-for-agent/backend/internal/simulation"
)

func TestSimulateTrade_HitsTarget(t *testing.T) {
	// 매수가 10000, 목표가 +3% = 10300
	// 1분봉에서 고가 10350 등장 → 목표가 10300 에서 청산
	candles := []simulation.MinuteCandle{
		{High: 10050, Low: 9980, Close: 10050},
		{High: 10350, Low: 10100, Close: 10300}, // 목표 도달
	}
	params := simulation.SimParams{
		TakeProfitPct:      3.0,
		StopLossPct:        2.0,
		TrailingTriggerPct: 0,
		TrailingStopPct:    0,
	}
	result := simulation.SimulateTrade(10000, candles, params)
	if result.ExitReason != "target" {
		t.Errorf("want exit reason 'target', got %q", result.ExitReason)
	}
	if result.ExitPrice != 10300 {
		t.Errorf("want exit price 10300, got %f", result.ExitPrice)
	}
}

func TestSimulateTrade_HitsStop(t *testing.T) {
	// 매수가 10000, 손절가 -2% = 9800
	candles := []simulation.MinuteCandle{
		{High: 10020, Low: 9980, Close: 9990},
		{High: 9900, Low: 9750, Close: 9800}, // 손절 도달
	}
	params := simulation.SimParams{
		TakeProfitPct: 3.0,
		StopLossPct:   2.0,
	}
	result := simulation.SimulateTrade(10000, candles, params)
	if result.ExitReason != "stop" {
		t.Errorf("want exit reason 'stop', got %q", result.ExitReason)
	}
	if result.ExitPrice != 9800 {
		t.Errorf("want exit price 9800, got %f", result.ExitPrice)
	}
}

func TestSimulateTrade_TrailingStop(t *testing.T) {
	// 매수가 10000, 트레일링 활성화 +1.5% = 10150
	// 고가 10300 도달 후 -0.7% 하락 → 10300*(1-0.007) = 10228 에서 청산
	candles := []simulation.MinuteCandle{
		{High: 10200, Low: 10100, Close: 10150}, // +1.5% 초과 → trailing 활성화
		{High: 10300, Low: 10150, Close: 10250}, // 신고가 갱신
		{High: 10250, Low: 10200, Close: 10220}, // 고가 10300 기준 -0.7% = 10228 미만
	}
	params := simulation.SimParams{
		TakeProfitPct:      5.0,
		StopLossPct:        2.0,
		TrailingTriggerPct: 1.5,
		TrailingStopPct:    0.7,
	}
	result := simulation.SimulateTrade(10000, candles, params)
	if result.ExitReason != "trailing" {
		t.Errorf("want exit reason 'trailing', got %q", result.ExitReason)
	}
}

func TestSimulateTrade_NoExit(t *testing.T) {
	// 어떤 조건도 충족하지 않으면 마지막 종가로 청산
	candles := []simulation.MinuteCandle{
		{High: 10020, Low: 9990, Close: 10010},
		{High: 10030, Low: 10000, Close: 10020},
	}
	params := simulation.SimParams{TakeProfitPct: 5.0, StopLossPct: 2.0}
	result := simulation.SimulateTrade(10000, candles, params)
	if result.ExitReason != "end_of_data" {
		t.Errorf("want exit reason 'end_of_data', got %q", result.ExitReason)
	}
	if result.ExitPrice != 10020 {
		t.Errorf("want last close 10020, got %f", result.ExitPrice)
	}
}
```

- [ ] **Step 2: 테스트 실행 → FAIL 확인**

```bash
cd backend && go test ./internal/simulation/... 2>&1 | head -20
```
Expected: `cannot find package`

- [ ] **Step 3: `simulator.go` 구현**

`backend/internal/simulation/simulator.go`:

```go
package simulation

import "math"

// MinuteCandle 은 1분봉 OHLC 데이터다.
type MinuteCandle struct {
	High  float64
	Low   float64
	Close float64
}

// SimParams 는 시뮬레이션에 사용할 파라미터 세트다.
type SimParams struct {
	TakeProfitPct      float64 // 목표 수익률 (%)
	StopLossPct        float64 // 손절률 (%)
	TrailingTriggerPct float64 // 트레일링 활성화 수익률 (0=비활성)
	TrailingStopPct    float64 // 트레일링 허용 하락폭 (%)
	StagnationPct      float64 // 횡보 판단 변동폭 (0=비활성) — 현재 미구현: 분봉에 충분한 정보 있음
}

// SimTradeResult 는 하나의 거래에 대한 시뮬레이션 결과다.
type SimTradeResult struct {
	EntryPrice     float64
	ExitPrice      float64
	ExitReason     string  // "target" | "stop" | "trailing" | "end_of_data"
	PnlPct         float64
	HoldingCandles int
}

// SimulateTrade 는 entryPrice 에서 매수 후, candles 를 순서대로 재생해
// SimParams 기준의 첫 번째 청산 조건에 도달하면 청산 결과를 반환한다.
// candles 는 매수 직후 첫 1분봉부터 시간순(오름차순)으로 정렬되어야 한다.
func SimulateTrade(entryPrice float64, candles []MinuteCandle, params SimParams) SimTradeResult {
	targetPrice := entryPrice * (1 + params.TakeProfitPct/100)
	stopPrice := entryPrice * (1 - params.StopLossPct/100)

	trailingActive := false
	trailingPeak := entryPrice

	for i, c := range candles {
		// 트레일링 스탑 피크 갱신
		if trailingActive && c.High > trailingPeak {
			trailingPeak = c.High
		}

		// 목표가 도달 여부 (고가 기준)
		if c.High >= targetPrice {
			return SimTradeResult{
				EntryPrice:     entryPrice,
				ExitPrice:      targetPrice,
				ExitReason:     "target",
				PnlPct:         pnlPct(entryPrice, targetPrice),
				HoldingCandles: i + 1,
			}
		}

		// 손절가 도달 여부 (저가 기준)
		if c.Low <= stopPrice {
			return SimTradeResult{
				EntryPrice:     entryPrice,
				ExitPrice:      stopPrice,
				ExitReason:     "stop",
				PnlPct:         pnlPct(entryPrice, stopPrice),
				HoldingCandles: i + 1,
			}
		}

		// 트레일링 스탑 활성화 조건 확인
		if !trailingActive && params.TrailingTriggerPct > 0 {
			gain := (c.High/entryPrice - 1) * 100
			if gain >= params.TrailingTriggerPct {
				trailingActive = true
				trailingPeak = c.High
			}
		}

		// 트레일링 스탑 발동 (저가 기준)
		if trailingActive && params.TrailingStopPct > 0 {
			trailStop := trailingPeak * (1 - params.TrailingStopPct/100)
			if c.Low <= trailStop {
				exitPrice := math.Max(c.Low, trailStop)
				return SimTradeResult{
					EntryPrice:     entryPrice,
					ExitPrice:      exitPrice,
					ExitReason:     "trailing",
					PnlPct:         pnlPct(entryPrice, exitPrice),
					HoldingCandles: i + 1,
				}
			}
		}
	}

	// 데이터 소진 → 마지막 종가
	lastClose := entryPrice
	if len(candles) > 0 {
		lastClose = candles[len(candles)-1].Close
	}
	return SimTradeResult{
		EntryPrice:     entryPrice,
		ExitPrice:      lastClose,
		ExitReason:     "end_of_data",
		PnlPct:         pnlPct(entryPrice, lastClose),
		HoldingCandles: len(candles),
	}
}

func pnlPct(entry, exit float64) float64 {
	if entry == 0 {
		return 0
	}
	return (exit/entry - 1) * 100
}
```

- [ ] **Step 4: 테스트 실행 → PASS 확인**

```bash
cd backend && go test ./internal/simulation/... -v 2>&1
```
Expected: 4개 테스트 모두 PASS

- [ ] **Step 5: 커밋**

```bash
git add backend/internal/simulation/simulator.go backend/internal/simulation/simulator_test.go
git commit -m "feat(simulation): implement core trade simulation engine with trailing stop support"
```

---

## Task 2: 시나리오 감도 분석 정의

**Files:**
- Create: `backend/internal/simulation/scenarios.go`
- Create: `backend/internal/simulation/scenarios_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

`backend/internal/simulation/scenarios_test.go`:

```go
package simulation_test

import (
	"testing"

	"github.com/micro-trading-for-agent/backend/internal/simulation"
)

func TestGenerateScenarios_CountAndBaseline(t *testing.T) {
	base := simulation.SimParams{
		TakeProfitPct:      3.0,
		StopLossPct:        2.0,
		TrailingTriggerPct: 1.5,
		TrailingStopPct:    0.5,
	}
	scenarios := simulation.GenerateScenarios(base)

	// 베이스라인 포함, 각 파라미터 4개 × 변이 5개 = 최소 20개 이상
	if len(scenarios) < 20 {
		t.Errorf("want at least 20 scenarios, got %d", len(scenarios))
	}

	// 베이스라인이 첫 번째 시나리오여야 함
	if scenarios[0].Label != "현재 설정" {
		t.Errorf("first scenario should be baseline, got %q", scenarios[0].Label)
	}
}

func TestGenerateScenarios_LabelNotEmpty(t *testing.T) {
	base := simulation.SimParams{TakeProfitPct: 2.0, StopLossPct: 1.5}
	scenarios := simulation.GenerateScenarios(base)
	for i, s := range scenarios {
		if s.Label == "" {
			t.Errorf("scenario[%d] has empty label", i)
		}
	}
}
```

- [ ] **Step 2: 테스트 실행 → FAIL 확인**

```bash
cd backend && go test ./internal/simulation/... 2>&1 | grep FAIL
```

- [ ] **Step 3: `scenarios.go` 구현**

`backend/internal/simulation/scenarios.go`:

```go
package simulation

import "fmt"

// Scenario 는 레이블이 붙은 파라미터 세트다.
type Scenario struct {
	Label  string
	Params SimParams
}

// GenerateScenarios 는 base 파라미터를 기준으로 1D 감도 분석용 시나리오 목록을 생성한다.
// 한 번에 파라미터 하나씩 변화시키고 나머지는 base 값을 유지한다.
func GenerateScenarios(base SimParams) []Scenario {
	scenarios := []Scenario{
		{Label: "현재 설정", Params: base},
	}

	// 목표 수익률 변이: 현재 ×0.5 ~ ×2.0
	for _, mult := range []float64{0.5, 0.75, 1.25, 1.5, 2.0} {
		p := base
		p.TakeProfitPct = roundPct(base.TakeProfitPct * mult)
		scenarios = append(scenarios, Scenario{
			Label:  fmt.Sprintf("목표가 %.1f%%", p.TakeProfitPct),
			Params: p,
		})
	}

	// 손절률 변이: 현재 ×0.5 ~ ×2.0
	for _, mult := range []float64{0.5, 0.75, 1.25, 1.5, 2.0} {
		p := base
		p.StopLossPct = roundPct(base.StopLossPct * mult)
		scenarios = append(scenarios, Scenario{
			Label:  fmt.Sprintf("손절 %.1f%%", p.StopLossPct),
			Params: p,
		})
	}

	// 트레일링 트리거 변이 (기본값이 0이면 0.5%~2.0% 시도)
	triggerBase := base.TrailingTriggerPct
	if triggerBase == 0 {
		triggerBase = 1.0
	}
	for _, val := range []float64{0, triggerBase * 0.5, triggerBase * 0.75, triggerBase * 1.25, triggerBase * 1.5} {
		if val == base.TrailingTriggerPct {
			continue
		}
		p := base
		p.TrailingTriggerPct = roundPct(val)
		label := fmt.Sprintf("트레일링 트리거 %.1f%%", p.TrailingTriggerPct)
		if val == 0 {
			label = "트레일링 비활성"
		}
		scenarios = append(scenarios, Scenario{Label: label, Params: p})
	}

	// 트레일링 스탑 허용 하락폭 변이
	stopBase := base.TrailingStopPct
	if stopBase == 0 {
		stopBase = 0.5
	}
	for _, mult := range []float64{0.5, 0.75, 1.25, 1.5} {
		p := base
		p.TrailingStopPct = roundPct(stopBase * mult)
		scenarios = append(scenarios, Scenario{
			Label:  fmt.Sprintf("트레일링 스탑 %.1f%%", p.TrailingStopPct),
			Params: p,
		})
	}

	return scenarios
}

func roundPct(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
```

- [ ] **Step 4: 테스트 실행 → PASS 확인**

```bash
cd backend && go test ./internal/simulation/... -v 2>&1
```
Expected: 모든 테스트 PASS

- [ ] **Step 5: 커밋**

```bash
git add backend/internal/simulation/scenarios.go backend/internal/simulation/scenarios_test.go
git commit -m "feat(simulation): add 1D sensitivity scenario generator"
```

---

## Task 3: 시뮬레이션 결과 모델 & Firestore CRUD

**Files:**
- Modify: `backend/internal/models/models.go` (기존 파일 말미에 추가)
- Modify: `backend/internal/database/db.go` (기존 파일 말미에 추가)

- [ ] **Step 1: 모델 추가**

`backend/internal/models/models.go` 파일 말미에 추가:

```go
// SimulationResult 는 장 종료 후 시뮬레이션 실행 결과를 저장한다.
type SimulationResult struct {
	ID          int64     `firestore:"id"`
	Date        string    `firestore:"date"`         // YYYY-MM-DD (당일)
	ScenariosJSON string  `firestore:"scenarios_json"` // JSON: []ScenarioSummary
	RecommendedJSON string `firestore:"recommended_json"` // JSON: RecommendedSettings
	CreatedAt   time.Time `firestore:"created_at"`
	ExpireAt    time.Time `firestore:"expire_at"`    // TTL 30일
}
```

- [ ] **Step 2: DB CRUD 추가**

`backend/internal/database/db.go` 에 컬렉션 상수 추가 (상단 const 블록):

```go
colSimResults  = "simulation_results"
```

그리고 파일 말미에 CRUD 함수 추가:

```go
// UpsertSimulationResult 는 당일 시뮬레이션 결과를 저장(덮어쓰기)한다.
func (db *DB) UpsertSimulationResult(ctx context.Context, r models.SimulationResult) error {
	if r.ID == 0 {
		r.ID = newID()
	}
	r.CreatedAt = time.Now().UTC()
	r.ExpireAt = time.Now().UTC().Add(30 * 24 * time.Hour)
	_, err := db.client.Collection(colSimResults).Doc(r.Date).Set(ctx, r)
	return err
}

// GetSimulationResult 는 특정 날짜의 시뮬레이션 결과를 조회한다.
func (db *DB) GetSimulationResult(ctx context.Context, date string) (*models.SimulationResult, error) {
	snap, err := db.client.Collection(colSimResults).Doc(date).Get(ctx)
	if err != nil {
		return nil, err
	}
	var r models.SimulationResult
	if err := snap.DataTo(&r); err != nil {
		return nil, err
	}
	return &r, nil
}
```

- [ ] **Step 3: 빌드 확인**

```bash
cd backend && go build ./...
```

- [ ] **Step 4: 커밋**

```bash
git add backend/internal/models/models.go backend/internal/database/db.go
git commit -m "feat(simulation): add SimulationResult model and Firestore CRUD"
```

---

## Task 4: 장 종료 후 시뮬레이션 실행 (RunDailySimulation)

**Files:**
- Create: `backend/internal/simulation/runner.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: `runner.go` 구현**

`backend/internal/simulation/runner.go`:

```go
package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/kis"
	"github.com/micro-trading-for-agent/backend/internal/models"
)

// ScenarioSummary 는 하나의 시나리오에 대한 집계 결과다.
type ScenarioSummary struct {
	Label             string    `json:"label"`
	Params            SimParams `json:"params"`
	TotalPnlPct       float64   `json:"total_pnl_pct"`
	AvgPnlPct         float64   `json:"avg_pnl_pct"`
	WinRatePct        float64   `json:"win_rate_pct"`
	AvgHoldingMinutes float64   `json:"avg_holding_minutes"`
	TradeCount        int       `json:"trade_count"`
	DeltaVsActualPct  float64   `json:"delta_vs_actual_pct"` // 실제 결과와의 차이
}

// RecommendedSettings 는 시뮬레이션 결과 기준 최적 파라미터와 그 이유다.
type RecommendedSettings struct {
	Label          string    `json:"label"`
	Params         SimParams `json:"params"`
	Reason         string    `json:"reason"`
	ExpectedGainPct float64  `json:"expected_gain_pct"`
}

// RunDailySimulation 은 date(YYYY-MM-DD) 당일 체결 거래를 기반으로 시뮬레이션을 실행한다.
func RunDailySimulation(ctx context.Context, db *database.DB, kisClient *kis.Client, date string) error {
	trades, err := db.GetCompletedTradesBySoldDate(ctx, date)
	if err != nil {
		return fmt.Errorf("fetch trades: %w", err)
	}
	if len(trades) == 0 {
		log.Printf("[simulation] no trades on %s, skipping", date)
		return nil
	}

	settings, err := db.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("fetch settings: %w", err)
	}

	base := SimParams{
		TakeProfitPct:      settings.TakeProfitPct,
		StopLossPct:        settings.StopLossPct,
		TrailingTriggerPct: settings.TrailingTriggerPct,
		TrailingStopPct:    settings.TrailingStopPct,
	}
	scenarios := GenerateScenarios(base)

	// 당일 실제 총 손익 계산
	var actualTotalPnl float64
	for _, t := range trades {
		actualTotalPnl += t.ProfitPct
	}

	// 시나리오별 집계
	summaries := make([]ScenarioSummary, 0, len(scenarios))
	for _, scenario := range scenarios {
		summary, err := runScenarioForTrades(ctx, kisClient, trades, scenario, date)
		if err != nil {
			log.Printf("[simulation] scenario %q skipped: %v", scenario.Label, err)
			continue
		}
		summary.DeltaVsActualPct = summary.TotalPnlPct - actualTotalPnl
		summaries = append(summaries, summary)
	}

	// 최고 총 손익 시나리오를 추천
	recommended := pickBestScenario(summaries, base)

	scenariosJSON, _ := json.Marshal(summaries)
	recommendedJSON, _ := json.Marshal(recommended)

	result := models.SimulationResult{
		Date:            date,
		ScenariosJSON:   string(scenariosJSON),
		RecommendedJSON: string(recommendedJSON),
	}
	if err := db.UpsertSimulationResult(ctx, result); err != nil {
		return fmt.Errorf("save simulation result: %w", err)
	}
	log.Printf("[simulation] completed for %s: %d scenarios, best=%q", date, len(summaries), recommended.Label)
	return nil
}

func runScenarioForTrades(
	ctx context.Context,
	kisClient *kis.Client,
	trades []models.TradeReport,
	scenario Scenario,
	date string,
) (ScenarioSummary, error) {
	var totalPnl, totalHold float64
	var wins, count int

	kisDate := strings.ReplaceAll(date, "-", "") // YYYYMMDD

	for _, trade := range trades {
		candles, err := fetchHoldingCandles(ctx, kisClient, trade, kisDate)
		if err != nil || len(candles) == 0 {
			continue
		}
		result := SimulateTrade(trade.BuyPrice, candles, scenario.Params)
		totalPnl += result.PnlPct
		totalHold += float64(result.HoldingCandles)
		count++
		if result.PnlPct > 0 {
			wins++
		}
	}
	if count == 0 {
		return ScenarioSummary{}, fmt.Errorf("no candle data available")
	}
	return ScenarioSummary{
		Label:             scenario.Label,
		Params:            scenario.Params,
		TotalPnlPct:       roundPct(totalPnl),
		AvgPnlPct:         roundPct(totalPnl / float64(count)),
		WinRatePct:        roundPct(float64(wins) / float64(count) * 100),
		AvgHoldingMinutes: roundPct(totalHold / float64(count)),
		TradeCount:        count,
	}, nil
}

// fetchHoldingCandles 는 해당 거래의 보유 기간(매수~매도 시각) 1분봉을 조회한다.
// KIS GetDayMinuteChart 는 지정 시각 이전 최대 120개 1분봉을 반환하므로,
// 매도 시각 기준으로 조회 후 매수 시각 이후 구간만 필터링한다.
func fetchHoldingCandles(ctx context.Context, kisClient *kis.Client, trade models.TradeReport, kisDate string) ([]MinuteCandle, error) {
	if trade.SoldAt == nil {
		return nil, fmt.Errorf("trade not closed")
	}
	kst, _ := time.LoadLocation("Asia/Seoul")
	sellKST := trade.SoldAt.In(kst)
	buyKST := trade.CreatedAt.In(kst)

	inputHour := sellKST.Format("150405")
	bars, err := kisClient.GetDayMinuteChart(ctx, trade.StockCode, kisDate, inputHour)
	if err != nil {
		return nil, err
	}

	// bars 는 newest-first → 반전 후 시간순 정렬
	for i, j := 0, len(bars)-1; i < j; i, j = i+1, j-1 {
		bars[i], bars[j] = bars[j], bars[i]
	}

	// 매수 시각 이후 구간만 선택
	var candles []MinuteCandle
	for _, b := range bars {
		barTime, err := time.ParseInLocation("20060102 150405", kisDate+" "+b.Time, kst)
		if err != nil {
			continue
		}
		if barTime.Before(buyKST) {
			continue
		}
		high, _ := strconv.ParseFloat(b.High, 64)
		low, _ := strconv.ParseFloat(b.Low, 64)
		close_, _ := strconv.ParseFloat(b.Close, 64)
		candles = append(candles, MinuteCandle{High: high, Low: low, Close: close_})
	}
	return candles, nil
}

func pickBestScenario(summaries []ScenarioSummary, base SimParams) RecommendedSettings {
	if len(summaries) == 0 {
		return RecommendedSettings{Label: "현재 설정", Params: base, Reason: "시뮬레이션 데이터 없음"}
	}
	best := summaries[0]
	for _, s := range summaries[1:] {
		if s.TotalPnlPct > best.TotalPnlPct {
			best = s
		}
	}
	return RecommendedSettings{
		Label:           best.Label,
		Params:          best.Params,
		Reason:          fmt.Sprintf("당일 거래 시뮬레이션 기준 총 손익 최대 (%.2f%%)", best.TotalPnlPct),
		ExpectedGainPct: best.DeltaVsActualPct,
	}
}
```

참고: 파일 상단 import 에 `"strings"` 추가 필요.

- [ ] **Step 2: 스케줄러에 15:25 트리거 추가**

`backend/cmd/server/main.go` 의 `runMarketScheduler()` 함수에서 기존 "15:20" 케이스 아래에 추가:

```go
case t.Hour() == 15 && t.Minute() == 25:
    if !scheduledTasks["15:25"] {
        scheduledTasks["15:25"] = true
        go func() {
            date := t.In(kst).Format("2006-01-02")
            if err := simulation.RunDailySimulation(context.Background(), db, kisClient, date); err != nil {
                log.Printf("[scheduler] simulation failed: %v", err)
            }
        }()
    }
```

참고: 파일 상단 import 에 `"github.com/micro-trading-for-agent/backend/internal/simulation"` 추가.

- [ ] **Step 3: 빌드 확인**

```bash
cd backend && go build ./...
```

- [ ] **Step 4: 커밋**

```bash
git add backend/internal/simulation/runner.go backend/cmd/server/main.go
git commit -m "feat(simulation): add RunDailySimulation and 15:25 post-market scheduler trigger"
```

---

## Task 5: 시뮬레이션 결과 HTTP API

**Files:**
- Modify: `backend/internal/api/handlers.go`
- Modify: `backend/internal/api/router.go`

- [ ] **Step 1: 핸들러 추가**

`handlers.go` 파일 말미에 추가:

```go
// HandleGetSimulationResult handles GET /api/simulation/:date
func (h *Handler) HandleGetSimulationResult(c *gin.Context) {
	date := c.Param("date")
	if date == "" {
		kst, _ := time.LoadLocation("Asia/Seoul")
		date = time.Now().In(kst).Format("2006-01-02")
	}
	result, err := h.db.GetSimulationResult(c.Request.Context(), date)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "simulation result not found for " + date})
		return
	}

	// ScenariosJSON, RecommendedJSON 을 파싱해서 응답
	var scenarios []simulation.ScenarioSummary
	var recommended simulation.RecommendedSettings
	_ = json.Unmarshal([]byte(result.ScenariosJSON), &scenarios)
	_ = json.Unmarshal([]byte(result.RecommendedJSON), &recommended)

	c.JSON(http.StatusOK, gin.H{
		"date":        result.Date,
		"scenarios":   scenarios,
		"recommended": recommended,
		"created_at":  result.CreatedAt,
	})
}

// HandleRunSimulation handles POST /api/simulation/run?date=YYYY-MM-DD (수동 트리거)
func (h *Handler) HandleRunSimulation(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		kst, _ := time.LoadLocation("Asia/Seoul")
		date = time.Now().In(kst).Format("2006-01-02")
	}
	go func() {
		if err := simulation.RunDailySimulation(context.Background(), h.db, h.kisClient, date); err != nil {
			log.Printf("[api] manual simulation failed: %v", err)
		}
	}()
	c.JSON(http.StatusAccepted, gin.H{"message": "simulation started for " + date})
}
```

참고: `handlers.go` 에 `"github.com/micro-trading-for-agent/backend/internal/simulation"` import 추가.  
참고: `Handler` 구조체에 `kisClient *kis.Client` 필드가 있어야 함. 기존 handler 생성 코드 확인 후 없으면 추가.

- [ ] **Step 2: 라우터에 등록**

`router.go` 에 추가:

```go
// 기존 reports 그룹 아래에 simulation 그룹 추가
sim := api.Group("/simulation")
{
    sim.GET("/:date", h.HandleGetSimulationResult)
    sim.POST("/run", h.HandleRunSimulation)
}
```

- [ ] **Step 3: 빌드 확인**

```bash
cd backend && go build ./...
```

- [ ] **Step 4: 수동 스모크 테스트**

```bash
curl -X POST "http://localhost:8080/api/simulation/run?date=2026-05-13"
# → {"message": "simulation started for 2026-05-13"}
# 잠시 후:
curl "http://localhost:8080/api/simulation/2026-05-13" | jq '.recommended'
```

- [ ] **Step 5: 커밋**

```bash
git add backend/internal/api/handlers.go backend/internal/api/router.go
git commit -m "feat(api): add GET /api/simulation/:date and POST /api/simulation/run endpoints"
```

---

## Task 6: 프론트엔드 시뮬레이션 탭 UI

**Files:**
- Modify: `frontend/src/pages/DailyReports.jsx`

- [ ] **Step 1: 시뮬레이션 데이터 fetch 로직 추가**

`DailyReports.jsx` 의 기존 state/effect 영역에 추가:

```jsx
const [simResult, setSimResult] = useState(null);
const [simLoading, setSimLoading] = useState(false);
const [simRunning, setSimRunning] = useState(false);
const [activeTab, setActiveTab] = useState('daily'); // 'daily' | 'simulation'

const fetchSimulation = async (date) => {
  setSimLoading(true);
  try {
    const res = await fetch(
      `${import.meta.env.VITE_API_BASE_URL}/api/simulation/${date}`
    );
    if (!res.ok) { setSimResult(null); return; }
    setSimResult(await res.json());
  } catch {
    setSimResult(null);
  } finally {
    setSimLoading(false);
  }
};

const runSimulation = async (date) => {
  setSimRunning(true);
  try {
    await fetch(
      `${import.meta.env.VITE_API_BASE_URL}/api/simulation/run?date=${date}`,
      { method: 'POST' }
    );
    // 30초 후 결과 조회 (백그라운드 실행)
    setTimeout(() => fetchSimulation(date), 30000);
  } finally {
    setSimRunning(false);
  }
};
```

- [ ] **Step 2: 탭 UI 추가**

기존 DailyReports 페이지의 헤더/필터 아래에 탭 버튼 추가:

```jsx
<div className="flex gap-2 border-b mb-4">
  <button
    onClick={() => setActiveTab('daily')}
    className={`px-4 py-2 text-sm font-medium border-b-2 ${
      activeTab === 'daily'
        ? 'border-indigo-600 text-indigo-600'
        : 'border-transparent text-gray-500 hover:text-gray-700'
    }`}
  >
    일별 리포트
  </button>
  <button
    onClick={() => { setActiveTab('simulation'); fetchSimulation(selectedDate || todayKST()); }}
    className={`px-4 py-2 text-sm font-medium border-b-2 ${
      activeTab === 'simulation'
        ? 'border-indigo-600 text-indigo-600'
        : 'border-transparent text-gray-500 hover:text-gray-700'
    }`}
  >
    시뮬레이션
  </button>
</div>
```

`todayKST()` 헬퍼 (컴포넌트 파일 내 또는 상단):
```jsx
const todayKST = () => {
  return new Date(Date.now() + 9 * 60 * 60 * 1000)
    .toISOString().slice(0, 10);
};
```

- [ ] **Step 3: 시뮬레이션 탭 콘텐츠 추가**

기존 daily 리포트 렌더링을 `{activeTab === 'daily' && ...}` 로 감싸고, 아래에 시뮬레이션 탭 추가:

```jsx
{activeTab === 'simulation' && (
  <div className="space-y-4">
    {simLoading && <p className="text-sm text-gray-500">시뮬레이션 결과 로딩 중...</p>}
    {!simLoading && !simResult && (
      <div className="text-center py-8 space-y-3">
        <p className="text-gray-500 text-sm">이 날짜의 시뮬레이션 결과가 없습니다.</p>
        <button
          onClick={() => runSimulation(selectedDate || todayKST())}
          disabled={simRunning}
          className="px-4 py-2 bg-indigo-600 text-white rounded text-sm hover:bg-indigo-700 disabled:opacity-50"
        >
          {simRunning ? '실행 중...' : '지금 시뮬레이션 실행'}
        </button>
        {simRunning && <p className="text-xs text-gray-400">30초 후 자동으로 결과를 불러옵니다.</p>}
      </div>
    )}
    {simResult && (
      <>
        {/* 추천 설정 카드 */}
        <div className="bg-indigo-50 dark:bg-indigo-900/20 rounded-lg p-4 border border-indigo-200 dark:border-indigo-700">
          <h3 className="font-semibold text-indigo-700 dark:text-indigo-300 mb-1">
            추천 설정: {simResult.recommended.label}
          </h3>
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">{simResult.recommended.reason}</p>
          <div className="flex flex-wrap gap-2 text-xs">
            {simResult.recommended.params.take_profit_pct > 0 && (
              <span className="bg-white dark:bg-gray-800 px-2 py-1 rounded border">
                목표가 {simResult.recommended.params.take_profit_pct}%
              </span>
            )}
            {simResult.recommended.params.stop_loss_pct > 0 && (
              <span className="bg-white dark:bg-gray-800 px-2 py-1 rounded border">
                손절 {simResult.recommended.params.stop_loss_pct}%
              </span>
            )}
          </div>
          <div className="mt-3 flex gap-2">
            <button
              onClick={async () => {
                const p = simResult.recommended.params;
                await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/settings`, {
                  method: 'PATCH',
                  headers: { 'Content-Type': 'application/json' },
                  body: JSON.stringify({
                    take_profit_pct: p.take_profit_pct,
                    stop_loss_pct: p.stop_loss_pct,
                    trailing_trigger_pct: p.trailing_trigger_pct,
                    trailing_stop_pct: p.trailing_stop_pct,
                  }),
                });
                alert('설정이 적용되었습니다.');
              }}
              className="px-3 py-1.5 bg-indigo-600 text-white rounded text-xs hover:bg-indigo-700"
            >
              이 설정 적용
            </button>
          </div>
        </div>

        {/* 시나리오 비교 테이블 */}
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-gray-500">
                <th className="py-2 pr-4">시나리오</th>
                <th className="py-2 pr-4 text-right">총 손익률</th>
                <th className="py-2 pr-4 text-right">승률</th>
                <th className="py-2 pr-4 text-right">실제 대비</th>
                <th className="py-2 pr-4 text-right">평균 보유(분)</th>
              </tr>
            </thead>
            <tbody>
              {simResult.scenarios.map((s, i) => (
                <tr key={i} className={`border-b ${s.label === '현재 설정' ? 'bg-gray-50 dark:bg-gray-800/50 font-medium' : ''}`}>
                  <td className="py-2 pr-4">{s.label}</td>
                  <td className={`py-2 pr-4 text-right ${s.total_pnl_pct >= 0 ? 'text-green-600' : 'text-red-500'}`}>
                    {s.total_pnl_pct >= 0 ? '+' : ''}{s.total_pnl_pct.toFixed(2)}%
                  </td>
                  <td className="py-2 pr-4 text-right">{s.win_rate_pct.toFixed(1)}%</td>
                  <td className={`py-2 pr-4 text-right ${s.delta_vs_actual_pct > 0 ? 'text-green-600' : s.delta_vs_actual_pct < 0 ? 'text-red-500' : ''}`}>
                    {s.delta_vs_actual_pct > 0 ? '+' : ''}{s.delta_vs_actual_pct.toFixed(2)}%
                  </td>
                  <td className="py-2 pr-4 text-right">{s.avg_holding_minutes.toFixed(0)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </>
    )}
  </div>
)}
```

- [ ] **Step 4: 프론트엔드 빌드 확인**

```bash
cd frontend && npm run build 2>&1 | tail -10
```
Expected: `✓ built in`

- [ ] **Step 5: 커밋**

```bash
git add frontend/src/pages/DailyReports.jsx
git commit -m "feat(ui): add simulation tab to DailyReports with scenario comparison table and apply button"
```

---

## Verification

1. `cd backend && go test ./internal/simulation/... -v` → 모든 테스트 PASS
2. `cd backend && go build ./...` → 빌드 에러 없음
3. `cd frontend && npm run build` → 빌드 에러 없음
4. `POST /api/simulation/run?date=YYYY-MM-DD` → 202 Accepted
5. 30초 후 `GET /api/simulation/YYYY-MM-DD` → scenarios 배열 + recommended 포함 응답
6. DailyReports "시뮬레이션" 탭 클릭 → 시나리오 비교 테이블 표시, "이 설정 적용" 버튼 동작 확인
7. `runMarketScheduler` 에서 15:25 KST 에 자동 실행 → 서비스 로그에서 `[simulation] completed` 확인
