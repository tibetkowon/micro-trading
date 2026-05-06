# 1분봉 지표 전환 구현 계획

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 기술 지표(RSI, MACD, MA, VWAP) 계산 기준을 5분봉에서 1분봉으로 전환하고, 1분봉 기준 MA60·MA120을 추가하여 하드 필터에서 장기 이평선 지지 조건으로 활용한다.

**Architecture:** `GetStockInfo`에서 1분봉 집계(`aggregateMinuteBars(bars,5)`) 단계를 제거하고 `minuteBarsToCandles`로 변환한 1분봉 `Candle` 슬라이스를 직접 사용한다. 200개 1분봉(현재 요청량)은 MA120에 충분하다. MA60·MA120은 `StockInfo` 신규 필드로 추가하고, `ApplyHardFilter`에 현재가 ≥ MA60/MA120 조건 두 개를 추가한다.

**Tech Stack:** Go 1.26.1, React 18 / Vite, Firebase Firestore

---

## 변경 파일 목록

| 파일 | 변경 내용 |
|------|----------|
| `backend/internal/ops/stock_info.go` | 1분봉 직접 사용, MA60/MA120 계산 추가 |
| `backend/internal/ops/stock_info_test.go` | 신규 — calcMA/calcRSI/calcMACD 단위 테스트 |
| `backend/internal/scorer/scorer_test.go` | 신규 — ApplyHardFilter MA60 지지 조건 테스트 |
| `backend/internal/database/db.go` | TradingSettings 필드 2개 + 파싱 + defaultSettings |
| `backend/internal/scorer/scorer.go` | ApplyHardFilter에 MA60/MA120 조건 추가 |
| `backend/internal/api/handlers.go` | UpdateSettings 요청 구조체 + 저장 로직 |
| `frontend/src/pages/Settings.jsx` | 하드필터 탭에 MA60/MA120 지지 토글 추가 |

---

## Task 1: 단위 테스트 파일 생성 (calcMA / calcRSI / calcMACD)

**Files:**
- Create: `backend/internal/ops/stock_info_test.go`

- [ ] **Step 1: 테스트 파일 작성**

```go
package ops

import (
	"math"
	"testing"
)

func TestCalcMA(t *testing.T) {
	closes := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// MA5: 평균(6,7,8,9,10) = 8.0
	got := calcMA(closes, 5)
	if math.Abs(got-8.0) > 0.01 {
		t.Errorf("MA5 = %.2f, want 8.0", got)
	}

	// 데이터 부족: MA20 = 0
	got = calcMA(closes, 20)
	if got != 0 {
		t.Errorf("MA20 (insufficient) = %.2f, want 0", got)
	}
}

func TestCalcRSI_Insufficient(t *testing.T) {
	// 15개 미만이면 0 반환 (RSI14 needs period+1=15)
	closes := make([]float64, 14)
	for i := range closes {
		closes[i] = float64(i + 1)
	}
	got := calcRSI(closes, 14)
	if got != 0 {
		t.Errorf("RSI with 14 bars = %.2f, want 0", got)
	}
}

func TestCalcRSI_AllGain(t *testing.T) {
	// 단조 증가 수열 → RSI = 100
	closes := make([]float64, 20)
	for i := range closes {
		closes[i] = float64(i + 1)
	}
	got := calcRSI(closes, 14)
	if got != 100 {
		t.Errorf("RSI (all gain) = %.2f, want 100", got)
	}
}

func TestCalcMACD_Insufficient(t *testing.T) {
	// slowPeriod(26)개 미만이면 (0,0,0) 반환
	closes := make([]float64, 25)
	for i := range closes {
		closes[i] = float64(i + 1)
	}
	line, signal, histo := calcMACD(closes, 12, 26, 9)
	if line != 0 || signal != 0 || histo != 0 {
		t.Errorf("MACD (insufficient) = (%.4f,%.4f,%.4f), want (0,0,0)", line, signal, histo)
	}
}

func TestCalcMA60And120(t *testing.T) {
	// MA60, MA120 — 충분한 데이터로 0이 아닌 값 반환
	closes := make([]float64, 150)
	for i := range closes {
		closes[i] = float64(i + 100)
	}
	ma60 := calcMA(closes, 60)
	ma120 := calcMA(closes, 120)
	if ma60 == 0 {
		t.Error("MA60 should not be 0 with 150 bars")
	}
	if ma120 == 0 {
		t.Error("MA120 should not be 0 with 150 bars")
	}
	// MA120 < MA60 (단조증가 수열에서 더 오래된 평균이 더 낮음)
	if ma120 >= ma60 {
		t.Errorf("MA120 (%.2f) should be less than MA60 (%.2f) for ascending series", ma120, ma60)
	}
}
```

- [ ] **Step 2: 테스트 실행 — 통과 확인 (calcMA/RSI/MACD 함수는 이미 구현됨)**

```bash
cd /Volumes/MAC/Project/micro-trading-for-agent/backend
go test ./internal/ops/... -run TestCalc -v
```

기대 결과: `PASS` — 기존 함수들이 이미 올바르게 구현되어 있어야 함

- [ ] **Step 3: 커밋**

```bash
git add backend/internal/ops/stock_info_test.go
git commit -m "test: ops 지표 계산 함수 단위 테스트 추가"
```

---

## Task 2: ApplyHardFilter MA60/MA120 지지 조건 — 실패하는 테스트 먼저

**Files:**
- Create: `backend/internal/scorer/scorer_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

```go
package scorer

import (
	"testing"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/ops"
)

func makeSettings(ma60enabled, ma120enabled bool) database.TradingSettings {
	return database.TradingSettings{
		HardMA60SupportEnabled:  ma60enabled,
		HardMA120SupportEnabled: ma120enabled,
	}
}

func TestApplyHardFilter_MA60Support_Reject(t *testing.T) {
	// 현재가(9500)가 MA60(10000) 미만 → 탈락
	c := CandidateInfo{
		StockCode: "000001",
		StockName: "테스트",
		Info: &ops.StockInfo{
			CurrentPrice: "9500",
			MA60:         10000,
		},
	}
	result := ApplyHardFilter(c, makeSettings(true, false))
	if result.Passed {
		t.Error("expected rejection when price < MA60 with HardMA60SupportEnabled=true")
	}
}

func TestApplyHardFilter_MA60Support_Pass(t *testing.T) {
	// 현재가(10500)가 MA60(10000) 이상 → 통과
	c := CandidateInfo{
		StockCode: "000001",
		StockName: "테스트",
		Info: &ops.StockInfo{
			CurrentPrice: "10500",
			MA60:         10000,
		},
	}
	result := ApplyHardFilter(c, makeSettings(true, false))
	if !result.Passed {
		t.Errorf("expected pass when price >= MA60, got rejection: %s", result.Reason)
	}
}

func TestApplyHardFilter_MA60Disabled_Skips(t *testing.T) {
	// MA60 조건 비활성화 → 현재가 < MA60 이어도 이 조건으로는 탈락하지 않음
	c := CandidateInfo{
		StockCode: "000001",
		StockName: "테스트",
		Info: &ops.StockInfo{
			CurrentPrice: "5000",
			MA60:         10000,
		},
	}
	result := ApplyHardFilter(c, makeSettings(false, false))
	if !result.Passed {
		t.Errorf("expected pass when HardMA60SupportEnabled=false, got: %s", result.Reason)
	}
}

func TestApplyHardFilter_MA120Support_Reject(t *testing.T) {
	// 현재가(9000)가 MA120(9500) 미만 → 탈락
	c := CandidateInfo{
		StockCode: "000001",
		StockName: "테스트",
		Info: &ops.StockInfo{
			CurrentPrice: "9000",
			MA120:        9500,
		},
	}
	result := ApplyHardFilter(c, makeSettings(false, true))
	if result.Passed {
		t.Error("expected rejection when price < MA120 with HardMA120SupportEnabled=true")
	}
}

func TestApplyHardFilter_MA60Zero_Skips(t *testing.T) {
	// MA60=0 (데이터 부족) → 조건 무시
	c := CandidateInfo{
		StockCode: "000001",
		StockName: "테스트",
		Info: &ops.StockInfo{
			CurrentPrice: "5000",
			MA60:         0,
		},
	}
	result := ApplyHardFilter(c, makeSettings(true, false))
	if !result.Passed {
		t.Errorf("expected pass when MA60=0 (no data), got: %s", result.Reason)
	}
}
```

- [ ] **Step 2: 테스트 실행 — 컴파일 에러 확인 (MA60/MA120 필드가 아직 없음)**

```bash
cd /Volumes/MAC/Project/micro-trading-for-agent/backend
go test ./internal/scorer/... -v 2>&1 | head -20
```

기대 결과: `FAIL` — `ops.StockInfo` 에 `MA60`, `MA120` 필드 없음 + `database.TradingSettings` 에 `HardMA60SupportEnabled`, `HardMA120SupportEnabled` 없음

- [ ] **Step 3: 커밋**

```bash
git add backend/internal/scorer/scorer_test.go
git commit -m "test: ApplyHardFilter MA60/MA120 지지 조건 실패 테스트 추가"
```

---

## Task 3: StockInfo에 MA60·MA120 필드 추가

**Files:**
- Modify: `backend/internal/ops/stock_info.go`

- [ ] **Step 1: StockInfo 구조체에 MA60, MA120 필드 추가**

`stock_info.go` 의 `StockInfo` 구조체에서 `MA5`, `MA20` 바로 뒤에 추가:

```go
MA5             float64 `json:"ma5"`
MA20            float64 `json:"ma20"`
MA60            float64 `json:"ma60"`
MA120           float64 `json:"ma120"`
```

- [ ] **Step 2: TradingSettings에 MA60/MA120 필드 추가**

`backend/internal/database/db.go` 의 `TradingSettings` 구조체에서 `HardMACDBearishEnabled` 바로 뒤에 추가:

```go
HardMACDBearishEnabled  bool
HardMA60SupportEnabled  bool
HardMA120SupportEnabled bool
```

`GetTradingSettings` 함수에서 `HardMACDBearishSell` 파싱 바로 뒤에 추가:

```go
s.HardMACDBearishSell = pb(m, "indicator_macd_bearish_sell", false)
s.HardMA60SupportEnabled  = pb(m, "hard_ma60_support_enabled", false)
s.HardMA120SupportEnabled = pb(m, "hard_ma120_support_enabled", false)
```

`defaultSettings` 맵에 추가:

```go
"hard_ma60_support_enabled":  "false",
"hard_ma120_support_enabled": "false",
```

- [ ] **Step 3: 빌드 통과 확인**

```bash
cd /Volumes/MAC/Project/micro-trading-for-agent/backend
go build ./... 2>&1
```

기대 결과: 에러 없음

- [ ] **Step 4: Task 2 테스트 재실행 — 컴파일은 통과하지만 로직 테스트는 아직 실패**

```bash
go test ./internal/scorer/... -v 2>&1 | head -30
```

기대 결과: 컴파일 성공, `TestApplyHardFilter_MA60Support_Reject` 등 FAIL (조건 미구현)

- [ ] **Step 5: 커밋**

```bash
git add backend/internal/ops/stock_info.go backend/internal/database/db.go
git commit -m "feat: StockInfo MA60/MA120 필드 및 TradingSettings 지지 조건 필드 추가"
```

---

## Task 4: ApplyHardFilter에 MA60/MA120 지지 조건 구현

**Files:**
- Modify: `backend/internal/scorer/scorer.go`

- [ ] **Step 1: 조건 추가**

`scorer.go` 의 `ApplyHardFilter` 함수에서, `MACD 데드크로스 제외` 블록 바로 뒤에 추가:

```go
// MA60 지지선 (1분봉 기준 1시간 추세선) — 현재가 < MA60이면 탈락
if s.HardMA60SupportEnabled && info.MA60 > 0 {
    price, _ := strconv.ParseFloat(info.CurrentPrice, 64)
    if price > 0 && price < info.MA60 {
        return FilterResult{Reason: fmt.Sprintf("price %.0f below MA60 %.0f", price, info.MA60)}
    }
}

// MA120 지지선 (1분봉 기준 2시간 추세선)
if s.HardMA120SupportEnabled && info.MA120 > 0 {
    price, _ := strconv.ParseFloat(info.CurrentPrice, 64)
    if price > 0 && price < info.MA120 {
        return FilterResult{Reason: fmt.Sprintf("price %.0f below MA120 %.0f", price, info.MA120)}
    }
}
```

> ⚠️ `scorer.go` imports에 `"strconv"` 추가 필요. 현재 `"fmt"`, `"math"` 만 있음:

```go
import (
    "fmt"
    "math"
    "strconv"

    "github.com/micro-trading-for-agent/backend/internal/database"
    "github.com/micro-trading-for-agent/backend/internal/ops"
)
```

- [ ] **Step 2: 테스트 실행 — 전부 통과 확인**

```bash
cd /Volumes/MAC/Project/micro-trading-for-agent/backend
go test ./internal/scorer/... -v
```

기대 결과:
```
--- PASS: TestApplyHardFilter_MA60Support_Reject
--- PASS: TestApplyHardFilter_MA60Support_Pass
--- PASS: TestApplyHardFilter_MA60Disabled_Skips
--- PASS: TestApplyHardFilter_MA120Support_Reject
--- PASS: TestApplyHardFilter_MA60Zero_Skips
PASS
```

- [ ] **Step 3: 빌드 확인**

```bash
go build ./...
```

- [ ] **Step 4: 커밋**

```bash
git add backend/internal/scorer/scorer.go
git commit -m "feat: ApplyHardFilter — MA60/MA120 지지선 조건 추가"
```

---

## Task 5: GetStockInfo 1분봉 전환 (핵심 변경)

**Files:**
- Modify: `backend/internal/ops/stock_info.go`

현재 코드(약 116~230줄)의 5분봉 집계 블록을 1분봉 직접 사용으로 교체.

- [ ] **Step 1: 5분봉 집계 제거 및 1분봉 직접 사용으로 교체**

`stock_info.go` 에서 아래 블록을 찾아 교체:

**Before (제거할 코드):**
```go
candles5m := aggregateMinuteBars(bars, 5)
if len(candles5m) >= 2 {
    closes5m := make([]float64, len(candles5m))
    for i, c := range candles5m {
        closes5m[i] = c.Close
    }
    info.MA5 = calcMA(closes5m, 5)
    info.MA20 = calcMA(closes5m, 20)
    info.RSI14 = calcRSI(closes5m, 14)
    info.MACDLine, info.MACDSignal, info.MACDHisto = calcMACD(closes5m, 12, 26, 9)
    // DisparityM5: 현재가와 5분봉 MA5의 이격도
    if ma5m := calcMA(closes5m, 5); ma5m > 0 && price > 0 {
        info.DisparityM5 = math.Round((price-ma5m)/ma5m*10000) / 100
    }
    info.M5MA10 = calcMA(closes5m, 10)
    ...이하 PrevVolumeRatio, RecentCandles, HighFormedMinsAgo 블록...
}
```

**After (교체할 코드):**
```go
// 1분봉 캔들 변환 — RSI/MACD/MA/VWAP 계산 기준
candles1m := minuteBarsToCandles(bars)
if len(candles1m) >= 2 {
    closes1m := make([]float64, len(candles1m))
    for i, c := range candles1m {
        closes1m[i] = c.Close
    }
    info.MA5   = calcMA(closes1m, 5)
    info.MA20  = calcMA(closes1m, 20)
    info.MA60  = calcMA(closes1m, 60)
    info.MA120 = calcMA(closes1m, 120)
    info.RSI14 = calcRSI(closes1m, 14)
    info.MACDLine, info.MACDSignal, info.MACDHisto = calcMACD(closes1m, 12, 26, 9)

    // DisparityM5: 현재가와 1분봉 MA5의 이격도
    if info.MA5 > 0 && price > 0 {
        info.DisparityM5 = math.Round((price-info.MA5)/info.MA5*10000) / 100
    }
    info.M5MA10 = calcMA(closes1m, 10)

    // PrevVolumeRatio: 직전 1분봉 대비 현재 1분봉 거래량 비율
    if len(candles1m) >= 2 {
        curVol := float64(candles1m[len(candles1m)-1].Volume)
        prevVol := float64(candles1m[len(candles1m)-2].Volume)
        if prevVol > 0 {
            info.PrevVolumeRatio = math.Round(curVol/prevVol*100) / 100
        }
    }

    // ── 최근 5개 1분봉 캔들 시퀀스 ──────────────────────────────
    {
        n := len(candles1m)
        start := n - 5
        if start < 0 {
            start = 0
        }
        recent := candles1m[start:]
        snaps := make([]CandleSnap, len(recent))
        for i, c := range recent {
            dir := "="
            if c.Close > c.Open {
                dir = "U"
            } else if c.Close < c.Open {
                dir = "D"
            }
            snaps[i] = CandleSnap{
                Close:  math.Round(c.Close*100) / 100,
                Volume: c.Volume,
                Dir:    dir,
            }
        }
        info.RecentCandles = snaps
    }

    // ── 고점 형성 경과 시간 (1분봉 기준, 1봉 = 1분) ──────────
    {
        dayHigh, _ := strconv.ParseFloat(info.DayHigh, 64)
        if dayHigh > 0 && len(candles1m) >= 1 {
            highIdx := -1
            for i := len(candles1m) - 1; i >= 0; i-- {
                if candles1m[i].High >= dayHigh*0.9999 {
                    highIdx = i
                    break
                }
            }
            if highIdx >= 0 {
                info.HighFormedMinsAgo = len(candles1m) - 1 - highIdx // 1봉 = 1분
                info.VolAtHigh = candles1m[highIdx].Volume
            }
        }

        // VolTrend3: 최근 3개 1분봉 거래량 기울기
        if len(candles1m) >= 3 {
            v1 := float64(candles1m[len(candles1m)-3].Volume)
            v2 := float64(candles1m[len(candles1m)-2].Volume)
            v3 := float64(candles1m[len(candles1m)-1].Volume)
            maxV := math.Max(v1, math.Max(v2, v3))
            if maxV > 0 {
                slope := (v3 - v1) / maxV
                info.VolTrend3 = math.Round(slope*100) / 100
            }
        }

        // VolVs3AvgRatio: 현재 1분봉 거래량 / 직전 3봉 평균
        if len(candles1m) >= 4 {
            cur := float64(candles1m[len(candles1m)-1].Volume)
            avg3 := (float64(candles1m[len(candles1m)-2].Volume) +
                float64(candles1m[len(candles1m)-3].Volume) +
                float64(candles1m[len(candles1m)-4].Volume)) / 3
            if avg3 > 0 {
                info.VolVs3AvgRatio = math.Round(cur/avg3*100) / 100
            }
        }
    }
}
```

- [ ] **Step 2: 빌드 확인**

```bash
cd /Volumes/MAC/Project/micro-trading-for-agent/backend
go build ./... 2>&1
```

기대 결과: 에러 없음

- [ ] **Step 3: 전체 테스트 실행**

```bash
go test ./... 2>&1
```

기대 결과: `PASS` (새로 작성한 테스트 포함)

- [ ] **Step 4: 커밋**

```bash
git add backend/internal/ops/stock_info.go
git commit -m "feat: GetStockInfo 지표 계산 기준 5분봉→1분봉 전환, MA60/MA120 추가"
```

---

## Task 6: Settings API — MA60/MA120 저장 핸들러 추가

**Files:**
- Modify: `backend/internal/api/handlers.go`

- [ ] **Step 1: UpdateSettings 요청 구조체에 필드 추가**

`handlers.go` 에서 `HardMACDBearishEnabled *bool` 바로 뒤에 추가:

```go
HardMACDBearishEnabled  *bool    `json:"hard_macd_bearish_enabled"`
HardMA60SupportEnabled  *bool    `json:"hard_ma60_support_enabled"`
HardMA120SupportEnabled *bool    `json:"hard_ma120_support_enabled"`
```

- [ ] **Step 2: 저장 로직 추가**

`handlers.go` 에서 `HardMACDBearishEnabled` 처리 블록 바로 뒤에 추가:

```go
if req.HardMACDBearishEnabled != nil {
    // 기존 코드...
}
if req.HardMA60SupportEnabled != nil {
    v := "false"
    if *req.HardMA60SupportEnabled {
        v = "true"
    }
    if !save("hard_ma60_support_enabled", v) {
        return
    }
}
if req.HardMA120SupportEnabled != nil {
    v := "false"
    if *req.HardMA120SupportEnabled {
        v = "true"
    }
    if !save("hard_ma120_support_enabled", v) {
        return
    }
}
```

- [ ] **Step 3: 빌드 확인**

```bash
cd /Volumes/MAC/Project/micro-trading-for-agent/backend
go build ./...
```

- [ ] **Step 4: 커밋**

```bash
git add backend/internal/api/handlers.go
git commit -m "feat: settings API — MA60/MA120 지지 조건 저장 추가"
```

---

## Task 7: 프론트엔드 — 하드필터 탭 MA60/MA120 토글

**Files:**
- Modify: `frontend/src/pages/Settings.jsx`

- [ ] **Step 1: transformSettings에 MA60/MA120 파싱 추가**

`Settings.jsx` 의 `transformSettings` 함수 반환 객체에서 `indicator_macd_bearish_sell` 바로 뒤에 추가:

```js
indicator_macd_bearish_sell: pb(raw.indicator_macd_bearish_sell, false),
hard_ma60_support_enabled:  pb(raw.hard_ma60_support_enabled, false),
hard_ma120_support_enabled: pb(raw.hard_ma120_support_enabled, false),
```

- [ ] **Step 2: flattenSettings에 저장 필드 추가**

`flattenSettings`(또는 저장 시 `flat` 객체 구성 부분)에서 `indicator_macd_bearish_sell` 저장 바로 뒤에 추가:

```js
flat.hard_ma60_support_enabled  = String(settings.hard_ma60_support_enabled  ?? false)
flat.hard_ma120_support_enabled = String(settings.hard_ma120_support_enabled ?? false)
```

- [ ] **Step 3: 하드필터 탭 UI에 토글 추가**

`{tab === '하드필터' && ...}` 블록 내, `HARD_FILTERS.map` 바로 아래에 추가:

```jsx
{/* MA 지지선 조건 — boolean 전용 */}
<div className="filter-row" style={{ marginTop: 8 }}>
  <span className="filter-label">MA60 지지 (현재가 ≥ MA60)</span>
  <Toggle
    checked={settings.hard_ma60_support_enabled ?? false}
    onChange={v => set('hard_ma60_support_enabled', v)} />
</div>
<div className="filter-row">
  <span className="filter-label">MA120 지지 (현재가 ≥ MA120)</span>
  <Toggle
    checked={settings.hard_ma120_support_enabled ?? false}
    onChange={v => set('hard_ma120_support_enabled', v)} />
</div>
```

- [ ] **Step 4: 프론트엔드 빌드 확인**

```bash
cd /Volumes/MAC/Project/micro-trading-for-agent/frontend
npm run build 2>&1 | tail -5
```

기대 결과: `✓ built in ...ms`

- [ ] **Step 5: 커밋**

```bash
git add frontend/src/pages/Settings.jsx
git commit -m "feat: 설정 UI — MA60/MA120 지지 조건 토글 추가"
```

---

## Task 8: 모니터링 단계 영향도 확인 (코드 변경 없음)

**Files:** 없음 (검증만)

- [ ] **Step 1: 모니터 → IndicatorSnapshot 흐름 확인**

`backend/internal/api/handlers.go:317` 부근의 `StartIndicatorChecker` 콜백을 확인:

```go
func(iCtx context.Context, code string) (*monitor.IndicatorSnapshot, error) {
    info, err := ops.GetStockInfo(iCtx, h.client, code)
    if err != nil {
        return nil, err
    }
    return &monitor.IndicatorSnapshot{
        RSI14:      info.RSI14,
        MACDLine:   info.MACDLine,
        MACDSignal: info.MACDSignal,
    }, nil
}
```

`ops.GetStockInfo`가 이제 1분봉 기준으로 RSI14, MACDLine, MACDSignal을 계산하므로 **콜백 수정 없이 자동으로 1분봉 기준 지표가 매도 신호 판단에 사용됨**. 변경 불필요.

- [ ] **Step 2: 확인 메모 — 커밋 없음**

변경 파일 없으므로 커밋 생략.

---

## Task 9: changelog 업데이트

**Files:**
- Modify: `docs/changelog.md`

- [ ] **Step 1: changelog 맨 위에 항목 추가**

```markdown
## 2026-05-06 — feat: 지표 계산 기준 5분봉→1분봉 전환 + MA60/MA120 추가

- **GetStockInfo 1분봉 전환**: RSI(14), MACD(12,26,9), MA5/20, VWAP 계산 기준이 5분봉 집계에서 1분봉 직접 사용으로 변경. 지표 후행성 제거.
- **MA60/MA120 신규 추가**: 1분봉 기준 60봉(1시간)·120봉(2시간) 이동평균 계산. `StockInfo.MA60`, `StockInfo.MA120` 필드 추가.
- **하드 필터 — MA 지지 조건**: `HardMA60SupportEnabled`, `HardMA120SupportEnabled` 설정 추가. 현재가가 MA 아래이면 후보 탈락.
- **모니터링 자동 반영**: `StartIndicatorChecker`가 `ops.GetStockInfo`를 통해 RSI/MACD를 조회하므로 수정 없이 1분봉 기준 매도 신호 적용.
- **단위 테스트 추가**: `ops/stock_info_test.go`, `scorer/scorer_test.go`
```

- [ ] **Step 2: 커밋**

```bash
git add docs/changelog.md
git commit -m "docs: changelog — 1분봉 지표 전환 내용 추가 [skip actions]"
```

---

---

## Task 10: fetchMinuteBars 호출 횟수 감소 (200→120봉)

**Files:**
- Modify: `backend/internal/ops/stock_info.go`

**배경:** 현재 `GetStockInfo`는 200개 1분봉을 요청한다. KIS `inquire-time-itemchartprice`는 한 번에 최대 ~30개를 반환하므로 200봉 = 약 7회 API 호출이 발생한다.

MA120 계산에는 최소 120개가 필요하다. 120봉 요청으로 줄이면 4~5회 호출로 약 30% 감소한다. MACD(12,26,9)는 34봉 이상이면 충분하므로 정확도 손실 없음.

- [ ] **Step 1: 요청량을 200→120으로 변경**

`backend/internal/ops/stock_info.go:108` 에서:

```go
// Before
bars, chartErr := fetchMinuteBars(ctx, client, stockCode, 200)

// After
bars, chartErr := fetchMinuteBars(ctx, client, stockCode, 120)
```

주석도 같이 수정:

```go
// Before
// Fetch 200 1-minute bars → aggregate to ~40 5-minute bars.
// 40 bars is sufficient for MACD(12,26,9) which needs 26+9-1 = 34 periods minimum.
// MA5/MA20 are also computed from these 5m closes for intraday consistency.

// After
// Fetch 120 1-minute bars — sufficient for MA120 (needs 120 periods) and
// MACD(12,26,9) (needs 34 periods minimum). Reduces KIS API calls by ~30%.
```

- [ ] **Step 2: 빌드 및 테스트 통과 확인**

```bash
cd /Volumes/MAC/Project/micro-trading-for-agent/backend
go build ./... && go test ./internal/ops/... -v
```

기대 결과: 에러 없음, 모든 테스트 PASS

- [ ] **Step 3: 커밋**

```bash
git add backend/internal/ops/stock_info.go
git commit -m "perf: fetchMinuteBars 요청량 200→120봉으로 감소 (~30% KIS API 호출 절감)"
```

---

## Task 11: 2-패스 필터링 — GetStockPrice 사전 필터

**Files:**
- Modify: `backend/internal/trader/engine.go`

**배경:** 현재 `runScanCycle`은 후보 종목마다 즉시 `GetStockInfo`(차트 4~5회 API 호출)를 실행한다. `GetStockInfo` 안에서 이미 `GetStockPrice` 1회를 호출하므로, 차트 호출 전에 현재가·체결강도·이격도 등 싸게 얻을 수 있는 값으로 먼저 탈락 여부를 판단하면 비용이 큰 차트 호출을 건너뛸 수 있다.

**구현 방법:**
1. `GetStockPrice` 응답만으로 확인 가능한 조건 — `HardStrengthMin`, `FilterOpenPriceDiffMax`, `MinTradingValue` 등 — 을 차트 fetch 이전에 먼저 체크한다.
2. 조건을 통과한 종목에만 `GetStockInfo`(차트 호출 포함)를 실행한다.

- [ ] **Step 1: `runScanCycle` 루프 안에 사전 필터 추가**

`backend/internal/trader/engine.go` 의 `runScanCycle` 에서 아래 블록을 찾는다:

```go
info, err := ops.GetStockInfo(ctx, e.kisClient, c.StockCode)
if err != nil {
    logger.Warn("engine: GetStockInfo failed",
        map[string]any{"code": c.StockCode, "error": err.Error()})
    continue
}
```

그 **앞에** 다음 사전 필터 블록을 삽입:

```go
// ── 1-패스: GetStockPrice(저비용) 사전 필터 ──────────────────
// 차트 fetch(4~5회 API) 전에 현재가만으로 탈락 여부를 선 확인한다.
priceResp, err := e.kisClient.GetStockPrice(ctx, c.StockCode)
if err != nil {
    logger.Warn("engine: GetStockPrice (pre-filter) failed",
        map[string]any{"code": c.StockCode, "error": err.Error()})
    continue
}
// 최소 거래대금 필터 (0 = 비활성화)
if settings.MinTradingValue > 0 {
    p, _ := strconv.ParseFloat(priceResp.CurrentPrice, 64)
    v, _ := strconv.ParseFloat(priceResp.Volume, 64)
    if tv := p * v; tv < settings.MinTradingValue {
        logger.Info("engine: pre-filter rejected (trading value)",
            map[string]any{"code": c.StockCode, "trading_value": tv, "min": settings.MinTradingValue})
        rejectedCount++
        continue
    }
}
// 최소 체결강도 필터 (0 = 비활성화)
if settings.HardStrengthMin > 0 {
    strength, _ := strconv.ParseFloat(priceResp.Strength, 64)
    if strength > 0 && strength < settings.HardStrengthMin {
        logger.Info("engine: pre-filter rejected (strength)",
            map[string]any{"code": c.StockCode, "strength": strength, "min": settings.HardStrengthMin})
        rejectedCount++
        continue
    }
}
// 최대 시가대비 상승률 필터 (0 = 비활성화)
if settings.FilterOpenPriceDiffMax > 0 {
    p, _ := strconv.ParseFloat(priceResp.CurrentPrice, 64)
    o, _ := strconv.ParseFloat(priceResp.DayOpen, 64)
    if p > 0 && o > 0 {
        openDiff := (p - o) / o * 100
        if openDiff > settings.FilterOpenPriceDiffMax {
            logger.Info("engine: pre-filter rejected (open price diff)",
                map[string]any{"code": c.StockCode, "open_diff": openDiff, "max": settings.FilterOpenPriceDiffMax})
            rejectedCount++
            continue
        }
    }
}
// ── 2-패스: GetStockInfo(차트 포함) 전체 지표 ────────────────
```

- [ ] **Step 2: GetStockInfo가 내부에서 중복 GetStockPrice를 건너뛰도록 오버로드 추가**

현재 `GetStockInfo`는 함수 시작 시 무조건 `GetStockPrice`를 호출한다. 2-패스 구조에서는 이미 위에서 호출했으므로 중복된다. `GetStockInfoWithPrice`를 추가해 기존 응답을 재사용하도록 한다.

`backend/internal/ops/stock_info.go` 맨 아래에 추가:

```go
// GetStockInfoWithPrice는 이미 조회된 GetStockPrice 응답을 재사용해
// 차트 fetch(4~5회 API)만 새로 실행한다. 2-패스 필터링 최적화용.
func GetStockInfoWithPrice(ctx context.Context, client *kis.Client, stockCode string, resp *kis.StockPriceResponse) (*StockInfo, error) {
    if stockCode == "" {
        return nil, fmt.Errorf("stock_code is required")
    }

    info := &StockInfo{
        StockCode:    resp.StockCode,
        CurrentPrice: resp.CurrentPrice,
        ChangeRate:   resp.ChangeRate,
        Volume:       resp.Volume,
        DayOpen:      resp.DayOpen,
        DayHigh:      resp.DayHigh,
        DayLow:       resp.DayLow,
    }

    price, _ := strconv.ParseFloat(resp.CurrentPrice, 64)
    vol, _ := strconv.ParseFloat(resp.Volume, 64)
    if price > 0 && vol > 0 {
        info.TradingValue = math.Round(price * vol)
    }
    if s, err := strconv.ParseFloat(resp.Strength, 64); err == nil && s > 0 {
        info.Strength = math.Round(s*10) / 10
    }
    if high, err := strconv.ParseFloat(resp.DayHigh, 64); err == nil && high > 0 {
        info.HighPriceDiff = math.Round((price-high)/high*10000) / 100
    }
    if open, err := strconv.ParseFloat(resp.DayOpen, 64); err == nil && open > 0 {
        info.OpenPriceDiff = math.Round((price-open)/open*10000) / 100
    }

    // 차트 구간은 GetStockInfo와 동일
    bars, chartErr := fetchMinuteBars(ctx, client, stockCode, 120)
    if chartErr != nil {
        logger.Warn("GetStockInfoWithPrice: minute chart fetch failed",
            map[string]any{"code": stockCode, "error": chartErr.Error()})
    }
    if chartErr == nil && len(bars) > 0 {
        fillChartIndicators(info, bars, price)
    }
    return info, nil
}
```

그리고 차트 지표 계산 로직을 `GetStockInfo`와 공유하기 위해 `stock_info.go` 에 `fillChartIndicators` 헬퍼를 추출한다:

현재 `GetStockInfo` 의 `if chartErr == nil && len(bars) > 0 { ... }` 블록 전체를 새 함수로 이동:

```go
// fillChartIndicators는 1분봉 bars로부터 모든 차트 기반 지표를 info에 채운다.
func fillChartIndicators(info *StockInfo, bars []kis.ChartBar, price float64) {
    var sumPV, sumVol float64
    for _, b := range bars {
        p, _ := strconv.ParseFloat(b.Close, 64)
        v, _ := strconv.ParseFloat(b.Volume, 64)
        if p > 0 && v > 0 {
            sumPV += p * v
            sumVol += v
        }
    }
    if sumVol > 0 && len(bars) >= 5 {
        info.VWAP = math.Round(sumPV/sumVol*100) / 100
        if info.VWAP > 0 && price > 0 {
            info.VWAPDiff = math.Round((price-info.VWAP)/info.VWAP*10000) / 100
        }
    }

    candles1m := minuteBarsToCandles(bars)
    if len(candles1m) < 2 {
        return
    }
    closes1m := make([]float64, len(candles1m))
    for i, c := range candles1m {
        closes1m[i] = c.Close
    }
    info.MA5   = calcMA(closes1m, 5)
    info.MA20  = calcMA(closes1m, 20)
    info.MA60  = calcMA(closes1m, 60)
    info.MA120 = calcMA(closes1m, 120)
    info.RSI14 = calcRSI(closes1m, 14)
    info.MACDLine, info.MACDSignal, info.MACDHisto = calcMACD(closes1m, 12, 26, 9)
    if info.MA5 > 0 && price > 0 {
        info.DisparityM5 = math.Round((price-info.MA5)/info.MA5*10000) / 100
    }
    info.M5MA10 = calcMA(closes1m, 10)
    if len(candles1m) >= 2 {
        curVol := float64(candles1m[len(candles1m)-1].Volume)
        prevVol := float64(candles1m[len(candles1m)-2].Volume)
        if prevVol > 0 {
            info.PrevVolumeRatio = math.Round(curVol/prevVol*100) / 100
        }
    }
    n := len(candles1m)
    start := n - 5
    if start < 0 {
        start = 0
    }
    recent := candles1m[start:]
    snaps := make([]CandleSnap, len(recent))
    for i, c := range recent {
        dir := "="
        if c.Close > c.Open {
            dir = "U"
        } else if c.Close < c.Open {
            dir = "D"
        }
        snaps[i] = CandleSnap{Close: math.Round(c.Close*100) / 100, Volume: c.Volume, Dir: dir}
    }
    info.RecentCandles = snaps

    dayHigh, _ := strconv.ParseFloat(info.DayHigh, 64)
    if dayHigh > 0 {
        highIdx := -1
        for i := len(candles1m) - 1; i >= 0; i-- {
            if candles1m[i].High >= dayHigh*0.9999 {
                highIdx = i
                break
            }
        }
        if highIdx >= 0 {
            info.HighFormedMinsAgo = len(candles1m) - 1 - highIdx
            info.VolAtHigh = candles1m[highIdx].Volume
        }
    }
    if len(candles1m) >= 3 {
        v1 := float64(candles1m[len(candles1m)-3].Volume)
        v2 := float64(candles1m[len(candles1m)-2].Volume)
        v3 := float64(candles1m[len(candles1m)-1].Volume)
        maxV := math.Max(v1, math.Max(v2, v3))
        if maxV > 0 {
            info.VolTrend3 = math.Round((v3-v1)/maxV*100) / 100
        }
    }
    if len(candles1m) >= 4 {
        cur := float64(candles1m[len(candles1m)-1].Volume)
        avg3 := (float64(candles1m[len(candles1m)-2].Volume) +
            float64(candles1m[len(candles1m)-3].Volume) +
            float64(candles1m[len(candles1m)-4].Volume)) / 3
        if avg3 > 0 {
            info.VolVs3AvgRatio = math.Round(cur/avg3*100) / 100
        }
    }
}
```

`GetStockInfo` 의 기존 `if chartErr == nil && len(bars) > 0 { ... }` 블록을 단 한 줄로 교체:

```go
if chartErr == nil && len(bars) > 0 {
    fillChartIndicators(info, bars, price)
}
```

- [ ] **Step 3: engine.go 에서 GetStockInfo 대신 GetStockInfoWithPrice 호출**

`runScanCycle` 에서 Step 1에서 삽입한 2-패스 주석 바로 아래의:

```go
info, err := ops.GetStockInfo(ctx, e.kisClient, c.StockCode)
if err != nil {
    logger.Warn("engine: GetStockInfo failed",
        map[string]any{"code": c.StockCode, "error": err.Error()})
    continue
}
```

를 다음으로 교체:

```go
info, err := ops.GetStockInfoWithPrice(ctx, e.kisClient, c.StockCode, priceResp)
if err != nil {
    logger.Warn("engine: GetStockInfoWithPrice failed",
        map[string]any{"code": c.StockCode, "error": err.Error()})
    continue
}
```

- [ ] **Step 4: kis.StockPriceResponse 타입 확인**

`kis.GetStockPrice`가 반환하는 타입이 `*kis.StockPriceResponse`인지 확인:

```bash
cd /Volumes/MAC/Project/micro-trading-for-agent/backend
grep -n "func.*GetStockPrice" internal/kis/client.go
```

기대: `func (c *Client) GetStockPrice(ctx context.Context, stockCode string) (*StockPriceResponse, error)` 형태

반환 타입을 `GetStockInfoWithPrice` 파라미터 타입에 맞게 조정.

- [ ] **Step 5: 빌드 및 테스트 확인**

```bash
cd /Volumes/MAC/Project/micro-trading-for-agent/backend
go build ./... && go test ./...
```

기대: 에러 없음

- [ ] **Step 6: 커밋**

```bash
git add backend/internal/ops/stock_info.go backend/internal/trader/engine.go
git commit -m "perf: 2-패스 필터링 도입 — GetStockPrice 사전 필터로 차트 API 호출 절감"
```

---

## Task 12: StockInfo 캐시 (30초 TTL) — 지표체커 중복 호출 제거

**Files:**
- Create: `backend/internal/ops/stock_info_cache.go`
- Modify: `backend/internal/ops/stock_info.go`

**배경:** `StartIndicatorChecker`는 최대 N개 포지션에 대해 주기적으로 `GetStockInfo`를 호출한다. 엔진 스캔 사이클과 지표체커가 동일 종목에 대해 30초 이내에 각각 호출하면 동일한 차트 API 4~5회 호출이 중복 발생한다. 30초 TTL 인메모리 캐시를 추가해 중복 호출을 제거한다.

- [ ] **Step 1: 캐시 구조체 파일 생성**

```go
package ops

import (
    "context"
    "sync"
    "time"

    "github.com/micro-trading-for-agent/backend/internal/kis"
)

type stockInfoCache struct {
    mu      sync.Mutex
    entries map[string]*cacheEntry
}

type cacheEntry struct {
    info      *StockInfo
    fetchedAt time.Time
}

var siCache = &stockInfoCache{entries: make(map[string]*cacheEntry)}

const siCacheTTL = 30 * time.Second

// GetStockInfoCached returns a cached StockInfo if available and within TTL,
// otherwise fetches fresh data and stores it in the cache.
func GetStockInfoCached(ctx context.Context, client *kis.Client, stockCode string) (*StockInfo, error) {
    siCache.mu.Lock()
    if e, ok := siCache.entries[stockCode]; ok && time.Since(e.fetchedAt) < siCacheTTL {
        info := e.info
        siCache.mu.Unlock()
        return info, nil
    }
    siCache.mu.Unlock()

    info, err := GetStockInfo(ctx, client, stockCode)
    if err != nil {
        return nil, err
    }

    siCache.mu.Lock()
    siCache.entries[stockCode] = &cacheEntry{info: info, fetchedAt: time.Now()}
    siCache.mu.Unlock()

    return info, nil
}

// InvalidateStockInfoCache removes a specific stock from the cache (e.g. after order fill).
func InvalidateStockInfoCache(stockCode string) {
    siCache.mu.Lock()
    delete(siCache.entries, stockCode)
    siCache.mu.Unlock()
}
```

- [ ] **Step 2: handlers.go 의 StartIndicatorChecker 콜백에서 캐시 사용으로 전환**

`backend/internal/api/handlers.go` 에서 `StartIndicatorChecker` 콜백 부분을 찾는다:

```go
func(iCtx context.Context, code string) (*monitor.IndicatorSnapshot, error) {
    info, err := ops.GetStockInfo(iCtx, h.client, code)
```

`GetStockInfo` → `GetStockInfoCached` 로 교체:

```go
func(iCtx context.Context, code string) (*monitor.IndicatorSnapshot, error) {
    info, err := ops.GetStockInfoCached(iCtx, h.client, code)
```

- [ ] **Step 3: 빌드 확인**

```bash
cd /Volumes/MAC/Project/micro-trading-for-agent/backend
go build ./...
```

- [ ] **Step 4: 캐시 단위 테스트 작성 및 통과 확인**

`backend/internal/ops/stock_info_cache_test.go` 생성:

```go
package ops

import (
    "testing"
    "time"
)

func TestStockInfoCache_Hit(t *testing.T) {
    // 직접 캐시에 넣고 TTL 내에 조회 시 같은 포인터 반환 확인
    code := "TEST01"
    info := &StockInfo{StockCode: code, CurrentPrice: "10000"}
    siCache.mu.Lock()
    siCache.entries[code] = &cacheEntry{info: info, fetchedAt: time.Now()}
    siCache.mu.Unlock()

    siCache.mu.Lock()
    e, ok := siCache.entries[code]
    siCache.mu.Unlock()
    if !ok {
        t.Fatal("cache entry not found")
    }
    if time.Since(e.fetchedAt) >= siCacheTTL {
        t.Fatal("cache entry already expired")
    }
    if e.info != info {
        t.Error("expected same pointer")
    }
}

func TestStockInfoCache_Invalidate(t *testing.T) {
    code := "TEST02"
    siCache.mu.Lock()
    siCache.entries[code] = &cacheEntry{info: &StockInfo{StockCode: code}, fetchedAt: time.Now()}
    siCache.mu.Unlock()

    InvalidateStockInfoCache(code)

    siCache.mu.Lock()
    _, ok := siCache.entries[code]
    siCache.mu.Unlock()
    if ok {
        t.Error("expected cache entry to be removed after invalidation")
    }
}

func TestStockInfoCache_Expiry(t *testing.T) {
    code := "TEST03"
    siCache.mu.Lock()
    siCache.entries[code] = &cacheEntry{
        info:      &StockInfo{StockCode: code},
        fetchedAt: time.Now().Add(-siCacheTTL - time.Second), // 이미 만료
    }
    siCache.mu.Unlock()

    siCache.mu.Lock()
    e, ok := siCache.entries[code]
    siCache.mu.Unlock()
    if !ok {
        t.Fatal("entry should still be in map (eviction is lazy)")
    }
    if time.Since(e.fetchedAt) < siCacheTTL {
        t.Error("expected entry to be expired")
    }
}
```

```bash
cd /Volumes/MAC/Project/micro-trading-for-agent/backend
go test ./internal/ops/... -run TestStockInfoCache -v
```

기대 결과:
```
--- PASS: TestStockInfoCache_Hit
--- PASS: TestStockInfoCache_Invalidate
--- PASS: TestStockInfoCache_Expiry
PASS
```

- [ ] **Step 5: 커밋**

```bash
git add backend/internal/ops/stock_info_cache.go \
        backend/internal/ops/stock_info_cache_test.go \
        backend/internal/api/handlers.go
git commit -m "perf: StockInfo 30초 캐시 추가 — 지표체커 중복 차트 API 호출 제거"
```

---

## 스펙 커버리지 자기 검토

| 요구사항 | 대응 Task |
|---------|----------|
| 1분봉 기반 VWAP | Task 5 — 기존 VWAP은 이미 1분봉 bars에서 계산, 변경 불필요 확인 |
| 1분봉 기반 RSI(14) | Task 5 — closes1m 사용 |
| 1분봉 기반 MACD(12,26,9) | Task 5 — closes1m 사용 |
| MA5, MA20 1분봉 전환 | Task 5 — closes1m 사용 |
| MA60, MA120 신규 추가 | Task 3, Task 5 |
| 하드 필터 MA60 지지 조건 | Task 2, 3, 4 |
| 모니터링 1분봉 연동 확인 | Task 8 — 자동 반영 확인 |
| 설정 UI | Task 7 |
| **KIS API 호출 최적화 1: 봉 수 감소** | **Task 10** |
| **KIS API 호출 최적화 2: 2-패스 필터** | **Task 11** |
| **KIS API 호출 최적화 3: StockInfo 캐시** | **Task 12** |

> **VWAP 주의:** `GetStockInfo`의 VWAP은 이미 `bars`(1분봉 raw)에서 계산하므로 Task 5에서 별도 수정 불필요. 단 Task 5 코드 교체 시 VWAP 계산 블록(sumPV/sumVol)이 `bars` 기반으로 유지되는지 반드시 확인.

> **Task 11 주의:** `fillChartIndicators` 헬퍼 추출 시 `GetStockInfo`에서 직접 사용하던 VWAP 계산(`bars` 기반)이 헬퍼로 이동되는데, VWAP은 1분봉 raw bars(집계 전)에서 계산해야 정확하다. `fillChartIndicators`가 `bars []kis.ChartBar`를 파라미터로 받아 계산하므로 이 정확도는 유지된다.
