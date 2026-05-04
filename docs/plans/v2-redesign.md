# v2 전면 재설계 계획서

## 배경 및 목표

현재 v1은 Claude LLM 기반 종목 선정을 핵심으로 설계되었으나, 실제 운영에서 다음 문제가 확인됨:
- LLM 호출로 인한 토큰 비용 낭비 (거래 없이 API 소모)
- 하드 필터만으로 충분히 종목 선별 가능
- 설정 항목 100개 이상으로 관리 복잡
- UI 페이지 분산으로 정보 파악 어려움
- SQLite 특성상 외부에서 데이터 조회 불가

v2는 **LLM 없이 수치 기반 자동 거래**에 집중하고, 서버·DB·UI를 전면 재설계한다.

---

## 주요 결정 사항

| 항목 | v1 | v2 |
|---|---|---|
| 종목 선정 | Claude LLM | 점수 시스템 (규칙 기반) |
| DB | SQLite | Firebase Firestore |
| 프론트 호스팅 | GCP VM (정적 파일 서빙) | Firebase Hosting |
| 서버 | NCP Micro (1GB) | GCP e2-small (2GB, 서울) |
| VM 운영 | 24/7 | 장중만 가동 (자동 시작/종료) |
| DNS | 고정 IP | DuckDNS (자동 IP 업데이트) |
| 알림 | 없음 | Discord Webhook |
| UI | 10페이지 | 5페이지 통합 |
| 마켓 | KR + US (미완) | KR 집중 |

---

## 아키텍처

### 거래 플로우

```
[v1]
순위 조회 → 하드 필터 → Claude LLM 선정 → 주문 → 모니터링

[v2]
순위 조회 → 하드 필터 → 점수 계산 → 상위 N종목 자동 주문 → 모니터링
```

### 인프라 구성

```
Firebase Hosting
  └── React 앱 (항상 접속 가능, CDN, HTTPS 자동)
      ├── VM 켜짐: 백엔드 API 호출 → 실시간 기능 사용
      └── VM 꺼짐: Firestore 직접 조회 → 데이터 읽기만 가능

GCP e2-small (asia-northeast3, 서울)
  └── Go 백엔드 (장중만 가동)
      ├── 08:30 자동 시작 (GCP 인스턴스 스케줄)
      └── 16:10 자동 종료

DuckDNS
  └── my-trading.duckdns.org → GCP VM IP 자동 갱신 (5분 주기)
      └── React → 백엔드 API 호출 시 이 도메인 사용

Firebase Firestore (무료 티어)
  ├── 모든 데이터 저장
  ├── React에서 직접 조회 가능 (VM 꺼져도)
  └── Firebase Console에서 모바일 포함 언제든 조회

Discord Webhook
  └── VM 시작/에러/거래체결/일일요약 알림
```

### VM 꺼짐/켜짐 시 기능 비교

| 기능 | VM 켜짐 | VM 꺼짐 |
|---|---|---|
| 주문/포지션/로그 조회 | ✅ | ✅ (Firestore 직접) |
| 설정 조회 | ✅ | ✅ (Firestore 직접) |
| 실시간 가격 | ✅ | ❌ |
| 주문 실행 / 강제 매도 | ✅ | ❌ |
| 순위 조회 | ✅ | ❌ |

### 비용

```
GCP e2-small 스케줄 운영 (08:30~16:10, 평일 22일 기준)
  VM: 165시간 × $0.0134 = $2.21/월
  디스크 HDD 20GB:        $0.80/월
  합계:                   $3.01/월
  Google 크레딧 $10 차감: $0/월 (완전 무료)

Firebase Hosting: 무료 티어
Firebase Firestore: 무료 티어
DuckDNS: 무료
Discord: 무료
```

### DB 컬렉션 설계 (Firestore)

| 컬렉션 | 내용 | 문서 ID |
|---|---|---|
| `settings` | 전체 설정 (단일 문서) | `config` |
| `orders` | 주문 내역 | `{timestamp_micro}` |
| `monitored_positions` | 보유 포지션 | `{stock_code}` |
| `scan_logs` | 스캔 결과 (순위+필터+점수) | `{timestamp_micro}` |
| `service_logs` | 시스템 로그 (CRITICAL/WARN/INFO) | `{timestamp_micro}` |
| `tokens` | KIS 토큰 | `current` |
| `stock_masters` | 종목 마스터 | `{stock_code}` |
| `trade_reports` | 거래 리포트 (매수→매도 lifecycle) | `{timestamp_micro}` |
| `daily_reports` | 일별 요약 | `{YYYY-MM-DD}` |

---

## 점수 시스템 설계

하드 필터를 통과한 종목에 지표별 점수를 합산해 순위를 매기고,
`min_score_threshold` 이상인 종목 중 상위 `max_positions`개 자동 매수.

| 지표 | 점수 기준 | 가중치 설정 키 |
|---|---|---|
| 체결강도 | 120 이상 고점수, 100 미만 0점 | `score_weight_strength` |
| RSI | 40~60 구간 고점수, 70 이상 0점 | `score_weight_rsi` |
| MACD | 골든크로스 방향이면 고점수 | `score_weight_macd` |
| 매수호가비율 | 1.5 이상 고점수 | `score_weight_bidask` |
| VWAP 이격도 | 적정 범위 내 고점수 | `score_weight_vwap` |
| 거래량 증가율 | 급증이면 고점수 | `score_weight_volume` |

```
min_score_threshold   — 매수 최소 기준 점수 (0=비활성)
score_weight_*        — 지표별 가중치 (6개)
```

---

## 설정 항목

총 73개. v1 대비 약 1/3 감소 (100개 → 73개).

### Tab 1 — 거래 조건 (26개)

포지션 모니터링 및 수익/손실 관리 기준.
`fee_and_tax_rate` + `slippage_pct`는 왕복(매수+매도) 합산 기준으로 손익 계산에 사용.

| 설정 키 | 이름 | 기본값 | 범위/옵션 |
|---|---|---|---|
| `trading_enabled` | 자동 거래 활성화 | true | ON/OFF |
| `max_positions` | 최대 동시 보유 종목 수 | 1 | 1~5 |
| `order_amount_pct` | 가용자금 대비 주문 비율 | 95% | 10~100% |
| `daily_max_loss_pct` | 일일 최대 손실 한도 | 0 (비활성) | 0~20% |
| `daily_target_profit_pct` | 일일 목표 수익 달성 시 매수 중단 | 0 (비활성) | 0~20% |
| `stock_take_profit_pct` | 주식 익절 기준 | 1.5% | 0.1~20% |
| `stock_stop_loss_pct` | 주식 손절 기준 | 1.0% | 0.1~20% |
| `etf_take_profit_pct` | ETF 익절 기준 | 0.5% | 0.1~10% |
| `etf_stop_loss_pct` | ETF 손절 기준 | 0.4% | 0.1~10% |
| `fee_and_tax_rate` | 수수료 및 제세금 비율 (왕복 기준) | 0.25% | 0~1% |
| `slippage_pct` | 예상 슬리피지 비율 | 0.2% | 0~1% |
| `trailing_trigger_pct` | 트레일링 스탑 활성 기준 수익률 | 0 (비활성) | 0~10% |
| `trailing_stop_pct` | 최고가 대비 하락 허용폭 | 1.0% | 0.1~10% |
| `partial_tp_enabled` | 부분 익절 활성화 | false | ON/OFF |
| `partial_tp_pct` | 부분 익절 트리거 수익률 | 1.0% | 0.1~10% |
| `partial_tp_ratio` | 부분 익절 매도 비율 | 50% | 10~90% |
| `partial_tp_raise_stop` | 부분 익절 후 손절가 → 매수가로 상향 | true | ON/OFF |
| `sell_conditions` | 매도 조건 우선순위 | target, stop | 순서 설정 |
| `indicator_check_interval_min` | 지표 체크 주기 | 1분 | 1~15분 |
| `indicator_rsi_sell_threshold` | RSI 과매수 시 매도 기준값 | 70 | 50~90 |
| `indicator_macd_bearish_sell` | MACD 데드크로스 시 매도 | false | ON/OFF |
| `stagnation_threshold_pct` | 횡보 판단 변동폭 | 1.0% | 0.1~5% |
| `stagnation_duration_min` | 횡보 지속 시간 | 30분 | 5~120분 |
| `stagnation_partial_exit_enabled` | 횡보 시 부분 청산 | false | ON/OFF |
| `stagnation_bid_ask_sell_threshold` | 횡보 즉시 전량 청산 기준 (매수호가비율) | 1.0 | 0.5~2.0 |
| `max_holding_time_min` | 최대 보유 허용 시간 (초과 시 시장가 청산) | 0 (비활성) | 0~120분 |

### Tab 2 — 하드 필터 (18개)

스캔 사이클 2단계. 하나라도 미달하면 매수 제외.
> ⚠️ `hard_max_spread_pct` 구현 시 호가 API 별도 호출 필요 여부 확인 필요.

| 설정 키 | 이름 | 기본값 | 범위/옵션 |
|---|---|---|---|
| `hard_exclude_stock_types` | 제외 종목군 (우선주/스팩/환기) | 우선주, 스팩, 환기 | 복수 선택 |
| `hard_max_spread_pct` | 최대 허용 호가 스프레드 | 1.0% | 0.1~5% |
| `hard_rsi_max` | RSI 상한 (초과 시 제외) | 70 | 50~90 |
| `hard_strength_min` | 체결강도 하한 (미달 시 제외) | 100 | 80~150 |
| `hard_disparity_m5_min` | 5분봉 이격도 하한 | -1.5% | -10~0% |
| `hard_disparity_m5_max` | 5분봉 이격도 상한 | 3.0% | 0~10% |
| `hard_high_price_diff_min` | 고가 대비 최솟값 | -5.0% | -20~0% |
| `hard_high_price_diff_max` | 고가 대비 최댓값 | -0.5% | -10~0% |
| `hard_open_price_diff_max` | 시가 대비 상승률 상한 | 15.0% | 5~30% |
| `hard_macd_bearish_enabled` | MACD 데드크로스 종목 제외 | false | ON/OFF |
| `hard_high_formed_mins_max` | 고점 형성 후 경과 시간 상한 | 0 (비활성) | 0~120분 |
| `hard_vol_vs_3avg_ratio_min` | 거래량 회복 비율 최솟값 | 0 (비활성) | 0~3.0 |
| `hard_relative_strength_min` | 시장 대비 상대강도 최솟값 | 0 (비활성) | -5~10% |
| `hard_prev_vol_ratio_max` | 전봉 대비 거래량 비율 상한 | 1.2 | 0.5~5.0 |
| `min_trading_value` | 최소 거래대금 (하드필터) | 0 (비활성) | 0~100억 |
| `min_market_cap` | 최소 시가총액 | 0 (비활성) | 0~1000억 |
| `index_codes` | 지수 필터 (코스피/코스닥) | [] (비활성) | 복수 선택 |
| `index_drop_threshold_pct` | 지수 하락 시 매수 중단 기준 | -1.0% | -5~0% |

### Tab 3 — 점수 시스템 (8개)

하드 필터 통과 종목 전체 대상. 가중치 합계 = 100.
> ⚠️ `score_weight_imbalance` 구현 시 호가 잔량 데이터 출처 확인 필요.

| 설정 키 | 이름 | 기본값 | 범위/옵션 |
|---|---|---|---|
| `min_score_threshold` | 매수 최소 기준 점수 | 0 (비활성) | 0~100점 |
| `score_weight_strength` | 체결강도 가중치 | 25 | 0~100 |
| `score_weight_rsi` | RSI 가중치 | 20 | 0~100 |
| `score_weight_macd` | MACD 가중치 | 10 | 0~100 |
| `score_weight_bidask` | 매수호가비율 가중치 (가격 비율) | 15 | 0~100 |
| `score_weight_imbalance` | 호가 잔량 비율 가중치 (물량 비율) | 15 | 0~100 |
| `score_weight_vwap` | VWAP 이격도 가중치 | 10 | 0~100 |
| `score_weight_volume` | 거래량 증가율 가중치 | 5 | 0~100 |

### Tab 4 — 순위 조회 (15개)

스캔 사이클 1단계. `ranking_trading_value_min`은 API 레벨 사전 필터.

| 설정 키 | 이름 | 기본값 | 범위/옵션 |
|---|---|---|---|
| `ranking_types` | 순위 유형 | volume, strength | 복수 선택 |
| `ranking_condition` | 종목 교집합/합집합 | OR | AND / OR |
| `ranking_trading_value_min` | API 사전 조회용 최소 거래대금 | 50억 | 0~500억 |
| `ranking_price_min` | 최소 주가 | 5,000원 | 0~100만 |
| `ranking_price_max` | 최대 주가 | 100,000원 | 0~100만 |
| `ranking_exchanges` | 거래소 | KOSPI, KOSDAQ | 복수 선택 |
| `ranking_top_n` | 유형별 상위 N개 | 20 | 5~100 |
| `ranking_volume_min_incrrate` | 거래량 증가율 최솟값 | 0 (비활성) | 0~500% |
| `ranking_strength_min` | 체결강도 최솟값 | 100 | 80~200 |
| `ranking_fluctuation_min_rate` | 등락률 하한 | 0 (비활성) | -30~30% |
| `ranking_fluctuation_max_rate` | 등락률 상한 | 0 (비활성) | -30~30% |
| `ranking_vi_kind_code` | VI 종류 필터 | 전체 | 전체/정적/동적 |
| `ranking_volume_blng_cls_codes` | 거래량 분류 기준 | 증가율, 거래대금 | 복수 선택 |
| `rank_lease_duration_min` | 순위 이탈 종목 유지 시간 | 5분 | 0~60분 |
| `hard_watch_symbols` | 강제 감시 종목 | [] | 종목코드 목록 |

### Tab 5 — 스케줄 (6개)

| 설정 키 | 이름 | 기본값 | 범위/옵션 |
|---|---|---|---|
| `trading_start_time` | 거래 엔진 시작 시간 | 09:15 | HH:MM |
| `trading_end_time` | 거래 엔진 종료 시간 | 15:15 | HH:MM |
| `force_liquidate_on_end` | 종료 시 보유 포지션 전량 청산 | true | ON/OFF |
| `trading_days` | 거래 요일 | 월~금 | 복수 선택 |
| `buy_pause_start` | 매수 중단 시작 | 11:00 | HH:MM |
| `buy_pause_end` | 매수 중단 종료 | 14:00 | HH:MM |

### 제거 항목 (v1 → v2)

```
AI 관련
  claude_model / max_claude_candidates
  adaptive_threshold_* (3개)
  escalation_* (4개)
  market_phase_relax_* (3개)
  hard_rule_feedback_* (3개)

통합/정리
  take_profit_pct / stop_loss_pct  → stock_*/etf_* 로 대체
  stock_tax_rate                   → fee_and_tax_rate 로 통합
  min_expected_profit_pct          → fee_and_tax_rate + slippage_pct 로 대체
```

---

## UI 재설계 (5페이지)

### 1. 대시보드
- 현재 보유 포지션 + 실시간 손익률
- 오늘 실현 손익
- 시스템 상태 (엔진 ON/OFF, WebSocket 연결, 마켓 상태)
- 최근 에러 요약

### 2. 거래 현황
- 주문 내역 (오늘 기본, 날짜 검색)
- 거래 리포트 (매수→매도 lifecycle)
- 일별 요약 (날짜별 손익 차트)

### 3. 스캐너 로그
- 오늘 날짜 기본 로드 (날짜 검색 가능)
- 스캔 목록: 시각 + 통과종목수 + 거부종목수 (기본 접힘)
- 펼치면: 종목별 점수 및 필터 통과/거부 상세

### 4. 설정
- 탭 구성: 거래조건 / 하드필터 / 점수시스템 / 스케줄
- 슬라이더·토글 위주 직관적 UX

### 5. 로그
- 서비스 로그 (CRITICAL / WARN / INFO)
- KIS API 에러 로그
- 날짜 필터, 레벨 필터

---

## Discord 알림 항목

| 이벤트 | 레벨 |
|---|---|
| VM 시작 (현재 IP 포함) | INFO |
| 매수 체결 | INFO |
| 매도 체결 + 손익 | INFO |
| WebSocket 재연결 | WARN |
| KIS API 에러 | WARN |
| 일일 손익 요약 (15:20) | INFO |
| 시스템 오류 (거래 불가 상태) | CRITICAL |

---

## 구현 단계

### Phase 0 — 사전 정리 (코딩 전)
- [ ] docs/ 정리 (완료된 계획서, 리뷰 문서 삭제)
- [ ] Firebase 프로젝트 확인 (이미 생성 완료)
- [ ] Discord Webhook URL 발급

### Phase 1 — 백엔드 기반
- [ ] `go.mod` 의존성 정리 (sqlite3 제거, firestore 추가)
- [ ] `internal/database/` — Firestore 클라이언트로 전면 재작성
- [ ] `internal/models/` — firestore 태그 추가, US 관련 필드 제거
- [ ] `internal/config/` — 환경변수 추가 (FIREBASE_CREDENTIALS_JSON, DISCORD_WEBHOOK_URL, FRONTEND_ORIGIN)
- [ ] `internal/notify/` — Discord Webhook 패키지 신규 작성
- [ ] CORS 설정 — Firebase Hosting 도메인 허용

### Phase 2 — 거래 엔진 재작성
- [ ] `internal/trader/claude.go` 제거
- [ ] `internal/report/optimization.go` 제거
- [ ] `internal/scorer/` — 점수 시스템 신규 작성
- [ ] 거래 플로우 재작성: 순위 → 하드필터 → 점수 → 자동 주문
- [ ] 스캔 로그 Firestore 저장

### Phase 3 — 기존 패키지 정리
- [ ] `internal/agent/` — US 마켓 분기 제거
- [ ] `internal/monitor/` — Firestore 연동
- [ ] `internal/mst/` — Firestore 연동
- [ ] `internal/kis/token.go` — Firestore 연동

### Phase 4 — UI 전면 재작성
- [ ] Firebase SDK 추가 (Firestore 직접 조회용)
- [ ] VM 꺼짐 감지 → Firestore 직접 조회 fallback 처리
- [ ] 대시보드: 실시간 포지션 + 손익
- [ ] 거래 현황: 주문 + 리포트 통합
- [ ] 스캐너 로그: 당일 기본 + 날짜 검색 + 접기/펼치기
- [ ] 설정: 탭 구조 + 슬라이더 UX
- [ ] 로그: 레벨 필터 + 날짜 필터

### Phase 5 — 인프라 구성 (코드 완료 후 별도 진행)

> 코드 작업 완료 후 요청 시 단계별 안내 제공

- [ ] **GCP**
  - [ ] e2-small 인스턴스 생성 (asia-northeast3)
  - [ ] 인스턴스 시작/종료 스케줄 설정 (08:30 / 16:10, 평일)
  - [ ] 방화벽 규칙 설정 (백엔드 포트)
  - [ ] VM 시작 스크립트 작성 (앱 자동 실행)

- [ ] **DuckDNS**
  - [ ] 도메인 생성
  - [ ] VM 크론잡 등록 (5분 주기 IP 갱신)

- [ ] **Firebase**
  - [ ] Firestore Security Rules 설정 (프론트 읽기 허용)
  - [ ] Firebase Hosting 배포 설정
  - [ ] React 빌드 → `firebase deploy`

- [ ] **Discord**
  - [ ] 서버 채널 생성
  - [ ] Webhook URL 발급 → `.env`에 등록

- [ ] **검증**
  - [ ] DuckDNS 갱신 확인
  - [ ] Firebase Hosting 접속 확인
  - [ ] VM 꺼진 상태에서 Firestore 데이터 조회 확인
  - [ ] Discord 알림 테스트
  - [ ] 장 중 실거래 테스트 (1~2주 모니터링)
  - [ ] NCP 서버 종료

---

## 제거 대상

```
backend/internal/trader/claude.go
backend/internal/report/optimization.go
frontend/src/pages/OptimizationReports.jsx
frontend/src/pages/SelectionLogs.jsx
```

---

## 위험 및 대응

| 위험 | 대응 |
|---|---|
| Firestore SUM 집계 없음 | Go 코드에서 계산, 데이터 양 적어 성능 무관 |
| VM 시작 후 DuckDNS 갱신 5분 소요 | 거래 시작(09:15)보다 충분히 일찍 시작(08:30) |
| Firebase Hosting → 백엔드 CORS | FRONTEND_ORIGIN 환경변수로 허용 도메인 관리 |
| 점수 시스템 초기 튜닝 필요 | 2주 모니터링 후 가중치 조정 |
| KIS WebSocket 레이턴시 (GCP 서울) | NCP 대비 차이 미미, 동일 리전 |
