# Changelog

## 2026-04-27 — Hard Rule 거부 사유 상세 로깅 및 룰별 자동 완화 피드백 루프

- **trader/engine.go**: `evaluateHardRules()` 추가 — Claude 호출 전 후보 종목 전체에 대해 12개 Hard Rejection Rule 위반 여부를 서버 사이드에서 사전 집계
- **trader/engine.go**: LLM 선택 실패 시 `fail_reason`에 상위 위반 룰 상세 포함 (e.g. `hard_strength_min:7/8, hard_rsi_max:3/8`)
- **trader/engine.go**: `ruleHitRecord` 링 버퍼 추가 — 최근 N 사이클(기본 5) Hard Rule 통계 누적
- **trader/engine.go**: `detectBottleneckRule()` — 링 버퍼에서 threshold%(기본 80%) 이상 사이클에서 발동한 룰 감지
- **trader/engine.go**: `relaxSpecificRule()` — bottleneck 룰의 임계값만 선택적 완화 (전체 완화가 아닌 targeted)
- **trader/engine.go**: 선택 성공 시 링 버퍼 리셋
- **database/db.go**: `trader_selection_logs` 테이블에 `hard_rule_stats TEXT` 컬럼 추가
- **database/db.go**: `TradingSettings`에 `HardRuleFeedbackEnabled`, `HardRuleFeedbackWindow`, `HardRuleFeedbackThresholdPct` 필드 추가 (기본값: false / 5 / 80)
- **models/models.go**: `TraderSelectionLog`에 `HardRuleStats` 필드 추가
- **report/optimization.go**: no-trade 분석 summary에 `hard_rule_stats` 누적 집계 포함 — Claude가 구체적 룰 완화 제안 가능
- **trader/claude.go**: `AnalyzeNoTradeDay` 프롬프트에 `hard_rule_stats` 해석 가이드 추가

## 2026-04-21 — KIS TPS 에러 방지 강화 (속도 감소 + 지수 백오프 + 일시 스로틀)

- **kis/ratelimiter.go**: `Throttle(slowRPS, duration)` 메서드 추가 — TPS 에러 발생 시 3 RPS로 3초간 일시 감속 후 자동 복원 (`sync.Mutex` 안전, 중복 호출 시 타이머 연장)
- **kis/client.go**: RPS 7→5로 낮춤 (~200ms 간격); 고정 500ms 재시도 → 지수 백오프(500ms·1s·2s)로 변경; TPS 에러 시 `Throttle()` 호출 추가 (GET·POST 모두)

## 2026-04-21 — 거래량 회복 필터 + 시장 대비 상대강도 필터 추가

- **agent/stock_info.go**: `VolVs3AvgRatio` 필드 추가 — 현재 5분봉 거래량 / 직전 3봉 평균 거래량 (거래량 회복 비율)
- **trader/claude.go**: `RankItem`에 `vol_vs_3avg_ratio`, `relative_strength_vs_market` 필드 추가; `TradingRules`에 `HardVolVs3AvgRatioMin`, `HardRelativeStrengthMin` 추가; Claude 프롬프트 Hard Rule 11·12 조건부 삽입; New Data Fields 가이드 설명 추가; `allowedSettingsKeys`에 두 키 등록
- **trader/engine.go**: rankings 정보 수집 시 `VolVs3AvgRatio`·`RelativeStrengthVsMkt` 할당; TradingRules 빌드 시 새 설정 적용
- **database/db.go**: `TradingSettings`에 두 필드 추가; `GetTradingSettings` 쿼리·파싱·반환값 확장; 기본값 `0` (비활성) INSERT OR IGNORE
- **frontend/Settings.jsx**: Hard Rejection Rules 섹션에 "거래량 회복 비율 최솟값", "시장 대비 상대강도 최솟값" 입력 UI 추가

## 2026-04-13 — Settings UI Hard Rejection 신규 항목 추가 + AI 개선 제안 설정 누락 수정

- **frontend/Settings.jsx**: MACD 베어리시 차단(토글), 고점 경과 시간 상한(분) UI 항목 추가
- **report/optimization.go**: `collectCurrentSettings`에 `hard_macd_bearish_enabled`, `hard_high_formed_mins_max` 등록 → AI 일일 개선 제안에 해당 설정값 포함되도록 수정
- **report/optimization.go**: `settingsKeyLabel`, `settingConstraints`에 두 키 레이블·범위 추가

## 2026-04-11 — 자동 매도 실패 시 포지션 모니터링 유지 버그 수정 + Hard Rejection 룰 2개 추가

- **monitor/monitor.go**: `executeSell` 반환값 변경 — 주문 실패 시 `-1` 반환, 0은 잔고 없음, 양수는 성공. 실패 시 `Remove` 차단하여 실제 잔고 보존
- **monitor/monitor.go**: `HandlePrice`(목표가/손절가/트레일링), `checkIndicators`, `LiquidateAll` — `executeSell` 결과 검사 후 `Remove` 조건부 실행으로 통일
- **database/db.go**: `hard_macd_bearish_enabled`(false), `hard_high_formed_mins_max`(0) 설정 키 추가
- **trader/claude.go**: Hard Rejection Rule 9(MACD bearish), Rule 10(고점 경과 시간 상한) 조건부 추가
- **trader/engine.go**: 신규 TradingRules 필드 settings 매핑 추가

## 2026-04-11 — Claude 종목 선정 데이터 품질 개선 (1~5순위)

- **agent/stock_info.go**: `CandleSnap` 타입 추가; `StockInfo`에 `RecentCandles`(최근 5분봉 시퀀스), `HighFormedMinsAgo`(고점 경과 시간), `VolTrend3`(3봉 거래량 기울기), `VolAtHigh`(고점 봉 거래량) 필드 추가 및 계산 로직 구현
- **kis/client.go**: `OrderBookSnapshot` 타입 추가; `GetOrderBookSnapshot()` — 동일 FHKST01010200 호출로 `NearBidAskRatio`(±2% 근거리 비율), `TopAskWall`(최대 매도 벽 위치), `TopAskWallSize` 추가 반환; `GetBidAskRatio()` 위임으로 하위호환 유지
- **trader/claude.go**: `RankItem`에 신규 7개 필드 추가; `SelectStocks` 프롬프트에 장 시간대(SessionPhase) 주입 및 신규 필드 해석 가이드 추가
- **trader/engine.go**: `StockInfo`→`RankItem` 매핑에 신규 4개 필드 추가; `GetBidAskRatio` → `GetOrderBookSnapshot` 교체 (추가 API 호출 0회)

## 2026-04-09 — 전체 프론트엔드 모바일 최적화

### Orders.jsx
- 모바일용 카드 뷰 추가 (`sm:hidden`): 종목명·유형·상태·가격·수량·시각·매도사유·액션 버튼
- 기존 데스크탑 테이블은 `sm:` 이상에서 그대로 유지

### Dashboard.jsx
- 헤더 오른쪽 컨트롤(새로고침 주기 + 새로고침 버튼)에 `flex-wrap` 추가
- PnL 그래프 헤더 (실현 손익 텍스트 + %/원·기간 버튼)에 `flex-wrap` 추가

### Monitor.jsx
- 새로고침 주기 버튼 그룹 + 새로고침 + 전체 매도 버튼 컨테이너에 `flex-wrap` 추가

### DailyReports.jsx / OptimizationReports.jsx
- 날짜 범위 필터 inner div에 `flex-wrap` 및 `min-w-0` 추가 — 좁은 화면에서 줄 바꿈 처리

## 2026-04-09 — KIS TPS 초과(EGW00201) 자동 재시도 추가

### KIS API 클라이언트 (`kis/client.go`)
- `EGW00201` (초당 거래건수 초과) 응답 수신 시 500ms 대기 후 최대 3회 재시도
- `get()` / `placeOrder()` 모두 적용
- 재시도 중 context 취소 시 즉시 중단

## 2026-04-09 — 눌림목 모멘텀 스코어링 + 단계적 횡보 청산 로직 추가

### 복합 모멘텀 스코어링 (`trader/claude.go`, `trader/engine.go`)
- `RankItem`에 `MomentumScore float64` 필드 추가 — Claude 프롬프트에도 노출
- BidAskRatio 조회 완료 후 0~100점 스코어 계산: `bid_ask(40pt) + 체결강도(40pt) + 거래량감소(20pt)`
- 신규 설정 `momentum_score_min` (기본 0=비활성): 이 값 미만 종목은 Claude 전달 전 제거
- 오늘 기준 비츠로셀 97.6점 / 온코닉테라퓨틱스 49.4점 → threshold=60 설정 시 자동 차별화

### 단계적 횡보 청산 (`monitor/monitor.go`, `cmd/server/main.go`)
- `MonitoredEntry`에 `PartialExitDone bool` 필드 추가
- `Monitor.SetStagnationExitConfig()` 메서드 추가
- `Monitor.executePartialSell()` 메서드 추가 (보유량 절반 시장가 매도)
- `checkIndicators` stagnation case 분기 로직:
  - `stagnation_partial_exit_enabled=false` → 기존 전량 청산 유지 (하위 호환)
  - `bid_ask < stagnation_bid_ask_sell_threshold` → 즉시 전량 청산 ("횡보 중 매도우세 전환")
  - 1차 횡보 + 매수 우세 → 절반 청산 + 횡보 타이머 리셋
  - 2차 횡보 (절반 청산 후 재횡보) → 전량 청산
- 신규 설정 `stagnation_partial_exit_enabled` (기본 false), `stagnation_bid_ask_sell_threshold` (기본 1.0)
- `cmd/server/main.go`에 `mon.SetStagnationExitConfig()` 호출 추가

### DB 설정 추가 (`database/db.go`)
- `momentum_score_min`, `stagnation_partial_exit_enabled`, `stagnation_bid_ask_sell_threshold` 3개 키 추가

## 2026-04-09 — KIS API TPS 제한 10 → 7 하향

- **`kis/client.go`**: `NewRateLimiter(10, 1)` → `NewRateLimiter(7, 1)` — 초당 거래건수 초과 방지

## 2026-04-08 — UI/시스템 정합성 정리 및 API 성능 개선

### Backend
- **`trader/claude.go`**: Anthropic API 529 overloaded_error 발생 시 지수 백오프(2s→4s→8s) 최대 3회 재시도 로직 추가 (`SelectStocks`, `AnalyzeDailyReport`)
- **`api/handlers.go`**: `GetServerStatus`에서 `GetInquireBalance` 결과를 30초 캐시 (`sync.Mutex` 기반) — 새로고침 빈도가 높을 때 KIS API 중복 호출 방지

### Frontend
- **`Dashboard.jsx`**: 백엔드에 구현되지 않은 `미장(US) 상태`·`미장 트레이더` 섹션 제거
- **`Orders.jsx`**: 미장(US) 주문이 없는 시스템이므로 시장 필터(국장/미장) 제거
- **`StockLogs.jsx`**: 엔진에서 사용하지 않는 `대량체결`·`이격도` 순위 카운트 UI 제거; RANKING_TYPE_LABELS을 실제 사용 타입(거래량/체결강도/등락률/VI발동)으로 교체; 조회 로그 수 축소(ranking 100→20, selection 200→30)
- **`DailyReports.jsx`**, **`TradeReports.jsx`**, **`OptimizationReports.jsx`**: 하드코딩된 `bg-gray-800/gray-700` 버튼을 테마 변수(`bg-th-surface/bg-th-surface-high`, `text-th-on-muted`)로 교체 — 라이트 테마에서도 글자 가시성 확보

## 2026-04-08 — lease 등록 시점 변경으로 GetStockInfo 과다 호출 해결

- **`trader/engine.go`**: lease 등록을 `getRankings` 직후에서 **모든 서버 필터(시가총액/거래대금/현금/RSI/이격도) 통과 후**로 이동
- 기존: 품질 미검증 ~40개가 lease에 쌓여 매 사이클 96개+ GetStockInfo 호출
- 변경: 실제 후보 품질 종목(소수)만 lease에 등록 → 다음 사이클 lease 복원 종목 수 대폭 감소
- lease 복원 로직은 `getRankings` 내에 그대로 유지

## 2026-04-08 — TPS 초과 및 Claude 토큰 부족 수정

### 문제 1: KIS API TPS 초과 (EGW00201)
- **`agent/stock_info.go`**: `GetBidAskRatio` 호출 제거 — LLM 보조 지표는 최종 후보에만 별도 호출
- **`trader/engine.go`**: `GetStockInfo` 루프를 세마포어(동시 3개) 기반 goroutine 병렬 처리로 변경
- **`trader/engine.go`**: 거래대금 필터 중복 `GetStockInfo` 제거 → 1차 루프에서 채운 `item.TradingValue` 직접 사용
- **`trader/engine.go`**: BidAskRatio를 서버 필터 + 점수화 후 최종 후보에만 병렬 호출

### 문제 2: Claude 응답 truncated (JSON array 없음)
- **`trader/claude.go`**: `MaxTokens` 2048 → 4096 증가
- **`trader/claude.go`**: 프롬프트에 `DO NOT explain your reasoning process. Output ONLY the final JSON array.` 추가
- **`trader/engine.go`**: 서버 사전 점수화(MA배열·MACD·RSI구간·VWAPDiff·PrevVolumeRatio) 후 상위 N개만 Claude 전달
- **`database/db.go`**: `max_claude_candidates` 설정키 추가 (기본값 15, `MaxClaudeCandidates` 필드)

### 효과
- KIS API 호출 ~264회 순차 → ~106회 병렬(3개씩) — 약 60% 감소
- 종목 선정 소요시간 ~30초 → ~4~6초
- Claude 입력 후보 40개+ → 최대 15개

## 2026-04-08 — AI 자동 최적화 제안 기능 추가

- **`models/models.go`**: `OptimizationSuggestion`, `OptimizationReport` 구조체 추가
- **`database/db.go`**: `optimization_reports` 테이블 신규 + CRUD 함수 3개 + `optimization_apply_mode` 기본 설정 추가
- **`trader/claude.go`** (신규 메서드): `AnalyzeDailyReport()` — 일별 리포트·현재 설정을 Claude에 전달해 개선 제안(settings/feature 2카테고리) JSON 반환
- **`report/optimization.go`** (신규): `GenerateOptimizationSuggestions()` + `ApplySuggestionByID()` + `RejectSuggestionByID()` — 분석 파이프라인 및 자동/수동 적용 로직
- **`cmd/server/main.go`**: 15:20 스케줄러 블록에 `GenerateOptimizationSuggestions()` 연계, `claudeClient` 파라미터 추가
- **`api/handlers.go`**: Handler에 `claude` 필드 + `SetClaudeClient()` + `optimization_apply_mode` GetSettings/UpdateSettings 처리 + 핸들러 4개 추가
- **`api/router.go`**: `/api/reports/optimization` 라우트 4개 등록
- **`pages/OptimizationReports.jsx`** (신규): AI 개선 제안 목록 페이지 (탭 필터, [적용]/[무시] 버튼)
- **`pages/Settings.jsx`**: `optimization_apply_mode` 셀렉트 추가 (전체 자동/전체 수동)
- **`App.jsx`**: "AI 개선 제안" 메뉴 + 라우트 추가

## 2026-04-07 — 거래 학습 리포트 기능 추가

### DB 스키마
- **`trade_reports`** 테이블 신규: 매수 시 RankItem 스냅샷·Claude 선정 근거·체결가/수량 저장, 매도 후 지표 재수집 및 손익 자동 계산
- **`daily_reports`** 테이블 신규: 하루 종료 후 승률·총손익·평균수익률·최고/최하 거래 집계

### 백엔드
- **`models/models.go`**: `TradeReport`, `DailyReport` 구조체 추가
- **`database/db.go`**: 테이블 마이그레이션 + `InsertTradeReport`, `UpdateTradeReportOnSell`, `GetTradeReports`, `InsertOrUpdateDailyReport`, `GetDailyReports` CRUD 메서드 추가
- **`trader/engine.go`**: 매수 체결 성공 후 `InsertTradeReport` goroutine 비동기 호출
- **`monitor/monitor.go`**: `executeSell` / `LiquidateAll` 매도 후 `agent.GetStockInfo` 재호출로 매도 시점 지표 수집 + `UpdateTradeReportOnSell` 호출
- **`report/report.go`** 신규: `GenerateDailyReport` — 당일 완료 거래 집계 → `daily_reports` UPSERT
- **`cmd/server/main.go`**: 15:20 스케줄에 `GenerateDailyReport` 자동 호출 추가
- **`api/handlers.go`**: `GET /api/reports/trades`, `GET /api/reports/daily`, `POST /api/reports/daily/generate` 핸들러 추가
- **`api/router.go`**: `/api/reports/*` 라우트 등록

### 프론트엔드
- **`TradeReports.jsx`** 신규: 거래별 리포트 — 매수/매도 지표 팝오버, 손익 색상, 날짜·종목 필터, 페이지네이션
- **`DailyReports.jsx`** 신규: 일별 요약 카드 — 승률·총손익·최고/최하 거래·거래 목록 토글
- **`App.jsx`**: "거래 리포트", "일별 리포트" 네비게이션 탭 추가

## 2026-04-06 — hard_watch 종목 현금 필터 silent drop 버그 수정

- **`trader/engine.go`**: GetStockInfo 보강 루프에서 `CurrentPrice` 갱신 추가
  - hard_watch 종목은 `getRankings()`에서 `CurrentPrice=""`로 추가됨
  - 보강 루프에서 `CurrentPrice`가 복사되지 않아 현금 필터 `price > 0` 조건 실패 → silent drop
  - `info.CurrentPrice != ""`이면 `rankings[i].CurrentPrice` 갱신하도록 수정 → 하드필터/LLM 정상 전달

## 2026-04-06 — 지수 필터 개선 및 종목로그 버그 수정

### 기능 개선
- **`trader/engine.go`**: 지수 필터 동작 방식 변경
  - 이전: 설정 지수 중 하나라도 임계값 이하 하락 시 **전체 매수 중단**
  - 이후: 하락한 지수에 대응하는 **거래소만 순위 조회에서 제외**, 나머지 거래소는 정상 조회
    - 예) KOSPI 하락 → KOSPI 거래소 제외, KOSDAQ만 순위 조회
    - 예) 모든 지수 하락 → 기존과 동일하게 매수 중단
  - `force=true` 강제 실행 시 지수 필터 완전 무시 — 기존 동작 유지
  - 거래소 제외 발생 시 INFO 로그 기록 (제외 거래소, 활성 거래소, 임계값)

### 버그 수정
- **`trader/engine.go`**: `lease`·`hard_watch` 타입 종목 추가 시 `StockName` 미설정 수정 — MST store에서 종목명 조회 후 설정 (`lookupStockName` 헬퍼 추가)
  - 증상: 종목로그에서 lease/하드감시 종목이 코드만 표시되고 종목명 없음
- **`trader/engine.go`**: `GetInquireBalance` 실패·가용현금 없음·일일손실 초과·현금부족 등 early return 시 selection log 없이 종료되던 문제 수정
  - `insertFailedSelectionLog` 헬퍼 추가 — 조기 종료 사유를 selection log의 `fail_reason`에 기록하고 `filtered_stocks`도 함께 갱신
  - 증상: 종목로그에서 "하드필터 전체통과"인데 "LLM 선정 단계 미실행 — 순위/필터 단계에서 후보 없음"으로 표시되어 실제 실패 원인 파악 불가

## 2026-04-04 — 설정화면 0값 저장 버그 수정 및 하드감시 PATCH 누락 필드 추가

### 버그 수정: 설정값 0 입력 시 기본값으로 대체되는 문제
- **`frontend/src/pages/Settings.jsx`**: `parseFloat(x) || nonZeroDefault` 패턴을 `parseFloat(x)`로 변경 (20개 필드). 0은 JS에서 falsy라 기본값으로 대체되던 문제 해결
- **`backend/internal/database/db.go`**: `f64Default(key, default)` / `i64Default(key, default)` 헬퍼 추가. 기존 `if value == 0 { value = default }` 패턴을 "키가 없거나 빈 문자열일 때만 기본값 적용"으로 교체 (25개 필드)
- **`backend/internal/trader/engine.go`**: db.go에서 이미 기본값이 적용되므로 엔진 레벨의 이중 `== 0` 폴백 제거 (14개 필드)

### 버그 수정: 종목 목록 하드감시 등록/해제 항상 실패하는 문제
- **`backend/internal/api/handlers.go`**: `UpdateSettings` 핸들러 struct에 `hard_watch_symbols []string` 및 `rank_lease_duration_min *int` 필드 누락으로 PATCH 요청이 무시되던 문제 수정. 저장 로직 추가

## 2026-04-03 — 순위 세부옵션 추가 및 미장 필터 제거

### 등락률·VI 세부 필터 추가
- **`database/db.go`**: `ranking_fluctuation_min_rate`, `ranking_fluctuation_max_rate`, `ranking_vi_kind_code` 설정 키 추가
- **`trader/engine.go`**: fluctuation 케이스에 등락률 범위 필터, vi_status 케이스에 정적/동적 종류 필터 추가
- **`api/handlers.go`**: GetSettings/UpdateSettings에 신규 필드 3개 반영
- **`Settings.jsx`**: 등락률 체크 시 최소/최대 등락률 입력 표시, VI 체크 시 전체/정적/동적 토글 버튼 표시
- **`StockLogs.jsx`**: 국장/미장 탭 및 카드 배지 제거 (미장 미운용)

## 2026-04-03 — 트레이딩 시스템 고도화 2차 (Phase A~G)

### Phase A — RecoverFromHoldings() AssetType 버그 수정
- **`monitor/monitor.go`**: `mstStore *mst.Store` 필드 추가, `New()` 시그니처에 포함
- `RecoverFromHoldings()`: MST 조회로 AssetType 결정 → ETF/주식별 익절·손절 분리 적용 (0이면 기본값 폴백)
- **`cmd/server/main.go`**: mstStore 생성을 monitor 초기화 이전으로 이동, `monitor.New()`에 mstStore 주입, 중복 생성 제거

### Phase B — VI 발동현황 (FHPST01390000)
- **`kis/client.go`**: `VIStatusItem` 구조체 추가, `GetVIStatus()` 함수 추가
- **`agent/ranking.go`**: `GetVIStatus()` 래퍼 추가
- **`trader/engine.go`**: `getRankings()` switch-case에 `vi_status` 처리 추가 (미해제 건 제외)
- **`api/handlers.go`**: `GET /api/ranking/vi-status` 핸들러 추가
- **`api/router.go`**: `/ranking/vi-status` 라우트 추가
- **`frontend/Settings.jsx`**: VI 발동현황 체크박스 추가

### Phase C — MST 확장 (listed_shares + min_market_cap 필터)
- **`mst/parser.go`**: `StockMaster`에 `ListedShares int64` 필드 추가
- **`mst/store.go`**: Upsert/GetByCode에 `listed_shares` 컬럼 추가
- **`database/db.go`**: `stock_masters` 테이블에 `listed_shares` 컬럼 추가(DDL + ALTER 마이그레이션), `TradingSettings`에 `MinMarketCap`·`MinExpectedProfitPct` 추가, 기본값 INSERT OR IGNORE
- **`trader/engine.go`**: 시가총액 필터 추가 (상장주식수 × 현재가 ÷ 1억 ≥ MinMarketCap)
- **`api/handlers.go`**: GetSettings/UpdateSettings에 `min_market_cap`·`min_expected_profit_pct` 추가
- **`frontend/Settings.jsx`**: 하드 필터 섹션에 최소 시가총액 입력 추가

### Phase D — 세금보정 기대수익률
- **`trader/claude.go`**: `RankItem`에 `TradingValue`·`ApplicableTaxRate` 추가; `TradingRules`에 `MinExpectedProfitPct`·`StockTaxRate` 추가; SelectStocks 프롬프트에 세금보정 규칙 삽입
- **`trader/engine.go`**: RankItem 보강 단계에서 `TradingValue`·`ApplicableTaxRate` 설정; TradingRules 빌드 시 `MinExpectedProfitPct`·`StockTaxRate` 반영
- **`frontend/Settings.jsx`**: 주식 세후 최소 기대수익(%) 입력 추가

### Phase E — 패닉셀 버튼
- **`api/handlers.go`**: `POST /api/monitor/liquidate-all` 핸들러 추가 (비동기 실행)
- **`api/router.go`**: `/monitor/liquidate-all` 라우트 추가
- **`frontend/Monitor.jsx`**: 헤더에 "전체 매도" 빨간색 버튼 추가 (2중 확인, 포지션 있을 때만 표시)

### Phase F — 프리셋 동기화 개선
- **`database/db.go`**: `active_preset_id` 기본값 추가, `UpdateSettingsPreset()` 함수 추가
- **`api/handlers.go`**: `ApplyPreset`·`CreatePreset` 후 `active_preset_id` 갱신; `UpdateSettings` 완료 시 활성 프리셋 스냅샷 동시 업데이트; GetSettings에 `active_preset_id` 반환
- **`frontend/Settings.jsx`**: 활성 프리셋에 "적용 중" 뱃지 표시

### Phase G — 문서 정리
- **`docs/kis-api/순위분석.md`**: VI 발동현황 상태 "🆕 신규" → "✅ 구현됨"으로 업데이트

---

## 2026-04-03 — 트레이딩 시스템 고도화 (Phase 1~6 완료)

### Phase 1 — US 마켓 코드 제거 / ExecCount·Disparity 순위 제거
- **`monitor/monitor.go`**: `SubscribeOverseasPrice`, `exchCodeToEXCD` 등 US 잔존 코드 제거
- **`kis/client.go`**: `ExecCountRankItem`, `DisparityRankItem` 구조체 및 `Get*Rank` 함수 제거
- **`agent/ranking.go`**, **`trader/engine.go`**, **`api/handlers.go`**, **`api/router.go`**: 관련 핸들러·라우트·분기 제거

### Phase 2 — MST 종목 마스터 파싱 & stock_masters 테이블
- **`backend/internal/mst/`** 신규 패키지 (parser.go, downloader.go, store.go)
  - KOSPI(288B)/KOSDAQ(282B) 고정폭 바이너리 파싱, EUC-KR→UTF-8 변환
  - 국내주식형 ETF 키워드 분류 (`classifyDomesticEquityETF`)
  - KOSPI/KOSDAQ ZIP 다운로드 + SQLite UPSERT
- **`database/db.go`**: `stock_masters` 테이블 마이그레이션 추가
- **`cmd/server/main.go`**: 서버 시작 시 stock_masters 자동 다운로드, 08:40 KST 일일 갱신 cron

### Phase 3 — 자산 타입 태깅 + ETF/주식별 수익/손절 분리
- **`trader/engine.go`**: `resolveAssetType()`, `resolveProfitLoss()` 헬퍼 추가; 종목 선정 후 MST 조회로 AssetType 태깅
- **`database/db.go`**: `ETFTakeProfitPct`, `ETFStopLossPct`, `StockTakeProfitPct`, `StockStopLossPct`, `StockTaxRate` 설정 키 추가
- **`monitor/monitor.go`**: `MonitoredEntry`에 `AssetType` 필드 추가

### Phase 4 — GetFluctuationRank 추가
- **`kis/client.go`**: `FluctuationRankItem` + `GetFluctuationRank()` (FHPST01700000)
- **`agent/ranking.go`**, **`trader/engine.go`**: `fluctuation` 분기 추가
- **`api/handlers.go`**, **`api/router.go`**: `/api/ranking/fluctuation` 핸들러·라우트 추가

### Phase 5 — Hard Watch Symbols + Lease TTL + /api/stocks 엔드포인트
- **`database/db.go`**: `HardWatchSymbols`, `RankLeaseDurationMin` 설정 키 추가
- **`trader/engine.go`**: 순위 lease TTL 로직 + Hard Watch 종목 강제 삽입
- **`api/handlers.go`**: `GetStocks()` 핸들러 추가 (stock_masters 검색, hard_watch 여부 포함)
- **`api/router.go`**: `GET /api/stocks` 라우트 등록; handler에 `SetMSTStore()` 연결

### Phase 6 — 프론트엔드 업데이트
- **`Settings.jsx`**: 미장 탭 제거; exec_count/disparity 체크박스 제거, fluctuation 추가; ETF/주식 전용 수익/손절 UI; 하드 감시 종목 추가/삭제 UI; rank_lease_duration_min 설정
- **`StockList.jsx`** (신규): 종목 마스터 검색/필터(ETF·거래소), 하드 감시 등록/해제
- **`App.jsx`**: `/stock-list` 라우트 및 사이드바 메뉴 추가

## 2026-03-26 — 순위 조회 개선: 가격 필터 클라이언트 이전, 거래소 분리, BLNG_CLS 다중 조회

- **`kis/client.go`**: `GetVolumeRank`에 `inputIscd` 파라미터 추가 (`FID_INPUT_ISCD` 하드코딩 제거)
- **`agent/ranking.go`**: `GetVolumeRank` 래퍼 시그니처에 `inputIscd` 추가
- **`database/db.go`**: `TradingSettings`에 `RankingExchanges`, `RankingVolumeBlngClsCodes` 필드 추가 및 기본값 설정
- **`trader/engine.go`**: 4개 순위 case를 거래소×BLNG_CLS 다중 API 호출+dedup 구조로 교체; 가격 필터 API 전달 제거 (클라이언트 필터만 사용)
- **`api/handlers.go`**: GET/PATCH 설정 핸들러에 신규 필드 추가; 수동 랭킹 핸들러에 `input_iscd` 쿼리 파라미터 추가
- **`Settings.jsx`**: 조회 거래소(KOSPI/KOSDAQ/KOSPI200) 및 거래량 분류 코드(0~4) 체크박스 UI 추가

## 2026-03-26 — 버그 수정 6종 (모바일 포커스·잘림, 미장 설정 완전 분리, Hard Rejection 표시, P&L %, 멀티 거래소)

- **`Settings.jsx`**: `PresetPanel`을 컴포넌트 외부로 이동 → 모바일 입력 포커스 버그 수정 (이슈1); sticky 헤더 `top-14 md:top-0` 수정 (이슈2)
- **`Settings.jsx`**: 미장 전용 매매기준·소프트필터·하드리젝션 10개 신규 UI 섹션 추가; 거래소 단일선택 → 복수선택(NASDAQ/NYSE/AMEX) (이슈3, 6)
- **`db.go`**: `TradingSettings` 10개 미장 전용 필드 추가; `USRankingExchanges []string`; DB default INSERT 및 GetTradingSettings 파싱 (이슈3, 6)
- **`handlers.go`**: 설정 응답/요청 구조체에 미장 전용 10개 필드 + `us_ranking_exchanges` 추가, save 로직 반영 (이슈3, 6)
- **`engine.go`**: `selectAndBuyUS` 미장 전용 설정값 적용; 멀티 거래소 루프 + 중복제거; KR fallback 제거 (이슈3, 6)
- **`StockLogs.jsx`**: `SelectionSection`에 `filteredStocks` prop 추가 — Hard Rejection으로 전체 제거 시 종목·이유 목록 표시 (이슈4)
- **`Dashboard.jsx`**: `PnLGraph`에 원/% 토글 추가 — 총 평가금액 기준 수익률 % 표시, 툴팁에 원화 병기 (이슈5)

## 2026-03-25 — 설정 페이지 탭 분할 (국장 / 미장 / AI·서버)

- **`Settings.jsx`**: 단일 스크롤 페이지를 **국장 / 미장 / AI·서버** 3개 탭으로 분할. `hidden` CSS 클래스로 비활성 탭 숨김(unmount 없음) — 단일 `<form>` 내에서 모든 설정값 유지. 국장·미장 탭 각각에 해당 시장 프리셋 패널 배치. AI·서버 탭은 별도 저장 버튼 사용

## 2026-03-25 — 강제매도 버튼 / 국장·미장 프리셋 분리

### 강제매도 버튼 (모니터링 페이지)
- **`monitor.go`** (`ForceSell`): 특정 종목에 시장가 매도 주문 후 모니터링 해제하는 공개 메서드 추가
- **`handlers.go`** (`ForceSellMonitorPosition`): `POST /api/monitor/positions/:code/sell` 핸들러 추가
- **`router.go`**: `POST /api/monitor/positions/:code/sell` 라우트 등록
- **`Monitor.jsx`**: 모니터링 테이블/카드에 **강제매도** 버튼 추가 — 클릭 시 확인 후 시장가 전량 매도, 기존 "해제" 버튼은 모니터링 해제만(매도 없음) 유지

### 국장·미장 프리셋 완전 분리 (설정 페이지)
- **`db.go`**: `settings_presets` 테이블에 `market TEXT NOT NULL DEFAULT 'KR'` 컬럼 추가 (ALTER TABLE 마이그레이션). `SettingsPreset` 구조체에 `Market` 필드 추가. `CreateSettingsPreset` 시그니처에 `market` 파라미터 추가
- **`handlers.go`** (`CreatePreset`): `market` 파라미터(`"KR"` | `"US"`) 수신. KR 저장 시 `us_` 접두사 키 제외, US 저장 시 `us_` 접두사 키만 포함하여 스냅샷 필터링
- **`Settings.jsx`**: 프리셋 패널을 **국장 프리셋** / **미장 프리셋** 두 섹션으로 완전 분리. 각 섹션에 전용 이름·설명 입력란 및 저장 버튼 배치. 국장 프리셋 적용 시 미장 설정 불변, 미장 프리셋 적용 시 국장 설정 불변

### 기타
- `frontend/package.json`: `msw` 패키지 devDependency 추가 (기존 빌드 오류 수정)

## 2026-03-24 — 강제 실행 시 스케줄/시장 조건 체크 우회

- **`engine.go`** (`selectAndBuy`, `selectAndBuyUS`): `force bool` 파라미터 추가. `force=true`일 때 요일 체크, 매수 중단 시간대, 지수 하락 필터를 건너뜀
- **`engine.go`** (`ForceRun`): `selectAndBuy` 호출 시 `force=true` 전달하도록 수정 — 강제 실행 버튼이 스케줄 조건에 막히지 않음
- **`engine.go`** (`runCycle`): 일반 사이클은 `force=false` 전달 (기존 동작 유지)

## 2026-03-24 — 트레이딩 중지 사유 표시 / 강제 실행 버튼 / 종목 수 불일치 수정

### 버그 수정
- **`engine.go`** (`selectAndBuy`, `selectAndBuyUS`): 가격 필터(현금 부족), 거래대금 필터, 이미 거래된 종목 제외 등 중간 필터 단계에서 제거된 종목이 `filtered_stocks`에 기록되지 않던 문제 수정.
  - 이제 모든 필터 단계 제거 종목이 `filtered_stocks` JSON에 누적 기록되어 "합집합 9종목 - 필터 8종목 = LLM 전달 1종목" 등 수치가 일치하도록 개선.
  - 각 제거 사유: `오늘 이미 거래된 종목`, `거래대금 미달 (N억 < M억)`, `현금 부족 (주가 N원 > 가용 M원)`, 기존 하드필터 사유 그대로 유지.

### 신규 기능
- **`engine.go`**: `haltReason` 필드 추가. 사이클 실패 시 마지막 에러 메시지 저장, 성공 시 초기화.
  - `GetHaltReason() string` 메서드 추가 (thread-safe).
  - `ForceRun(ctx)` 메서드 추가: goroutine으로 즉시 매수 사이클 트리거 (강제 실행용).
- **`handlers.go`**: `GET /api/server/status` 응답에 `halt_reason`, `halt_reason_us` 필드 추가.
- **`handlers.go`**: `POST /api/trader/force-run?market=KR|US` 엔드포인트 추가.
- **`router.go`**: `/api/trader/force-run` 라우트 등록.
- **`Dashboard.jsx`**: 국장/미장 트레이더 상태 행에 다음 UI 추가:
  - 중지 사유 텍스트 (amber 색상, halt_reason이 있을 때만 표시).
  - "강제 실행" 버튼 (항상 표시, 클릭 시 POST /api/trader/force-run 호출).

---

## 2026-03-23 — 미장 순위 버그 수정 + 하드필터 UI 노출

### 버그 수정
- **`engine.go`**: `selectAndBuyUS()`에서 `getRankings()` (국장용 KIS API 호출)를 잘못 호출하던 버그 수정 → `getRankingsUS()`로 변경. 미장 활성화 후 순위 조회 이후 로직이 중단되던 근본 원인.

### UI 개선
- **`RankingLogs.jsx`**: 순위 조회 로그에 하드필터 제거 종목 목록 노출 추가
  - 헤더에 `하드필터 -N` 배지 표시 (제거 종목이 있을 때)
  - `filtered_stocks` JSON을 파싱해 종목코드·종목명·제거사유 목록을 collapsible 섹션으로 표시

## 2026-03-22 — 트레이딩 고도화 Phase 4 (프리셋 시스템) 완료

### Phase 4 — 매매 프리셋 시스템
- **`db.go`**: `settings_presets` 테이블 신규 생성 (id, name, description, settings_json, created_at, updated_at)
- **`db.go`**: `ListSettingsPresets`, `CreateSettingsPreset`, `GetSettingsPreset`, `DeleteSettingsPreset` CRUD 함수 추가
- **`handlers.go`**: `ListPresets`, `CreatePreset` (현재 설정 전체 스냅샷), `ApplyPreset`, `DeletePreset` 핸들러 추가
- **`handlers.go`**: `GetSettings`에 누락된 `trading_days` + AI 매매 기준값 13개 필드 추가
- **`router.go`**: `/api/presets` 라우트 그룹 추가 (GET/POST + /:id/apply POST + /:id DELETE)
- **`Settings.jsx`**: 설정 페이지 상단에 프리셋 패널 추가 (목록 조회/적용/삭제 + 현재 설정 저장)
- **`mocks/handlers.js`**: `/api/presets` CRUD mock 추가

## 2026-03-22 — 트레이딩 고도화 Phase 1-3 + Phase 2d (BidAskRatio) 완료

### Phase 2d — BidAskRatio KIS API 통합
- **`kis/client.go`**: `GetBidAskRatio(ctx, stockCode)` 신규 함수 추가 (`inquire-asking-price-exp-ccn`, TR_ID `FHKST01010200`)
- **`agent/stock_info.go`**: `BidAskRatio float64` 필드 추가, `GetStockInfo`에서 호출 (실패 시 0으로 무시)
- **`trader/claude.go`**: `RankItem.BidAskRatio` 필드 추가
- **`trader/engine.go`**: KR/US 양쪽 getRankings에서 `rankings[i].BidAskRatio = info.BidAskRatio` 반영

### Phase 3 + Phase 1 — 프론트엔드 완성 (Settings.jsx)
- **요일 스케줄 체크박스**: 거래 제어 섹션에 월~일 체크박스 추가, `trading_days` 배열로 저장
- **AI 매매 기준값 섹션 신설**: 하드 리젝션 룰 8개 + 랭킹 기준 5개 수치 입력 UI
  - 하드 룰: `hard_disparity_m5_min/max`, `hard_high_price_diff_max/min`, `hard_prev_vol_ratio_max`, `hard_strength_min`, `hard_rsi_max`, `hard_open_price_diff_max`
  - 랭킹 기준: `vwap_diff_min/max`, `rsi_buy_min/max`, `bid_ask_ratio_min`
- **`handlers.go` UpdateSettings**: `trading_days` JSON 배열 파싱 + 13개 신규 AI 기준값 필드 저장 처리
- **`mocks/handlers.js`**: 신규 설정 필드 기본값 추가

### Phase 2a-2c — 신규 기술 지표 (이전 커밋)
- VWAP, VWAPDiff, M5MA10, PrevVolumeRatio 계산 및 Claude 프롬프트 전달

## 2026-03-20 — 라이트/다크 테마 완전 동작 + 실현 손익 그래프 추가

- **전 페이지 테마 토큰화**: 모든 hardcoded hex 색상(`bg-[#131316]` 등) → CSS semantic token (`bg-th-*`) 교체, 라이트/다크 모드 완전 분리
- **index.css**: `--th-sidebar` 토큰 추가, `.glass-panel` 라이트 모드 지원
- **tailwind.config.js**: `th-sidebar` 토큰 등록
- **Dashboard.jsx**: 실현 손익 그래프 추가 (recharts, 1주일/1달 탭, 일별 Bar + 누적 Line)
- **Backend `db.go`**: `GetDailyPnL(days)` 함수 추가 — AGENT SELL 기준 일별 집계
- **Backend `handlers.go`**: `GET /api/stats/daily-pnl?days=N` 핸들러 추가
- **Backend `router.go`**: `/api/stats` 라우트 그룹 등록
- **MSW mock**: `/api/stats/daily-pnl` 목 데이터 추가

## 2026-03-19 — UI 개선: ThemeToggle 복원, 설정 섹션 재배치, 자동새로고침, 로그 레벨 정리

- **App.jsx**: ThemeToggle 복원 (사이드바 하단), 모바일 햄버거 버튼 LEFT 이동
- **Settings.jsx**: `(FID_TRGT_EXLS_CLS_CODE)` 레이블 제거, "현재 값" 이진 표시 제거, `step="60"` 추가 (time input 초 단위 숨김), 지수하락임계값 → 거래 제어 섹션, 최소거래대금 → 하드 필터 섹션, AI 설정 → 최하단 이동
- **StockLogs.jsx**: 순위 유형 한글 표시 (volume→거래량 등), 자동 새로고침 셀렉터 추가
- **Dashboard.jsx**: 자동 새로고침 셀렉터 추가 (10초/30초/1분/5분)
- **ErrorLogs.jsx**: 서비스 로그 레벨 표시를 ERROR 고정 (INFO/WARN 개념 제거)
- **trader/engine.go**: `InsertServiceLog` WARN → ERROR로 통일 (일일손실한도, 미체결타임아웃)
- **mocks/handlers.js**: 서비스 로그 더미 데이터 모두 ERROR 레벨로 수정

## 2026-03-19 — MSW Mock 데이터 환경 구성

- **frontend/src/mocks/handlers.js** (신규): 전 API 엔드포인트 더미 데이터 핸들러 (서버상태, 잔고, 보유종목, 주문, 모니터, 로그, 설정)
- **frontend/src/mocks/browser.js** (신규): MSW Service Worker 셋업
- **frontend/src/main.jsx**: `VITE_USE_MOCK=true` 환경변수 시 MSW 자동 활성화
- **frontend/.env.development.local** (신규): 로컬 개발 시 mock 모드 ON
- **frontend/.eslintignore** (신규): `public/` 폴더 ESLint 제외 (mockServiceWorker.js)

## 2026-03-19 — Stitch "Digital Obsidian" 디자인 시스템 적용

- **App.jsx**: 상단 Navbar → 좌측 고정 사이드바(w-64, `#0E0E11`) + 모바일 드로어로 레이아웃 전환, Material Symbols Outlined 아이콘 적용
- **Dashboard.jsx**: 자산 총액 히어로 패널(glassmorphism, `text-4xl~5xl`), 서버 상태 tonal 배경, 보유종목 테이블 `divide-white/[0.04]` + `hover:bg-white/[0.02]`
- **Monitor.jsx / Orders.jsx / ErrorLogs.jsx**: 카드 보더 제거, `bg-[#1F1F22]` tonal 레이어링, 로그 좌측 accent `border-l-2` 적용
- **Settings.jsx**: 인라인 `th-*` 토큰 → 다크 전용 `gray-*` / Tailwind 색상으로 교체, glass-panel 스티키 헤더
- **index.css**: No-Line Rule 적용 (`.card` 보더 제거), `.glass-panel` 유틸 추가, Material Symbols 기본 font-variation-settings
- **index.html**: Material Symbols Outlined Google Fonts 링크 추가
- **StockLogs.jsx**: `th-*` 다크 클래스 교체, useMemo 의존성 경고 수정
- **ThemeContext.jsx**: `eslint-disable react-refresh/only-export-components` 추가 (lint 경고 해소)

## 2026-03-19 — 프론트엔드 전면 리뉴얼 + 하드필터 로그 연결

- **frontend/tailwind.config.js, index.css**: CSS 커스텀 속성 기반 다크/라이트 테마 토큰 시스템 도입 (`th-*` 변수), Inter+Manrope 폰트
- **frontend/src/contexts/ThemeContext.jsx** (신규): 다크모드 토글 컨텍스트, localStorage 영속화
- **frontend/src/App.jsx**: ThemeProvider 래핑, ThemeToggle 버튼, `/stock-logs` 라우트 추가
- **frontend/src/pages/Dashboard.jsx**: 수익률 0% 버그 수정 (`calcRate` fallback), th-* 토큰 리뉴얼
- **frontend/src/pages/Monitor.jsx**: 자동 새로고침 주기 선택(수동/5초/10초/30초/1분), 리뉴얼
- **frontend/src/pages/Orders.jsx**: 무한스크롤, 시장/유형/상태 필터, PENDING 주문 취소 버튼, 수동 동기화 days 선택
- **frontend/src/pages/ErrorLogs.jsx**: 서비스로그/KIS API 탭 구조, 출처 필터, 테마 토큰 적용
- **frontend/src/pages/Settings.jsx**: 저장 버튼 sticky 헤더로 이동 (form association), 테마 토큰 전면 교체
- **frontend/src/pages/StockLogs.jsx** (신규): 순위조회→하드필터→LLM선정 3단계 통합 뷰, 국장/미장 필터
- **backend/internal/models/models.go**: `TraderRankingLog.FilteredStocks`, `TraderSelectionLog.RankingLogID` 필드 추가
- **backend/internal/database/db.go**: `filtered_stocks`/`ranking_log_id` ALTER TABLE, `InsertRankingLog` ID 반환으로 변경
- **backend/internal/trader/engine.go**: 하드필터 제거 종목+사유 JSON 기록, selection log에 ranking_log_id 연결
- **backend/internal/api/handlers.go**: `GetSelectionLogs`에 `ranking_log_id` 스캔 추가

## 2026-03-18 — UI B스타일 전체 적용 (Settings.jsx 포함)

### Task 4: UI 개선 — B스타일 (미니멀 클린) + 한국식 색상 통일
- **Settings.jsx**: zinc 팔레트 전환, `rounded-xl` 카드, `rounded-lg` 입력, `rounded-full` 뱃지, 익절=빨강·손절=파랑 레이블 힌트, AND/OR·거래소·거래량 버튼 zinc ring 스타일, 저장 버튼 `rounded-xl`
- **Badge/WsBadge**: `rounded-full` pill + border 스타일
- **Dashboard/Monitor/Orders/ErrorLogs**: B스타일 + 한국식 색상 (매수=빨강, 매도=파랑, 목표가=빨강, 손절가=파랑) 전면 적용
- **App.jsx**: `bg-zinc-950` 배경, `bg-zinc-900` 네비, active nav `bg-zinc-800 ring-1 ring-zinc-700`
- **Card.jsx**: zinc 팔레트, `rounded-xl`

---

## 2026-03-18 — MQTT 제거, KR/US 손실 한도 분리, 서비스 에러 로그 확장

### Task 6: MQTT 코드 전면 제거
- **mqtt/publisher.go** 파일 및 `mqtt/` 디렉터리 완전 삭제
- **config/config.go**: `MQTTBrokerURL`, `MQTTClientID` 필드 제거
- **cmd/server/main.go**: MQTT Publisher 초기화 블록 제거, `monitor.New()` / `NewEngine()` 인자에서 mqttPub 제거
- **monitor/monitor.go**: `mqttPub` 필드·파라미터·`PublishAlert` 호출 5곳 제거, 미사용 `sellQty` 변수 정리
- **trader/engine.go**: `mqttPub` 필드·파라미터 제거
- **api/handlers.go**: 상태 응답에서 `mqtt_broker_url`, `mqtt_client_id` 제거
- **go.mod**: `paho.mqtt.golang` 의존성 제거 + `go mod tidy`
- **Settings.jsx**: MQTT 브로커/클라이언트 ID 표시 행 제거
- **Monitor.jsx**: MQTT 토픽 안내 문구 제거
- **docs/guides/mqtt-setup.md** 삭제
- **docs/architecture.md**: MQTT 디렉터리·데이터 흐름·토픽 맵 섹션 제거

### Task 1: KR/US 손실 한도 및 최소 거래대금 분리
- **database/db.go**: `GetTodayRealizedPnLByMarket(ctx, market)` 추가 (market 필터); `TradingSettings`에 `USDailyMaxLossPct`, `USMinTradingValue` 추가; `defaultSettings`에 `us_daily_max_loss_pct`(0), `us_min_trading_value`(0) 추가
- **trader/engine.go**: KR 손실 체크 → `GetTodayRealizedPnLByMarket("KR")`; US 손실 체크 → USD 기준 `USDailyMaxLossPct` (0이면 KR 값 fallback); US 최소 거래대금 → `USMinTradingValue` (0이면 KR 값 fallback)
- **api/handlers.go**: `GetSettings`/`UpdateSettings`에 두 신규 필드 추가
- **Settings.jsx**: 미장 설정 블록에 `us_daily_max_loss_pct`, `us_min_trading_value` 입력 필드 추가

### Task 3: 서비스 에러 로그 확장
- **database/db.go**: `service_logs` 테이블 마이그레이션 추가; `InsertServiceLog()`, `GetServiceLogs()` (7일 자동 삭제, source 필터) 추가
- **models/models.go**: `ServiceLog` 구조체 추가
- **api/handlers.go**: `GetServiceLogs` 핸들러 추가 (`GET /api/logs/service?limit=100&source=ALL`)
- **api/router.go**: `/api/logs/service` 라우트 등록
- **trader/engine.go**: 일일 손실 한도 도달(KR/US), 미체결 타임아웃(KR/US) 4곳에 `InsertServiceLog` 추가
- **monitor/monitor.go**: GetHoldings/PlaceSellOrder 실패(KR/US) 4곳에 `InsertServiceLog` 추가
- **frontend/src/pages/ErrorLogs.jsx** 신규 생성: 탭 2개 — "서비스 로그"(source 필터, ERROR=빨강/WARN=노랑 배지) + "KIS API 에러"(기존 KISLogs UI 통합)
- **App.jsx**: `KISLogs` → `ErrorLogs` import 교체, 네비 레이블 "KIS 에러 로그" → "에러 로그"

## 2026-03-17 — 하드 필터 설정화, market 컬럼 추가, UI 개선

- **models/models.go**: `TraderSelectionLog`, `TraderRankingLog`에 `Market string` 필드 추가
- **database/db.go**: `TradingSettings`에 `FilterRsiMax`, `FilterDisparityM5Max`, `FilterHighPriceDiffMin`, `FilterOpenPriceDiffMax`, `IndexDropThresholdPct` 5개 필드 추가; `trader_selection_logs`·`trader_ranking_logs` 테이블에 `market` 컬럼 ALTER; 5개 신규 설정값 기본값 INSERT; `InsertRankingLog`·`GetRankingLogs` market 컬럼 반영
- **trader/engine.go**: KR·US 하드 필터를 `settings.*` 필드 기반으로 교체 (RSI/Disparity/HighPriceDiff/OpenPriceDiff); KR·US `trader_selection_logs` INSERT에 market 컬럼 추가; `getRankings` KR·US `InsertRankingLog`에 Market 필드 추가; 지수 하락 임계값 `IndexDropThresholdPct` 적용
- **trader/claude.go**: KR·US 프롬프트에 `open_price_diff > 20%% skip` 룰 추가; 눌림목·최적 진입 구간 강조 개선
- **api/handlers.go**: `Handler`에 `usEngine` 필드 추가; `SetUSEngine()` 메서드; `GetServerStatus`에 `us_market_open`·`trader_state_us` 반환; `GetSelectionLogs` market 컬럼 SELECT/Scan; GetSettings·UpdateSettings에 5개 신규 필터 설정 반영
- **cmd/server/main.go**: `handler.SetUSEngine(usEngine)` 호출 추가
- **kis/client.go**: `GetRawBalance()` 디버깅 함수 제거; `GetOverseasDailyChart()`·`OverseasDailyBar` 제거 (GetOverseasMinuteChart로 대체)
- **Settings.jsx**: 하드 필터(매수 품질) 섹션 신규 추가 (RSI/5분봉이격도/고가대비/시가대비/지수하락임계값); 레이아웃 max-w-lg 제거
- **Dashboard.jsx**: "장 운영"을 "국장(KR)"/"미장(US)"으로 분리; "미장 트레이더" 상태 추가; 서버 상태 그리드 반응형 개선
- **Monitor.jsx**: 모바일 카드 그리드 추가 (sm:hidden)
- **Orders.jsx**: `fmtPrice(price, market)` US→`$X.XX`/KR→`X원`; "시장" 컬럼 (미장/국장 배지); 일부 헤더 hidden sm:table-cell 반응형 처리
- **KISLogs.jsx**: 로그 카드 레이아웃 컴팩트화
- **SelectionLogs.jsx**: market 배지 (미장/국장) 카드 헤더 추가
- **RankingLogs.jsx**: market 배지 추가; 가격 범위 US→`$X~$Y`/KR→`X~Y원` 포맷

## 2026-03-17 — MA5/MA20 일봉→5분봉 전환, US RSI/MACD/DisparityM5 지표 추가

- **kis/client.go**: `OverseasMinuteBar` 구조체 및 `GetOverseasMinuteChart()`(HHDFS76950200) 추가 — 5분봉 OHLCV 최대 120개, NMIN=5/PINC=1/NREC=120
- **agent/stock_info.go**: KR `GetStockInfo()` MA5/MA20 계산을 일봉(GetDailyChart)에서 5분봉 집계(`closes5m`)로 전환 — 단타 전략에서 5분봉 MA가 실제 단기 추세를 반영; `GetOverseasStockInfo(ctx, client, excd, symb)` 신규 추가 — 해외주식 5분봉 기반 MA5/MA20/RSI14/MACD/DisparityM5 일괄 계산
- **trader/engine.go**: US enrichment 루프를 `GetOverseasStockInfo()` 단일 호출로 교체 (`GetOverseasPrice` + `GetOverseasDailyChart` 인라인 제거); US 하드 필터에 RSI≥80 및 disparity_m5>3% 조건 추가
- **trader/claude.go**: US 프롬프트 업데이트 — Hard Rejection에 disparity_m5>3% / rsi14≥80 추가, Ranking 기준에 MACD·RSI·DisparityM5 참조 추가, "RSI/MACD not available" 주석 제거

## 2026-03-17 — 미장(US) 엔진 품질 개선 — KR 개선 기능 적용

- **kis/client.go**: `OverseasPriceResponse`에 `Open`, `High`, `Low`, `Tamt` 필드 추가 (API 기존 반환값 매핑), `OverseasDailyBar` 타입 및 `GetOverseasDailyChart()`(HHDFS76240000) 신규 구현 (일봉 OHLCV 배열 반환)
- **trader/claude.go**: `SelectStocks()` 시그니처에 `market string` 파라미터 추가, `market=="US"` 시 US 전용 프롬프트 사용 (RSI/MACD/disparity_m5 제거, USD 기준, NASDAQ/NYSE/AMEX 컨텍스트)
- **trader/engine.go**: `selectAndBuyUS()`에 품질 필터 5종 추가 — ① 매수 중단 시간대, ② 일일 최대 손실 한도, ③ 지표 enrichment(GetOverseasPrice·GetOverseasDailyChart로 MA5/MA20/HighPriceDiff/OpenPriceDiff 계산), ④ 최소 거래대금 필터(USD), ⑤ 하드 필터(HighPriceDiff < -5% / MA5 < MA20); 트레일링 스탑 누락 버그 수정(`TrailingTriggerPct` / `TrailingStopPct` MonitoredEntry에 전달)

## 2026-03-16 — 미장(미국주식) 듀얼 엔진 구현

- **models/models.go**: `Order.Market`, `MonitoredPosition.Market` 필드 추가 (`"KR"` / `"US"`)
- **database/db.go**: `TradingSettings`에 US 설정 10개 필드 추가, `alterStmts`에 `orders.market` / `monitored_positions.market` 컬럼 추가 마이그레이션, `defaultSettings`에 US 기본값 추가, `GetTradingSettings` 쿼리·파싱·반환 확장
- **kis/client.go**: 해외 주문 API (TTTT1002U/1006U), 잔고 (TTTS3012R), 가격 (HHDFS00000300), 거래량 순위 (HHDFS76310010), 주문가능 조회 (TTTS3007R) 7개 함수 및 관련 타입 추가
- **kis/websocket.go**: `TrIDOverseasPrice` / `TrIDOverseasExecNotice` 상수 추가, `SubscribeOverseasPrice` / `UnsubscribeOverseasPrice` 함수 추가, `parseOverseasPriceData` 구현, `handleMessage` switch에 해외 가격/체결통보 케이스 추가
- **monitor/monitor.go**: `MonitoredEntry`에 `Market` / `ExchCode` 필드 추가, `Register` / `Remove` / `LiquidateAll` / `LoadFromDB` / `ResubscribeAll`을 US 시장 분기 처리로 개선, `executeOverseasSell` 신규 함수, `LiquidateAll` 시그니처를 variadic으로 변경 (market 필터 지원), `exchCodeToEXCD` 헬퍼 추가
- **trader/engine.go**: `Engine.market` 필드 추가, `NewEngine` 시그니처에 `market string` 파라미터 추가, `countPendingOrders` / `getTodayTradedCodes` market 필터 적용, `getRankings` US 분기 추가, `getRankingsUS` / `excdToExchCode` / `selectAndBuyUS` 신규 함수
- **cmd/server/main.go**: KR/US 엔진 동시 생성, `runMarketScheduler`에 US 엔진 스케줄링 추가 (`isActiveUSTrading` 자정 크로스오버 처리), `LiquidateAll` 호출에 market 인자 전달
- **api/handlers.go**: `GetSettings` / `UpdateSettings`에 US 설정 10개 항목 추가
- **agent/history.go**: `GetLocalOrderHistory` SELECT에 `market` 컬럼 추가
- **frontend/Settings.jsx**: 미장 설정 섹션 (ON/OFF, DST, 거래소, 가격범위, 순위유형, 거래량필터, 상위N) UI 추가

## 2026-03-16 — Settings input step 수정 + 지수 필터 복수 지정

- **Settings.jsx**: `min_trading_value` step `100000000` → `any` (자유 입력). `trailing_trigger_pct`, `trailing_stop_pct`, `daily_max_loss_pct` step `0.5` → `0.1`
- **database/db.go**: `IndexCode string` → `IndexCodes []string`, DB 키 `index_code` → `index_codes` (JSON 배열, 기본값 `"[]"`)
- **trader/engine.go**: 단일 지수 코드 체크 → `IndexCodes` 배열 루프
- **api/handlers.go**: `index_code *string` → `index_codes []string`, JSON 직렬화 저장
- **Settings.jsx**: 지수 필터 텍스트 입력 → 코스피(0001)/코스닥(1001) 체크박스 2개

## 2026-03-16 — KIS API 명세 업데이트 + 국내·미장 동시 트레이딩 아키텍처 설계

- **docs/kis-api/주문계좌.md**: 현재 사용 중인 TR_ID 중심으로 전면 재작성 (TTTC0012U/0011U/0013U/0084R/0081R/8434R/8908R)
- **docs/kis-api/기본시세.md**: FHKST01010100/FHKST03010100/FHKST03010200 최신화
- **docs/kis-api/순위분석.md**: 사용 중인 4종 API 상세화 + 전체 API 목록 추가
- **docs/kis-api/실시간.md**: H0STCNT0/H0STCNI0/H0STASP0 WebSocket URL·필드 최신화
- **docs/kis-api/해외주식_주문계좌.md** (신규): TTTT1002U/1006U/1004U/TTTS3007R/3018R/3012R/3035R 명세
- **docs/kis-api/해외주식_기본시세.md** (신규): HHDFS00000300/HHDFS76950200/HHDFS76240000 명세
- **docs/kis-api/해외주식_시세분석.md** (신규): 8종 순위 API (HHDFS76310010 등) 명세
- **docs/kis-api/해외주식_실시간.md** (신규): HDFSCNT0/HDFSASP0/H0GSCNI0 WebSocket 명세
- **docs/reviews/2026-03-16-dual-market-architecture.md** (신규): 국내·미장 동시 운영 전체 아키텍처 설계 (DB 스키마, Engine 분리, KIS Client 신규 함수, 스케줄러, UI 확장 계획)

---

## 2026-03-16 — 매매 품질 개선 8종 (OHLC·하드필터·트레일링스탑·점심제한·거래대금·일일손실한도·지수필터)

### A. 당일 OHLC + 파생 지표
- **kis/client.go**: `StockPriceResponse`에 `DayOpen`, `DayHigh`, `DayLow` 필드 추가 (KIS API가 이미 반환하는 필드)
- **agent/stock_info.go**: `StockInfo`에 `DayOpen`, `DayHigh`, `DayLow`, `HighPriceDiff`, `OpenPriceDiff`, `DisparityM5` 추가. `HighPriceDiff = (현재가-고가)/고가×100`, `OpenPriceDiff = (현재가-시가)/시가×100`, `DisparityM5`는 기존 5분봉 데이터로 계산 (추가 API 호출 없음)
- **trader/claude.go**: `RankItem`에 6개 파생 지표 필드 추가
- **trader/engine.go**: `GetStockInfo` 결과에서 신규 필드 매핑

### B. 코드 레벨 하드 필터
- **trader/engine.go**: LLM 호출 전 서버 측 제거 — RSI≥80, DisparityM5>3%, HighPriceDiff<-5%

### C. 프롬프트 전면 개선
- **trader/claude.go**: 불 트랩 회피 + 눌림목 매수 전략 프롬프트로 교체. `MaxTokens: 2048`

### D. 거래대금 하한선
- **database/db.go**: `TradingSettings.MinTradingValue` 추가, 기본값 `0`(비활성)
- **trader/engine.go**: GetStockInfo 루프 이후 거래대금 미달 종목 필터링
- **api/handlers.go**: `min_trading_value` 읽기/쓰기 추가
- **Settings.jsx**: 매수 설정 섹션에 "최소 거래대금" 입력 UI 추가

### E. 점심시간 매수 중단
- **database/db.go**: `BuyPauseStart`, `BuyPauseEnd` 추가, 기본값 `11:00`/`14:00`
- **trader/engine.go**: `selectAndBuy()` 진입 시 중단 시간대 체크
- **api/handlers.go**: `buy_pause_start`, `buy_pause_end` 읽기/쓰기 추가
- **Settings.jsx**: 거래 제어 섹션에 매수 중단 시작/종료 시간 입력 UI 추가

### F. 트레일링 스탑
- **database/db.go**: `TrailingTriggerPct`, `TrailingStopPct` 추가, 기본값 `0`/`1.0`
- **monitor/monitor.go**: `MonitoredEntry`에 `TrailingTriggerPct`, `TrailingStopPct`, `TrailingActivated`, `PeakPrice` 필드 추가. `HandlePrice()`에서 활성화 기준 도달 시 트레일링 활성화, 최고가 추적, 최고가 대비 하락 폭 초과 시 자동 매도
- **trader/engine.go**: MonitoredEntry 등록 시 trailing 파라미터 전달
- **api/handlers.go**: `trailing_trigger_pct`, `trailing_stop_pct` 읽기/쓰기 추가
- **Settings.jsx**: 매도 설정 섹션에 트레일링 스탑 UI 추가

### G. 일일 최대 손실 제한
- **database/db.go**: `DailyMaxLossPct` 추가, 기본값 `0`(비활성); `GetTodayRealizedPnL()` 함수 추가 (당일 SELL 주문의 실현 손익 합산)
- **trader/engine.go**: GetInquireBalance 이후 손실 한도 초과 시 매수 중단
- **api/handlers.go**: `daily_max_loss_pct` 읽기/쓰기 추가
- **Settings.jsx**: 매도 설정 섹션에 일일 최대 손실 UI 추가

### H. 지수 필터
- **database/db.go**: `IndexCode` 추가, 기본값 `""`(비활성)
- **kis/client.go**: `GetIndexPrice(ctx, indexCode)` 함수 추가 — `inquire-index-price` (FHPUP02100000) 엔드포인트
- **trader/engine.go**: `selectAndBuy()` 진입 시 지수가 시가 대비 -1% 이상 하락이면 매수 중단
- **api/handlers.go**: `index_code` 읽기/쓰기 추가
- **Settings.jsx**: 거래 제어 섹션에 지수 코드 입력 UI 추가

### 기타
- **kis/client.go**: `withinPriceRange()` 헬퍼로 KIS API가 무시하는 가격 파라미터를 서버에서 재필터링

## 2026-03-16 — 주문 정렬 변경 + 리포트 기능 제거 + 순위 AND/OR 조건 추가

- **agent/history.go**: 주문 내역 정렬 `ORDER BY created_at DESC, id DESC` → `ASC, ASC` (오래된 주문이 위로)
- **리포트 기능 완전 제거**: `engine.go` `GenerateDailyReport()` + `tradeRow` 제거, `claude.go` `ReportSummary` + `GenerateReport()` 제거, `handlers.go` `GetReports()` + `GetReport()` 제거, `router.go` `/api/reports` 라우트 제거, `database/db.go` `SaveReport()` 제거, `models/models.go` `Report` 구조체 제거, `cmd/server/main.go` 15:20 리포트 스케줄러 제거, `App.jsx` 리포트 임포트/라우트/네비 제거, `pages/Reports.jsx` 파일 삭제 (DB `reports` 테이블은 보존)
- **순위 AND/OR 조건**: `database/db.go` `TradingSettings.RankingCondition` 필드 + 기본값 `"AND"` 추가, `engine.go` `getRankings()` OR 합집합 로직 분기 추가, `handlers.go` `GetSettings`/`UpdateSettings`에 `ranking_condition` 포함, `Settings.jsx` AND/OR 토글 버튼 UI 추가

## 2026-03-15 — WebSocket 자동 연결 버그 수정 + 매도 판단 근거 저장 및 표시

- **cmd/server/main.go**: 08:50 스케줄러에서 `wsClient.SetReconnectCancel(wsCancel)` 호출 추가. 매일 16:00 `Disconnect()`가 `intentionalStop = true`로 설정하여 다음날 `StartWithReconnect`가 즉시 종료되는 버그 수정 → 이제 `SetReconnectCancel()`이 플래그를 리셋하여 자동 연결 정상 동작
- **database/db.go**: `orders` 테이블에 `sell_reason TEXT NOT NULL DEFAULT ''` 컬럼 마이그레이션 추가
- **models/models.go**: `Order` 구조체에 `SellReason string` 필드 추가 (`json:"sell_reason"`)
- **monitor/monitor.go**: `executeSell()` 시그니처에 `reason string` 파라미터 추가; 매도 주문 성공 시 `orders` 테이블에 INSERT (자동 매도 주문이 기존에 DB에 기록되지 않던 버그도 수정); `LiquidateAll()`도 동일하게 `orders` INSERT 추가 (reason=`"일일 자동 청산"`); `HandlePrice()` 목표가→`"목표가 도달"`, 손절가→`"손절가 도달"`; `checkIndicators()`→ `triggerReason` 변수 그대로 전달 (RSI/MACD/횡보 상세 메시지 포함)
- **agent/history.go**: `GetLocalOrderHistory()` SELECT/Scan에 `sell_reason` 포함
- **Orders.jsx**: 주문 목록 테이블에 `매도사유` 컬럼 추가 (SELL 주문에만 값 표시)

## 2026-03-11 — 이미 거래한 종목 서버 필터링 + 종목선정 UI 화살표 중복 수정

- **trader/engine.go**: AND 교집합 이후, 지표 조회 전에 `excludedCodes`로 서버 측 필터링 추가. Claude API 호출 전에 이미 오늘 거래한 종목을 미리 제거하여 불필요한 API 토큰 낭비 방지
- **trader/claude.go**: `SelectStocks()`의 Claude 프롬프트에서 `excluded stocks` 섹션 제거 (서버 필터링으로 완전 대체). 프롬프트 간결화로 토큰 절약
- **SelectionLogs.jsx**: `<details><summary>` 브라우저 기본 삼각형 마커와 직접 삽입한 `▶` 문자가 중복 표시되던 버그 수정 → `list-none [&::-webkit-details-marker]:hidden` CSS 적용

## 2026-03-11 — 거래 시간 설정 + 횡보 감지 자동 매도 + 설정 화면 재구성

- **database/db.go**: `TradingSettings` 구조체에 4개 필드 추가 (`TradingStartTime`, `TradingEndTime`, `StagnationThresholdPct`, `StagnationDurationMin`); 기본값 설정 (`09:15`, `15:15`, `1.0`, `30`)
- **api/handlers.go**: GET `/api/settings` 응답 및 PATCH `/api/settings` 요청 구조체에 4개 신규 설정 추가; 거래 시간 형식(`HH:MM`) 및 `start < end` 유효성 검사; 횡보 파라미터 범위 검사
- **cmd/server/main.go**: `parseHHMM()` 헬퍼 추가; 스케줄러 루프에서 매 틱 DB에서 시작/종료 시간 조회; 하드코딩 `915`, `1515` → 동적 값으로 교체; 엔진 시작 시 `mon.SetStagnationConfig()` 호출
- **monitor/monitor.go**: `Monitor` 구조체에 `stagnantSince` 맵 + `stagnationThresholdPct/DurationMin` 필드 추가; `SetStagnationConfig()` 메서드 추가; `HandlePrice()` default 케이스에 횡보 추적 로직 추가 (±N% 이내 진입 시 타이머 시작, 초과 시 리셋); `checkIndicators()` 에 `"stagnation"` 조건 추가; `Remove()`에 `stagnantSince` 정리 추가
- **Settings.jsx**: 섹션 5개로 재구성 (거래 제어 / 종목 선정 / 매수 설정 / 매도 설정 / AI 설정); 거래 시작·종료 시간 `<input type="time">` 추가; `SELL_CONDITIONS`에 `stagnation` 항목 추가; 횡보 감지 파라미터 UI (`stagnation` 조건 활성 시 표시)

## 2026-03-10 — 순위 상위 N개 설정 + 선정 실패 로그 개선

- **database/db.go**: `TradingSettings`에 `RankingTopN` 필드 추가; 기본값 `ranking_top_n=20` 설정; `trader_selection_logs`에 `fail_reason` 컬럼 ALTER TABLE 마이그레이션 추가
- **models/models.go**: `TraderSelectionLog`에 `FailReason string` 필드 추가
- **trader/engine.go**: `getRankings()` — 각 순위 타입별 필터 통과 후 `RankingTopN`개로 제한 (AND 교집합 이전 적용); `selectAndBuy()` — Claude 호출 전 로그 INSERT, 호출 후 `llm_result` UPDATE, LLM 오류/주문 전체 실패 시 `fail_reason` UPDATE
- **api/handlers.go**: `ranking_top_n` 설정 CRUD 추가 (GET/PATCH /api/settings); 선정 로그 쿼리에 `fail_reason` 컬럼 포함
- **Settings.jsx**: 순위 조회 설정 섹션에 "각 순위별 상위 종목 수 (1~30)" 입력 필드 추가
- **SelectionLogs.jsx**: 선정 실패 시 빨간 배지 + 실패 사유 텍스트 표시; 기존 "적합 종목 없음" 배지는 LLM이 정상 응답했으나 매수 불가 판단 시 유지

## 2026-03-10 — LLM 선정 로그 요청/응답 표시 개선, 주문내역 구분 컬럼 제거

- **SelectionLogs.jsx**: 요청(전달 후보 종목)·응답(Claude 순위) 섹션 명칭 명확히 분리; 기술 지표 컬럼 추가(MA5, MA20, 체결강도, 거래량증가율, 이격도D20); 응답 섹션 기본 펼침; "미체결" → "적합 종목 없음" 배지로 변경; 응답 비어있을 때 안내 문구 표시
- **Orders.jsx**: 정확하지 않은 구분(에이전트/수동) 컬럼 제거

## 2026-03-10 — 순위 조회 로그 기능 추가 + 모바일 UI 개선

- **database/db.go**: `trader_ranking_logs` 테이블 마이그레이션 추가 (timestamp, ranking_types, price_min, price_max, 타입별 count, intersection_count, error_message)
- **models/models.go**: `TraderRankingLog` 구조체 추가
- **trader/engine.go**: `getRankings()` — 교집합 계산 후 타입별 결과 수 및 교집합 수 DB INSERT; 오류 시에도 error_message 포함 기록
- **database/db.go**: `InsertRankingLog()`, `GetRankingLogs()` DB 메서드 추가 (30일 자동 삭제)
- **api/handlers.go**: `GetRankingLogs` 핸들러 추가 (`GET /api/logs/ranking?limit=50`)
- **api/router.go**: `/api/logs/ranking` 라우트 등록
- **RankingLogs.jsx**: 신규 페이지 — 타임스탬프·교집합 배지·가격범위 헤더, 타입별 카운트 펼치기
- **App.jsx**: `/ranking-logs` 라우트 및 "순위 조회 로그" 네비게이션 링크 추가; 반응형 햄버거 메뉴(md 미만 화면) 구현으로 모바일 UI 개선

## 2026-03-10 — 재시작 시 미체결 주문 이중 주문 방지

- **trader/engine.go**: `countPendingOrders()` 헬퍼 추가 — 오늘 AGENT 접수한 BUY PENDING 주문 수 조회
- **trader/engine.go**: `runCycle`에서 `currentCount = mon.Count() + countPendingOrders()` — 재시작 후 미체결 주문이 남아있을 때 추가 매수 차단
- **취약점**: 주문 접수 후 서버 재시작 시 `mon.Count()=0`으로 인식해 이중 주문이 가능했던 버그 수정

## 2026-03-10 — 트레이더 상태 MONITORING/SEARCHING 분리 (포지션 없을 때 표시 버그 수정)

- **trader/engine.go**: `SEARCHING` 상태 추가 — 포지션이 없어서 종목을 탐색 중일 때 표시
- **trader/engine.go**: `runCycle` 수정 — `currentCount < maxPositions` 구간 진입 시 `StateSearching`으로 전환, 포지션 보유 중일 때만 `StateMonitoring` 표시
- **Dashboard.jsx**: `SEARCHING` 상태를 녹색 + "(종목탐색)" 한국어 레이블로 표시
- **원인**: 포지션 0개여도 `MONITORING`이 표시되어 사용자가 시스템이 멈춘 것처럼 오인하는 문제

## 2026-03-09 — LLM 종목 선정 로그 화면 표시 기능 추가

- **database/db.go**: `trader_selection_logs` 테이블 마이그레이션 추가 (id, timestamp, sent_count, candidates, llm_result, selected_code, selected_reason)
- **models/models.go**: `TraderSelectionLog` 구조체 추가
- **trader/engine.go**: `selectAndBuy()` — Claude 호출 직후 선정 로그 DB INSERT, 체결 성공 후 selected_code·selected_reason UPDATE
- **api/handlers.go**: `GetSelectionLogs` 핸들러 추가 (`GET /api/logs/selection?limit=20`, 30일 이상 자동 삭제)
- **api/router.go**: `/api/logs/selection` 라우트 등록
- **SelectionLogs.jsx**: 신규 페이지 — Claude 순위 결과 목록, 전달 종목 펼치기 테이블
- **App.jsx**: `/selection-logs` 라우트 및 "선정 로그" 네비게이션 링크 추가

## 2026-03-09 — 대시보드 잔고 필드 정확도 개선

- **kis/client.go**: `InquireBalanceOutput2`에 `OrderableAmt`(`prvs_rcdl_excc_amt`), `StockEvalAmt`(`scts_evlu_amt`) 필드 추가
- **agent/balance.go**: `AccountBalance`에 `OrderableAmt` 필드 추가 및 파싱 로직 반영
- **api/handlers.go**: `GetServerStatus`의 `available_cash` 소스를 `dnca_tot_amt` → `prvs_rcdl_excc_amt`(D+2 주문가능금액 근사값)으로 변경
- **Dashboard.jsx**: 출금가능금액 카드 sub 레이블 "예수금" → "출금가능"으로 정리

## 2026-03-08 — WebSocket 재연결 안정화 & 매도 종목 정리

- **kis/websocket.go**: `reconnectMaxAttempts` 제한 제거 → 무제한 재연결 시도
- **kis/websocket.go**: ping/pong keepalive 추가 (30초 ping, 70초 pong wait) — 유휴 연결 끊김 자동 감지 후 재연결 트리거
- **monitor/monitor.go**: `PurgeStalePositions()` 추가 — KIS 실제 보유 종목과 비교해 이미 매도된 종목을 `monitored_positions`에서 제거
- **cmd/server/main.go**: WebSocket 연결 후 `PurgeStalePositions` 호출
- **api/handlers.go**: 수동 WS 연결 핸들러에서도 `PurgeStalePositions` 호출

## 2026-03-06 — 순위별 필터 설정 (거래량증가율/체결강도/순매수/이격도)

- **database/db.go**: `TradingSettings`에 순위 필터 필드 5개 추가, 기본값 자동 삽입
- **trader/engine.go**: `getRankings()`에 필터 로직 추가, `RankItem`에 지표 필드 전달
- **trader/claude.go**: `RankItem`에 `VolIncrRate`, `Strength`, `NetBuyQty`, `DisparityD20` 필드 추가
- **api/handlers.go**: `GetSettings`/`UpdateSettings` 신규 키 처리
- **frontend/Settings.jsx**: 순위 유형별 체크박스 아래 필터 입력창 표시

## 2026-03-06 — 매도 조건 우선순위 UI + CLAUDE.md 스킬 체크리스트 강화

- **frontend/Settings.jsx**: 매도 조건 체크박스 → 순서 변경 가능한 우선순위 리스트로 교체 (▲▼ 버튼, ＋/✕ 토글)
- **CLAUDE.md**: 스킬 트리거 조건 강화 — MANDATORY POST-TASK CHECKLIST 테이블 추가

## 2026-03-06 — 자율 트레이딩 엔진 (Claude API 기반) 도입

- **trader/claude.go** (신규): Claude API 클라이언트 — `SelectStock` (JSON 응답 파싱), `GenerateReport` (한국어 마크다운 일일 리포트)
- **trader/engine.go** (신규): 자율 트레이딩 엔진 — IDLE/SELECTING/ORDERING/WAITING_FILL/MONITORING 상태 머신, ExecCh 체결 대기(5분 타임아웃), 미체결 시 취소 후 재선정
- **database/db.go**: `reports` 테이블 추가, `TradingSettings` 구조체 + `GetTradingSettings()` 헬퍼, 신규 설정 키 12개 기본값 자동 삽입 (`INSERT OR IGNORE`)
- **models/models.go**: `Report` 구조체 추가
- **config/config.go**: `AnthropicAPIKey` 필드 추가 (`ANTHROPIC_API_KEY` 환경변수)
- **monitor/monitor.go**: `MonitoredEntry.SoldCh chan<- string` 추가 (매도 완료 시 엔진 신호), `StartIndicatorChecker()` 추가 (RSI과매수/MACD데드크로스 주기적 평가)
- **api/handlers.go**: `Handler.engine` 필드 + `SetEngine()`, `GetServerStatus`에 `trader_state` 추가, `GetSettings`/`UpdateSettings` 신규 설정 키 처리, `GetReports`/`GetReport` 핸들러 추가
- **api/router.go**: `/api/reports`, `/api/reports/:date` 라우트 추가
- **main.go**: `ClaudeClient` 초기화, `Engine` 생성 및 `handler.SetEngine` 주입, `runMarketScheduler` 확장 (09:00 거래 가능 확인, 09:15 엔진 시작 + 지표체커 시작, 15:15 엔진 중지, 15:20 일일 리포트 생성/저장)
- **frontend**: `Reports.jsx` 신규, `Settings.jsx` 신규 설정 섹션 6개 추가, `Dashboard.jsx` 트레이더 상태 표시, `Orders.jsx` 날짜 드롭다운 제거, `App.jsx` 리포트 라우트 추가

## 2026-03-05 — 목표/손절가 도달 시 자동 매도 + MQTT 페이로드 개선

- **monitor/monitor.go**: `HandlePrice` — 목표/손절 도달 시 KIS 시장가 매도(`executeSell`) 후 MQTT 발행. `isTest=true`면 매도 스킵 (테스트 전용)
- **monitor/monitor.go**: `executeSell()` 신규 — GetHoldings → PlaceSellOrder(시장가) → sellQty 반환
- **monitor/monitor.go**: `LiquidateAll` — 청산 시 현재가(근사 매도가) + 매도수량을 MQTT 페이로드에 포함
- **mqtt/publisher.go**: `AlertPayload`에 `sell_qty`, `profit_amount` 필드 추가; `PublishAlert` 시그니처에 `sellQty int` 파라미터 추가

## 2026-03-05 — Debug API + UI (장 외 테스트 기능)

- **mqtt/publisher.go**: `AlertPayload`에 `IsTest bool` (`is_test`) 필드 추가; `PublishAlert` 시그니처에 `isTest bool` 파라미터 추가
- **monitor/monitor.go**: `HandlePrice(stockCode, price, isTest bool)` 시그니처 변경; `PublishAlert` 3곳 — 정상 호출 `false`, Debug 주입 `true` 전달
- **api/handlers.go**: Debug 핸들러 5종 추가 — `DebugWSConnect`, `DebugWSDisconnect`, `DebugInjectPrice`, `DebugRegisterMonitor`, `DebugLiquidate`
- **api/router.go**: `/api/debug/*` 라우트 그룹으로 정리 (기존 `GET /balance` 포함 6종)
- **frontend/src/pages/Debug.jsx** (신규): WebSocket 제어, 포지션 등록, 가격 주입, LiquidateAll UI + 응답 로그 패널
- **frontend/src/App.jsx**: `/debug` 라우트 및 네비게이션 "디버그" 항목 추가

## 2026-03-04 — Phase 1: KIS WebSocket + 실시간 모니터링 + MQTT

- **kis/websocket.go** (신규): `WebSocketClient` — KIS WebSocket 연결/구독/재연결, AES-256-CBC 복호화, `PriceCh`/`ExecCh` 채널
- **mqtt/publisher.go** (신규): `Publisher` — Paho MQTT 클라이언트, `PublishAlert()` 포함, 브로커 미연결 시 graceful 처리
- **monitor/monitor.go** (신규): `Monitor` — 목표가/손절가 실시간 체크, DB 영속화, 서버 재시작 복구, `LiquidateAll()` (15:15)
- **kis/client.go**: `GetApprovalKey()` 추가 — WebSocket approval_key 발급 (`POST /oauth2/Approval`)
- **config/config.go**: `KISHTSID`, `MQTTBrokerURL`, `MQTTClientID` 환경변수 추가
- **models/models.go**: `Order`에 `TargetPct`/`StopPct` 추가, `MonitoredPosition` 구조체 신규
- **database/db.go**: `monitored_positions` 테이블, `orders.target_pct`/`stop_pct` 컬럼 마이그레이션 추가
- **agent/order.go**: `PlaceOrderRequest`에 `TargetPct`/`StopPct` 추가, INSERT에 포함
- **api/handlers.go**: `PlaceOrder` target/stop 수신 → 모니터 등록, `GetServerStatus`/`GetMonitorPositions`/`RemoveMonitorPosition` 신규
- **api/router.go**: `/api/server/status`, `/api/monitor/positions`, `/api/monitor/positions/:code` 라우트 등록
- **main.go**: MQTT/WebSocket/Monitor 초기화, 장운영 스케줄러(08:50/15:15/16:00), 폴링 주기 3분→5분 조정

## 2026-03-03 — KIS inquire-daily-ccld API 파라미터 버그 수정

- **kis/client.go**: `GetOrderHistory()` 쿼리에 누락된 필수 파라미터 3개 추가 (`INQR_DVSN_1`, `INQR_DVSN_3`, `EXCG_ID_DVSN_CD=ALL`)
- **kis/client.go**: 스펙에 없는 `CANC_YN` 파라미터 제거 — API 게이트웨이가 HTML 오류 반환하던 원인
- **kis/client.go**: `get()` 헬퍼에 `Content-Type: application/json; charset=utf-8` 헤더 추가 (GET 요청에도 명세 required)
- **영향**: 체결 완료 거래가 대기(PENDING) 상태로 표시되던 현상 해결

## 2026-03-02 — 장운영일 체크 + Order Sync 스케줄러 최적화

- **KIS client**: `HolidayInfo` DTO 및 `GetMarketHolidayInfo()` 메서드 추가 (`CTCA0903R`)
- **agent/market.go** (신규): `IsMarketOpen()` 함수 — KST 평일·장 운영 시간·KIS 영업일 3중 체크, 당일 메모리 캐시로 KIS 호출 하루 1회 제한
- **agent/history.go**: `StartOrderSyncScheduler` ticker 블록에 `IsMarketOpen()` 가드 추가 — 장 마감·공휴일 시 sync skip
- **api/handlers.go**: `GET /api/market/status` 핸들러 추가 (reason 필드: open/weekend/outside_hours/holiday/check_failed)
- **api/router.go**: `/api/market/status` 라우트 등록
- **main.go**: `import _ "time/tzdata"` 추가 — NCP Micro tzdata 미설치 환경 대비
