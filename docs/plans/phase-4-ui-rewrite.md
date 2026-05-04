# Phase 4: UI 전면 재작성 계획

## Goal

Claude Design에서 완성된 `trading-dashboard.html` 디자인을 기준으로 React 앱을 픽셀 단위로 재구현한다.
v1 레거시 페이지를 제거하고 7개 핵심 화면으로 정리하며, 데스크톱 사이드바 + 모바일 하단 탭바 반응형 레이아웃을 적용한다.

---

## 핵심 결정 사항

| 항목 | 결정 |
|------|------|
| CSS 전략 | Tailwind **완전 제거** → 디자인의 CSS 변수 시스템으로 교체 |
| 폰트 | Inter/Manrope 제거 → **Pretendard + IBM Plex Mono** |
| 아이콘 | Material Symbols 제거 → 디자인의 텍스트 심볼(◈ ⬡ ≡ ◎ ▤) 사용 |
| 테마 | ThemeContext 제거 → body에 `.light-mode` 클래스 직접 토글 |
| 화면 수 | 10개 → **7개** (v1 레거시 5개 제거) |
| 라우팅 | React Router 유지, 경로 간소화 |

---

## 화면 구성 (Before → After)

| Before | After | 비고 |
|--------|-------|------|
| Dashboard | Dashboard (`/`) | 재작성 |
| Monitor | Positions (`/positions`) | 파일명 변경 + 재작성 |
| Orders | Orders (`/orders`) | 재작성 |
| TradeReports | TradeReports (`/reports/trades`) | 재작성 |
| DailyReports | DailyReports (`/reports/daily`) | 재작성 |
| Settings | Settings (`/settings`) | 재작성 (4탭) |
| ErrorLogs + KISLogs | Logs (`/logs`) | 통합 재작성 |
| OptimizationReports | **삭제** | v1 AI 기능 |
| SelectionLogs | **삭제** | v1 |
| RankingLogs | **삭제** | v1 |
| StockLogs | **삭제** | v1 |
| StockList | **삭제** | v1 (필요시 나중에 추가) |

---

## 영향 파일 목록

### 수정
- `frontend/index.html` — 폰트 교체, Material Symbols 제거
- `frontend/src/index.css` — 디자인 CSS 시스템으로 전면 교체
- `frontend/src/App.jsx` — 새 사이드바 + 모바일 레이아웃 + 라우팅
- `frontend/package.json` + `tailwind.config.js` — Tailwind 의존성 제거

### 신규 생성
- `frontend/src/components/shared.jsx` — Modal, Toggle, Badge, BotStateBadge, PriceProgressBar
- `frontend/src/pages/Positions.jsx` — 새 포지션 모니터 화면

### 전면 재작성
- `frontend/src/pages/Dashboard.jsx`
- `frontend/src/pages/Orders.jsx`
- `frontend/src/pages/TradeReports.jsx`
- `frontend/src/pages/DailyReports.jsx`
- `frontend/src/pages/Settings.jsx`
- `frontend/src/pages/Logs.jsx`

### 삭제
- `frontend/src/pages/Monitor.jsx`
- `frontend/src/pages/ErrorLogs.jsx`
- `frontend/src/pages/KISLogs.jsx`
- `frontend/src/pages/OptimizationReports.jsx`
- `frontend/src/pages/SelectionLogs.jsx`
- `frontend/src/pages/RankingLogs.jsx`
- `frontend/src/pages/StockLogs.jsx`
- `frontend/src/pages/StockList.jsx`
- `frontend/src/contexts/ThemeContext.jsx`
- `frontend/src/components/Card.jsx`
- `frontend/src/components/StatusBadge.jsx`
- `frontend/tailwind.config.js`
- `frontend/postcss.config.js`

---

## API 연결 계획

| 화면 | 사용 엔드포인트 |
|------|----------------|
| Dashboard | `GET /api/balance`, `GET /api/positions`, `GET /api/trader/status`, `GET /api/scan-logs?limit=5` |
| Positions | `GET /api/monitor/positions`, `DELETE /api/monitor/positions/:code`, `POST /api/monitor/positions/:code/sell`, `POST /api/monitor/liquidate-all` |
| Orders | `GET /api/orders`, `POST /api/orders?sync=true&days=N` |
| TradeReports | `GET /api/reports/trades` |
| DailyReports | `GET /api/reports/daily` |
| Settings | `GET /api/settings`, `PATCH /api/settings` |
| Logs | `GET /api/logs/kis`, `DELETE /api/logs/kis/:id`, `GET /api/logs/service` |

---

## 구현 단계

### Phase A — CSS·레이아웃 기반 (선행 필수)
1. `index.html`: 폰트 교체 (Pretendard + IBM Plex Mono), Material Symbols 제거
2. `index.css`: Tailwind 제거 → 디자인 CSS 변수 + 커스텀 클래스 전체 이식
3. `package.json`: tailwindcss, postcss, autoprefixer 제거; `tailwind.config.js`, `postcss.config.js` 삭제
4. `src/components/shared.jsx`: Modal / Toggle / Badge / BotStateBadge / PriceProgressBar / ProgressBar 공유 컴포넌트
5. `src/App.jsx`: 새 사이드바(데스크톱) + 하단 탭바(모바일) + 더보기 드로어 + 라우팅

### Phase B — 핵심 화면
6. `Dashboard.jsx`: 5 stat 카드 + 봇 상태 + 보유 포지션 테이블 + 스캔 로그 확장 카드
7. `Positions.jsx`: 카드 그리드, PriceProgressBar, 강제매도/전체청산 확인 모달
8. `Orders.jsx`: 날짜/타입/상태 필터 칩, KIS 동기화 스피너, 페이지네이션

### Phase C — 리포트·설정
9. `TradeReports.jsx`: 날짜 그룹, 거래 카드, 지표 펼침
10. `DailyReports.jsx`: 요약 stat 카드, SVG 바 차트, 일별 행 확장
11. `Settings.jsx`: 4탭(거래조건/하드필터/점수시스템/스케줄), DonutChart SVG, 저장 플래시

### Phase D — 로그·정리
12. `Logs.jsx`: KIS API 탭(체크박스 선택 삭제) + 서비스 로그 탭(터미널 스트림)
13. v1 파일 삭제 (8개 페이지 + ThemeContext + Card + StatusBadge)
14. `npm run build` 빌드 검증

---

## 주요 디자인 스펙

```
배경:   #0F172A (다크) / #F1F5F9 (라이트)
카드:   #1E293B
테두리: #2D3F55
액센트: #EA6C10
상승:   #EF4444 (한국식 빨강)
하락:   #2563EB (한국식 파랑)
폰트-숫자: IBM Plex Mono
폰트-UI:   Pretendard
```

---

## 검증

- `npm run build` 에러 없음
- 데스크톱(1280px) + 모바일(390px) 레이아웃 확인
- 7개 화면 모두 라우팅 정상
- 다크/라이트 토글 정상
- 모바일 드로어 정상 동작
