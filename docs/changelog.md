# Changelog

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
