# Database Schema

> Engine: SQLite (WAL mode, foreign keys enabled)
> Last updated: 2026-03-15 (rev 10 — orders에 sell_reason 컬럼 추가)

---

## Table: `settings`

**Purpose:** 애플리케이션 설정 키-값 저장소. 서버 기동 시 미존재 키는 `INSERT OR IGNORE`로 기본값 자동 삽입.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `key` | TEXT | PRIMARY KEY | 설정 식별자 |
| `value` | TEXT | NOT NULL, DEFAULT '' | 설정값 |
| `updated_at` | DATETIME | NOT NULL, DEFAULT `datetime('now')` | 마지막 업데이트 시각 |

**Known keys:**

| Key | Default | Description |
|-----|---------|-------------|
| `kis_credentials_hash` | — | SHA-256(`KIS_APP_KEY:KIS_APP_SECRET`); 자격증명 변경 감지·토큰 무효화 |
| `trading_enabled` | `"true"` | `"false"` 시 주문 API 차단 |
| `ranking_excl_cls` | `"1111111111"` | 순위 조회 제외 종목 10자리 플래그 (FID_TRGT_EXLS_CLS_CODE) |
| `take_profit_pct` | `"3.0"` | 익절 기준 % |
| `stop_loss_pct` | `"2.0"` | 손절 기준 % |
| `ranking_types` | `["volume","strength","exec_count","disparity"]` | 순위 유형 우선순위 (JSON 배열) |
| `ranking_price_min` | `"5000"` | 순위 조회 최소 주가 (원) |
| `ranking_price_max` | `"100000"` | 순위 조회 최대 주가 (원) |
| `max_positions` | `"1"` | 동시 보유 최대 종목 수 |
| `order_amount_pct` | `"95"` | 주문가능금액 대비 실제 주문 비율 (%) |
| `sell_conditions` | `["target_pct","stop_pct"]` | 매도 조건 우선순위 배열 (JSON). 가능한 값: `target_pct`, `stop_pct`, `rsi_overbought`, `macd_bearish`, `stagnation` |
| `indicator_check_interval_min` | `"5"` | 지표 확인 주기 (분) |
| `indicator_rsi_sell_threshold` | `"70"` | RSI 과매수 기준값 (이상이면 매도) |
| `indicator_macd_bearish_sell` | `"false"` | MACD 데드크로스 시 매도 여부 |
| `claude_model` | `"claude-sonnet-4-6"` | 종목 선정·리포트에 사용할 Claude 모델 ID |
| `ranking_volume_min_incrrate` | `"0"` | 거래량 순위 필터: 전일대비 거래량 증가율 최솟값 (%, 0=필터없음) |
| `ranking_strength_min` | `"100"` | 체결강도 순위 필터: 최소 체결강도 (%, 0=필터없음, 100=매수우세 이상) |
| `ranking_execcount_net_buy_only` | `"true"` | 대량체결 순위 필터: 순매수체결량 > 0 종목만 허용 |
| `ranking_disparity_d20_min` | `"0"` | 이격도 순위 필터: 20일 이격도 최솟값 (0=필터없음) |
| `ranking_disparity_d20_max` | `"0"` | 이격도 순위 필터: 20일 이격도 최댓값 (0=필터없음) |
| `ranking_top_n` | `"20"` | 각 순위 타입별 필터 통과 후 상위 N개만 AND 교집합 대상으로 사용 (0=전체) |
| `trading_start_time` | `"09:15"` | 자율 거래 엔진 시작 시간 (`HH:MM` 형식, KST) |
| `trading_end_time` | `"15:15"` | 자율 거래 엔진 종료 시간 및 전량 청산 트리거 (`HH:MM` 형식, KST) |
| `stagnation_threshold_pct` | `"1.0"` | 횡보 감지 기준 변동폭 (%). 진입가 대비 ±이 값 이내이면 횡보로 판단 |
| `stagnation_duration_min` | `"30"` | 횡보 지속 기준 시간 (분). 이 시간 이상 지속 시 `stagnation` 매도 조건 트리거 |

---

## Table: `tokens`

**Purpose:** KIS OAuth 액세스 토큰 영속화. 서버 재시작 시에도 토큰 유지. 가장 최근 토큰(highest `id`)만 사용.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT | 대리 키 |
| `access_token` | TEXT | NOT NULL | KIS Bearer 토큰 문자열 |
| `issued_at` | DATETIME | NOT NULL, DEFAULT `datetime('now')` | 토큰 발급 시각 |
| `expires_at` | DATETIME | NOT NULL | 만료 시각 (발급 후 24시간; 20시간 기준 선제 갱신) |

---

## Table: `orders`

**Purpose:** 모든 주문의 완전한 감사 로그. 자율 트레이딩 엔진 주문(AGENT)과 KIS 앱 수동 거래(MANUAL) 모두 포함.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT | 대리 키 |
| `stock_code` | TEXT | NOT NULL | KIS 종목코드 (e.g., `005930`) |
| `stock_name` | TEXT | NOT NULL, DEFAULT '' | 종목명 (e.g., `삼성전자`) |
| `order_type` | TEXT | NOT NULL, CHECK IN ('BUY','SELL') | 매매 방향 |
| `qty` | INTEGER | NOT NULL, CHECK > 0 | 주문 수량 |
| `price` | REAL | NOT NULL, CHECK >= 0 | 주문 단가; 0 = 시장가 |
| `filled_price` | REAL | NOT NULL, DEFAULT 0 | 평균 체결가; 엔진이 ExecCh 체결 확인 후 업데이트 |
| `status` | TEXT | NOT NULL, DEFAULT 'PENDING', CHECK IN ('PENDING','FILLED','PARTIALLY_FILLED','CANCELLED','FAILED') | 주문 상태 |
| `kis_order_id` | TEXT | NOT NULL, DEFAULT '' | KIS 주문번호 (`odno`) |
| `source` | TEXT | NOT NULL, DEFAULT 'AGENT' | `AGENT`=자율 트레이딩 엔진 / `MANUAL`=HTS/MTS 수동 거래 |
| `target_pct` | REAL | NOT NULL, DEFAULT 0 | 목표 수익률 (%); 0이면 모니터링 미등록 |
| `stop_pct` | REAL | NOT NULL, DEFAULT 0 | 손절 비율 (%); 0이면 모니터링 미등록 |
| `sell_reason` | TEXT | NOT NULL, DEFAULT '' | 매도 사유 (자동 매도 시만 값 있음; 예: `"목표가 도달"`, `"손절가 도달"`, `"RSI 78.50 >= threshold 70.00"`, `"일일 자동 청산"`) |
| `created_at` | DATETIME | NOT NULL, DEFAULT `datetime('now')` | 주문 시각 |

**정렬 기준:** `created_at DESC, id DESC`

---

## Table: `balances`

**Purpose:** 계좌 잔고 시점 스냅샷. `GetAccountBalance()` 호출 시 자동 삽입.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT | 대리 키 |
| `total_eval` | REAL | NOT NULL, DEFAULT 0 | 총 평가금액 (KRW) |
| `available_amount` | REAL | NOT NULL, DEFAULT 0 | 출금가능금액 (`dnca_tot_amt`, KRW) |
| `recorded_at` | DATETIME | NOT NULL, DEFAULT `datetime('now')` | 스냅샷 시각 |

---

## Table: `kis_api_logs`

**Purpose:** 모든 KIS API 오류 응답 기록. CLAUDE.md 기준으로 `error_code`, `timestamp`, `raw_response` 3개 필드는 필수.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT | 대리 키 |
| `endpoint` | TEXT | NOT NULL | API 엔드포인트 경로 |
| `error_code` | TEXT | NOT NULL, DEFAULT '' | KIS `msg_cd` 또는 HTTP 상태 코드 |
| `error_message` | TEXT | NOT NULL, DEFAULT '' | KIS `msg1` 오류 메시지 |
| `raw_response` | TEXT | NOT NULL, DEFAULT '' | KIS API 전체 원본 JSON 응답 |
| `timestamp` | DATETIME | NOT NULL, DEFAULT `datetime('now')` | 오류 발생 시각 |

**보존 정책:** `GET /api/logs/kis` 호출 시 2일 이상 된 로그 자동 삭제.

---

## Table: `monitored_positions`

**Purpose:** 실시간 모니터링 중인 매수 포지션. 서버 재시작 시 이 테이블에서 포지션 복구.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT | 대리 키 |
| `stock_code` | TEXT | NOT NULL, UNIQUE | KIS 종목코드 (종목당 1개 포지션) |
| `stock_name` | TEXT | NOT NULL, DEFAULT '' | 종목명 |
| `filled_price` | REAL | NOT NULL, DEFAULT 0 | 체결가 (목표가/손절가 계산 기준) |
| `target_price` | REAL | NOT NULL, DEFAULT 0 | 목표가 (원). `filled_price × (1 + take_profit_pct/100)` |
| `stop_price` | REAL | NOT NULL, DEFAULT 0 | 손절가 (원). `filled_price × (1 - stop_loss_pct/100)` |
| `order_id` | INTEGER | NOT NULL, DEFAULT 0 | 연결된 `orders.id` |
| `created_at` | DATETIME | NOT NULL, DEFAULT `datetime('now')` | 모니터링 등록 시각 |

> `MonitoredEntry.SoldCh` (인메모리 전용) — DB에는 저장되지 않음. 엔진이 살아있을 때만 유효.

**생애주기:**
1. 트레이딩 엔진: 체결 확인 후 `Monitor.Register()` → INSERT
2. `HandlePrice()`: 목표가/손절가 도달 → `executeSell()` → `SoldCh <- code` → `Remove()` → DELETE
3. `StartIndicatorChecker()`: RSI/MACD/횡보 감지 조건 충족 → `executeSell()` → `SoldCh <- code` → `Remove()` → DELETE
4. 15:15 `LiquidateAll()` → DELETE
5. `DELETE /api/monitor/positions/:code` → DELETE

---

## Table: `reports`

**Purpose:** Claude AI가 생성한 일일 트레이딩 리포트 저장. 15:20에 자동 생성, UI에서 날짜별 조회 가능.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT | 대리 키 |
| `report_date` | TEXT | NOT NULL, UNIQUE | 리포트 날짜 (`YYYY-MM-DD` 형식) |
| `content` | TEXT | NOT NULL, DEFAULT '' | 한국어 마크다운 리포트 내용 |
| `created_at` | DATETIME | NOT NULL, DEFAULT `datetime('now')` | 생성/갱신 시각 |

**생성 방식:** 매일 15:20 KST에 `Engine.GenerateDailyReport()` → `Claude.GenerateReport()` → `db.SaveReport()`. 같은 날 재생성 시 ON CONFLICT DO UPDATE (덮어쓰기).

---

## Table: `trader_selection_logs`

**Purpose:** 트레이딩 엔진이 Claude API를 호출해 종목을 선정할 때마다 입력(후보 목록)과 출력(LLM 순위)을 영속화. UI `/selection-logs` 페이지에서 조회.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT | 대리 키 |
| `timestamp` | DATETIME | NOT NULL, DEFAULT `datetime('now')` | 선정 시도 시각 |
| `sent_count` | INTEGER | NOT NULL, DEFAULT 0 | LLM에 전달한 후보 종목 수 |
| `candidates` | TEXT | NOT NULL, DEFAULT '' | LLM에 전달한 `[]RankItem` JSON 배열 (기술지표 포함) |
| `llm_result` | TEXT | NOT NULL, DEFAULT '' | Claude가 반환한 `[]StockCandidate` JSON 배열 (우선순위 순) |
| `selected_code` | TEXT | NOT NULL, DEFAULT '' | 최종 체결된 종목코드; 미체결 시 빈 문자열 |
| `selected_reason` | TEXT | NOT NULL, DEFAULT '' | 체결 종목에 대한 Claude의 선정 이유 (한국어 1문장) |
| `fail_reason` | TEXT | NOT NULL, DEFAULT '' | 선정 실패 사유 (LLM 오류, 주문 전체 실패 등); 정상 선정 시 빈 문자열 |

**보존 정책:** `GET /api/logs/selection` 호출 시 30일 이상 된 로그 자동 삭제.

**생애주기:**
1. `selectAndBuy()` — Claude 호출 **전** INSERT (sent_count, candidates만 저장)
2. Claude 오류 시 — `UPDATE ... SET fail_reason='LLM 오류: ...' WHERE id=?` 후 반환
3. Claude 응답 정상 — `UPDATE ... SET llm_result=? WHERE id=?`
4. 체결 성공 시 — `UPDATE ... SET selected_code=?, selected_reason=? WHERE id=?`
5. 모든 후보 주문 실패 시 — `UPDATE ... SET fail_reason='모든 후보 N개 주문 실패' WHERE id=?`

---

---

## Table: `trader_ranking_logs`

**Purpose:** 트레이딩 엔진의 `getRankings()` 호출 기록. 언제/어떤 조건으로 순위를 조회했고 타입별/교집합 결과가 몇 개였는지 추적. UI `/ranking-logs` 페이지에서 조회.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT | 대리 키 |
| `timestamp` | DATETIME | NOT NULL, DEFAULT `datetime('now')` | 조회 시각 |
| `ranking_types` | TEXT | NOT NULL, DEFAULT '' | 활성화된 순위 타입 JSON 배열 (e.g., `["volume","strength"]`) |
| `price_min` | TEXT | NOT NULL, DEFAULT '' | 최소 주가 필터 (원) |
| `price_max` | TEXT | NOT NULL, DEFAULT '' | 최대 주가 필터 (원) |
| `volume_count` | INTEGER | NOT NULL, DEFAULT -1 | 거래량 순위 필터 통과 종목 수; -1 = 타입 미사용 |
| `strength_count` | INTEGER | NOT NULL, DEFAULT -1 | 체결강도 순위 필터 통과 종목 수; -1 = 타입 미사용 |
| `exec_count_count` | INTEGER | NOT NULL, DEFAULT -1 | 대량체결 순위 필터 통과 종목 수; -1 = 타입 미사용 |
| `disparity_count` | INTEGER | NOT NULL, DEFAULT -1 | 이격도 순위 필터 통과 종목 수; -1 = 타입 미사용 |
| `intersection_count` | INTEGER | NOT NULL, DEFAULT 0 | AND 교집합 결과 종목 수 (Claude에 전달되는 후보 수) |
| `error_message` | TEXT | NOT NULL, DEFAULT '' | 오류 발생 시 메시지; 정상이면 빈 문자열 |

**보존 정책:** `GET /api/logs/ranking` 호출 시 30일 이상 된 로그 자동 삭제.

**생애주기:**
1. `getRankings()` — 각 타입 API 호출 및 AND 교집합 계산 완료 후 INSERT
2. `no ranking types configured` 오류 시에도 error_message 포함 INSERT

---

## 마이그레이션 전략

- 새 테이블: `stmts` 슬라이스에 `CREATE TABLE IF NOT EXISTS` 추가
- 새 컬럼: `alterStmts` 슬라이스에 `ALTER TABLE ... ADD COLUMN` 추가 (중복 오류 무시)
- 신규 설정 기본값: `defaultSettings` 슬라이스에 추가 → `INSERT OR IGNORE` (기존 사용자 값 보존)
