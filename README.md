# micro-trading

Claude AI가 KIS(한국투자증권) API를 통해 종목 선정부터 매수·모니터링·매도·일일 리포트까지 완전 자율 수행하는 국내 주식 자동매매 시스템.  
NCP Micro (1GB RAM) 환경에서 효율적으로 동작하도록 설계되었습니다.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.26.1, Gin |
| Database | Firebase Firestore |
| Realtime | KIS WebSocket (gorilla/websocket) |
| Frontend | React 18, Vite, Firebase SDK |
| Hosting | Firebase Hosting (Frontend) · GCS + NCP VM (Backend) |
| CI/CD | GitHub Actions |

## 시스템 개요

서버가 매일 장 시간 중 다음 사이클을 자율 반복합니다:

```
순위 조회 → 하드 필터 → 스코어링 → 매수 주문
→ WebSocket 체결 확인 → 포지션 모니터 등록
→ 목표가 / 손절가 / 트레일링스탑 / 상한가 / 횡보 감시 → 자동 매도
→ 다음 사이클
```

매일 15:20에 당일 거래 내역을 분석한 일일 리포트를 Firestore에 저장합니다.

---

## Quick Start

### 1. 환경변수 설정

```bash
cp .env.example .env
```

| 키 | 필수 | 설명 |
|----|------|------|
| `KIS_APP_KEY` | ✅ | KIS Open API 앱 키 |
| `KIS_APP_SECRET` | ✅ | KIS Open API 시크릿 |
| `KIS_ACCOUNT_NO` | ✅ | 계좌번호 앞 8자리 |
| `KIS_ACCOUNT_TYPE` | | `01`=종합계좌 (기본값) |
| `KIS_BASE_URL` | ✅ | 실전: `https://openapi.koreainvestment.com:9443` |
| `KIS_HTS_ID` | | HTS 아이디 — 실시간 체결통보 수신 시 필요 |
| `FIREBASE_PROJECT_ID` | ✅ | Firebase 프로젝트 ID |
| `FIREBASE_CREDENTIALS_JSON` | ✅ | 서비스 계정 JSON 파일 경로 |
| `SERVER_PORT` | | 기본값 `8080` |
| `FRONTEND_ORIGIN` | | CORS 허용 프론트엔드 도메인 |

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
```

---

## 자동 스케줄러

| 시각 (KST) | 동작 |
|-----------|------|
| **08:40** | 종목 마스터(MST) 파일 다운로드 |
| **08:50** | KIS 토큰 발급 → WebSocket 연결 → 체결통보 구독 |
| **09:00** | `trading_enabled` 확인 → `tradingReady` 세팅 + 연속손절 카운터 초기화 |
| **`trading_start_time`** (기본: 09:15) | 자율 트레이딩 엔진 시작 + 지표 감시 시작 |
| **`trading_end_time`** (기본: 15:15) | 엔진 정지 → 전량 시장가 청산 |
| **15:20** | 일일 리포트 생성 |
| **16:00** | WebSocket 연결 해제 |

> 거래 시작·종료 시간은 Settings 화면에서 변경 가능하며, 변경 시 다음 틱(30초 이내) 반영됩니다.

---

## 매도 조건 우선순위

`HandlePrice()` 에서 아래 순서로 평가됩니다:

1. **트레일링 스탑** — `trailing_trigger_pct` 달성 후 고점 대비 `trailing_stop_pct` 하락 시 전량 매도
2. **분할 익절** — `partial_tp_pct` 달성 시 `partial_tp_ratio` 비율 매도 (1회 한정)
3. **상한가 매도** — `sell_on_upper_limit=true` 이고 당일 상한가 도달 시 즉시 전량 매도
4. **목표가 (익절)** — `take_profit_pct` 달성 시 전량 매도
5. **손절가 (손절)** — `stop_loss_pct` 도달 시 전량 매도
6. **횡보 감지** — `stagnation_threshold_pct` 범위 내 `stagnation_duration_min` 지속 시 분할 청산

---

## 트레이딩 설정

Settings 화면 또는 `PATCH /api/settings` 로 변경합니다. 모든 설정은 Firestore `settings/config` 에 저장됩니다.

### 기본 설정

| 설정 키 | 기본값 | 설명 |
|---------|--------|------|
| `trading_enabled` | true | OFF 시 주문 API 차단 |
| `trading_start_time` | 09:15 | 엔진 시작 시간 (KST) |
| `trading_end_time` | 15:15 | 엔진 종료 + 전량 청산 시간 (KST) |
| `max_positions` | 1 | 동시 보유 최대 종목 수 |
| `order_amount_pct` | 95.0 | 주문가능금액 대비 실제 주문 비율 (%) |
| `buy_pause_start` / `buy_pause_end` | "" | 매수 중단 시간대 (HH:MM) |
| `daily_max_loss_pct` | 0 | 일일 최대 손실 한도 (%, 0=비활성) |
| `daily_target_profit_pct` | 0 | 일일 목표 수익 한도 (%, 0=비활성) |

### 익절 / 손절

| 설정 키 | 기본값 | 설명 |
|---------|--------|------|
| `take_profit_pct` | 3.0 | 기본 익절 기준 (%) |
| `stop_loss_pct` | 2.0 | 기본 손절 기준 (%) |
| `etf_take_profit_pct` | 0.5 | ETF 익절 기준 (%) |
| `etf_stop_loss_pct` | 1.0 | ETF 손절 기준 (%) |
| `stock_take_profit_pct` | 1.5 | 주식 익절 기준 (%) |
| `stock_stop_loss_pct` | 1.0 | 주식 손절 기준 (%) |

### 트레일링 스탑

| 설정 키 | 기본값 | 설명 |
|---------|--------|------|
| `trailing_trigger_pct` | 0 | 트레일링 스탑 활성화 기준 수익률 (0=비활성) |
| `trailing_stop_pct` | 0 | 고점 대비 허용 하락폭 (%) |

### 분할 익절

| 설정 키 | 기본값 | 설명 |
|---------|--------|------|
| `partial_tp_enabled` | false | 분할 익절 사용 여부 |
| `partial_tp_pct` | 1.0 | 분할 익절 기준 수익률 (%) |
| `partial_tp_ratio` | 0.5 | 분할 익절 비율 (예: 0.5 = 50%) |
| `partial_tp_raise_stop` | false | 분할 익절 후 손절가 본전 상향 여부 |

### 매도 조건 제어

| 설정 키 | 기본값 | 설명 |
|---------|--------|------|
| `sell_conditions` | ["take_profit","stop_loss"] | 활성화할 매도 조건 목록 |
| `sell_on_upper_limit` | false | 당일 상한가 도달 시 즉시 매도 |
| `max_consecutive_losses` | 0 | 연속 손절 허용 횟수 (0=비활성, 초과 시 당일 매수 중단) |
| `consecutive_loss_reset_on_profit` | true | 익절 시 연속손절 카운터 리셋 |

### 횡보 감지

| 설정 키 | 기본값 | 설명 |
|---------|--------|------|
| `stagnation_threshold_pct` | 0 | 횡보 기준 변동폭 (%, 0=비활성) |
| `stagnation_duration_min` | 0 | 횡보 지속 기준 시간 (분) |
| `stagnation_partial_exit_enabled` | false | 횡보 감지 시 분할 청산 여부 |
| `stagnation_bidask_sell_threshold` | 1.0 | 호가잔량비율 미만 시 즉시 청산 기준 |

### 지표 감시

| 설정 키 | 기본값 | 설명 |
|---------|--------|------|
| `indicator_check_interval_min` | 5 | 지표 확인 주기 (분) |
| `indicator_rsi_sell_threshold` | 70.0 | RSI 과매수 기준값 |
| `indicator_macd_bearish_sell` | false | MACD 데드크로스 시 매도 여부 |

### 순위 조회 설정

| 설정 키 | 기본값 | 설명 |
|---------|--------|------|
| `ranking_types` | ["volume","strength"] | 순위 유형 (volume/strength/fluctuation/vi_status) |
| `ranking_condition` | OR | AND=교집합, OR=합집합 |
| `ranking_price_min` | 5000 | 최소 주가 (원) |
| `ranking_price_max` | 200000 | 최대 주가 (원) |
| `ranking_top_n` | 30 | 타입별 상위 N개 |
| `ranking_exchanges` | ["0001","1001"] | 거래소 코드 |
| `ranking_exclude_cls` | 1111111111 | 순위 조회 제외 종목 플래그 |

### 하드 필터 (진입 전 제외 기준)

| 설정 키 | 기본값 | 설명 |
|---------|--------|------|
| `max_bidask_spread_pct` | 0 | 최대 허용 호가 스프레드 (%, 0=비활성) |
| `min_trading_value` | 0 | 최소 거래대금 (원, 0=비활성) |
| `index_drop_threshold_pct` | -1.0 | 지수 낙폭 임계값 — 초과 시 매수 중단 (%) |
| `index_codes` | [] | 지수 필터 코드 목록 |
| `hard_rsi_max` | 70.0 | RSI 상한 |
| `hard_strength_min` | 100.0 | 최소 체결강도 (%) |
| `hard_disparity_m5_min` | -1.5 | 5분봉 이격도 하한 (%) |
| `hard_disparity_m5_max` | 3.0 | 5분봉 이격도 상한 (%) |
| `hard_high_price_diff_max` | -0.5 | 당일 고점 대비 최대 하락 (%) |
| `hard_high_price_diff_min` | -5.0 | 당일 고점 대비 최소 하락 (%) |
| `hard_open_price_diff_max` | 15.0 | 시가 대비 상승률 상한 (%) |
| `hard_macd_bearish_enabled` | false | MACD 데드크로스 진입 차단 |

### 스코어링 가중치

| 설정 키 | 기본값 | 설명 |
|---------|--------|------|
| `min_score_threshold` | 0 | 최소 점수 기준 (0=비활성) |
| `score_weight_strength` | 30 | 체결강도 가중치 |
| `score_weight_rsi` | 20 | RSI 가중치 |
| `score_weight_macd` | 20 | MACD 가중치 |
| `score_weight_bidask` | 15 | 호가잔량 가중치 |
| `score_weight_vwap` | 10 | VWAP 가중치 |
| `score_weight_volume` | 5 | 거래량 가중치 |

---

## UI 페이지

| 경로 | 페이지 | 설명 |
|------|--------|------|
| `/` | 대시보드 | 계좌 잔고, 포지션, 일별 수익 요약 |
| `/positions` | 포지션 | 모니터링 중인 보유 종목 |
| `/orders` | 주문 내역 | 전체 주문 이력 |
| `/reports/trades` | 거래 리포트 | 매수·매도 완료 건별 상세 기록 |
| `/reports/daily` | 일일 리포트 | 일별 P&L 요약 |
| `/settings` | 설정 | KIS 자격증명 + 모든 트레이딩 설정 |
| `/logs` | 로그 | 서비스·KIS API·스캔 로그 |

---

## API Endpoints

### 계좌 / 잔고

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/balance` | 계좌 잔고 조회 |
| GET | `/api/positions` | 현재 보유 종목 조회 |

### 종목 / 차트

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/stock/:code` | 종목 현재가 + 기술지표 (RSI, MACD, MA) |
| GET | `/api/stock/:code/chart` | 캔들 차트 (`?interval=1m\|5m\|1h`) |
| GET | `/api/stocks` | 종목 마스터 목록 |

### 주문

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/orders` | 주문 내역 조회 |
| POST | `/api/orders` | 수동 주문 실행 |
| POST | `/api/orders/:id/cancel` | KIS 미체결 주문 취소 |
| DELETE | `/api/orders/:id` | 주문 단건 삭제 |
| GET | `/api/orders/feasibility` | 주문가능수량 / 주문가능금액 조회 |

### 모니터링

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/monitor/positions` | 모니터링 중인 포지션 목록 |
| DELETE | `/api/monitor/positions/:code` | 모니터링 포지션 수동 해제 |
| POST | `/api/monitor/positions/:code/sell` | 특정 포지션 강제 매도 |
| POST | `/api/monitor/liquidate-all` | 전량 즉시 청산 |

### 순위

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/ranking/volume` | 거래량 순위 |
| GET | `/api/ranking/strength` | 체결강도 순위 |
| GET | `/api/ranking/fluctuation` | 등락률 순위 |
| GET | `/api/ranking/vi-status` | VI 발동 현황 |

### 설정

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/settings` | 모든 설정 조회 |
| PATCH | `/api/settings` | 설정 변경 |

### 리포트

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/reports/trades` | 거래 리포트 목록 |
| GET | `/api/reports/daily` | 일일 리포트 목록 |
| POST | `/api/reports/daily/generate` | 일일 리포트 수동 생성 |

### 로그

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/logs/kis` | KIS API 에러 로그 |
| DELETE | `/api/logs/kis/:id` | 에러 로그 단건 삭제 |
| GET | `/api/logs/service` | 서비스 전체 로그 |
| DELETE | `/api/logs/service/:id` | 서비스 로그 단건 삭제 |
| GET | `/api/logs/scan` | 스캔 사이클 로그 |

### 서버 / 트레이더

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/server/status` | 통합 서버 상태 |
| GET | `/api/market/status` | 장운영 여부 |
| POST | `/api/trader/force-run` | 자율 트레이더 즉시 실행 |
| GET | `/api/stats/daily-pnl` | 일별 P&L 집계 |
| POST | `/api/ws/connect` | WebSocket 수동 연결 |
| POST | `/api/ws/disconnect` | WebSocket 수동 해제 |
| GET | `/health` | 헬스 체크 |

---

## Project Structure

자세한 구조와 패키지 역할: [`docs/architecture.md`](docs/architecture.md)

DB 스키마 상세: [`docs/db_schema.md`](docs/db_schema.md)

---

## Security

- 모든 민감 정보 (KIS 자격증명, Firebase 서비스 계정)는 `.env` 파일로만 관리
- `.env` 및 `firebase-credentials.json`은 `.gitignore`에 의해 커밋에서 제외됩니다
- 트레이딩 설정은 Firestore `settings/config` 문서에 저장됩니다
- KIS API 에러는 Firestore `kis_api_logs`에, 서비스 이벤트는 `service_logs`에 자동 기록됩니다

## License

Private
