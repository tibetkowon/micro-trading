# Simulation P2 버그 수정 플랜

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Codex 리뷰에서 발견된 시뮬레이션 기능의 P2 버그 3개를 수정한다.

**Architecture:**  
모두 `backend/internal/simulation/runner.go` 와 `backend/internal/models/models.go` 수정으로 해결된다.  
Issue 1: delta 계산 기준을 candle 조회 성공 subset으로 통일.  
Issue 2: `GetDayMinuteChart` 페이지네이션으로 전체 보유 기간 커버.  
Issue 3: `SimulationResult.ExpireAt` Firestore 태그를 `expireAt` (camelCase) 로 수정.

**Tech Stack:** Go 1.26.1, Firestore SDK

---

## 파일 맵

| 파일 | 변경 내용 |
|---|---|
| `backend/internal/simulation/runner.go` | Issue 1: delta 계산 위치 이동 / Issue 2: fetchHoldingCandles 페이지네이션 추가 |
| `backend/internal/simulation/runner_test.go` | 신규: 두 수정 사항에 대한 단위 테스트 |
| `backend/internal/models/models.go:168` | Issue 3: firestore 태그 `expire_at` → `expireAt` |

---

## Task 1: Issue 1 — delta 계산 기준 통일

**문제:** `actualTotalPnl` 은 전체 `trades` 기준으로 계산하지만, 시뮬레이션은 candle 조회에 성공한 `prepared` subset 기준으로 돌아감. 결과적으로 `delta_vs_actual_pct` 가 다른 모수를 비교하여 수치가 왜곡됨.

**수정:** `actualTotalPnl` 계산을 `prepared` 빌드 완료 이후로 이동.

**Files:**
- Modify: `backend/internal/simulation/runner.go:66-92`
- Create: `backend/internal/simulation/runner_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

`backend/internal/simulation/runner_test.go` 신규 생성:

```go
package simulation_test

import (
	"testing"

	"github.com/micro-trading-for-agent/backend/internal/simulation"
)

// deltaOnlyCountsPreparedTrades 는 delta 가 prepared subset 기준인지 검증한다.
// RunDailySimulation 의 내부 로직을 직접 테스트할 수 없으므로
// 헬퍼 함수 computeActualPnl 을 export 해서 검증한다.
func TestComputeActualPnlUsesSubset(t *testing.T) {
	items := []simulation.TradeCandles{
		{ProfitPct: 2.0},
		{ProfitPct: -1.0},
	}
	got := simulation.ComputeActualPnl(items)
	want := 1.0
	if got != want {
		t.Errorf("ComputeActualPnl = %f, want %f", got, want)
	}
}

func TestComputeActualPnlEmpty(t *testing.T) {
	got := simulation.ComputeActualPnl(nil)
	if got != 0 {
		t.Errorf("ComputeActualPnl(nil) = %f, want 0", got)
	}
}
```

- [ ] **Step 2: 테스트 실행 → FAIL 확인**

```bash
cd backend && go test ./internal/simulation/... 2>&1 | grep -E "FAIL|undefined"
```
Expected: `undefined: simulation.TradeCandles` 또는 `undefined: simulation.ComputeActualPnl`

- [ ] **Step 3: runner.go 수정**

`backend/internal/simulation/runner.go` 에서:

1. `tradeCandles` 구조체를 export (`TradeCandles`) 하고 `ProfitPct` 필드 추가:

```go
// TradeCandles pairs a completed trade with its fetched minute candles.
type TradeCandles struct {
	trade     models.TradeReport
	candles   []MinuteCandle
	ProfitPct float64 // actual profit % for delta comparison
}
```

2. `ComputeActualPnl` 헬퍼 함수 추가 (파일 말미):

```go
// ComputeActualPnl returns the sum of actual ProfitPct for the prepared subset.
func ComputeActualPnl(items []TradeCandles) float64 {
	var total float64
	for _, item := range items {
		total += item.ProfitPct
	}
	return total
}
```

3. `RunDailySimulation` 에서 기존 delta 계산 블록(line 66-68) 을 삭제하고, `prepared` 빌드 코드 및 delta 계산 위치를 수정:

```go
// 기존 코드 (삭제):
// var actualTotalPnl float64
// for _, t := range trades {
//     actualTotalPnl += t.ProfitPct
// }

kisDate := strings.ReplaceAll(date, "-", "")
prepared := make([]TradeCandles, 0, len(trades))
for _, trade := range trades {
    candles, err := fetchHoldingCandles(ctx, kisClient, trade, kisDate)
    if err != nil || len(candles) == 0 {
        continue
    }
    prepared = append(prepared, TradeCandles{
        trade:     trade,
        candles:   candles,
        ProfitPct: trade.ProfitPct, // 실제 수익률 저장
    })
}
if len(prepared) == 0 {
    return fmt.Errorf("no candle data available")
}

// prepared subset 기준 실제 총 손익 계산 (delta 기준 통일)
actualTotalPnl := ComputeActualPnl(prepared)
```

4. `runScenarioForTrades` 시그니처도 `[]TradeCandles` 로 맞게 수정:

```go
func runScenarioForTrades(prepared []TradeCandles, scenario Scenario) (ScenarioSummary, error) {
    var totalPnl, totalHold float64
    var wins, count int

    for _, item := range prepared {
        result := SimulateTrade(item.trade.BuyPrice, item.candles, scenario.Params)
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
```

- [ ] **Step 4: 테스트 실행 → PASS 확인**

```bash
cd backend && go test ./internal/simulation/... -v -run TestComputeActual 2>&1
```
Expected: `PASS`

- [ ] **Step 5: 빌드 확인**

```bash
cd backend && go build ./... 2>&1
```

- [ ] **Step 6: 커밋**

```bash
git add backend/internal/simulation/runner.go backend/internal/simulation/runner_test.go
git commit -m "fix(simulation): compute delta against candle-prepared subset only"
```

---

## Task 2: Issue 2 — 전체 보유 기간 분봉 페이지네이션

**문제:** `GetDayMinuteChart` 1회 호출은 최대 120분봉(2시간)만 반환. 매수~매도 기간이 2시간을 초과하는 거래는 초반 캔들이 누락되어 시뮬레이션에서 목표가/손절가 도달 케이스를 놓침.

**수정:** oldest bar의 시각을 cursor로 삼아 `buyKST` 이전까지 커버될 때까지 반복 호출.

**Files:**
- Modify: `backend/internal/simulation/runner.go` (`fetchHoldingCandles` 함수)
- Modify: `backend/internal/simulation/runner_test.go`

- [ ] **Step 1: 테스트 추가**

`runner_test.go` 에 추가:

```go
// TestFetchHoldingCandlesDeduplication 은 페이지네이션 시 중복 제거를 검증한다.
// 실제 KIS 호출 없이 순수 dedup 로직만 검증하기 위해 filterAndConvertBars 를 export 해서 테스트.
func TestFilterAndConvertBars_DeduplicatesAndFilters(t *testing.T) {
	kst, _ := time.LoadLocation("Asia/Seoul")
	kisDate := "20260513"
	buyKST := time.Date(2026, 5, 13, 9, 15, 0, 0, kst)

	// 중복 포함 bars (Time 기준)
	bars := []simulation.RawBar{
		{Time: "091500", High: "10100", Low: "10000", Close: "10050"},
		{Time: "091500", High: "10100", Low: "10000", Close: "10050"}, // 중복
		{Time: "091600", High: "10200", Low: "10050", Close: "10150"},
		{Time: "090000", High: "9900", Low: "9800", Close: "9850"},   // buyKST 이전 → 제외
	}

	candles, err := simulation.FilterAndConvertBars(bars, kisDate, buyKST, kst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candles) != 2 {
		t.Errorf("want 2 candles (dedup + filter), got %d", len(candles))
	}
}
```

- [ ] **Step 2: 테스트 실행 → FAIL 확인**

```bash
cd backend && go test ./internal/simulation/... -run TestFilterAndConvert 2>&1 | grep -E "FAIL|undefined"
```
Expected: `undefined: simulation.RawBar` 또는 `undefined: simulation.FilterAndConvertBars`

- [ ] **Step 3: runner.go 의 `fetchHoldingCandles` 교체**

`backend/internal/simulation/runner.go` 에서 기존 `fetchHoldingCandles` 함수 전체를 아래로 교체:

```go
// RawBar is an exported alias for kis.ChartBar fields used in tests.
type RawBar struct {
	Time  string
	High  string
	Low   string
	Close string
}

// FilterAndConvertBars deduplicates bars by Time, filters to [buyKST, ∞),
// and converts string OHLC fields to MinuteCandle. Exported for testing.
func FilterAndConvertBars(bars []RawBar, kisDate string, buyKST time.Time, kst *time.Location) ([]MinuteCandle, error) {
	seen := make(map[string]bool, len(bars))
	var candles []MinuteCandle
	for _, b := range bars {
		if seen[b.Time] {
			continue
		}
		seen[b.Time] = true
		barTime, err := time.ParseInLocation("20060102 150405", kisDate+" "+b.Time, kst)
		if err != nil {
			continue
		}
		if barTime.Before(buyKST) {
			continue
		}
		high, _ := strconv.ParseFloat(b.High, 64)
		low, _ := strconv.ParseFloat(b.Low, 64)
		closePrice, _ := strconv.ParseFloat(b.Close, 64)
		candles = append(candles, MinuteCandle{High: high, Low: low, Close: closePrice})
	}
	return candles, nil
}

// fetchHoldingCandles fetches all 1-minute candles covering the trade's holding period.
// Paginates backwards from sellKST until buyKST is covered (each call returns ≤120 bars).
func fetchHoldingCandles(ctx context.Context, kisClient *kis.Client, trade models.TradeReport, kisDate string) ([]MinuteCandle, error) {
	if trade.SoldAt == nil {
		return nil, fmt.Errorf("trade not closed")
	}
	kst, _ := time.LoadLocation("Asia/Seoul")
	sellKST := trade.SoldAt.In(kst)
	buyKST := trade.CreatedAt.In(kst)

	var allRaw []RawBar
	seen := make(map[string]bool)
	cursor := sellKST

	for {
		bars, err := kisClient.GetDayMinuteChart(ctx, trade.StockCode, kisDate, cursor.Format("150405"))
		if err != nil || len(bars) == 0 {
			break
		}

		// bars are newest-first; reverse to chronological order
		for i, j := 0, len(bars)-1; i < j; i, j = i+1, j-1 {
			bars[i], bars[j] = bars[j], bars[i]
		}

		var oldestTime time.Time
		for _, b := range bars {
			if seen[b.Time] {
				continue
			}
			seen[b.Time] = true
			allRaw = append(allRaw, RawBar{Time: b.Time, High: b.High, Low: b.Low, Close: b.Close})
			t, err := time.ParseInLocation("20060102 150405", kisDate+" "+b.Time, kst)
			if err == nil && (oldestTime.IsZero() || t.Before(oldestTime)) {
				oldestTime = t
			}
		}

		// Stop if we've covered buyKST or can't go further back
		if oldestTime.IsZero() || !oldestTime.After(buyKST) {
			break
		}
		// Move cursor 1 minute before the oldest fetched bar
		cursor = oldestTime.Add(-1 * time.Minute)
		if !cursor.After(buyKST) {
			break
		}
	}

	// Sort allRaw chronologically before filtering
	// (already in order due to append order, but sort to be safe)
	return FilterAndConvertBars(allRaw, kisDate, buyKST, kst)
}
```

참고: `kis.ChartBar` 에서 직접 `RawBar` 로 변환하므로 `kis` 패키지 import 는 유지.

- [ ] **Step 4: 테스트 실행 → PASS 확인**

```bash
cd backend && go test ./internal/simulation/... -v 2>&1
```
Expected: 모든 테스트 PASS

- [ ] **Step 5: 빌드 확인**

```bash
cd backend && go build ./... 2>&1
```

- [ ] **Step 6: 커밋**

```bash
git add backend/internal/simulation/runner.go backend/internal/simulation/runner_test.go
git commit -m "fix(simulation): paginate fetchHoldingCandles to cover full holding period"
```

---

## Task 3: Issue 3 — SimulationResult TTL 필드명 수정

**문제:** Firestore TTL 정책은 `simulation_results.expireAt` (camelCase) 기준이지만, `SimulationResult` 모델의 `ExpireAt` 필드가 `firestore:"expire_at"` (snake_case) 로 저장되어 TTL 정책에 매칭되지 않음. 다른 모델(`DailyReport`, `TradeReport` 등)은 모두 `firestore:"expireAt"` 사용 중.

**Files:**
- Modify: `backend/internal/models/models.go:168`

- [ ] **Step 1: 수정**

`backend/internal/models/models.go` 의 `SimulationResult` 구조체에서:

```go
// 변경 전:
ExpireAt time.Time `json:"expire_at" firestore:"expire_at"`

// 변경 후:
ExpireAt time.Time `json:"expire_at" firestore:"expireAt"`
```

- [ ] **Step 2: 빌드 확인**

```bash
cd backend && go build ./... 2>&1
```

- [ ] **Step 3: 커밋**

```bash
git add backend/internal/models/models.go
git commit -m "fix(models): use expireAt firestore tag on SimulationResult to match TTL policy"
```

---

## Verification

1. `cd backend && go test ./internal/simulation/... -v` → 모든 테스트 PASS (기존 4개 + 신규 3개)
2. `cd backend && go build ./...` → 빌드 에러 없음
3. `POST /api/simulation/run?date=YYYY-MM-DD` 실행 후 `GET /api/simulation/YYYY-MM-DD` 조회:
   - `scenarios` 배열의 `"현재 설정"` 시나리오의 `delta_vs_actual_pct` 가 `0.0` 에 가까운지 확인 (동일 파라미터이므로)
4. 보유 시간이 2시간 초과인 거래가 있는 날짜로 시뮬레이션 실행 시 에러 없이 완료되는지 확인
5. Firestore `simulation_results` 컬렉션에서 문서의 `expireAt` 필드가 정상적으로 설정되는지 확인
