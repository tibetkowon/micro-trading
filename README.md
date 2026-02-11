# Micro Trading

1vCPU/1GB 서버에서 동작하는 초경량 자동매매 서비스

## Tech Stack

- **Backend**: FastAPI, SQLAlchemy (async), APScheduler
- **Broker**: 한국투자증권(KIS) OpenAPI / Paper Trading
- **DB**: SQLite (aiosqlite)
- **Deploy**: Docker, GitHub Actions → NCP

## Features

- 실시간 시세 조회 및 주문 실행
- 내장 전략: DCA, 이동평균, RSI 리밸런싱
- 모의투자(Paper Trading) 모드 지원
- 웹 대시보드 (포트폴리오, 포지션, 주문 내역)

## Quick Start

```bash
# 환경변수 설정
cp .env.example .env

# 실행
pip install .
python run.py
```

### Docker

```bash
docker build -t micro-trading .
docker run -p 8000:8000 --env-file .env micro-trading
```

서버 실행 후 `http://localhost:8000` 에서 대시보드 확인
