# 2026-03-19 프론트엔드 개선 작업 계획

## Goal
2026-03-19 작업 요청사항 처리 — 버그 수정 3건 + 하드필터 로그 추가 + 종목로그 통합 + Stitch 기반 전체 화면 디자인 개선

---

## 요구사항

### 버그 수정
1. **대시보드 보유종목 수익률 0.0% 고정** — `/api/positions` 응답의 `evlu_erng_rt` 값 확인 및 수정
2. **주문내역 매도사유 사라짐** — `sell_reason` 필드가 API에서 비어오는지 또는 UI 렌더링 문제인지 확인
3. **설정 저장 버튼 위치** — 버튼이 화면 맨 아래 있어 UX 불편 → 상단 헤더 고정 또는 sticky footer로 이동

### 신규 기능: 하드필터 로그
- 순위조회(7개) → 하드필터 통과(3개) → LLM선정 흐름에서 **"어떤 종목이 어떤 필터 조건에 걸렸는지"** 기록 필요
- `trader_ranking_logs` 테이블에 `filtered_stocks` 컬럼 추가 (JSON: `[{code, name, filter_reason}]`)
- `trader_selection_logs`에 `ranking_log_id` 추가 (순위로그 ↔ 선정로그 연결)
- 엔진에서 하드필터 후 제거 종목 + 이유 기록

### 종목로그 통합 메뉴 (RankingLogs + SelectionLogs → StockLogs)
- 현재: `순위 조회 로그` / `선정 로그` 두 개의 별도 페이지
- 변경: `종목로그` 하나로 통합. 순위조회 항목 클릭 시 → 하드필터 결과 + LLM선정 결과를 하나의 화면에서 확인
- 화면 구성: 순위조회 목록 (최신순) → 선택하면 오른쪽/하단에 하드필터/LLM선정 상세 표시

### Stitch 디자인 적용 (전체 화면)
- Google Stitch "Dashboard 개선안" 프로젝트 HTML 코드 기반으로 각 화면 리뉴얼
- 디자인 시스템: "Digital Obsidian" (다크 테마, `#131316` 배경, 오렌지 `#F97316` 액센트)
- 화면 개선.md의 기능 스펙과 Stitch 디자인 결합
- 대상 화면: 대시보드, 모니터링, 주문내역, 에러로그, 종목로그(통합), 설정

---

## 영향 파일

### 백엔드
| 파일 | 변경 내용 |
|------|---------|
| `backend/internal/database/db.go` | `filtered_stocks`, `ranking_log_id` 컬럼 ALTER + InsertRankingLog/InsertSelectionLog 수정 |
| `backend/internal/models/models.go` | `TraderRankingLog.FilteredStocks` 필드, `TraderSelectionLog.RankingLogID` 필드 추가 |
| `backend/internal/trader/engine.go` | 하드필터 후 제거 종목 및 이유 기록 로직 추가 |
| `backend/internal/api/handlers.go` | `GetRankingLogs` 응답에 `filtered_stocks` 포함 확인, `/api/positions` 수익률 확인 |

### 프론트엔드
| 파일 | 변경 내용 |
|------|---------|
| `frontend/src/pages/Dashboard.jsx` | 수익률 버그 수정 + Stitch 디자인 적용 |
| `frontend/src/pages/Orders.jsx` | 매도사유 버그 확인 + 필터/취소버튼/무한스크롤 + Stitch 디자인 |
| `frontend/src/pages/Monitor.jsx` | 새로고침 주기 선택박스 + Stitch 디자인 |
| `frontend/src/pages/ErrorLogs.jsx` | 탭 구조(트레이더/모니터/시스템/KIS API) + Stitch 디자인 |
| `frontend/src/pages/Settings.jsx` | 저장 버튼 sticky 이동 + Stitch 디자인 |
| `frontend/src/pages/StockLogs.jsx` | **신규** — 순위/하드필터/LLM선정 통합 뷰 |
| `frontend/src/pages/RankingLogs.jsx` | 삭제 (StockLogs로 통합) |
| `frontend/src/pages/SelectionLogs.jsx` | 삭제 (StockLogs로 통합) |
| `frontend/src/App.jsx` | 라우트 변경 (`/stock-logs`), 네비 수정 |

---

## 구현 단계

### Phase 1: 버그 수정 (백엔드 조사 → 프론트 수정)
1. `/api/positions` 엔드포인트 응답 로그 확인 — `evlu_erng_rt` 값이 실제로 0으로 오는지 확인
2. `/api/orders` 응답에서 `sell_reason` 필드 확인
3. `Settings.jsx` 저장 버튼 → sticky footer 또는 상단 헤더 영역으로 이동

> **예상 소요:** 소규모. 백엔드 코드 수정 가능성 있음.

### Phase 2: 하드필터 로그 (DB + 백엔드)
1. `db.go`: `ALTER TABLE trader_ranking_logs ADD COLUMN filtered_stocks TEXT NOT NULL DEFAULT '[]'`
2. `db.go`: `ALTER TABLE trader_selection_logs ADD COLUMN ranking_log_id INTEGER NOT NULL DEFAULT 0`
3. `models.go`: 두 struct에 필드 추가
4. `engine.go`: 하드필터 실행 후 제거된 종목 목록과 이유(`filter_reason`) 수집, `InsertRankingLog` 호출 시 전달
5. `engine.go`: `InsertSelectionLog` 호출 시 연관 `ranking_log_id` 전달

> **filter_reason 종류:** `min_trade_amount`, `rsi_overbought`, `disparity_5m`, `high_pct`, `open_pct`

> **예상 소요:** 백엔드 중간 규모.

### Phase 3: 종목로그 통합 (프론트엔드)
1. `StockLogs.jsx` 신규 생성
   - 상단: 순위조회 로그 목록 (최신순, 마켓/시간/종목수 요약)
   - 선택 시 하단에 3단계 패널 표시:
     - **순위조회**: 시장, 조건, 종목목록 (result_stocks)
     - **하드필터**: 통과 종목 + 필터링된 종목 (filtered_stocks + 이유)
     - **LLM선정**: candidates, 선정결과, LLM 응답 코멘트, 상세 종목정보
2. `App.jsx`: `/stock-logs` 라우트 추가, 기존 `/selection-logs` `/ranking-logs` 제거
3. `RankingLogs.jsx`, `SelectionLogs.jsx` 파일 삭제

### Phase 4: Stitch 디자인 적용 (프론트엔드)
Stitch 프로젝트 `10699970814357457810`의 HTML 코드를 화면별로 가져와 React 컴포넌트로 변환.

| 화면 | Stitch 스크린 ID | 적용 대상 |
|------|-----------------|---------|
| 대시보드 | `4e13b5fe...` (최종 정리) | Dashboard.jsx |
| 모니터링 | `3d7307fd...` (최종 정리) | Monitor.jsx |
| 주문내역 | `fb2c1926...` (최종 정리) | Orders.jsx |
| 에러로그 | `b7d7289...` (최종 정리) | ErrorLogs.jsx |
| 종목로그 | `dd2c3be...` (최종 정리) | StockLogs.jsx |
| 설정 | `b5fc51be...` (최종 정리) | Settings.jsx |

**디자인 토큰 (CSS 변수 또는 Tailwind 커스텀):**
- 배경: `#131316`, Surface: `#1F1F22`, High: `#2A2A2D`
- Primary(CTA): `#F97316`, Secondary(수익): `#4AE176`, Tertiary(손실): `#FF6A64`
- 폰트: Inter (헤드라인), Manrope (데이터/숫자)

---

## 검증
- `go build ./...` 성공
- `npm run build` 성공
- 대시보드 수익률 실제 값 표시 확인
- 주문내역 매도사유 표시 확인
- 종목로그 3단계 연결 확인 (순위 선택 → 하드필터 → LLM선정)
- 설정 저장 버튼 접근성 확인

---

## 작업 순서 권장
**Phase 1 → Phase 2 → Phase 3 → Phase 4** 순서로 진행.
Phase 1, 2는 독립적이므로 병렬 진행 가능.
Phase 4는 분량이 가장 많으므로 각 화면별로 커밋 분리.
