# micro-trading-for-agent

Claude AI가 KIS(한국투자증권) API를 통해 종목 선정부터 매수·모니터링·매도·일일 리포트까지 완전 자율 수행하는 주식 자동매매 시스템.
NCP Micro (1GB RAM) 환경에서 효율적으로 동작하도록 설계되었습니다.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.24, Gin, SQLite (go-sqlite3) |
| AI | Anthropic Claude API (`anthropic-sdk-go`) |
| Realtime | KIS WebSocket (`gorilla/websocket`) |
| Alerting | MQTT (`paho.mqtt.golang`) + Mosquitto |
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
해외 거래량 순위 조회 → 품질 필터(MA5/MA20 등) → Claude 종목 선정 → 해외 시장가 매수
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
| `MQTT_BROKER_URL` | | MQTT 브로커 주소 (기본: `tcp://localhost:1883`) |
| `MQTT_CLIENT_ID` | | MQTT 클라이언트 ID (기본: `micro-trading-server`) |

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

| 설정 | 기본값 | 설명 |
|------|--------|------|
| 거래 ON/OFF | ON | OFF 시 주문 API 차단 |
| 익절 기준 | 3.0% | 목표가 도달 시 자동 매도 |
| 손절 기준 | 2.0% | 손절가 도달 시 자동 매도 |
| 최대 동시 보유 종목 | 1 | 이 수에 도달하면 매도 신호 대기 |
| 주문 금액 비율 | 95% | 주문가능금액 대비 실제 주문 비율 |
| 순위 조회 유형 | volume, strength, exec_count, disparity | Claude에 제공할 데이터 소스 |
| 순위 가격 범위 | 5,000~100,000원 | 주가 필터 |
| 매도 조건 | target_pct, stop_pct | 조건 우선순위 배열 |
| 지표 확인 주기 | 5분 | RSI/MACD 체크 간격 |
| RSI 과매수 기준 | 70 | RSI ≥ 이 값이면 매도 트리거 |
| MACD 데드크로스 매도 | OFF | MACD선 < 시그널선 시 매도 |
| 트레일링 스탑 트리거 | 0% (비활성) | 이 수익률 달성 시 트레일링 스탑 활성화 |
| 트레일링 스탑 간격 | 0% (비활성) | 고점 대비 이 % 하락 시 매도 |
| 매수 중단 시간대 | "" (비활성) | 예: `11:20~13:00` → 해당 시간대 매수 안 함 |
| 최소 거래대금 | 0 (비활성) | KRW/USD 기준, 이 미만 종목 제거 |
| 일일 최대 손실 한도 | 0% (비활성) | 당일 실현손실이 총평가 대비 이 % 초과 시 매수 중단 |
| 지수 필터 | [] (비활성) | 예: `["0001","1001"]` — 지수 -1% 이상 하락 시 매수 중단 |
| US 거래 활성화 | OFF | US 엔진 ON/OFF |
| US 거래 거래소 | NAS | NAS/NYS/AMS |
| US 가격 범위 | "" (제한없음) | USD 기준 최소/최대 주가 필터 |
| Claude 모델 | claude-sonnet-4-6 | 종목 선정 및 리포트 생성에 사용 |

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

`GET /api/server/status` 응답 예시:
```json
{
  "market_open": true,
  "trading_enabled": true,
  "available_cash": 500000,
  "ws_connected": true,
  "monitored_count": 1,
  "trader_state": "MONITORING"
}
```

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

## MQTT 알림 (사용자 알림용)

매도 이벤트 발생 시 Mosquitto MQTT 브로커로 알림을 발행합니다.

| 토픽 | 이벤트 | 발행 시점 |
|------|--------|----------|
| `trading/alert/{stock_code}` | `TARGET_HIT` | 현재가 ≥ 목표가 → 자동 매도 |
| `trading/alert/{stock_code}` | `STOP_HIT` | 현재가 ≤ 손절가 → 자동 매도 |
| `trading/liquidation` | `LIQUIDATION` | 15:15 전량 청산 |

페이로드 예시:
```json
{
  "event": "TARGET_HIT",
  "stock_code": "005930",
  "stock_name": "삼성전자",
  "trigger_price": 73100,
  "target_price": 73000,
  "stop_price": 69700,
  "sell_qty": 10,
  "profit_pct": 3.5,
  "profit_amount": 31000,
  "timestamp": "2026-03-06T10:30:00+09:00",
  "is_test": false
}
```

MQTT 브로커 설치 가이드: [`docs/guides/mqtt-setup.md`](docs/guides/mqtt-setup.md)

---

## Project Structure

자세한 구조와 패키지 역할: [`docs/architecture.md`](docs/architecture.md)

DB 스키마 상세: [`docs/db_schema.md`](docs/db_schema.md)

---

## Security

- 모든 민감 정보 (API 키, 계좌번호, Anthropic 키)는 `.env` 파일로만 관리
- `.env` 파일은 `.gitignore`에 의해 절대 커밋되지 않습니다
- KIS API 에러는 `kis_api_logs` 테이블에 자동 기록됩니다

## License

Private
