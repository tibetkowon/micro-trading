# Changelog

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
