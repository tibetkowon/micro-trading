# 계획: 눌림목 모멘텀 스코어링 + 단계적 횡보 청산 로직

**날짜:** 2026-04-09  
**브랜치:** `claude/momentum-scoring-exit-logic-eM44l`

---

## 목표

오늘 비츠로셀(고수익)과 온코닉테라퓨틱스(손실)의 결과 차이를 반영하여 두 가지 로직을 구현한다.

1. **복합 모멘텀 스코어링**: 진입 신호의 질을 수치화해 저품질 눌림목 신호를 사전 필터링
2. **단계적 횡보 청산**: 횡보 감지 시 무조건 전량 청산 대신 bid_ask_ratio를 보고 반액 or 즉시 전량 청산

---

## 요구사항

### Feature 1: 복합 모멘텀 스코어 (MomentumScore)

- `bid_ask_ratio`, `체결강도(strength)`, `prev_volume_ratio` 세 지표를 가중 합산
- 점수 범위: 0 ~ 100
- 기본 가중치: bid_ask 40 / strength 40 / vol_decline 20
- `momentum_score_min` 설정값(기본 0 = 비활성) 이상일 때만 Claude에 전달
- `MomentumScore` 필드를 `RankItem`에 추가해 Claude 프롬프트에도 노출

**점수 공식:**
```
bid_ask_score  = min(bid_ask_ratio / 5.0, 1.0) × 40
strength_score = min(max(strength_pct - 100, 0) / 100, 1.0) × 40
vol_score      = max(1.0 - prev_volume_ratio, 0) × 20
momentum_score = bid_ask_score + strength_score + vol_score
```

**예시 (오늘 결과 검증):**
| 종목 | bid_ask | strength | prev_vol | 점수 | 결과 |
|------|---------|----------|----------|------|------|
| 비츠로셀 | 6.91 | 198% | 0.08 | 97.6 | 성공 ✅ |
| 온코닉 | 2.62 | 128% | 0.14 | 49.4 | 실패 ❌ |

threshold=60 설정 시 온코닉 자동 필터링됨.

### Feature 2: 단계적 횡보 청산

현재: 횡보 N분 지속 → 전량 청산  
변경: 횡보 N분 지속 시 bid_ask_ratio 조회 후:

| 조건 | 처리 |
|------|------|
| bid_ask_ratio < threshold (기본 1.0) | 즉시 **전량 청산** ("횡보 중 매도우세 전환") |
| bid_ask_ratio ≥ threshold + PartialExitDone=false | **절반 청산** 후 횡보 타이머 리셋, PartialExitDone=true |
| bid_ask_ratio ≥ threshold + PartialExitDone=true | **전량 청산** (두 번째 횡보 경고) |

`stagnation_partial_exit_enabled=false` 이면 기존 전량 청산 동작 유지 (하위 호환).

---

## 영향 파일

| 파일 | 변경 내용 |
|------|----------|
| `backend/internal/database/db.go` | TradingSettings 필드 3개 추가, defaultSettings 3개 추가, GetTradingSettings 파싱 추가 |
| `backend/internal/trader/claude.go` | RankItem에 MomentumScore 필드 추가, allowedSettingsKeys에 신규 키 추가 |
| `backend/internal/trader/engine.go` | BidAskRatio fetch 직후 MomentumScore 계산 + 필터 로직 추가 |
| `backend/internal/monitor/monitor.go` | Monitor 구조체에 필드 추가, SetStagnationExitConfig(), executePartialSell(), checkIndicators 수정 |
| `backend/cmd/server/main.go` | mon.SetStagnationExitConfig() 호출 추가 |
| `docs/db_schema.md` | 신규 설정 키 3개 문서화 |
| `docs/changelog.md` | 변경 이력 추가 |

**새 파일 없음.**

---

## 구현 단계

### Phase 1: DB 스키마 & 설정 (database/db.go)

신규 설정 키:
- `momentum_score_min` (float, 기본 0) — 모멘텀 스코어 최솟값. 0=비활성
- `stagnation_partial_exit_enabled` (bool, 기본 false) — 단계적 청산 활성화
- `stagnation_bid_ask_sell_threshold` (float, 기본 1.0) — 즉시 전량청산 bid_ask 임계값

### Phase 2: RankItem 스코어 필드 (trader/claude.go)

`MomentumScore float64 json:"momentum_score,omitempty"` 추가 및 allowedSettingsKeys 확장.

### Phase 3: 스코어 계산 + 필터 (trader/engine.go)

BidAskRatio 병렬 조회(line 710~730) 직후, MomentumScore 계산 후 모든 RankItem에 설정.
`MomentumScoreMin > 0` 이면 미달 종목 제거.

### Phase 4: 단계적 횡보 청산 (monitor/monitor.go)

- `MonitoredEntry`에 `PartialExitDone bool` 추가
- `Monitor` 구조체에 `stagnationPartialExitEnabled`, `stagnationBidAskSellThreshold` 추가
- `SetStagnationExitConfig()` 추가
- `executePartialSell()` 추가 (holdings 조회 후 절반 수량 매도)
- `checkIndicators` stagnation case 분기 로직 추가

### Phase 5: 서버 초기화 (cmd/server/main.go)

`mon.SetStagnationExitConfig(settings.StagnationPartialExitEnabled, settings.StagnationBidAskSellThreshold)` 호출.

### Phase 6: 검증 및 문서화

- `go build ./...`
- `docs/db_schema.md`, `docs/changelog.md` 업데이트
- 커밋 & 푸시

---

## 검증

- `go build ./...` 성공 확인
- 점수 계산 로직 수동 검증 (비츠로셀 97.6 / 온코닉 49.4)
- `stagnation_partial_exit_enabled=false` 시 기존 동작 유지 확인
