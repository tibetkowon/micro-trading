# micro-trading-for-agent

저사양 서버에서 동작하는 **규칙 기반 자동매매 시스템** (한국 주식 KRX).

한국투자증권(KIS) OpenAPI를 통해 종목 스크리닝, 자동 매수/매도, 포지션 모니터링을 수행하며 웹 대시보드에서 실시간으로 현황을 확인합니다.

---

## 목차

- [주요 기능](#주요-기능)
- [아키텍처](#아키텍처)
- [매매 사이클](#매매-사이클)
- [기술 스택](#기술-스택)
- [시작하기](#시작하기)
- [환경 변수](#환경-변수)
- [웹 대시보드](#웹-대시보드)
- [배포](#배포)
- [CI/CD](#cicd)
- [디렉토리 구조](#디렉토리-구조)

---

## 주요 기능

- **자동매매 엔진** — 거래량/체결강도 순위 조회 → 하드 필터 → 복합 스코어링 → 자동 매수
- **포지션 모니터링** — 목표가/손절가, 트레일링 스톱, 정체 감지, 분할 익절
- **KIS API 완전 연동** — REST + WebSocket(실시간 체결통보), 토큰 자동 갱신, TPS 속도 제한
- **규칙 기반 스코어링** — 체결강도, RSI, MACD, 매수/매도 비율, VWAP, 거래량 6개 지표 가중치 합산 (0–100점)
- **하드 필터** — RSI 과매수, 체결강도 하한, 이격도 범위, 고가 대비 이격, 시가 대비 상승률 등
- **웹 대시보드** — 잔고·수익률·주문내역·로그 모니터링
- **시장 라이프사이클 스케줄러** — 개장 전 MST 다운로드부터 폐장 후 WebSocket 종료까지 자동 관리

---

## 아키텍처

```
┌─────────────────────────────────────────────────────────────┐
│                        NCP VM (Backend)                     │
│                                                             │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │  Market      │───▶│  Trading     │───▶│  Position    │  │
│  │  Scheduler   │    │  Engine      │    │  Monitor     │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│         │                   │                   │           │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │  KIS Client  │    │   Scorer     │    │  KIS Client  │  │
│  │  (REST/WS)   │    │  (6지표)    │    │  (주문실행)  │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│                                                             │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              Gin HTTP Server (:8080)                   │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
          │                                       │
    ┌─────▼──────┐                     ┌──────────▼─────────┐
    │  KIS API   │                     │  Firebase Firestore │
    │ (한국투자) │                     │  (설정·주문·로그)  │
    └────────────┘                     └────────────────────┘
                                                 │
                                    ┌────────────▼────────────┐
                                    │  Firebase Hosting       │
                                    │  React Dashboard (SPA)  │
                                    └─────────────────────────┘
```

---

## 매매 사이클

매일 장중 아래 스케줄로 동작합니다.

```
[08:40] 종목 마스터(MST) 파일 다운로드 및 Firestore 갱신
[08:50] KIS 토큰 발급 → WebSocket 연결 → 체결통보(H0STCNI0) 구독
[09:00] trading_enabled 설정 + 장 개장 여부 확인
[09:15] 자동매매 엔진 + 지표 체커 시작  ← 설정으로 시간 변경 가능
         │
         ▼
    1. 거래량/체결강도 순위 조회 (KIS API)
         │
    2. 하드 필터링 ──── 탈락 → 스킵
         │
    3. 지표 수집 (RSI·MACD·VWAP·거래량 등)
         │
    4. 복합 스코어링 (0–100점, 가중치 합산)
         │
    5. 목표 점수 이상?
       YES → 매수 주문
       NO  → 다음 스캔까지 대기
         │
    6. 포지션 모니터링 (30초 주기)
       목표가 도달     → 매도
       손절가 도달     → 손절
       트레일링 스톱   → 매도
       지표 악화       → 조기 청산
       정체 감지       → 조기 청산 또는 분할 익절
         │
    └──────── (scan_interval 분마다 반복)
[15:15] 자동매매 엔진 정지 → 전량 청산  ← 설정으로 시간 변경 가능
[15:20] 일별 거래 리포트 생성
[16:00] WebSocket 연결 종료
```

---

## 기술 스택

| 영역 | 기술 |
|------|------|
| Backend | Go 1.26.1, Gin |
| Frontend | React 18, Vite 5, React Router 6 |
| Database | Firebase Firestore |
| API 연동 | KIS OpenAPI (REST + WebSocket) |
| Hosting | Firebase Hosting (프론트) · NCP VM (백엔드) |
| 스토리지 | Google Cloud Storage (바이너리 배포) |
| CI/CD | GitHub Actions |

---

## 시작하기

### 사전 요구사항

- Go 1.26.1+
- Node.js 20+
- Firebase 프로젝트 (Firestore 활성화)
- KIS OpenAPI 앱 키 ([KIS Developers](https://apiportal.koreainvestment.com) 발급)
- Firebase 서비스 계정 JSON

### 1. 저장소 클론

```bash
git clone https://github.com/your-org/micro-trading-for-agent.git
cd micro-trading-for-agent
```

### 2. 환경 변수 설정

```bash
cp .env.example .env
# .env 파일에 값 입력 (아래 환경 변수 섹션 참고)
```

### 3. Firebase 서비스 계정 준비

Firebase 콘솔 → 프로젝트 설정 → 서비스 계정 → 키 생성 후 `firebase-credentials.json`으로 저장.

### 4. 백엔드 실행

```bash
cd backend
go mod download
go run cmd/server/main.go
```

### 5. 프론트엔드 실행 (개발)

```bash
cd frontend
npm install
npm run dev
```

`http://localhost:5173` 에서 대시보드 접속.

---

## 환경 변수

| 변수 | 필수 | 설명 |
|------|:----:|------|
| `KIS_APP_KEY` | ✅ | KIS OpenAPI 앱 키 |
| `KIS_APP_SECRET` | ✅ | KIS OpenAPI 앱 시크릿 |
| `KIS_ACCOUNT_NO` | ✅ | 계좌번호 앞 8자리 |
| `KIS_ACCOUNT_TYPE` | ✅ | `01` (종합계좌) / `22` (선물옵션) |
| `KIS_BASE_URL` | ✅ | 실거래: `https://openapi.koreainvestment.com:9443`<br>모의: `https://openapivts.koreainvestment.com:29443` |
| `KIS_HTS_ID` | | 실시간 체결통보(H0STCNI0) 구독 시 필요 |
| `FIREBASE_PROJECT_ID` | ✅ | GCP 프로젝트 ID |
| `FIREBASE_CREDENTIALS_JSON` | ✅ | 서비스 계정 JSON 파일 경로 |
| `SERVER_PORT` | ✅ | 서버 포트 (기본 `8080`) |
| `FRONTEND_ORIGIN` | ✅ | CORS 허용 도메인 (Firebase Hosting URL) |

> **보안 주의:** API 키, 계좌번호 등 민감 정보는 절대 하드코딩하지 않습니다. 모든 설정은 `.env`로만 관리합니다.

---

## 웹 대시보드

| 페이지 | 기능 |
|--------|------|
| **Dashboard** | 총 자산·가용 현금·당일 손익·승률, 모니터링 포지션, 봇 상태, 최근 스캔 로그 |
| **Positions** | 보유 종목별 현재가·목표가·손익 실시간 현황 |
| **Orders** | 주문 내역 (상태·체결가·수량) |
| **Trade Reports** | 개별 매매 분석 |
| **Daily Reports** | 일별 거래 요약 및 손익 집계 |
| **Settings** | 매매 시간·최대 포지션·하드 필터·지표 가중치·매도 조건 등 전체 파라미터 설정 |
| **Logs** | KIS API 오류 로그, 서비스 로그, 스캔 로그 |

---

## 배포

### 백엔드 (NCP VM)

main 브랜치 push 시 GitHub Actions가 자동으로:

1. Go 빌드 → GCS 업로드
2. SSH로 VM 접속 → 바이너리 교체 → systemd 서비스 재시작

### 프론트엔드 (Firebase Hosting)

```bash
cd frontend && npm run build
firebase deploy --only hosting
```

또는 GitHub Actions에서 "Deploy Frontend" 워크플로우 수동 실행.

---

## CI/CD

| 워크플로우 | 트리거 | 동작 |
|-----------|--------|------|
| `ci.yml` | main push / PR / 수동 | 백엔드 포맷·빌드·테스트, 프론트엔드 린트·빌드, main push 시 자동 배포 |
| `deploy-backend.yml` | 수동 | 백엔드 단독 빌드 및 배포 |
| `deploy-frontend.yml` | 수동 | 프론트엔드 단독 빌드 및 배포 |

> 기능 변경 없는 커밋(문서 업데이트 등)은 메시지 끝에 `[skip actions]`를 추가하면 CI를 건너뜁니다.

---

## 디렉토리 구조

```
micro-trading-for-agent/
├── backend/
│   ├── cmd/server/          # 서버 진입점, 시장 스케줄러
│   └── internal/
│       ├── api/             # HTTP 핸들러 & 라우터
│       ├── config/          # 환경 변수 로딩
│       ├── database/        # Firestore 클라이언트
│       ├── kis/             # KIS API 클라이언트 (REST + WebSocket)
│       ├── logger/          # 구조화 JSON 로거
│       ├── models/          # 도메인 모델
│       ├── monitor/         # 포지션 모니터 (목표가·손절가·트레일링)
│       ├── ops/             # KIS API 래퍼 (잔고·주문·순위·차트)
│       ├── report/          # 일별 거래 리포트 생성
│       ├── scorer/          # 하드 필터 + 복합 스코어링 (6지표)
│       ├── stockmaster/     # 종목 마스터(MST) 다운로드 및 저장
│       └── trader/          # 자동매매 엔진 (스캔·필터·스코어·주문)
├── frontend/
│   └── src/
│       ├── pages/           # Dashboard, Positions, Orders, Settings, Logs 등
│       ├── hooks/           # useApi (REST 폴링)
│       ├── components/      # 공유 UI 컴포넌트
│       ├── lib/             # Firebase 초기화
│       └── utils/           # API URL 헬퍼
├── docs/
│   └── kis-api/             # KIS API 명세서 (TR_ID 기준, 186개 API)
├── deploy/
│   └── micro-trading.service  # systemd 서비스 파일
├── .github/workflows/       # CI/CD 파이프라인
├── .env.example             # 환경 변수 템플릿
├── firebase.json            # Firebase Hosting 설정
└── firestore.rules          # Firestore 보안 규칙
```
