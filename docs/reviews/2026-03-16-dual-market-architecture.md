# 국내·미장 동시 자동 트레이딩 아키텍처 설계

> 작성일: 2026-03-16
> 목적: 국내(KST 09:15~15:15)와 미국(KST 22:30~05:00 서머타임 / 23:30~06:00 비서머타임) 주식을 단일 프로세스에서 동시 운영하기 위한 설계 문서

---

## 1. 시간 분리 구조

두 시장의 운영 시간이 겹치지 않아 단일 프로세스로 순차 운영 가능:

```
KST 기준 하루:
 09:15~15:15  → 국내장 (KR engine 활성)
 15:15~22:30  → 대기
 22:30~05:00  → 미국장 (US engine 활성, 서머타임)
 23:30~06:00  → 미국장 (US engine 활성, 비서머타임)
```

---

## 2. DB 스키마 변경

### orders 테이블

```sql
ALTER TABLE orders ADD COLUMN market TEXT NOT NULL DEFAULT 'KR';
-- 'KR' = 국내주식, 'US' = 미국주식
```

### monitored_positions 테이블

```sql
ALTER TABLE monitored_positions ADD COLUMN market TEXT NOT NULL DEFAULT 'KR';
```

### settings 신규 기본값 (INSERT OR IGNORE)

| 키 | 기본값 | 설명 |
|----|--------|------|
| `us_trading_enabled` | `"false"` | 미장 자동매매 ON/OFF |
| `us_trading_start_time` | `"22:30"` | 미장 시작시간 (HH:MM, KST) |
| `us_trading_end_time` | `"05:00"` | 미장 종료시간 (HH:MM, KST) |
| `us_dst_enabled` | `"true"` | 서머타임 여부 (`true`=22:30, `false`=23:30) |
| `us_ranking_types` | `'["volume"]'` | 순위 조회 유형 JSON |
| `us_ranking_exchange` | `"NAS"` | 거래소코드 (NAS/NYS/AMS) |
| `us_ranking_price_min` | `"10"` | 최소 주가 (USD) |
| `us_ranking_price_max` | `"500"` | 최대 주가 (USD) |
| `us_ranking_vol_rang` | `"0"` | 거래량 조건 (0=전체, 1=100주↑, ...) |
| `us_ranking_top_n` | `"20"` | 상위 N개 종목 |

---

## 3. Engine 분리

```go
// cmd/server/main.go
krEngine := trader.NewEngine(db, kisClient, claudeClient, "KR")
usEngine := trader.NewEngine(db, kisClient, claudeClient, "US")
```

### trader.Engine 구조체 변경

```go
type Engine struct {
    market string // "KR" 또는 "US"
    // ... 기존 필드
}
```

- `market` 필드에 따라 API 호출 분기:
  - `getRankings()` → KR: 국내 순위 API / US: `GetOverseasVolumeRank()` 등
  - `placeOrder()` → KR: `PlaceOrder()` / US: `PlaceOverseasOrder()`
  - `getBalance()` → KR: `GetBalance()` / US: `GetOverseasBalance()`
  - `getPrice()` → KR: `GetStockPrice()` / US: `GetOverseasPrice()`
  - `cancelOrder()` → KR: `CancelOrder()` / US: `CancelOverseasOrder()`

### 스케줄러 동작 (cmd/server/main.go)

```go
// 매 틱(1분) DB에서 시간 설정 조회
// 국내장 스케줄
if currentHHMM == krStart {
    krEngine.Start()
}
if currentHHMM == krEnd {
    krEngine.Stop()
    monitor.LiquidateAll() // KR 포지션만
}

// 미장 스케줄 (us_trading_enabled=true 시)
if currentHHMM == usStart {
    usEngine.Start()
}
if currentHHMM == usEnd {
    usEngine.Stop()
    monitor.LiquidateAll() // US 포지션만
}
```

---

## 4. KIS Client 신규 함수 (kis/client.go)

```go
// 주문
func (c *Client) PlaceOverseasOrder(ctx context.Context, exchCode, code string, qty int, price float64, side string) (OrderResult, error)
func (c *Client) CancelOverseasOrder(ctx context.Context, ordNo, exchCode string) error

// 조회
func (c *Client) GetOverseasBalance(ctx context.Context, exchCode, crcy string) (OverseasBalance, error)
func (c *Client) GetOverseasPrice(ctx context.Context, excd, symb string) (OverseasPrice, error)
func (c *Client) GetOverseasVolumeRank(ctx context.Context, excd, prc1, prc2, volRang string) ([]OverseasRankItem, error)
func (c *Client) GetOverseasPendingOrders(ctx context.Context, exchCode string) ([]OverseasOrder, error)
func (c *Client) GetOverseasOrderHistory(ctx context.Context, exchCode, startDt, endDt string) ([]OverseasOrder, error)
```

---

## 5. WebSocket 신규 함수 (kis/websocket.go)

```go
// HDFSCNT0 구독: tr_key = "D" + excd(3자리) + symb
// 예: excd="NAS", symb="AAPL" → tr_key="DNASAAPL"
func (ws *Client) SubscribeOverseasPrice(excd, symb string) error
func (ws *Client) UnsubscribeOverseasPrice(excd, symb string) error

// H0GSCNI0 체결통보: tr_key = HTS ID (기존과 동일)
// 기존 SubscribeExecNotice() 재사용, ORDEN_COND 필드로 국내/해외 판별
```

---

## 6. 공통으로 재사용되는 설정

아래 설정은 KR/US 엔진이 공통으로 사용:

| 설정 키 | 설명 |
|--------|------|
| `take_profit_pct`, `stop_loss_pct` | 두 시장 동일 비율 적용 |
| `max_positions`, `order_amount_pct` | 공통 적용 (KR+US 포지션 합산) |
| `sell_conditions` | 공통 적용 |
| `indicator_check_interval_min` | 공통 적용 |
| `indicator_rsi_sell_threshold` | 공통 적용 |
| `indicator_macd_bearish_sell` | 공통 적용 |
| `stagnation_threshold_pct`, `stagnation_duration_min` | 공통 적용 |
| `claude_model` | 공통 사용 |

---

## 7. 미장 전용 순위 설정 (국내와 완전 분리)

국내 순위 API와 미장 순위 API는 파라미터 구조가 달라 별도 설정 필요:

| 설정 키 | 설명 | 기본값 |
|--------|------|--------|
| `us_ranking_types` | JSON 배열 (`volume`/`volume_surge`/`strength`/`updown_rate`/`new_highlow`) | `["volume"]` |
| `us_ranking_exchange` | EXCD 코드 (`NAS`/`NYS`/`AMS`) | `"NAS"` |
| `us_ranking_price_min` | 최소 주가 (USD) | `"10"` |
| `us_ranking_price_max` | 최대 주가 (USD) | `"500"` |
| `us_ranking_vol_rang` | 거래량 조건 | `"0"` |
| `us_ranking_top_n` | 상위 N개 | `"20"` |

> **AND/OR 조건**: 미장 순위 API는 각 타입이 별개 단일 호출 → OR(합산) 방식 고정.
> 국내의 교집합(AND) 로직은 미장에 적용하지 않음.

---

## 8. Settings.jsx UI 확장

기존 국내 섹션은 그대로 유지하고, 미장 섹션을 신규 추가:

```
기존 국내 섹션 (그대로 유지):
  ├ 거래 제어 (ON/OFF, 거래시간)
  ├ 종목 선정 (순위유형, 가격범위, AND/OR)
  ├ 매수 설정
  ├ 매도 설정
  └ AI 설정

신규 미장 섹션 추가:
  ├ 미장 자동매매 ON/OFF 토글 (us_trading_enabled)
  ├ 서머타임 ON/OFF (us_dst_enabled)
  ├ 미장 거래 시간 (us_trading_start_time ~ us_trading_end_time)
  ├ 거래소 선택 드롭다운 (NAS/NYS/AMS)
  ├ 순위 조회 유형 체크박스
  └ 가격 범위 (USD, us_ranking_price_min ~ max)
```

---

## 9. 구현 순서 (다음 플랜)

1. `database/db.go` — DB 마이그레이션 (market 컬럼, US 기본 설정값)
2. `models/models.go` — OverseasOrder, OverseasBalance, OverseasPrice, OverseasRankItem 구조체
3. `kis/client.go` — 해외주식 API 함수 6개 구현
4. `kis/websocket.go` — SubscribeOverseasPrice/Unsubscribe 구현; H0GSCNI0 ORDEN_COND 파싱 추가
5. `trader/engine.go` — market 필드 추가, API 분기 로직 구현
6. `cmd/server/main.go` — usEngine 생성, 스케줄러 미장 시간대 추가
7. `api/handlers.go` — US 설정 GET/PATCH 지원
8. `frontend/src/pages/Settings.jsx` — 미장 섹션 UI 추가

---

## 10. 거래소 코드 대응 요약

| EXCD (시세/WS) | OVRS_EXCG_CD (주문) | 거래소 |
|--------------|---------------------|--------|
| NAS | NASD | NASDAQ |
| NYS | NYSE | NYSE |
| AMS | AMEX | AMEX |
| HKS | SEHK | 홍콩 |
| SHS | SHAA | 중국 상해 |
| SZS | SZAA | 중국 심천 |
| TSE | TKSE | 일본 도쿄 |
