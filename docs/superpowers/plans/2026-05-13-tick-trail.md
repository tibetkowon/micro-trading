# 틱 기반 초정밀 트레일링 스탑 구현 플랜

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 기존 % 기반 트레일링 스탑을 유지하면서, 매수1호가(Bid1) + KRX 호가 단위 기반 다층(3-tier) 틱 트레일링 스탑을 추가하고 프론트엔드 설정 UI에서 모드 전환을 가능하게 한다.

**Architecture:** WebSocket `H0STCNT0` 응답의 `fields[11]`(BIDP1)을 파싱하여 `PriceEvent`에 포함, `HandlePrice()`에 `bid1` 인자 추가. `TrailingMode="tick"` 설정 시 3단계 상태머신(Tier0 손절 → Tier1 브레이크이븐 → Tier2 타이트)이 매 틱마다 평가된다. 기존 `TrailingMode="pct"` 로직은 완전히 보존.

**Tech Stack:** Go 1.26, Gin, Firebase Firestore, React 18 + Vite

---

## 파일 목록

| 역할 | 파일 | 유형 |
|------|------|------|
| KRX 틱 사이즈 계산 | `backend/internal/monitor/ticksize.go` | 신규 |
| 틱 사이즈 테스트 | `backend/internal/monitor/ticksize_test.go` | 신규 |
| 틱 트레일 로직 테스트 | `backend/internal/monitor/tick_trail_test.go` | 신규 |
| WebSocket 이벤트 구조체 | `backend/internal/kis/websocket.go` | 수정 |
| DB 설정 구조체·읽기·기본값 | `backend/internal/database/db.go` | 수정 |
| 포지션 모니터 (핵심) | `backend/internal/monitor/monitor.go` | 수정 |
| 매매 엔진 | `backend/internal/trader/engine.go` | 수정 |
| API 핸들러 | `backend/internal/api/handlers.go` | 수정 |
| 프론트엔드 설정 페이지 | `frontend/src/pages/Settings.jsx` | 수정 |
| 프론트엔드 MSW 목 | `frontend/src/mocks/handlers.js` | 수정 |

---

## Task 1: KRX 틱 사이즈 계산 함수

**Files:**
- Create: `backend/internal/monitor/ticksize.go`
- Create: `backend/internal/monitor/ticksize_test.go`

- [ ] **Step 1: 테스트 파일 작성**

`backend/internal/monitor/ticksize_test.go`:
```go
package monitor

import "testing"

func TestCalcTickSize(t *testing.T) {
	cases := []struct {
		price float64
		want  float64
	}{
		{500, 1},
		{999, 1},
		{1000, 5},
		{4999, 5},
		{5000, 10},
		{9999, 10},
		{10000, 50},
		{49999, 50},
		{50000, 100},
		{99999, 100},
		{100000, 500},
		{499999, 500},
		{500000, 1000},
		{1000000, 1000},
	}
	for _, tc := range cases {
		got := CalcTickSize(tc.price)
		if got != tc.want {
			t.Errorf("CalcTickSize(%.0f) = %.0f, want %.0f", tc.price, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: 테스트가 컴파일 실패하는지 확인**

```bash
cd backend && go test ./internal/monitor/... 2>&1 | head -5
```
Expected: `undefined: CalcTickSize`

- [ ] **Step 3: ticksize.go 구현**

`backend/internal/monitor/ticksize.go`:
```go
package monitor

// CalcTickSize returns the KRX 호가 단위 (tick size) for a given price.
func CalcTickSize(price float64) float64 {
	switch {
	case price < 1_000:
		return 1
	case price < 5_000:
		return 5
	case price < 10_000:
		return 10
	case price < 50_000:
		return 50
	case price < 100_000:
		return 100
	case price < 500_000:
		return 500
	default:
		return 1_000
	}
}
```

- [ ] **Step 4: 테스트 실행**

```bash
cd backend && go test ./internal/monitor/... -run TestCalcTickSize -v
```
Expected:
```
--- PASS: TestCalcTickSize (0.00s)
PASS
```

- [ ] **Step 5: 커밋**

```bash
git add backend/internal/monitor/ticksize.go backend/internal/monitor/ticksize_test.go
git commit -m "feat(monitor): KRX CalcTickSize 함수 추가"
```

---

## Task 2: PriceEvent에 Bid1Price 추가

**Files:**
- Modify: `backend/internal/kis/websocket.go`

- [ ] **Step 1: PriceEvent 구조체에 Bid1Price 추가**

`backend/internal/kis/websocket.go` — `PriceEvent` 구조체 수정:
```go
// PriceEvent carries a real-time price tick from KIS WebSocket.
type PriceEvent struct {
	StockCode string
	Price     float64
	Bid1Price float64   // BIDP1 (매수호가1) — H0STCNT0 fields[11]
	Qty       int
	Timestamp time.Time
}
```

- [ ] **Step 2: parsePriceData()에서 fields[11] 파싱**

`parsePriceData()` 함수 내에서 `qty` 파싱 바로 다음에 bid1 파싱 추가:

현재 코드 (websocket.go ~486):
```go
	qtyStr := fields[12]
	var qty int
	fmt.Sscanf(qtyStr, "%d", &qty)

	select {
	case c.PriceCh <- PriceEvent{
		StockCode: stockCode,
		Price:     price,
		Qty:       qty,
		Timestamp: time.Now(),
	}:
```

변경 후:
```go
	qtyStr := fields[12]
	var qty int
	fmt.Sscanf(qtyStr, "%d", &qty)

	var bid1 float64
	if len(fields) > 11 {
		fmt.Sscanf(fields[11], "%f", &bid1)
	}

	select {
	case c.PriceCh <- PriceEvent{
		StockCode: stockCode,
		Price:     price,
		Bid1Price: bid1,
		Qty:       qty,
		Timestamp: time.Now(),
	}:
```

- [ ] **Step 3: 빌드 확인**

```bash
cd backend && go build ./...
```
Expected: 오류 없음

- [ ] **Step 4: 커밋**

```bash
git add backend/internal/kis/websocket.go
git commit -m "feat(ws): PriceEvent에 Bid1Price 추가 (H0STCNT0 fields[11])"
```

---

## Task 3: TradingSettings 신규 필드 추가

**Files:**
- Modify: `backend/internal/database/db.go`

- [ ] **Step 1: TradingSettings 구조체에 6개 필드 추가**

`backend/internal/database/db.go` — `TradingSettings` 구조체의 `TrailingStopPct` 필드 바로 다음에 추가:

```go
	// 트레일링 스탑 (기존 유지)
	TrailingTriggerPct float64
	TrailingStopPct    float64
	// 트레일링 모드 선택 (공존)
	TrailingMode string // "pct" (기본) | "tick"
	// 틱 트레일 설정 (TrailingMode == "tick" 일 때만 적용)
	TickTier0StopLossTicks int     // X: 진입가 대비 -X틱 손절
	TickTier1TriggerPct    float64 // A: +A% 수익 시 Tier1 활성화
	TickTier1TrailTicks    int     // Y: 매수1호가 고점 대비 -Y틱
	TickTier2TriggerPct    float64 // B: +B% 수익 시 Tier2 활성화
	TickTier2TrailTicks    int     // Z: 매수1호가 고점 대비 -Z틱
```

- [ ] **Step 2: GetTradingSettings()에 6개 필드 읽기 추가**

`GetTradingSettings()` 함수에서 `s.TrailingStopPct = ...` 줄 바로 다음에 추가:

```go
	s.TrailingMode = ps(m, "trailing_mode", "pct")
	s.TickTier0StopLossTicks = pi(m, "tick_tier0_stop_loss_ticks", 3)
	s.TickTier1TriggerPct = pf(m, "tick_tier1_trigger_pct", 0.0)
	s.TickTier1TrailTicks = pi(m, "tick_tier1_trail_ticks", 5)
	s.TickTier2TriggerPct = pf(m, "tick_tier2_trigger_pct", 0.0)
	s.TickTier2TrailTicks = pi(m, "tick_tier2_trail_ticks", 2)
```

- [ ] **Step 3: Firestore 기본값 맵에 6개 키 추가**

`db.go` 내 기본값 맵(약 line 1062, `"trailing_trigger_pct": "0"` 바로 다음)에 추가:

```go
		"trailing_mode":              "pct",
		"tick_tier0_stop_loss_ticks": "3",
		"tick_tier1_trigger_pct":     "0",
		"tick_tier1_trail_ticks":     "5",
		"tick_tier2_trigger_pct":     "0",
		"tick_tier2_trail_ticks":     "2",
```

- [ ] **Step 4: 빌드 확인**

```bash
cd backend && go build ./...
```
Expected: 오류 없음

- [ ] **Step 5: 커밋**

```bash
git add backend/internal/database/db.go
git commit -m "feat(db): TradingSettings 틱 트레일 6개 필드 추가"
```

---

## Task 4: Monitor — TickTrailState 타입 정의 및 HandlePrice 확장

**Files:**
- Modify: `backend/internal/monitor/monitor.go`
- Create: `backend/internal/monitor/tick_trail_test.go`

- [ ] **Step 1: TickTrailState 구조체 정의**

`backend/internal/monitor/monitor.go` — `MonitoredEntry` 구조체 정의 바로 위에 삽입:

```go
// TickTrailState holds runtime state for tick-based multi-tier trailing stop.
type TickTrailState struct {
	// 설정값 (Register() 시점에 TradingSettings 에서 복사)
	Tier0StopLossTicks int
	Tier1TriggerPct    float64
	Tier1TrailTicks    int
	Tier2TriggerPct    float64
	Tier2TrailTicks    int
	// 런타임 상태 (HandlePrice 매 틱마다 갱신)
	CurrentTier   int     // 0=손절대기, 1=브레이크이븐, 2=타이트
	PeakBid1Price float64 // Tier1/2 활성 후 매수1호가 최고점
}
```

- [ ] **Step 2: MonitoredEntry에 TrailingMode + TickTrail 필드 추가**

`MonitoredEntry` 구조체의 `TrailingStopPct` 아래에 추가:

```go
	// 틱 트레일 (TrailingMode == "tick" 일 때 사용)
	TrailingMode string       // "pct" | "tick"
	TickTrail    TickTrailState
```

- [ ] **Step 3: evaluateTickTrail 함수 신규 추가**

`monitor.go` 파일 맨 아래에 추가:

```go
// evaluateTickTrail evaluates the tick-based multi-tier trailing stop.
// Returns (shouldSell bool, reason string).
// bid1 must be > 0; caller must guard.
func evaluateTickTrail(pos *MonitoredEntry, tradePrice, bid1 float64) (bool, string) {
	ts := &pos.TickTrail

	// Tier 2 승격 먼저 체크 (A/B% 동시 충족 시 바로 Tier2)
	if ts.CurrentTier < 2 && ts.Tier2TriggerPct > 0 {
		if tradePrice >= pos.FilledPrice*(1+ts.Tier2TriggerPct/100) {
			ts.CurrentTier = 2
			if bid1 > ts.PeakBid1Price {
				ts.PeakBid1Price = bid1
			}
		}
	}
	// Tier 1 승격
	if ts.CurrentTier < 1 && ts.Tier1TriggerPct > 0 {
		if tradePrice >= pos.FilledPrice*(1+ts.Tier1TriggerPct/100) {
			ts.CurrentTier = 1
			ts.PeakBid1Price = bid1
		}
	}

	switch ts.CurrentTier {
	case 0:
		if ts.Tier0StopLossTicks > 0 {
			stop := pos.FilledPrice - float64(ts.Tier0StopLossTicks)*CalcTickSize(pos.FilledPrice)
			if bid1 <= stop {
				return true, "틱트레일-Tier0-진입손절"
			}
		}
	case 1:
		if bid1 > ts.PeakBid1Price {
			ts.PeakBid1Price = bid1
		}
		stop := ts.PeakBid1Price - float64(ts.Tier1TrailTicks)*CalcTickSize(ts.PeakBid1Price)
		if bid1 <= stop {
			return true, "틱트레일-Tier1-브레이크이븐"
		}
	case 2:
		if bid1 > ts.PeakBid1Price {
			ts.PeakBid1Price = bid1
		}
		stop := ts.PeakBid1Price - float64(ts.Tier2TrailTicks)*CalcTickSize(ts.PeakBid1Price)
		if bid1 <= stop {
			return true, "틱트레일-Tier2-급등익절"
		}
	}
	return false, ""
}
```

- [ ] **Step 4: evaluateTickTrail 테스트 작성**

`backend/internal/monitor/tick_trail_test.go`:
```go
package monitor

import "testing"

func makeTrailPos(filledPrice float64, ts TickTrailState) *MonitoredEntry {
	return &MonitoredEntry{
		FilledPrice:  filledPrice,
		TrailingMode: "tick",
		TickTrail:    ts,
	}
}

func TestEvaluateTickTrail_Tier0_StopLoss(t *testing.T) {
	// 진입가 10000원, 틱사이즈=50원, 3틱 손절 → 9850 이하에서 청산
	pos := makeTrailPos(10000, TickTrailState{Tier0StopLossTicks: 3})
	// bid1=9851 → 유지
	sell, _ := evaluateTickTrail(pos, 10000, 9851)
	if sell {
		t.Error("9851 should not trigger stop (stop=9850)")
	}
	// bid1=9850 → 청산
	sell, reason := evaluateTickTrail(pos, 10000, 9850)
	if !sell {
		t.Error("9850 should trigger Tier0 stop")
	}
	if reason != "틱트레일-Tier0-진입손절" {
		t.Errorf("wrong reason: %s", reason)
	}
}

func TestEvaluateTickTrail_Tier1_Activation(t *testing.T) {
	// 진입가 10000원, Tier1 발동 +2%, 트레일 3틱(50원=150원)
	pos := makeTrailPos(10000, TickTrailState{
		Tier1TriggerPct: 2.0,
		Tier1TrailTicks: 3,
	})
	// 체결가 10200 (=+2%) → Tier1 활성화, peak=10200
	sell, _ := evaluateTickTrail(pos, 10200, 10200)
	if sell {
		t.Error("should not sell on activation tick")
	}
	if pos.TickTrail.CurrentTier != 1 {
		t.Errorf("expected Tier1, got %d", pos.TickTrail.CurrentTier)
	}
	if pos.TickTrail.PeakBid1Price != 10200 {
		t.Errorf("expected peak=10200, got %f", pos.TickTrail.PeakBid1Price)
	}
}

func TestEvaluateTickTrail_Tier1_Trail(t *testing.T) {
	// Tier1 이미 활성, peak=10400, 3틱=150원 → 10250 이하에서 청산
	pos := makeTrailPos(10000, TickTrailState{
		Tier1TriggerPct: 2.0,
		Tier1TrailTicks: 3,
		CurrentTier:     1,
		PeakBid1Price:   10400,
	})
	// bid1=10251 → 유지
	sell, _ := evaluateTickTrail(pos, 10300, 10251)
	if sell {
		t.Error("10251 should not trigger trail (stop=10250)")
	}
	// bid1=10250 → 청산
	sell, reason := evaluateTickTrail(pos, 10200, 10250)
	if !sell {
		t.Error("10250 should trigger Tier1 trail")
	}
	if reason != "틱트레일-Tier1-브레이크이븐" {
		t.Errorf("wrong reason: %s", reason)
	}
}

func TestEvaluateTickTrail_Tier2_SkipsTier1(t *testing.T) {
	// A=2%, B=4%, 동시에 +5% 도달 시 Tier2 직접 진입
	pos := makeTrailPos(10000, TickTrailState{
		Tier1TriggerPct: 2.0,
		Tier1TrailTicks: 5,
		Tier2TriggerPct: 4.0,
		Tier2TrailTicks: 2,
	})
	evaluateTickTrail(pos, 10500, 10500) // +5%
	if pos.TickTrail.CurrentTier != 2 {
		t.Errorf("expected Tier2, got %d", pos.TickTrail.CurrentTier)
	}
}

func TestEvaluateTickTrail_Tier2_Trail(t *testing.T) {
	// Tier2, peak=50000(틱사이즈=100원), 2틱=200원 → 49800 이하 청산
	pos := makeTrailPos(48000, TickTrailState{
		Tier2TriggerPct: 3.0,
		Tier2TrailTicks: 2,
		CurrentTier:     2,
		PeakBid1Price:   50000,
	})
	sell, reason := evaluateTickTrail(pos, 49900, 49800)
	if !sell {
		t.Error("49800 should trigger Tier2 trail")
	}
	if reason != "틱트레일-Tier2-급등익절" {
		t.Errorf("wrong reason: %s", reason)
	}
}

func TestEvaluateTickTrail_PeakUpdate(t *testing.T) {
	// Tier1 활성, peak 갱신 확인
	pos := makeTrailPos(10000, TickTrailState{
		Tier1TriggerPct: 2.0,
		Tier1TrailTicks: 3,
		CurrentTier:     1,
		PeakBid1Price:   10200,
	})
	evaluateTickTrail(pos, 10400, 10400)
	if pos.TickTrail.PeakBid1Price != 10400 {
		t.Errorf("peak should update to 10400, got %f", pos.TickTrail.PeakBid1Price)
	}
}
```

- [ ] **Step 5: 테스트 실행**

```bash
cd backend && go test ./internal/monitor/... -run TestEvaluateTickTrail -v
```
Expected: 모든 테스트 PASS

- [ ] **Step 6: HandlePrice() 시그니처 변경 및 틱 트레일 호출 추가**

`monitor.go`의 `HandlePrice` 함수 시그니처를 다음과 같이 변경:
```go
func (m *Monitor) HandlePrice(stockCode string, price float64, bid1 float64, isTest bool) {
```

함수 내 기존 `// 트레일링 스탑:` 블록(약 line 211) 바로 다음, `// 부분 익절:` 블록 이전에 틱 트레일 블록 추가:

```go
	// 틱 트레일 (TrailingMode == "tick")
	if pos.TrailingMode == "tick" && pos.FilledPrice > 0 && bid1 > 0 {
		m.mu.Lock()
		p, ok2 := m.positions[stockCode]
		if ok2 {
			shouldSell, reason := evaluateTickTrail(p, price, bid1)
			m.mu.Unlock()
			if shouldSell {
				logger.Info("monitor: TICK TRAIL hit",
					map[string]any{"stock_code": stockCode, "tier": p.TickTrail.CurrentTier,
						"bid1": bid1, "peak": p.TickTrail.PeakBid1Price, "reason": reason})
				if !isTest {
					if soldQty := m.executeSell(stockCode, p, reason); soldQty < 0 {
						return
					}
				}
				m.Remove(context.Background(), stockCode)
				return
			}
		} else {
			m.mu.Unlock()
		}
	}
```

- [ ] **Step 7: StartPriceConsumer() 호출부 업데이트**

`StartPriceConsumer()` 내 `HandlePrice` 호출 수정:
```go
			m.HandlePrice(ev.StockCode, ev.Price, ev.Bid1Price, false)
```

- [ ] **Step 8: 빌드 및 테스트**

```bash
cd backend && go build ./... && go test ./internal/monitor/...
```
Expected: 빌드 성공, 테스트 PASS

> **Note:** `HandlePrice` 시그니처 변경으로 인해 `trader/engine.go`에서 컴파일 에러가 발생할 수 있음. Task 5에서 즉시 수정.

- [ ] **Step 9: 커밋**

```bash
git add backend/internal/monitor/monitor.go backend/internal/monitor/tick_trail_test.go
git commit -m "feat(monitor): TickTrailState 추가 및 HandlePrice bid1 확장"
```

---

## Task 5: Engine — HandlePrice 호출부 및 MonitoredEntry 생성 업데이트

**Files:**
- Modify: `backend/internal/trader/engine.go`

- [ ] **Step 1: processPriceEvents HandlePrice 호출 업데이트**

`engine.go:1100` 부근, `processPriceEvents()` 함수 — `HandlePrice` 직접 호출 없이 `streamMon.AddTick(ev)`만 사용 중이므로 현재는 변경 불필요. 빌드 에러 여부 확인:

```bash
cd backend && go build ./internal/trader/...
```

컴파일 에러가 없으면 이 Step을 건너뛴다.

> `engine.go`는 `monitor.HandlePrice()`를 직접 호출하지 않고 `monitor.StartPriceConsumer()` 를 통해 간접 호출하므로 시그니처 변경 영향이 없다. 단, 테스트 파일 내 직접 호출이 있다면 수정.

- [ ] **Step 2: MonitoredEntry 생성 시 틱 트레일 필드 전달**

`engine.go:890` 부근 `entry := monitor.MonitoredEntry{...}` 블록에 신규 필드 추가:

```go
	entry := monitor.MonitoredEntry{
		StockCode:          c.StockCode,
		StockName:          c.StockName,
		FilledPrice:        filledPrice,
		TargetPrice:        targetPrice,
		StopPrice:          stopPrice,
		OrderID:            result.OrderID,
		Qty:                qty,
		Market:             "KR",
		AssetType:          c.Info.AssetType,
		SoldCh:             e.soldCh,
		TrailingTriggerPct: settings.TrailingTriggerPct,
		TrailingStopPct:    settings.TrailingStopPct,
		SellOnUpperLimit:   settings.SellOnUpperLimit,
		TrailingMode:       settings.TrailingMode,
		TickTrail: monitor.TickTrailState{
			Tier0StopLossTicks: settings.TickTier0StopLossTicks,
			Tier1TriggerPct:    settings.TickTier1TriggerPct,
			Tier1TrailTicks:    settings.TickTier1TrailTicks,
			Tier2TriggerPct:    settings.TickTier2TriggerPct,
			Tier2TrailTicks:    settings.TickTier2TrailTicks,
		},
	}
```

- [ ] **Step 3: 전체 빌드 및 테스트**

```bash
cd backend && go build ./... && go test ./...
```
Expected: 빌드 성공, 기존 테스트 PASS

- [ ] **Step 4: 커밋**

```bash
git add backend/internal/trader/engine.go
git commit -m "feat(engine): MonitoredEntry에 틱 트레일 설정 전달"
```

---

## Task 6: API 핸들러 — 신규 필드 추가

**Files:**
- Modify: `backend/internal/api/handlers.go`

- [ ] **Step 1: 요청 구조체에 신규 6개 필드 추가**

`handlers.go` 내 `updateTradingSettings` 핸들러의 익명 요청 구조체에서 `TrailingStopPct` 필드 바로 다음에 추가:

```go
		// 트레일링 모드 + 틱 트레일
		TrailingMode           *string  `json:"trailing_mode"`
		TickTier0StopLossTicks *int     `json:"tick_tier0_stop_loss_ticks"`
		TickTier1TriggerPct    *float64 `json:"tick_tier1_trigger_pct"`
		TickTier1TrailTicks    *int     `json:"tick_tier1_trail_ticks"`
		TickTier2TriggerPct    *float64 `json:"tick_tier2_trigger_pct"`
		TickTier2TrailTicks    *int     `json:"tick_tier2_trail_ticks"`
```

- [ ] **Step 2: validation + Firestore 저장 로직 추가**

`handlers.go` 내 `req.TrailingStopPct` 블록(약 line 1259) 바로 다음에 추가:

```go
	if req.TrailingMode != nil {
		if *req.TrailingMode != "pct" && *req.TrailingMode != "tick" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "trailing_mode는 'pct' 또는 'tick' 이어야 합니다"})
			return
		}
		if !save("trailing_mode", *req.TrailingMode) {
			return
		}
	}
	if req.TickTier0StopLossTicks != nil {
		if *req.TickTier0StopLossTicks < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tick_tier0_stop_loss_ticks는 0 이상이어야 합니다"})
			return
		}
		if !save("tick_tier0_stop_loss_ticks", strconv.Itoa(*req.TickTier0StopLossTicks)) {
			return
		}
	}
	if req.TickTier1TriggerPct != nil {
		if *req.TickTier1TriggerPct < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tick_tier1_trigger_pct는 0 이상이어야 합니다"})
			return
		}
		if !save("tick_tier1_trigger_pct", strconv.FormatFloat(*req.TickTier1TriggerPct, 'f', -1, 64)) {
			return
		}
	}
	if req.TickTier1TrailTicks != nil {
		if *req.TickTier1TrailTicks < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tick_tier1_trail_ticks는 0 이상이어야 합니다"})
			return
		}
		if !save("tick_tier1_trail_ticks", strconv.Itoa(*req.TickTier1TrailTicks)) {
			return
		}
	}
	if req.TickTier2TriggerPct != nil {
		if *req.TickTier2TriggerPct < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tick_tier2_trigger_pct는 0 이상이어야 합니다"})
			return
		}
		if !save("tick_tier2_trigger_pct", strconv.FormatFloat(*req.TickTier2TriggerPct, 'f', -1, 64)) {
			return
		}
	}
	if req.TickTier2TrailTicks != nil {
		if *req.TickTier2TrailTicks < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tick_tier2_trail_ticks는 0 이상이어야 합니다"})
			return
		}
		if !save("tick_tier2_trail_ticks", strconv.Itoa(*req.TickTier2TrailTicks)) {
			return
		}
	}
```

- [ ] **Step 3: 동일 요청 내 Tier 순서 교차 검증 추가**

`req.TickTier2TrailTicks` 처리 블록 다음, `DailyMaxLossPct` 블록 이전에 추가:

```go
	// Tier1 < Tier2 순서 교차 검증 (같은 요청에서 둘 다 제공된 경우)
	if req.TickTier1TriggerPct != nil && req.TickTier2TriggerPct != nil &&
		*req.TickTier1TriggerPct > 0 && *req.TickTier2TriggerPct > 0 &&
		*req.TickTier2TriggerPct <= *req.TickTier1TriggerPct {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tick_tier2_trigger_pct는 tick_tier1_trigger_pct보다 커야 합니다"})
		return
	}
```

- [ ] **Step 4: 전체 빌드**

```bash
cd backend && go build ./...
```
Expected: 오류 없음

- [ ] **Step 5: 커밋**

```bash
git add backend/internal/api/handlers.go
git commit -m "feat(api): 틱 트레일 설정 6개 필드 핸들러 추가"
```

---

## Task 7: 프론트엔드 Settings.jsx 업데이트

**Files:**
- Modify: `frontend/src/pages/Settings.jsx`
- Modify: `frontend/src/mocks/handlers.js`

- [ ] **Step 1: transformSettings()에 신규 필드 파싱 추가**

`Settings.jsx` 내 `transformSettings()` 함수의 `trailing` 객체를 다음과 같이 확장:

현재:
```js
    trailing: {
      trigger_pct: pf(raw.trailing_trigger_pct, 0),
      stop_pct: pf(raw.trailing_stop_pct, 0),
    },
```

변경 후:
```js
    trailing: {
      mode: raw.trailing_mode || 'pct',
      trigger_pct: pf(raw.trailing_trigger_pct, 0),
      stop_pct: pf(raw.trailing_stop_pct, 0),
      tick_tier0_stop_loss_ticks: pi(raw.tick_tier0_stop_loss_ticks, 3),
      tick_tier1_trigger_pct: pf(raw.tick_tier1_trigger_pct, 0),
      tick_tier1_trail_ticks: pi(raw.tick_tier1_trail_ticks, 5),
      tick_tier2_trigger_pct: pf(raw.tick_tier2_trigger_pct, 0),
      tick_tier2_trail_ticks: pi(raw.tick_tier2_trail_ticks, 2),
    },
```

- [ ] **Step 2: 저장 함수(handleSave)에 신규 필드 직렬화 추가**

`handleSave()` 내 `flat.trailing_stop_pct = ...` 줄 바로 다음에 추가:

```js
      flat.trailing_mode = settings.trailing?.mode || 'pct'
      flat.tick_tier0_stop_loss_ticks = String(settings.trailing?.tick_tier0_stop_loss_ticks ?? 3)
      flat.tick_tier1_trigger_pct = String(settings.trailing?.tick_tier1_trigger_pct ?? 0)
      flat.tick_tier1_trail_ticks = String(settings.trailing?.tick_tier1_trail_ticks ?? 5)
      flat.tick_tier2_trigger_pct = String(settings.trailing?.tick_tier2_trigger_pct ?? 0)
      flat.tick_tier2_trail_ticks = String(settings.trailing?.tick_tier2_trail_ticks ?? 2)
```

- [ ] **Step 3: 트레일링 스탑 UI 섹션 교체**

`Settings.jsx`에서 현재 `{/* ── 트레일링 스탑 ── */}` 섹션(약 line 670~692)을 다음으로 교체:

```jsx
          {/* ── 트레일링 스탑 ── */}
          <div style={{ marginBottom: 24 }}>
            <div className="form-label" style={{ fontSize: 13, fontWeight: 700, marginBottom: 12, color: 'var(--accent)' }}>
              트레일링 스탑
            </div>

            {/* 모드 선택 */}
            <div style={{ display: 'flex', gap: 20, marginBottom: 14 }}>
              {[['pct', '% 방식 (기존)'], ['tick', '틱 방식 (정밀)']].map(([val, label]) => (
                <label key={val} style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer', fontSize: 13 }}>
                  <input
                    type="radio"
                    name="trailing_mode"
                    value={val}
                    checked={(settings.trailing?.mode || 'pct') === val}
                    onChange={() => set('trailing.mode', val)}
                  />
                  {label}
                </label>
              ))}
            </div>

            {/* % 방식 */}
            {(settings.trailing?.mode || 'pct') === 'pct' && (
              <>
                <div className="muted" style={{ fontSize: 12, marginBottom: 10 }}>
                  목표가 전이라도 수익이 발생했다가 최고점 대비 일정 % 하락 시 즉시 익절. 0 = 비활성
                </div>
                <div className="form-row">
                  <div className="form-group">
                    <label className="form-label">활성화 기준 수익률 (%)</label>
                    <input className="form-input" type="number" step="0.1" min="0"
                      value={settings.trailing?.trigger_pct ?? 0}
                      onChange={e => set('trailing.trigger_pct', +e.target.value)} />
                  </div>
                  <div className="form-group">
                    <label className="form-label">최고점 대비 하락 허용폭 (%)</label>
                    <input className="form-input" type="number" step="0.1" min="0"
                      value={settings.trailing?.stop_pct ?? 0}
                      onChange={e => set('trailing.stop_pct', +e.target.value)} />
                  </div>
                </div>
              </>
            )}

            {/* 틱 방식 */}
            {settings.trailing?.mode === 'tick' && (
              <>
                <div className="muted" style={{ fontSize: 12, marginBottom: 14 }}>
                  매수1호가 기준 + KRX 호가 단위(틱)로 손절/트레일 거리 설정. 0 = 비활성
                </div>

                {/* Tier 0 */}
                <div style={{ marginBottom: 12, padding: '10px 12px', background: 'var(--bg-card, #1a1a2e)', borderRadius: 6, border: '1px solid var(--border)' }}>
                  <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 8, opacity: 0.7 }}>Tier 0 — 진입 손절</div>
                  <div className="form-row">
                    <div className="form-group">
                      <label className="form-label">진입가 대비 손절 (틱 수)</label>
                      <input className="form-input" type="number" step="1" min="0"
                        value={settings.trailing?.tick_tier0_stop_loss_ticks ?? 3}
                        onChange={e => set('trailing.tick_tier0_stop_loss_ticks', +e.target.value)} />
                    </div>
                  </div>
                </div>

                {/* Tier 1 */}
                <div style={{ marginBottom: 12, padding: '10px 12px', background: 'var(--bg-card, #1a1a2e)', borderRadius: 6, border: '1px solid var(--border)' }}>
                  <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 8, opacity: 0.7 }}>Tier 1 — 브레이크이븐 트레일</div>
                  <div className="form-row">
                    <div className="form-group">
                      <label className="form-label">발동 수익률 (%)</label>
                      <input className="form-input" type="number" step="0.1" min="0"
                        value={settings.trailing?.tick_tier1_trigger_pct ?? 0}
                        onChange={e => set('trailing.tick_tier1_trigger_pct', +e.target.value)} />
                    </div>
                    <div className="form-group">
                      <label className="form-label">트레일 거리 (틱 수)</label>
                      <input className="form-input" type="number" step="1" min="0"
                        value={settings.trailing?.tick_tier1_trail_ticks ?? 5}
                        onChange={e => set('trailing.tick_tier1_trail_ticks', +e.target.value)} />
                    </div>
                  </div>
                </div>

                {/* Tier 2 */}
                <div style={{ marginBottom: 12, padding: '10px 12px', background: 'var(--bg-card, #1a1a2e)', borderRadius: 6, border: '1px solid var(--border)' }}>
                  <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 8, opacity: 0.7 }}>Tier 2 — 급등 타이트 트레일</div>
                  <div className="form-row">
                    <div className="form-group">
                      <label className="form-label">발동 수익률 (%)</label>
                      <input className="form-input" type="number" step="0.1" min="0"
                        value={settings.trailing?.tick_tier2_trigger_pct ?? 0}
                        onChange={e => set('trailing.tick_tier2_trigger_pct', +e.target.value)} />
                    </div>
                    <div className="form-group">
                      <label className="form-label">트레일 거리 (틱 수)</label>
                      <input className="form-input" type="number" step="1" min="0"
                        value={settings.trailing?.tick_tier2_trail_ticks ?? 2}
                        onChange={e => set('trailing.tick_tier2_trail_ticks', +e.target.value)} />
                    </div>
                  </div>
                </div>

                {/* 틱 사이즈 안내 */}
                <div className="muted" style={{ fontSize: 11, lineHeight: 1.6 }}>
                  ℹ️ 가격대별 틱 사이즈: ~999원=1원 / 1,000~4,999원=5원 / 5,000~9,999원=10원<br />
                  10,000~49,999원=50원 / 50,000~99,999원=100원 / 100,000~499,999원=500원 / 500,000원~=1,000원
                </div>
              </>
            )}
          </div>
```

- [ ] **Step 4: MSW 목 데이터에 신규 필드 추가**

`frontend/src/mocks/handlers.js` 내 `trailing_trigger_pct` 항목 바로 다음에 추가:

```js
  trailing_mode: 'pct',
  tick_tier0_stop_loss_ticks: 3,
  tick_tier1_trigger_pct: 0,
  tick_tier1_trail_ticks: 5,
  tick_tier2_trigger_pct: 0,
  tick_tier2_trail_ticks: 2,
```

- [ ] **Step 5: 프론트엔드 빌드 확인**

```bash
cd frontend && npm run build 2>&1 | tail -10
```
Expected: `✓ built in` 또는 유사한 성공 메시지. 오류 없음.

- [ ] **Step 6: 커밋**

```bash
git add frontend/src/pages/Settings.jsx frontend/src/mocks/handlers.js
git commit -m "feat(ui): 트레일링 스탑 모드 토글 및 틱 트레일 설정 UI 추가"
```

---

## Task 8: 최종 통합 빌드 검증 및 정리

- [ ] **Step 1: 백엔드 전체 빌드 + 테스트**

```bash
cd backend && go fmt ./... && go build ./... && go test ./...
```
Expected: 
```
ok  	github.com/micro-trading-for-agent/backend/internal/monitor
ok  	github.com/micro-trading-for-agent/backend/internal/scorer
ok  	github.com/micro-trading-for-agent/backend/internal/ops
```

- [ ] **Step 2: 프론트엔드 최종 빌드**

```bash
cd frontend && npm run build
```
Expected: 빌드 성공

- [ ] **Step 3: 최종 커밋**

```bash
git add -A
git status  # 변경된 파일 없는지 확인 (모두 이미 커밋됨이어야 함)
```

---

## 구현 체크리스트 (스펙 커버리지)

| 스펙 항목 | 구현 Task |
|-----------|-----------|
| CalcTickSize KRX 호가 단위 | Task 1 |
| PriceEvent.Bid1Price (fields[11]) | Task 2 |
| TradingSettings 6개 신규 필드 | Task 3 |
| TickTrailState 타입 + MonitoredEntry | Task 4 |
| evaluateTickTrail 3-tier 상태머신 | Task 4 |
| HandlePrice bid1 파라미터 | Task 4 |
| StartPriceConsumer 업데이트 | Task 4 |
| Engine MonitoredEntry 생성 | Task 5 |
| API 핸들러 validation + 저장 | Task 6 |
| Firestore 기본값 | Task 3 |
| 프론트엔드 모드 라디오 토글 | Task 7 |
| 프론트엔드 조건부 입력폼 | Task 7 |
| 기존 % 방식 완전 유지 | Task 4 (분기 처리) |
| 하위 호환 (기본값 pct) | Task 3, Task 7 |
| bid1==0 안전 처리 | Task 4 |
