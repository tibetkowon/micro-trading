# 틱 기반 초정밀 트레일링 스탑 설계 명세

**날짜:** 2026-05-13  
**상태:** 승인됨  
**관련 기능:** 매도 자동화 고도화 — 생존력 강화

---

## 1. 목표 및 배경

### 문제

현재 트레일링 스탑은 퍼센트(%) 기반 거리로 구현되어 있다. 이 방식은 한국 주식시장의 **가격대별 호가 단위(틱 사이즈)** 를 무시하며, **체결가**를 기준으로 평가하기 때문에 호가창 움직임보다 반응이 느리다.

| 한계 | 설명 |
|------|------|
| 틱 단위 불일치 | 50,000원 종목의 1% = 500원 = 5틱, 하지만 5,000원 종목의 1% = 50원 = 5틱. 물리적 호가 단위를 무시함 |
| 체결가 기준 | 매수1호가(Bid1)보다 체결가가 하락 감지가 늦어 슬리피지 발생 |
| 단층 구조 | 수익 구간 무관하게 동일한 트레일 거리 적용 — 급등 후 윗꼬리 손실 미방어 |

### 해결 방향

- **기준 가격**: 체결가 → **매수1호가(Bid1, BIDP1)**  
- **트레일 거리 단위**: 퍼센트 → **KRX 호가 단위(틱 수)**  
- **트레일 구조**: 단층 → **3단계 다층(Multi-tier) 상태머신**  
- **기존 % 방식**: 완전 대체가 아닌 **공존(TrailingMode 플래그)** — 향후 A/B 테스트 및 Grid Search 대비

---

## 2. 아키텍처 개요

### 데이터 흐름

```
KIS WebSocket (H0STCNT0)
  └─ fields[2]  = STCK_PRPR  (체결가)
  └─ fields[11] = BIDP1      (매수1호가) ← 현재 파싱 안 됨, 신규 추가
        ↓
  PriceEvent{ StockCode, Price, Bid1Price, Qty, Timestamp }
        ↓ PriceCh
  monitor.HandlePrice(stockCode, price, bid1, isTest)
        ↓
  [TrailingMode == "pct"]  → 기존 % 로직 (변경 없음)
  [TrailingMode == "tick"] → 틱 트레일 상태머신 (신규)
```

### 구현 방식: Approach A — PriceEvent 구조체 확장

단일 채널(`PriceCh`)로 체결가와 Bid1를 함께 전달. 채널 분리나 별도 구독 없이 `H0STCNT0`에 이미 포함된 `fields[11]` 을 파싱해 활용.

---

## 3. KRX 호가 단위 (틱 사이즈)

`backend/internal/monitor/ticksize.go` (신규 파일)

```go
func CalcTickSize(price float64) float64 {
    switch {
    case price < 1_000:    return 1
    case price < 5_000:    return 5
    case price < 10_000:   return 10
    case price < 50_000:   return 50
    case price < 100_000:  return 100
    case price < 500_000:  return 500
    default:               return 1_000
    }
}
```

| 가격대 | 틱 사이즈 |
|--------|----------|
| ~ 999원 | 1원 |
| 1,000 ~ 4,999원 | 5원 |
| 5,000 ~ 9,999원 | 10원 |
| 10,000 ~ 49,999원 | 50원 |
| 50,000 ~ 99,999원 | 100원 |
| 100,000 ~ 499,999원 | 500원 |
| 500,000원 ~ | 1,000원 |

---

## 4. 데이터 구조 변경

### 4-A. `PriceEvent` (`backend/internal/kis/websocket.go`)

```go
// Before
type PriceEvent struct {
    StockCode string
    Price     float64
    Qty       int
    Timestamp time.Time
}

// After
type PriceEvent struct {
    StockCode string
    Price     float64
    Bid1Price float64   // BIDP1 — fields[11] of H0STCNT0
    Qty       int
    Timestamp time.Time
}
```

`parsePriceData()` 변경:
```go
// fields[11] 파싱 추가
var bid1 float64
if len(fields) > 11 {
    fmt.Sscanf(fields[11], "%f", &bid1)
}

c.PriceCh <- PriceEvent{
    StockCode: stockCode,
    Price:     price,
    Bid1Price: bid1,   // NEW
    Qty:       qty,
    Timestamp: time.Now(),
}
```

### 4-B. `TradingSettings` (`backend/internal/database/db.go`)

신규 필드 6개 추가 (기존 `TrailingTriggerPct`, `TrailingStopPct` 유지):

```go
// 트레일링 모드 선택
TrailingMode string  // "pct" (기본, 하위 호환) | "tick" (신규)

// 틱 트레일 설정 — TrailingMode == "tick" 일 때만 적용
TickTier0StopLossTicks int     // X: 진입가 대비 -X틱 (단순 손절)
TickTier1TriggerPct    float64 // A: +A% 수익 도달 시 Tier1 활성화
TickTier1TrailTicks    int     // Y: 매수1호가 고점 대비 -Y틱 (브레이크이븐)
TickTier2TriggerPct    float64 // B: +B% 수익 도달 시 Tier2 활성화
TickTier2TrailTicks    int     // Z: 매수1호가 고점 대비 -Z틱 (타이트 익절)
```

Firestore 기본값 (하위 호환 보장):
```
trailing_mode:              "pct"
tick_tier0_stop_loss_ticks: "3"
tick_tier1_trigger_pct:     "0"
tick_tier1_trail_ticks:     "5"
tick_tier2_trigger_pct:     "0"
tick_tier2_trail_ticks:     "2"
```

### 4-C. `TickTrailState` + `MonitoredEntry` (`backend/internal/monitor/monitor.go`)

```go
// TickTrailState holds runtime state for tick-based trailing stop.
type TickTrailState struct {
    // 설정값 (TradingSettings 에서 Register() 시점에 복사)
    Tier0StopLossTicks int
    Tier1TriggerPct    float64
    Tier1TrailTicks    int
    Tier2TriggerPct    float64
    Tier2TrailTicks    int
    // 런타임 상태
    CurrentTier   int     // 0=진입손절대기, 1=브레이크이븐, 2=타이트
    PeakBid1Price float64 // Tier1/2 활성 후 매수1호가 최고점
}

// MonitoredEntry — 기존 필드에 추가
type MonitoredEntry struct {
    // ... 기존 필드 전부 유지 ...
    TrailingMode string       // "pct" | "tick" — 설정에서 복사
    TickTrail    TickTrailState
}
```

---

## 5. 다층 트레일 상태머신 로직

### 상태 전이 다이어그램

```
진입 직후
   │
   ▼ [Tier 0 — 진입 손절 대기]
   │  bid1 ≤ filledPrice − X × tickSize(filledPrice)
   │  → SELL "틱트레일-Tier0-진입손절"
   │
   │  tradePrice ≥ filledPrice × (1 + B%)  ──────────────┐
   │  tradePrice ≥ filledPrice × (1 + A%)  ───────┐       │
   ▼                                               ▼       ▼
[Tier 1 — 브레이크이븐 트레일]           [Tier 2 — 타이트 트레일]
   peak = bid1 (활성화 시점)              peak 유지 (Tier1 peak 이어받음)
   bid1 > peak → peak 갱신               bid1 > peak → peak 갱신
   bid1 ≤ peak − Y × tickSize(peak)      bid1 ≤ peak − Z × tickSize(peak)
   → SELL "틱트레일-Tier1-브레이크이븐"  → SELL "틱트레일-Tier2-급등익절"
```

**주의 1:** Tier 2 조건을 먼저 평가 — A/B% 동시 충족 시 Tier 1을 건너뛰고 즉시 Tier 2 진입.  
**주의 2:** `tickSize`는 **peak 가격 기준**으로 재계산 — 가격이 오르면 틱 사이즈도 자동 조정됨.  
**주의 3:** Tier 2 직접 진입(Tier 0 → 2) 시 `PeakBid1Price`는 해당 틱의 `bid1`으로 초기화됨 (`PeakBid1Price`의 초기값은 0이므로 `bid1 > 0` 조건에서 항상 설정).

### `HandlePrice()` 시그니처 변경

```go
// Before
func (m *Monitor) HandlePrice(stockCode string, price float64, isTest bool)

// After
func (m *Monitor) HandlePrice(stockCode string, price float64, bid1 float64, isTest bool)
```

`bid1 == 0` 인 경우(파싱 실패, 장외 등): 틱 트레일 평가를 건너뜀.

### 내부 평가 함수 (신규)

```go
// evaluateTickTrail is called from HandlePrice when TrailingMode == "tick".
// Returns (shouldSell bool, reason string).
func evaluateTickTrail(pos *MonitoredEntry, tradePrice, bid1 float64) (bool, string) {
    ts := &pos.TickTrail

    // Tier 2 승격 먼저 체크
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
        if ts.Tier0StopLossTicks > 0 && bid1 > 0 {
            stop := pos.FilledPrice - float64(ts.Tier0StopLossTicks)*CalcTickSize(pos.FilledPrice)
            if bid1 <= stop {
                return true, "틱트레일-Tier0-진입손절"
            }
        }
    case 1:
        if bid1 > ts.PeakBid1Price { ts.PeakBid1Price = bid1 }
        stop := ts.PeakBid1Price - float64(ts.Tier1TrailTicks)*CalcTickSize(ts.PeakBid1Price)
        if bid1 <= stop {
            return true, "틱트레일-Tier1-브레이크이븐"
        }
    case 2:
        if bid1 > ts.PeakBid1Price { ts.PeakBid1Price = bid1 }
        stop := ts.PeakBid1Price - float64(ts.Tier2TrailTicks)*CalcTickSize(ts.PeakBid1Price)
        if bid1 <= stop {
            return true, "틱트레일-Tier2-급등익절"
        }
    }
    return false, ""
}
```

---

## 6. API 핸들러 변경 (`backend/internal/api/handlers.go`)

`updateTradingSettings` 요청 구조체에 신규 필드 추가:

```go
TrailingMode           *string  `json:"trailing_mode"`
TickTier0StopLossTicks *int     `json:"tick_tier0_stop_loss_ticks"`
TickTier1TriggerPct    *float64 `json:"tick_tier1_trigger_pct"`
TickTier1TrailTicks    *int     `json:"tick_tier1_trail_ticks"`
TickTier2TriggerPct    *float64 `json:"tick_tier2_trigger_pct"`
TickTier2TrailTicks    *int     `json:"tick_tier2_trail_ticks"`
```

Validation 규칙:
- `trailing_mode`: `"pct"` 또는 `"tick"` 이외 값은 400 반환
- 틱 수 필드: `< 0` 이면 400 반환
- `Tier2TriggerPct` > `Tier1TriggerPct` 강제 (0이 아닌 경우): 잘못된 순서 방지

---

## 7. 프론트엔드 UI 변경

대상 컴포넌트: `TradingSettings` 트레일링 스탑 섹션

### 상태 관리

```typescript
const [trailingMode, setTrailingMode] = useState<'pct' | 'tick'>('pct')
```

### 레이아웃 (조건부 렌더링)

```
[트레일링 스탑 모드]
  ◉ % 방식  ○ 틱 방식 (정밀)

━━━ % 방식 ━━━ (trailingMode === 'pct' 일 때만 표시)
  발동 기준:   [___] %
  트레일 거리: [___] %

━━━ 틱 방식 ━━━ (trailingMode === 'tick' 일 때만 표시)
  Tier 0  진입 손절          [__3__] 틱
  Tier 1  발동 수익률        [_1.5_] %
          트레일 거리        [__5__] 틱
  Tier 2  발동 수익률        [_4.0_] %
          트레일 거리        [__2__] 틱

  ℹ️ 틱 사이즈: ~999원=1원 / 1,000~4,999원=5원 / 5,000~9,999원=10원
     10,000~49,999원=50원 / 50,000~99,999원=100원 / 100,000~499,999원=500원
```

---

## 8. 변경 파일 목록

| 파일 | 유형 | 변경 내용 |
|------|------|----------|
| `backend/internal/kis/websocket.go` | 수정 | `PriceEvent`에 `Bid1Price` 추가; `parsePriceData()`에서 `fields[11]` 파싱 |
| `backend/internal/database/db.go` | 수정 | `TradingSettings` 6개 필드 추가; `GetTradingSettings()` 파싱; Firestore 기본값 |
| `backend/internal/monitor/monitor.go` | 수정 | `TickTrailState` 타입 정의; `MonitoredEntry`에 `TrailingMode`/`TickTrail` 추가; `HandlePrice()` 시그니처 변경; 틱 트레일 평가 로직; `StartPriceConsumer()` 호출부 업데이트 |
| `backend/internal/monitor/ticksize.go` | **신규** | `CalcTickSize()` — KRX 호가 단위 계산 |
| `backend/internal/trader/engine.go` | 수정 | `HandlePrice()` 호출부 `bid1` 인자 추가; `MonitoredEntry` 생성 시 신규 필드 전달 |
| `backend/internal/api/handlers.go` | 수정 | 신규 6개 요청 필드 추가, validation, Firestore 저장 |
| Frontend `TradingSettings` 컴포넌트 | 수정 | 모드 토글(radio/toggle); 조건부 입력폼 렌더링; API 연동 |

---

## 9. 하위 호환성 및 엣지 케이스

- `trailing_mode` 기본값 `"pct"` → 기존 사용자 설정 영향 없음
- 기존 `TrailingTriggerPct` / `TrailingStopPct` 필드 및 로직 **완전 유지**
- `Bid1Price == 0` 인 틱 이벤트 수신 시 틱 트레일 평가를 건너뜀 → 안전 처리
- **서버 재시작 복구**: `RecoverFromHoldings()` 호출 시 `TickTrailState`는 `CurrentTier = 0`, `PeakBid1Price = 0` 으로 초기화됨. 이는 설계상 허용되는 단순화 — 재시작 후 다음 틱 수신 시 정상 상태 재진입. Tier 1/2 활성화 조건 재평가가 즉시 이루어지므로 복구 지연은 최소 1틱.

---

## 10. 향후 확장 고려 (YAGNI 기준 현재 미구현)

- **Grid Search A/B 테스트**: `trailing_mode` 플래그로 % vs 틱 모드 교차 검증 가능한 구조 이미 확보
- **데이트레이딩 / 스윙 세팅**: % 방식 존치로 다양한 전략 프리셋 지원 가능
- **틱 트레일 부분 청산**: 현재 전량 청산만; 부분 청산 추가 시 `TickTrailState`에 `PartialExitTicks` 필드 추가 구조 가능
