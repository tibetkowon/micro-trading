# Project Architecture

> Last updated: 2026-05-05 (rev 10)

## Directory Tree

```
micro-trading-for-agent/
├── .github/
│   └── workflows/
│       ├── ci.yml                  # CI: 변경 감지 → Go build/fmt/test + React lint/build + GCS/Firebase 자동 배포
│       ├── deploy-backend.yml      # CD: backend/** 변경 시 Go linux/amd64 빌드 → GCS 업로드
│       └── deploy-frontend.yml     # CD: frontend/** 변경 시 React 빌드 → Firebase Hosting 배포
├── .claude/
│   └── skills/                     # AI 에이전트 행동 지침 파일 (.md)
├── backend/                        # Go backend root (Go 1.26.1)
│   ├── cmd/
│   │   └── server/
│   │       └── main.go             # 진입점; Firestore 초기화, WebSocket·Monitor·Engine, 장운영 스케줄러
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go           # godotenv로 .env 로드; KIS·Anthropic·Firebase·서버 설정
│   │   ├── database/
│   │   │   └── db.go               # Firestore 클라이언트 초기화; GetTradingSettings(), SaveReport(), 컬렉션 CRUD
│   │   ├── models/
│   │   │   └── models.go           # Firestore 컬렉션 1:1 Go 구조체 (Order, TradeReport, MonitoredPosition 등)
│   │   ├── logger/
│   │   │   └── logger.go           # 구조화 JSON 로깅; KISError()는 error_code·timestamp·raw_response 필수
│   │   ├── kis/
│   │   │   ├── client.go           # KIS REST API 클라이언트; 주문/잔고/가격/순위 조회
│   │   │   ├── websocket.go        # KIS WebSocket; H0STCNT0(체결가)/H0STCNI0(체결통보), AES-256-CBC 복호화
│   │   │   ├── chart.go            # KIS 차트 API: GetMinuteChart, GetDailyChart
│   │   │   ├── token.go            # OAuth 토큰 발급·갱신·캐시; 잔여 1시간 미만 자동 재발급
│   │   │   └── ratelimiter.go      # TPS 리미터 (15 req/s, golang.org/x/time/rate)
│   │   ├── monitor/
│   │   │   └── monitor.go          # 포지션 실시간 모니터; HandlePrice() 매도 로직, StartIndicatorChecker()
│   │   ├── ops/
│   │   │   ├── market.go           # IsMarketOpen(): KST 평일·장 시간·KIS 영업일 3중 체크
│   │   │   ├── stock_info.go       # GetStockInfo: 현재가 + MA5/MA20 + RSI14 + MACD(12,26,9)
│   │   │   ├── chart.go            # GetChart: OHLCV 캔들 (1m/5m/1h), 페이지네이션 + 집계
│   │   │   ├── balance.go          # GetAccountBalance: 잔고 조회 + Firestore 스냅샷 저장
│   │   │   ├── order.go            # PlaceOrder/CancelOrder/CheckOrderFeasibility
│   │   │   ├── ranking.go          # 거래량/체결강도/등락률/VI 순위 조회 래퍼
│   │   │   └── history.go          # KIS 체결 내역 동기화; StartOrderSyncScheduler (5분 ticker)
│   │   ├── report/
│   │   │   └── report.go           # GenerateDailyReport: 당일 거래 집계 + Firestore 저장
│   │   ├── scorer/
│   │   │   └── scorer.go           # 매수 후보 스코어링 (체결강도·RSI·MACD·호가잔량·VWAP·거래량 가중치 합산)
│   │   ├── stockmaster/
│   │   │   ├── downloader.go       # KIS MST 파일 일일 다운로드 (08:40)
│   │   │   ├── parser.go           # MST 파일 파싱 (종목코드·종목명·상한가 등)
│   │   │   └── store.go            # 종목 마스터 Firestore 저장 + 인메모리 캐시
│   │   └── trader/
│   │       └── engine.go           # Engine 상태 머신; 종목 스캔·스코어링·주문·체결 대기·모니터 등록 사이클
│   ├── data/                       # 런타임 캐시 파일 (git-ignored)
│   └── go.mod                      # Go 1.26.1 모듈 정의
├── frontend/                       # React 18 + Vite frontend
│   ├── src/
│   │   ├── main.jsx                # React 진입점; BrowserRouter, Firebase 초기화
│   │   ├── App.jsx                 # 루트 컴포넌트; 네비게이션 + 라우트 정의
│   │   ├── index.css               # Tailwind 베이스 스타일
│   │   ├── lib/
│   │   │   └── firebase.js         # Firebase 앱 초기화 및 Firestore 인스턴스 export
│   │   ├── hooks/
│   │   │   └── useApi.js           # 범용 fetch 훅 (loading/error/data/refetch)
│   │   ├── components/
│   │   │   └── shared.jsx          # 공통 UI 컴포넌트
│   │   ├── utils/
│   │   │   └── api.js              # API 엔드포인트 상수 정의
│   │   └── pages/
│   │       ├── Dashboard.jsx       # 대시보드: 잔고·포지션·일별 수익 요약
│   │       ├── Positions.jsx       # 모니터링 중인 보유 종목
│   │       ├── Orders.jsx          # 주문 내역
│   │       ├── TradeReports.jsx    # 거래 리포트 (매수·매도 건별)
│   │       ├── DailyReports.jsx    # 일일 리포트
│   │       ├── Settings.jsx        # 트레이딩 설정 (Firestore 직접 읽기·쓰기)
│   │       └── Logs.jsx            # 서비스·KIS·스캔 로그
│   ├── index.html
│   ├── vite.config.js              # /api 프록시 → :8080
│   └── package.json                # React 18, Firebase SDK, react-router-dom, MSW
├── docs/
│   ├── architecture.md             # 이 파일
│   ├── db_schema.md                # Firestore 컬렉션 스키마
│   ├── changelog.md                # 변경 이력
│   ├── kis-api/                    # KIS API 공식 명세서
│   ├── plans/                      # 기능 구현 계획 문서
│   └── reviews/                    # 한국어 코드 리뷰 문서
├── deploy/                         # 배포 관련 설정
├── firestore.rules                 # Firestore 보안 규칙
├── firebase.json                   # Firebase 프로젝트 설정
├── .env.example                    # 환경변수 템플릿
├── .gitignore                      # .env, firebase-credentials.json, data/ 제외
├── CLAUDE.md                       # AI 에이전트 프로젝트 지침
└── README.md
```

---

## Component Responsibilities

### `backend/internal/config`
환경변수 로드 (`.env` via godotenv). 관리 항목: KIS 자격증명, Anthropic API 키, HTS ID, Firebase 프로젝트 ID, 서비스 계정 경로, 서버 포트.

### `backend/internal/database`
Firebase Firestore 클라이언트 초기화 및 데이터 액세스 레이어.
- `GetTradingSettings(ctx)` — `settings/config` 문서에서 설정 50개 이상을 `TradingSettings` 구조체로 반환
- `SaveReport(ctx, date, content)` — `daily_reports/{date}` upsert
- `GetLatestCompletedTradeByStock(ctx, code)` — 연속손절 카운터용 최근 완료 거래 조회

### `backend/internal/models`
Firestore 컬렉션과 1:1 대응하는 Go 구조체.  
`Order`, `TradeReport`, `MonitoredPosition`, `Balance`, `KISAPILog`, `Token`, `DailyReport`, `StockMaster`

### `backend/internal/kis`
- **`client.go`** — REST API 요청/응답, 토큰 주입, Rate Limiting (15 req/s).  
  `GetOrderBookSnapshot` (호가 스프레드%), `GetStockPrice` (상한가 포함)
- **`websocket.go`** — WebSocket 연결/자동 재연결, `H0STCNT0`(체결가) / `H0STCNI0`(체결통보) 구독, AES-256-CBC 복호화.  
  `PriceCh chan PriceEvent` → monitor 소비 | `ExecCh chan ExecEvent` → engine 체결 확인
- **`token.go`** — OAuth 토큰 발급·갱신·캐시. 잔여 1시간 미만이면 자동 재발급.

### `backend/internal/monitor`
보유 포지션 실시간 모니터링.

**MonitoredEntry 주요 필드:**
```go
FilledPrice        float64       // 진입가
TargetPrice        float64       // 익절가
StopPrice          float64       // 손절가
TrailingTriggerPct float64       // 트레일링 스탑 활성화 기준 수익률
TrailingStopPct    float64       // 트레일링 스탑 고점 대비 허용 하락폭
PeakPrice          float64       // 보유 중 최고가
TrailingActivated  bool          // 트레일링 스탑 활성화 여부
PartialTPDone      bool          // 분할 익절 완료 여부 (1회 한정)
UpperLimitPrice    float64       // 당일 상한가
SellOnUpperLimit   bool          // 상한가 도달 시 즉시 매도
SoldCh             chan<- string // 매도 완료 → 엔진 신호
```

**HandlePrice() 매도 로직 (우선순위 순):**
1. 트레일링 스탑 — 활성화 기준 수익 달성 후 고점 대비 하락 시
2. 분할 익절 — 기준 수익 달성 시 `partialTPRatio` 비율 매도 (1회)
3. 상한가 — `SellOnUpperLimit=true` 이고 `price >= UpperLimitPrice`
4. 목표가 — `price >= TargetPrice`
5. 손절가 — `price <= StopPrice`
6. 횡보 감지 — 변동폭 임계 내 지속 시 분할 청산

`StartIndicatorChecker(ctx, intervalMin, ...)` — RSI 과매수·MACD 데드크로스 주기 평가, 조건 충족 시 매도.

`LiquidateAll(ctx)` — 전량 시장가 청산 (15:15 자동 호출).

### `backend/internal/ops`
KIS API와 Firestore를 연결하는 거래 액션 함수 모음.
- `IsMarketOpen()` — KST 평일·09:00~15:30·KIS 영업일 여부 체크, 당일 캐시
- `GetStockInfo(code)` — 현재가 + MA5/MA20 + RSI14 + MACD(12,26,9)
- `PlaceOrder()`, `CancelOrder()`, `CheckOrderFeasibility()`
- `StartOrderSyncScheduler()` — 5분 간격 KIS 체결 내역 Firestore 동기화

### `backend/internal/scorer`
매수 후보 종목 스코어링. 체결강도·RSI·MACD·호가잔량·VWAP·거래량을 가중치 합산하여 점수 계산. `min_score_threshold` 미달 종목 제외.

### `backend/internal/stockmaster`
매일 08:40 KIS MST 파일 다운로드·파싱 후 Firestore `stock_masters` 컬렉션 업데이트. 종목명·상한가 정보를 인메모리 캐시로 제공.

### `backend/internal/report`
`GenerateDailyReport(ctx)` — 당일 `orders` 컬렉션에서 AGENT 주문 집계 후 일일 리포트 생성 및 `daily_reports/{date}` 저장.

### `backend/internal/trader`
자율 트레이딩 엔진.
- **`engine.go`** — 상태 머신: IDLE → SCANNING → SCORING → ORDERING → WAITING_FILL → MONITORING.  
  `consecutiveLosses` / `consecutiveLossHalt` — 연속손절 카운터·중단 상태 관리.  
  `ResetConsecutiveLosses()` — 09:00 장 시작 시 자동 호출.

### `backend/internal/api`
HTTP 레이어. 입력 검증 → ops/db/engine 호출 → JSON 응답.

---

## 장운영 스케줄러

**타임존:** Asia/Seoul (KST), 평일만 동작

```
08:40 → stockmaster.Download()          종목 마스터 파일 갱신

08:50 → token.IssueToken()              KIS 토큰 발급
      → kisClient.GetApprovalKey()      WebSocket approval key 발급
      → wsClient.StartWithReconnect()   WebSocket 연결 + 자동 재연결
      → wsClient.SubscribeExecNotice()  체결통보(H0STCNI0) 구독

09:00 → db.GetSetting("trading_enabled") 확인
      → ops.IsMarketOpen() 확인
      → eng.ResetConsecutiveLosses()    연속손절 카운터 초기화
      → tradingReady = true

trading_start_time (기본 09:15)
      → krEngine.Start(ctx)             [tradingReady == true 시에만]
      → mon.StartIndicatorChecker(ctx, ...)

trading_end_time (기본 15:15)
      → krEngine stop()
      → indicatorChecker cancel()
      → mon.LiquidateAll(ctx)

15:20 → report.GenerateDailyReport(ctx)

16:00 → wsClient.Disconnect()
```

---

## 트레이딩 엔진 사이클

```
현재 포지션 수 < max_positions?

  YES → [SCANNING]
    1. 오늘 거래 종목 제외 목록 조회 (Firestore orders)
    2. 설정된 ranking_types 순위 API 순차 호출
    3. 하드 필터 순차 적용:
       a. 매수 중단 시간대(buy_pause_start~end) 해당 시 대기
       b. 연속손절 한도(max_consecutive_losses) 초과 시 중단
       c. 지수 필터 — index_codes 지수 임계 하락 시 중단
       d. 거래대금 하한선(min_trading_value) 미달 제거
       e. 호가 스프레드 상한(max_bidask_spread_pct) 초과 제거
       f. RSI/이격도/고가괴리/체결강도 하드 룰 제거
       g. 일일 손실·목표 한도 초과 시 중단

    4. [SCORING] scorer.Score(candidates) → 상위 종목 선정

    5. GetInquireBalance() → 가용자금 확인
    6. CheckOrderFeasibility(code) → orderableQty
    7. PlaceOrder(시장가, qty × order_amount_pct%)   [ORDERING]
    8. ExecCh drain-and-match (KISOrderID 매칭, 최대 5분)   [WAITING_FILL]
       └─ 타임아웃 → CancelOrder() → 1로

    9. Monitor.Register(pos, SoldCh)   [MONITORING]
       └─ sell_on_upper_limit=true 이면 GetStockPrice() → UpperLimitPrice 세팅
   10. updateConsecutiveLosses(code)   손절/익절에 따라 카운터 갱신
   11. 1로 (max_positions 미충족 시 즉시 다음 종목)

  NO  → soldCh 대기 (or 30초 주기 re-check)
        └─ sold 수신 → 1로
```

---

## 실시간 가격 플로우

```
KIS WebSocket (H0STCNT0)
  ↓ PriceCh (buffered 256)
monitor.StartPriceConsumer()
  ↓ HandlePrice(isTest=false)
  ├─ 트레일링 스탑 조건  → executeSell()      → SoldCh → Remove()
  ├─ 분할 익절 조건      → executePartialSell() → PartialTPDone=true
  ├─ 상한가 도달         → executeSell()      → SoldCh → Remove()
  ├─ price ≥ TargetPrice → executeSell()      → SoldCh → Remove()
  ├─ price ≤ StopPrice   → executeSell()      → SoldCh → Remove()
  └─ 횡보 감지           → executePartialSell() or executeSell()

KIS WebSocket (H0STCNI0 체결통보)
  ↓ ExecCh (buffered 64)
trader.Engine.waitForFill() — KISOrderID 매칭, 5분 타임아웃

monitor.StartIndicatorChecker() (intervalMin 주기, 별도 goroutine)
  → ops.GetStockInfo(code) → RSI14, MACDLine, MACDSignal
  → RSI 과매수 or MACD 데드크로스 → executeSell() → SoldCh → Remove()

15:15 LiquidateAll():
  → ops.GetHoldings() → PlaceSellOrder(시장가) × 전 종목
```

---

## Firestore 컬렉션

| 컬렉션 | 문서 ID | 설명 |
|--------|---------|------|
| `settings` | `config` | 모든 트레이딩 설정 (flat key-value) |
| `orders` | 주문 ID | 매수·매도 주문 기록 |
| `monitored_positions` | 종목 코드 | 현재 모니터링 중인 포지션 |
| `balances` | 잔고 ID | 잔고 스냅샷 |
| `trade_reports` | 리포트 ID | 매수+매도 완료 거래 상세 |
| `daily_reports` | YYYY-MM-DD | 일별 P&L 요약 리포트 |
| `service_logs` | 로그 ID | 서비스 운영 이벤트 로그 |
| `kis_api_logs` | 로그 ID | KIS API 에러 로그 |
| `scan_logs` | 스캔 ID | 트레이더 스캔 사이클 결과 |
| `tokens` | `current` | KIS OAuth 토큰 캐시 |
| `stock_masters` | 종목 코드 | 종목 마스터 데이터 (MST 파일) |

---

## API Endpoint Map

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/balance` | 계좌 잔고 |
| GET | `/api/positions` | 현재 보유 종목 |
| GET | `/api/stock/:code` | 종목 현재가 + 기술지표 |
| GET | `/api/stock/:code/chart` | 캔들 차트 |
| GET | `/api/stocks` | 종목 마스터 목록 |
| GET | `/api/orders` | 주문 내역 |
| POST | `/api/orders` | 수동 주문 |
| POST | `/api/orders/:id/cancel` | 주문 취소 |
| DELETE | `/api/orders/:id` | 주문 삭제 |
| GET | `/api/orders/feasibility` | 주문가능수량/금액 |
| GET | `/api/monitor/positions` | 모니터링 포지션 목록 |
| DELETE | `/api/monitor/positions/:code` | 포지션 해제 |
| POST | `/api/monitor/positions/:code/sell` | 강제 매도 |
| POST | `/api/monitor/liquidate-all` | 전량 청산 |
| GET | `/api/market/status` | 장운영 여부 |
| GET | `/api/ranking/volume` | 거래량 순위 |
| GET | `/api/ranking/strength` | 체결강도 순위 |
| GET | `/api/ranking/fluctuation` | 등락률 순위 |
| GET | `/api/ranking/vi-status` | VI 발동 현황 |
| GET | `/api/settings` | 설정 조회 |
| PATCH | `/api/settings` | 설정 변경 |
| GET | `/api/server/status` | 서버 통합 상태 |
| POST | `/api/trader/force-run` | 트레이더 즉시 실행 |
| GET | `/api/stats/daily-pnl` | 일별 P&L 집계 |
| GET | `/api/reports/trades` | 거래 리포트 목록 |
| GET | `/api/reports/daily` | 일일 리포트 목록 |
| POST | `/api/reports/daily/generate` | 일일 리포트 수동 생성 |
| GET | `/api/logs/kis` | KIS API 에러 로그 |
| DELETE | `/api/logs/kis/:id` | 에러 로그 단건 삭제 |
| GET | `/api/logs/service` | 서비스 로그 |
| DELETE | `/api/logs/service/:id` | 서비스 로그 단건 삭제 |
| GET | `/api/logs/scan` | 스캔 로그 |
| POST | `/api/ws/connect` | WebSocket 연결 |
| POST | `/api/ws/disconnect` | WebSocket 해제 |
| GET | `/health` | 헬스 체크 |
