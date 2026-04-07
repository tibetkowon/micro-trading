# 계획: 거래 학습 리포트 기능

## Goal

AI 트레이더의 성과를 분석하고 학습할 수 있도록 **3단계 리포트 파이프라인**을 구축한다.

1. **매수 시**: 종목 선정에 사용한 기술 지표 데이터, Claude 선정 근거, 매수 금액을 `trade_reports` 테이블에 저장
2. **매도 후**: 매도 시점의 기술 지표를 재수집하고 매도 사유·금액을 매수 레코드와 매칭하여 업데이트
3. **하루 종료 후**: 당일 거래 히스토리를 집계하여 `daily_reports` 테이블에 요약 저장 + UI에서 조회 가능

---

## Requirements

### 기능 요건
- 매수 체결 후 자동으로 선정 데이터(RankItem 스냅샷, Claude 근거) + 매수가/수량 저장
- 매도 체결 후 자동으로 매도 시점 지표 재수집 + 매도 사유/금액 매칭 저장, 손익 계산
- 15:20 스케줄에서 당일 `trade_reports` 집계 → `daily_reports` INSERT
- `/api/reports/trades` — 거래별 리포트 목록 조회 (날짜·종목 필터)
- `/api/reports/daily` — 일별 리포트 목록 조회
- 프론트엔드: 거래별 리포트 테이블 + 일별 요약 카드

### 비기능 요건
- 리포트 저장 실패가 거래 로직에 영향 주지 않도록 비동기 goroutine으로 처리
- 기존 `orders` / `trader_selection_logs` 테이블 스키마 변경 없음

---

## New DB Tables

### `trade_reports`

| 컬럼 | 타입 | 설명 |
|------|------|------|
| id | INTEGER PK | 자동증가 |
| date | TEXT | 거래 날짜 (YYYY-MM-DD) |
| stock_code | TEXT | 종목 코드 |
| stock_name | TEXT | 종목명 |
| buy_order_id | INTEGER | orders.id (매수 주문) |
| sell_order_id | INTEGER | orders.id (매도 주문, NULL until sold) |
| selection_log_id | INTEGER | trader_selection_logs.id |
| buy_price | REAL | 매수 체결가 |
| buy_qty | INTEGER | 매수 수량 |
| buy_amount | REAL | buy_price × buy_qty |
| buy_reason | TEXT | Claude 선정 근거 |
| buy_indicators | TEXT | JSON: 매수 시 RankItem 스냅샷 (기술 지표 포함) |
| sell_price | REAL | 매도 체결가 (NULL until sold) |
| sell_qty | INTEGER | 매도 수량 |
| sell_amount | REAL | sell_price × sell_qty |
| sell_reason | TEXT | 매도 사유 (monitor sell_reason) |
| sell_indicators | TEXT | JSON: 매도 시 재수집한 기술 지표 |
| profit_amount | REAL | sell_amount - buy_amount |
| profit_pct | REAL | (sell_price - buy_price) / buy_price × 100 |
| created_at | DATETIME | 매수 시각 |
| sold_at | DATETIME | 매도 시각 (NULL until sold) |

### `daily_reports`

| 컬럼 | 타입 | 설명 |
|------|------|------|
| id | INTEGER PK | 자동증가 |
| date | TEXT UNIQUE | 날짜 (YYYY-MM-DD) |
| total_trades | INTEGER | 당일 총 거래 수 |
| winning_trades | INTEGER | 수익 거래 수 |
| losing_trades | INTEGER | 손실 거래 수 |
| total_profit_amount | REAL | 총 손익 합계 (원) |
| avg_profit_pct | REAL | 평균 수익률 (%) |
| best_trade | TEXT | JSON: 최고 수익 거래 요약 |
| worst_trade | TEXT | JSON: 최대 손실 거래 요약 |
| trade_summary | TEXT | JSON: 전체 거래 배열 요약 |
| created_at | DATETIME | 리포트 생성 시각 |

---

## Affected Files

### 신규 파일
- `backend/internal/report/report.go` — DailyReportGenerator (daily_reports 생성 로직)

### 수정 파일
| 파일 | 변경 내용 |
|------|----------|
| `backend/internal/models/models.go` | TradeReport, DailyReport 모델 추가 |
| `backend/internal/database/db.go` | trade_reports / daily_reports 테이블 마이그레이션 + CRUD 메서드 추가 |
| `backend/internal/trader/engine.go` | 매수 체결 후 `InsertTradeReport` 호출 (goroutine) |
| `backend/internal/monitor/monitor.go` | 매도 체결 후 `UpdateTradeReportOnSell` 호출 (goroutine, stock_info 재수집 포함) |
| `backend/internal/api/handlers.go` | GET /api/reports/trades, GET /api/reports/daily 핸들러 추가 |
| `backend/internal/api/router.go` | 신규 라우트 등록 |
| `frontend/src/` | 리포트 페이지 컴포넌트 추가 (TradeReportList, DailyReportList) |
| `docs/db_schema.md` | 신규 테이블 명세 추가 |

---

## Implementation Phases

### Phase 1: DB 스키마 + 모델
- `models.go`에 `TradeReport`, `DailyReport` 구조체 추가
- `db.go` 마이그레이션에 `trade_reports`, `daily_reports` CREATE TABLE 추가
- DB CRUD 메서드: `InsertTradeReport`, `UpdateTradeReportOnSell`, `GetTradeReports`, `InsertDailyReport`, `GetDailyReports`

### Phase 2: 매수 후 리포트 저장 (engine.go)
- `selectAndBuy` 에서 체결 성공 후 — `selectionLogID`, 선택된 `RankItem` 스냅샷, Claude 근거(`chosenReason`), `filledPrice`, `filledQty` 를 이용해 `trade_reports` INSERT
- goroutine으로 비동기 처리 (거래 흐름에 영향 없음)

### Phase 3: 매도 후 리포트 업데이트 (monitor.go)
- `executeSell` 함수에서 매도 성공 후 — `agent.GetStockInfo` 재호출로 매도 시점 지표 수집
- `trade_reports` UPDATE: sell_price, sell_qty, sell_amount, sell_reason, sell_indicators, profit_amount, profit_pct, sold_at
- buy_order_id 기준으로 매칭 (orders 테이블 조회)
- goroutine으로 비동기 처리

### Phase 4: 일별 리포트 생성 (report.go + scheduler)
- `report.GenerateDailyReport(ctx, db, date)` 함수 — 당일 `trade_reports` 조회 후 집계
- 기존 15:20 스케줄러(`main.go` 또는 스케줄 goroutine)에 `GenerateDailyReport` 호출 추가
- `daily_reports` UPSERT (같은 날짜 재생성 시 덮어쓰기)

### Phase 5: API 엔드포인트
- `GET /api/reports/trades?date=YYYY-MM-DD&stock_code=XXXX&page=1&limit=20`
- `GET /api/reports/daily?from=YYYY-MM-DD&to=YYYY-MM-DD`

### Phase 6: 프론트엔드 UI
- 네비게이션에 "리포트" 탭 추가
- `TradeReportPage` — 거래별 리포트 테이블 (날짜/종목 필터, 수익률 색상 강조)
- `DailyReportPage` — 일별 요약 카드 (승률, 총 손익, 최고/최하 거래)

---

## Verification

1. `go build ./...` 빌드 확인
2. DB 마이그레이션 후 trade_reports / daily_reports 테이블 생성 확인
3. 엔진 ForceRun 실행 후 trade_reports 레코드 INSERT 확인
4. monitor 매도 후 trade_reports 레코드 UPDATE 확인 (profit_pct 계산 검증)
5. 15:20 GenerateDailyReport 수동 호출 → daily_reports 레코드 생성 확인
6. `/api/reports/trades`, `/api/reports/daily` 응답 JSON 구조 확인
7. 프론트엔드 리포트 페이지 렌더링 확인
