# 매도 제어 6개 기능 구현 계획

## Goal
단타 봇의 손익 보호 및 매도 로직을 고도화하는 6개 기능을 구현한다.

---

## 기능별 현황 분석

| # | 기능 | 백엔드 현황 | UI 현황 |
|---|------|------------|--------|
| 1 | 트레일링 스탑 | ✅ 완전 구현 (`trailing_trigger_pct`, `trailing_stop_pct` Firestore 키, monitor.go 로직) | ❌ Settings UI 없음 |
| 2 | 호가 스프레드 필터 | ❌ 미구현 | ❌ |
| 3 | 연속 손절 제한 | ❌ 미구현 | ❌ |
| 4 | 횡보 탐지 청산 | ✅ 완전 구현 (`stagnation_*` 4개 키, monitor.go 로직) | ❌ Settings UI 없음 |
| 5 | 매도 조건 순위 선정 | ✅ `sell_conditions` JSON 배열 존재, 순서가 곧 우선순위 | ❌ Settings UI 없음 |
| 6 | 상한가 매도 | ❌ 미구현 | ❌ |

---

## Phase 1 — Settings UI (백엔드 변경 없음)

**대상 기능: 트레일링 스탑(1), 횡보탐지(4), 매도조건 순위(5)**

이미 백엔드에 구현된 기능을 Settings.jsx에 노출한다. 새 탭 **`매도관리`**를 추가한다.

### 추가할 Firestore 키 ↔ UI 매핑

**트레일링 스탑**
- `trailing_trigger_pct` (float) — 활성화 기준 수익률 (예: 1.5%). 0=비활성
- `trailing_stop_pct` (float) — 최고점 대비 하락 허용폭 (예: 0.5%). 0=비활성

**횡보 탐지**
- `stagnation_threshold_pct` (float) — 횡보 판단 변동폭 (예: 0.5%). 0=비활성
- `stagnation_duration_min` (int) — 횡보 지속 기준 시간(분)
- `stagnation_partial_exit_enabled` (bool) — 1차 50%→2차 전량 청산 활성화
- `stagnation_bidask_sell_threshold` (float) — 호가비율 이 값 미만 시 즉시 전량 청산

**매도 조건 순위**
- `sell_conditions` (JSON 배열) — 순서가 우선순위. 예: `["rsi_overbought","macd_bearish","stagnation"]`
- 지원 조건값: `rsi_overbought`, `macd_bearish`, `stagnation`
- UI: 체크박스로 활성화 + 버튼(↑↓)으로 순서 변경

### 변경 파일
- `frontend/src/pages/Settings.jsx`
  - `transformSettings()`: 7개 신규 키 파싱 추가
  - `save()` 내 flat 맵: 7개 키 직렬화 추가
  - `매도관리` 탭 UI 섹션 추가

---

## Phase 2 — 연속 손절 제한 (Max Consecutive Losses)

**Firestore 신규 키**
- `max_consecutive_losses` (int, 기본 0=비활성) — 연속 손절 임계값
- `consecutive_loss_reset_on_profit` (bool, 기본 true) — 익절 시 카운터 리셋 여부

**백엔드 변경**

`database/db.go` — `TradingSettings` 필드 추가:
```go
MaxConsecutiveLosses      int
ConsecutiveLossResetOnProfit bool
```
`GetTradingSettings()` 파싱 추가.

`trader/engine.go`:
- `consecutiveLosses int` 필드 추가 (Engine 구조체)
- `haltReason` 체크와 별개로 `consecutiveLossHalt bool` 플래그 추가
- `runScanCycle()` 초입에 임계 초과 시 skip (로그 남김)
- `soldCh` 수신 후 해당 종목의 최신 `trade_report`를 DB 조회 → `profit_amount < 0` 이면 카운터 증가, `>= 0` 이면 (`ConsecutiveLossResetOnProfit=true` 시) 리셋
- 장 시작 시(`engineRunning` 플래그 세팅 시) 카운터 0으로 초기화

`cmd/server/main.go`:
- 엔진 시작 시 settings에서 `MaxConsecutiveLosses` 를 engine에 전달하는 setter 호출

**Settings UI**: `매도관리` 탭에 연속 손절 섹션 추가

**변경 파일**
- `backend/internal/database/db.go`
- `backend/internal/trader/engine.go`
- `backend/cmd/server/main.go`
- `frontend/src/pages/Settings.jsx`

---

## Phase 3 — 호가 스프레드 필터 (Max Bid-Ask Spread)

매수 후보 종목 스캔 단계에서 1호가 스프레드가 과도한 종목을 제거한다.

**Firestore 신규 키**
- `max_bidask_spread_pct` (float, 기본 0=비활성) — 최대 허용 스프레드 (%)

**백엔드 변경**

`kis/client.go` — `OrderBookSnapshot` 구조체에 필드 추가:
```go
BestAskPrice  float64 // 매도 1호가
BestBidPrice  float64 // 매수 1호가
SpreadPct     float64 // (매도1호가 - 매수1호가) / 매도1호가 × 100
```
`GetOrderBookSnapshot()` 내 `askp1`/`bidp1` 파싱 후 계산.

`ops/stock_info.go` — `StockInfo`에 `BidAskSpread float64` 추가.
`GetStockInfo()` 내 `GetOrderBookSnapshot()` 호출 시 `SpreadPct` 반영.

`database/db.go` — `TradingSettings`에 `MaxBidAskSpreadPct float64` 추가.

`trader/engine.go` — `fetchCandidates()` 내 필터:
```go
if settings.MaxBidAskSpreadPct > 0 && info.BidAskSpread > settings.MaxBidAskSpreadPct {
    // skip
}
```

**Settings UI**: `매도관리` 탭 (또는 `하드필터` 탭)에 추가

**변경 파일**
- `backend/internal/kis/client.go`
- `backend/internal/ops/stock_info.go`
- `backend/internal/database/db.go`
- `backend/internal/trader/engine.go`
- `frontend/src/pages/Settings.jsx`

---

## Phase 4 — 상한가 도달 시 매도

**Firestore 신규 키**
- `sell_on_upper_limit` (bool, 기본 false) — 상한가 도달 시 자동 매도

**백엔드 변경**

`kis/client.go` — `StockPriceResponse`에 `UpperLimitPrice string \`json:"stck_mxpr"\`` 추가.

`monitor/monitor.go` — `MonitoredEntry` 구조체에 추가:
```go
UpperLimitPrice  float64 // 당일 상한가. 0=미사용
SellOnUpperLimit bool
```
`HandlePrice()` 내 TP/SL 체크 직후에 추가:
```go
if pos.SellOnUpperLimit && pos.UpperLimitPrice > 0 && price >= pos.UpperLimitPrice {
    go m.executeSell(stockCode, pos, "상한가 도달")
    return
}
```

`trader/engine.go` — `placeOrder()` 완료 후 `MonitoredEntry` 생성 시:
- `GetStockPrice()` 응답의 `UpperLimitPrice` 파싱 후 `entry.UpperLimitPrice` 설정
- `entry.SellOnUpperLimit = settings.SellOnUpperLimit`

`database/db.go` — `TradingSettings`에 `SellOnUpperLimit bool` 추가.

**Settings UI**: `매도관리` 탭에 토글 추가

**변경 파일**
- `backend/internal/kis/client.go`
- `backend/internal/monitor/monitor.go`
- `backend/internal/trader/engine.go`
- `backend/internal/database/db.go`
- `frontend/src/pages/Settings.jsx`

---

## Requirements

- 모든 신규 설정값은 Firestore `settings/config` 문서에 `INSERT OR IGNORE` 기본값으로 초기화 (`db.go`의 `defaultSettings` 맵에 추가)
- 각 기능은 `0` 또는 `false`를 기본값으로 하여 **기본 비활성** — 기존 동작에 영향 없음
- 상한가 가격(`stck_mxpr`)은 Register 시 1회만 조회. 이후 장중 변동 없음 (KIS 상한가는 일일 고정)

---

## Verification

| Phase | 검증 방법 |
|-------|---------|
| 1 | Settings UI에서 값 저장 후 Firestore 콘솔에서 키 확인 |
| 2 | 테스트 종목 3회 손절 시 엔진 로그에 `consecutive loss halt` 확인 |
| 3 | 스프레드 높은 종목(코스닥 저유동성) 스캔 시 후보에서 제외 확인 |
| 4 | `sell_on_upper_limit=true` 설정 후 상한가 근접 종목 등록, 로그 확인 |

---

## 구현 순서

Phase 1 → Phase 3 → Phase 4 → Phase 2

- Phase 1은 백엔드 변경 없어 빠르게 처리 가능
- Phase 3(스프레드)은 KIS API 변경만 있어 난이도 낮음
- Phase 4(상한가)는 MonitoredEntry 변경이 있어 중간 난이도
- Phase 2(연속 손절)는 Engine 상태 관리가 가장 복잡
