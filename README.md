# micro-trading-for-agent

Claude AI가 KIS(한국투자증권) API를 통해 종목 선정부터 매수·모니터링·매도·일일 리포트까지 완전 자율 수행하는 주식 자동매매 시스템.
NCP Micro (1GB RAM) 환경에서 효율적으로 동작하도록 설계되었습니다.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.24, Gin, SQLite (go-sqlite3) |
| AI | Anthropic Claude API (`anthropic-sdk-go`) |
| Realtime | KIS WebSocket (`gorilla/websocket`) |
| Frontend | React 18, Vite, TailwindCSS |
| Database | SQLite (WAL mode) |
| CI/CD | GitHub Actions |

## 시스템 개요

국내(KR) + 미장(US) 동시 운영 듀얼 엔진으로 서버가 Claude API를 활용하여 각 시장 시간 중 다음 사이클을 자율 반복합니다:

**KR (국내):**
```
국내 순위 조회 → 품질 필터(지수·거래대금·RSI/이격도) → Claude 종목 선정 → KIS 시장가 매수
  → WebSocket 체결 확인 → 포지션 모니터 등록
  → 목표가/손절가/트레일링스탑/지표 감시 → 자동 매도 → 다음 사이클
```

**US (미장):**
```
해외 거래량 순위 조회 → 품질 필터(MA5/MA20/RSI/MACD/5분봉이격도) → Claude 종목 선정 → 해외 시장가 매수
  → WebSocket 체결 확인 → 포지션 모니터 등록
  → 목표가/손절가 감시 → 자동 매도 → 다음 사이클
```

매일 15:20에 Claude가 당일 거래 내역을 분석한 한국어 마크다운 일일 리포트를 생성하여 DB에 저장합니다.

---

## Quick Start

### 1. 환경변수 설정

```bash
cp .env.example .env
# .env 파일에 필요한 값을 입력하세요
```

| 키 | 필수 | 설명 |
|----|------|------|
| `KIS_APP_KEY` | ✅ | KIS Open API 앱 키 |
| `KIS_APP_SECRET` | ✅ | KIS Open API 시크릿 |
| `KIS_ACCOUNT_NO` | ✅ | 계좌번호 앞 8자리 |
| `KIS_ACCOUNT_TYPE` | | `01`=종합계좌 (기본값) |
| `KIS_BASE_URL` | | 실전: `https://openapi.koreainvestment.com:9443` |
| `KIS_HTS_ID` | | HTS 아이디 — 실시간 체결통보 수신 시 필요 |
| `ANTHROPIC_API_KEY` | ✅ | Claude API 키 — 자율 트레이딩 엔진 동작에 필요 |

> `ANTHROPIC_API_KEY` 미설정 시 서버는 기동되지만 자율 매매 엔진이 비활성화됩니다.

### 2. 백엔드 실행

```bash
cd backend
go mod download
go run cmd/server/main.go
# → http://localhost:8080
```

### 3. 프론트엔드 실행 (개발)

```bash
cd frontend
npm install
npm run dev
# → http://localhost:3000
```

---

## 자동 스케줄러 동작

| 시각 (KST) | 동작 | 조건 |
|-----------|------|------|
| **08:50** | KIS 토큰 갱신 + WebSocket 연결 + 체결통보 구독 | 평일 |
| **09:00** | `trading_enabled` 확인 + 장 개장 여부 확인 → tradingReady 세팅 | 평일 |
| **`trading_start_time`** (기본: 09:15) | 자율 트레이딩 엔진 시작 + 지표 감시 시작 | `tradingReady == true` |
| **`trading_end_time`** (기본: 15:15) | KR 엔진 정지 + 국내 포지션 전량 시장가 청산 | 평일 |
| **15:20** | Claude 일일 리포트 생성 → DB 저장 | 평일 |
| **16:00** | WebSocket 연결 해제 | 평일 |
| 5분 주기 | KIS 체결 내역 → DB 동기화 | 장 중에만 |
| **`us_trading_start_time`** (기본: 22:30 KST) | US 엔진 시작 | `us_trading_enabled == true` |
| **`us_trading_end_time`** (기본: 05:00 KST) | US 엔진 정지 + 미국 포지션 전량 청산 | - |

> 거래 시작·종료 시간은 Settings 화면에서 변경 가능합니다. 변경 시 다음 틱(30초 이내)부터 반영됩니다.

---

## 트레이딩 엔진 상태 머신

```
IDLE → SEARCHING → SELECTING → ORDERING → WAITING_FILL → MONITORING
           ↑                                                    │
           └──────────────── (매도 완료 신호) ───────────────────┘
```

- **SEARCHING**: 포지션 여유 있음 — 매수 종목 탐색 대기 단계
- **SELECTING**: 순위 API 조회 → 품질 필터 적용 → Claude에 종목 선정 요청
- **ORDERING**: KIS 시장가 매수 주문 실행
- **WAITING_FILL**: WebSocket ExecCh에서 체결 확인 (최대 5분 → 타임아웃 시 취소 후 재선정)
- **MONITORING**: Monitor에 포지션 등록 후 다음 종목 선정 대기

---

## 트레이딩 설정

Settings 화면 또는 `PATCH /api/settings` API로 변경합니다.

### 공통 설정

| 설정 키 | 기본값 | 설명 |
|---------|--------|------|
| `trading_enabled` | true | OFF 시 주문 API 차단 |
| `take_profit_pct` | 3.0 | 익절 기준 (%) |
| `stop_loss_pct` | 2.0 | 손절 기준 (%) |
| `max_positions` | 1 | 동시 보유 최대 종목 수 |
| `order_amount_pct` | 95 | 주문가능금액 대비 실제 주문 비율 (%) |
| `trading_start_time` | 09:15 | KR 엔진 시작 시간 (KST) |
| `trading_end_time` | 15:15 | KR 엔진 종료 + 전량 청산 시간 (KST) |
| `buy_pause_start` | "" | 매수 중단 시작 시간 |
| `buy_pause_end` | "" | 매수 중단 종료 시간 |
| `daily_max_loss_pct` | 0 | 일일 최대 손실 한도 (국장 KRW 기준, 0=비활성) |
| `min_trading_value` | 0 | 최소 거래대금 (국장 KRW 기준, 0=비활성) |
| `trailing_trigger_pct` | 0 | 트레일링 스탑 활성화 기준 수익률 (0=비활성) |
| `trailing_stop_pct` | 1.0 | 트레일링 스탑 고점 대비 허용 하락폭 (%) |
| `stagnation_threshold_pct` | 1.0 | 횡보 감지 기준 변동폭 (%) |
| `stagnation_duration_min` | 30 | 횡보 지속 기준 시간 (분) |
| `sell_conditions` | ["target_pct","stop_pct"] | 매도 조건 우선순위 배열 |
| `indicator_check_interval_min` | 5 | 지표 확인 주기 (분) |
| `indicator_rsi_sell_threshold` | 70 | RSI 과매수 기준값 |
| `indicator_macd_bearish_sell` | false | MACD 데드크로스 시 매도 여부 |
| `claude_model` | claude-sonnet-4-6 | 종목 선정·리포트에 사용할 모델 |

### 국장(KR) 순위 조회 설정

| 설정 키 | 기본값 | 설명 |
|---------|--------|------|
| `ranking_types` | ["volume","strength","exec_count","disparity"] | 순위 유형 |
| `ranking_condition` | AND | AND=교집합, OR=합집합 |
| `ranking_price_min` | 5000 | 최소 주가 (원) |
| `ranking_price_max` | 100000 | 최대 주가 (원) |
| `ranking_top_n` | 20 | 타입별 상위 N개 제한 |
| `ranking_volume_min_incrrate` | 0 | 거래량 전일대비 최소 증가율 (%) |
| `ranking_strength_min` | 100 | 최소 체결강도 (%) |
| `ranking_execcount_net_buy_only` | true | 순매수체결량 > 0 종목만 허용 |
| `ranking_disparity_d20_min` | 0 | 20일 이격도 최솟값 |
| `ranking_disparity_d20_max` | 0 | 20일 이격도 최댓값 |

### 국장(KR) 하드 필터

| 설정 키 | 기본값 | 설명 |
|---------|--------|------|
| `filter_rsi_max` | 80 | RSI 상한 (이상이면 후보 제외) |
| `filter_disparity_m5_max` | 3.0 | 5분봉 이격도 상한 (%) |
| `filter_high_price_diff_min` | -5.0 | 당일 고점 낙폭 하한 (%) |
| `filter_open_price_diff_max` | 20.0 | 당일 시가 대비 상승률 상한 (%) |
| `index_drop_threshold_pct` | -1.0 | 지수 낙폭 임계값 (%) |
| `index_codes` | [] | 지수 필터 코드 JSON 배열 |
| `ranking_excl_cls` | 1111111111 | 순위 조회 제외 종목 플래그 |

### 미장(US) 설정

| 설정 키 | 기본값 | 설명 |
|---------|--------|------|
| `us_trading_enabled` | false | US 엔진 ON/OFF |
| `us_exchange` | NAS | 거래소 코드 (NAS/NYS/AMS) |
| `us_trading_start_time` | 22:30 | US 엔진 시작 시간 (KST) |
| `us_trading_end_time` | 05:00 | US 엔진 종료 + 전량 청산 시간 (KST) |
| `us_price_min` | "" | 최소 주가 (USD) |
| `us_price_max` | "" | 최대 주가 (USD) |
| `us_daily_max_loss_pct` | 0 | 미장 일일 최대 손실 한도 (USD 기준, 0=국장 값 공유) |
| `us_min_trading_value` | 0 | 미장 최소 거래대금 (USD, 0=국장 값 공유) |

---

## UI 페이지

| 경로 | 페이지 | 설명 |
|------|--------|------|
| `/` | 대시보드 | 서버 상태, 잔고, 국장/미장 트레이더 상태 |
| `/monitor` | 모니터 | 모니터링 중인 포지션 목록 (목표가/손절가/현재가) |
| `/orders` | 주문 내역 | 전체 주문 이력 (KR/US 배지, 수동 동기화) |
| `/logs` | 에러 로그 | 서비스 로그(TRADER/MONITOR/SYSTEM) + KIS API 에러 로그 탭 |
| `/selection-logs` | 선정 로그 | LLM 종목 선정 시도 기록 (후보 목록, LLM 결과, 선정 사유) |
| `/ranking-logs` | 순위 조회 로그 | 순위 조회 파라미터 + 결과 종목 수 기록 |
| `/settings` | 설정 | KIS 자격증명 + 모든 트레이딩 설정 변경 |

---

## API Endpoints

### 계좌 / 잔고

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/balance` | 계좌 잔고 조회 |
| GET | `/api/positions` | 실시간 보유 종목 조회 |

### 종목 / 차트

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/stock/:code` | 종목 현재가 + MA5/MA20 + RSI14 + MACD |
| GET | `/api/stock/:code/chart` | 캔들 차트 (`?interval=1m\|5m\|1h`) |

### 주문

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/orders` | 주문 내역 조회 (`?sync=true` KIS 동기화) |
| POST | `/api/orders` | 수동 주문 실행 |
| POST | `/api/orders/:id/cancel` | KIS 미체결 주문 취소 |
| DELETE | `/api/orders/:id` | 주문 단건 삭제 |
| GET | `/api/orders/feasibility?code=` | 주문가능수량 / 주문가능금액 조회 |

### 모니터링

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/monitor/positions` | 모니터링 중인 포지션 목록 |
| DELETE | `/api/monitor/positions/:code` | 모니터링 포지션 수동 해제 |

### 순위

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/ranking/volume` | 거래량 순위 |
| GET | `/api/ranking/strength` | 체결강도 순위 |
| GET | `/api/ranking/exec-count` | 대량체결건수 순위 |
| GET | `/api/ranking/disparity` | 이격도 순위 |

### 서버 / 장 운영 상태

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/server/status` | 통합 서버 상태 |
| GET | `/api/market/status` | 장운영 여부 (KIS 영업일 기준) |

### 설정

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/settings` | 모든 설정 조회 |
| PATCH | `/api/settings` | 설정 변경 |

### 일일 리포트

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/reports` | 리포트 날짜 목록 (최근 30일) |
| GET | `/api/reports/:date` | 특정 날짜 리포트 전문 (`YYYY-MM-DD`) |

### 로그

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/logs/kis` | KIS API 에러 로그 (`?summary=true` raw 제외) |
| DELETE | `/api/logs/kis/:id` | 에러 로그 단건 삭제 |
| GET | `/api/logs/service` | 서비스 전체 로그 (`?limit=100&source=ALL\|TRADER\|MONITOR\|SYSTEM`) |
| GET | `/api/logs/selection` | LLM 종목 선정 로그 (`?limit=20`, 최신 순, 30일 자동 삭제) |
| GET | `/api/logs/ranking` | 순위 조회 로그 (최신 순, 30일 자동 삭제) |

### WebSocket 제어

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/ws/connect` | WebSocket 수동 연결 |
| POST | `/api/ws/disconnect` | WebSocket 수동 해제 |

### 헬스 체크

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | 서버 상태 확인 |

---

## Project Structure

자세한 구조와 패키지 역할: [`docs/architecture.md`](docs/architecture.md)

DB 스키마 상세: [`docs/db_schema.md`](docs/db_schema.md)

---

## Security

- 모든 민감 정보 (API 키, 계좌번호, Anthropic 키)는 `.env` 파일로만 관리
- `.env` 파일은 `.gitignore`에 의해 절대 커밋되지 않습니다
- KIS API 에러는 `kis_api_logs` 테이블에, 서비스 운영 이벤트는 `service_logs` 테이블에 자동 기록됩니다

## License

Private
