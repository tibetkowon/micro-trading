# Micro Trading

1vCPU / 1GB RAM 서버에서 동작하는 초경량 주식 모의투자 서비스

## Tech Stack

| 분류 | 기술 |
|:---|:---|
| Backend | Python 3.11+, FastAPI, SQLAlchemy 2.0 (async) |
| Database | SQLite (aiosqlite, WAL mode) |
| Scheduler | APScheduler (시세 갱신 30초, 전략 틱 1분) |
| Market Data | pykrx (한국), yfinance (미국) |
| Frontend | Jinja2 + HTMX + Chart.js |
| Deploy | Docker, GitHub Actions → NCP Micro |

## Features

| ID | 기능 | 설명 |
|:---|:---|:---|
| F-01 | 종목검색기 | 코드/명으로 KR·US 종목 검색 |
| F-02 | 종목 저장 | 관심종목 워치리스트 관리 |
| F-03 | 시세 수집기 | 3계층 캐시 (메모리→API→DB), 30초 자동 갱신 |
| F-04 | 가상 지갑 | .env 기반 초기자산, 거래 시 KRW/USD 잔고 자동 업데이트 |
| F-05 | 거래 엔진 | 시장가/지정가 주문, 수수료 반영 (기본 0.05%) |
| F-06 | 대시보드 | 실시간 자산현황, 관심종목 시세, 포트폴리오 추이 차트 |

## Architecture

```
app/
├── main.py              # FastAPI 앱 팩토리 + lifespan
├── config.py            # Pydantic Settings (.env)
├── database.py          # SQLAlchemy async engine + PRAGMA
├── models/              # ORM 모델 (Account, Order, Trade, Position, ...)
├── schemas/             # Pydantic 요청/응답 스키마
├── services/            # 비즈니스 로직 (OrderService, PortfolioService, ...)
├── broker/              # 브로커 추상화
│   ├── base.py          #   AbstractBroker 인터페이스
│   ├── paper/           #   모의투자 브로커 (PaperExecutionEngine)
│   ├── kis/             #   한국투자증권 실매매 브로커
│   └── free/            #   무료 시세 제공자 (pykrx + yfinance)
├── scheduler/           # APScheduler 잡 (시세 갱신, 전략 틱, 일일 스냅샷)
├── strategies/          # 내장 전략 (DCA, 이동평균, RSI)
├── api/                 # REST API (/api/market, /api/orders, ...)
└── web/                 # 웹 UI (Jinja2 템플릿, HTMX 파셜, 정적 파일)
```

## Quick Start

```bash
# 1. 환경 설정
cp .env.example .env    # PAPER_BALANCE_KRW, PAPER_BALANCE_USD 수정 가능

# 2. 의존성 설치
python -m venv .venv && source .venv/bin/activate
pip install .

# 3. 실행
python run.py
```

`http://localhost:8000` 에서 대시보드 확인

### Docker

```bash
docker compose up -d --build
```

### 테스트

```bash
pip install .[dev]
pytest -v
```

## Environment Variables

| 변수 | 기본값 | 설명 |
|:---|:---|:---|
| `TRADING_MODE` | `PAPER` | `PAPER` (모의투자) / `REAL` (실매매) |
| `PAPER_BALANCE_KRW` | `100000000` | 모의투자 초기 원화 잔고 |
| `PAPER_BALANCE_USD` | `100000.0` | 모의투자 초기 달러 잔고 |
| `PAPER_COMMISSION_RATE` | `0.0005` | 수수료율 (0.05%) |
| `KIS_APP_KEY` | - | 한국투자증권 앱 키 (실매매 시) |
| `KIS_APP_SECRET` | - | 한국투자증권 앱 시크릿 (실매매 시) |
| `DATABASE_URL` | `sqlite+aiosqlite:///./trading.db` | SQLite DB 경로 |

## API Endpoints

| Method | Path | 설명 |
|:---|:---|:---|
| GET | `/api/health` | 헬스체크 |
| GET | `/api/market/price/{symbol}` | 실시간 시세 조회 |
| GET | `/api/market/daily-prices/{symbol}` | 일봉 OHLCV |
| POST | `/api/orders` | 주문 생성 |
| GET | `/api/orders` | 주문 내역 조회 |
| GET | `/api/positions` | 보유 종목 조회 |
| GET | `/api/portfolio/summary` | 포트폴리오 요약 |
| GET | `/api/portfolio/snapshots` | 일일 스냅샷 이력 |

## Implementation Phases

| Phase | 상태 | 내용 |
|:---|:---|:---|
| Phase 1 | **완료** | 데이터 모델 설계 (ORM, FK 인덱스, PRAGMA 최적화) |
| Phase 2 | **완료** | 실시간 시장가 API 연동 (3계층 캐시, 30초 스케줄러) |
| Phase 3 | **완료** | 가상 거래 로직 (잔고검증, 수수료, KRW/USD 구분, 테스트) |
| Phase 4 | **완료** | 웹 대시보드 (자산현황, 관심종목 실시간 시세) |
| Phase 5 | **완료** | 웹 거래화면 (검색→추가, 주문 잔고/예상금액 표시) |
| Phase 6 | **완료** | CI/CD (GitHub Actions 테스트→배포→헬스체크) |

각 Phase별 구현 상세 및 학습 문서는 [`docs/study/`](docs/study/) 참조.

## License

Private
