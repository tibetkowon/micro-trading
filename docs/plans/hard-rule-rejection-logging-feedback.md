# Hard Rule 거부 사유 상세 로깅 및 자동 완화 피드백

## Goal

LLM 호출 결과가 "조건에 맞는 종목 없음"일 때, 어떤 Hard Rule이 몇 개 종목을 걸러냈는지 서버 사이드에서 사전 계산하여 로그에 기록한다. 또한 특정 룰이 최근 N 사이클의 80% 이상에서 발동했을 경우, 해당 룰의 임계값만 선택적으로 완화하는 인라인 피드백 루프를 추가한다.

---

## Requirements

### 기능 요건
1. `selectAndBuy`에서 Claude 호출 전, LLM 프롬프트의 Hard Rejection Rule 12개에 대해 각 종목이 위반 여부를 체크하고 `map[string]int` 형태로 집계한다.
2. Claude가 빈 배열(선택 실패)을 반환했을 때:
   - `trader_selection_logs.fail_reason`에 상위 위반 룰 정보를 포함한 상세 메시지 기록
   - 신규 컬럼 `hard_rule_stats`에 JSON `{"hard_strength_min":7,"hard_rsi_max":3,...}` 형태로 저장
3. `Engine`에 최근 N 사이클의 Hard Rule 통계를 보관하는 링 버퍼를 추가한다.
4. 링 버퍼 기준으로 특정 룰이 ≥80% 사이클에서 발동 시, 해당 룰의 임계값만 선택적으로 `EscalationStepPct`만큼 완화한다.
5. 신규 설정 키 3개: `hard_rule_feedback_enabled`, `hard_rule_feedback_window`(기본 5), `hard_rule_feedback_threshold_pct`(기본 80)
6. `optimization.go`의 no-trade 분석 요약에 per-rule 통계를 포함한다.

### 비기능 요건
- 서버 메모리 추가 사용 최소화 (링 버퍼는 최대 `hard_rule_feedback_window`개 항목)
- 기존 Adaptive Threshold / Escalation과 충돌 없음 (Rule-based 완화는 별도로 적용 후 max-wins로 처리)
- `hard_rule_feedback_enabled = false`(기본값)로 비활성이면 기존 동작 그대로

---

## Affected Files

### 수정
| 파일 | 주요 변경 내용 |
|------|--------------|
| `backend/internal/database/db.go` | ① `trader_selection_logs` 테이블에 `hard_rule_stats` 컬럼 추가 (ALTER TABLE), ② `TradingSettings` 구조체에 3개 필드 추가, ③ `GetTradingSettings` 파싱 및 기본값 INSERT |
| `backend/internal/trader/engine.go` | ① `evaluateHardRules` 함수 추가, ② `Engine` 구조체에 링 버퍼 필드 추가, ③ `pushRuleHit` / `detectBottleneckRule` / `relaxSpecificRule` 함수 추가, ④ `selectAndBuy`에서 Hard Rule 통계 수집·저장, ⑤ `runCycle`에서 피드백 루프 적용 |
| `backend/internal/report/optimization.go` | `noTradeSummary` 구조체에 `HardRuleStats map[string]int` 추가, per-cycle stats 집계 로직 추가 |

### 신규 생성 없음

---

## Implementation Phases

### Phase 1 — DB Schema & Settings (db.go)

1. `alterStmts`에 추가:
   ```sql
   ALTER TABLE trader_selection_logs ADD COLUMN hard_rule_stats TEXT NOT NULL DEFAULT '{}'
   ```
2. `TradingSettings` 구조체에 필드 추가:
   ```go
   HardRuleFeedbackEnabled      bool
   HardRuleFeedbackWindow       int
   HardRuleFeedbackThresholdPct float64
   ```
3. `GetTradingSettings`에서 `strconv.ParseBool` / `Atof` 파싱 추가
4. 기본값 `INSERT OR IGNORE`:
   - `hard_rule_feedback_enabled` = `"false"`
   - `hard_rule_feedback_window` = `"5"`
   - `hard_rule_feedback_threshold_pct` = `"80"`

---

### Phase 2 — Hard Rule 사전 평가 및 로깅 (engine.go)

`evaluateHardRules(items []RankItem, rules TradingRules) map[string]int` 신규 함수:
- Hard Rule 1~12에 대해 각 종목이 위반하는 룰 이름(`hard_strength_min`, `hard_rsi_max`, `hard_disparity_m5`, `hard_high_price_diff_max`, `hard_high_price_diff_vol`, `hard_ma_bearish`, `hard_open_price_diff_max`, `hard_macd_bearish`, `hard_high_formed_mins`, `hard_vol_vs_3avg`, `hard_relative_strength`)별로 위반 종목 수 집계
- Rule 1 (market_index_drop)은 전체 지수 조건이므로 미포함 (서버 사이드에서 이미 처리됨)

`selectAndBuy`에서:
1. Claude 호출 직전에 `ruleStats := evaluateHardRules(rankings, rules)` 호출
2. Claude가 빈 배열 반환 → `fail_reason`에 상위 2개 룰 포함:
   ```
   "LLM 오류: 조건에 맞는 종목 없음 — hard_strength_min:7/8, hard_rsi_max:3/8"
   ```
3. `hard_rule_stats` 컬럼에 `ruleStats` JSON 저장:
   ```sql
   UPDATE trader_selection_logs SET fail_reason=?, hard_rule_stats=? WHERE id=?
   ```
4. 정상 통과(후보 있음)일 때도 `hard_rule_stats` 저장 (참고용)

---

### Phase 3 — 피드백 루프 (engine.go)

`Engine` 구조체에 추가:
```go
ruleHitMu      sync.Mutex
ruleHitHistory []ruleHitRecord  // 링 버퍼 (최대 HardRuleFeedbackWindow개)
```

```go
type ruleHitRecord struct {
    candidateCount int
    hitByRule      map[string]int
}
```

헬퍼 함수 3개:
- `(e *Engine) pushRuleHit(r ruleHitRecord, window int)` — 링 버퍼에 추가 (window 초과 시 oldest 제거)
- `(e *Engine) detectBottleneckRule(total int, thresholdPct float64) (ruleName string, ok bool)` — 링 버퍼에서 특정 룰이 thresholdPct% 이상 사이클에서 발동하는지 확인
- `relaxSpecificRule(s database.TradingSettings, ruleName string, pct float64) database.TradingSettings` — 해당 룰의 임계값만 완화

`runCycle`에서:
```
선택 실패 후:
  if settings.HardRuleFeedbackEnabled:
    e.pushRuleHit(ruleHitRecord{...}, settings.HardRuleFeedbackWindow)
    if ruleName, ok := e.detectBottleneckRule(len(rankings), settings.HardRuleFeedbackThresholdPct); ok:
      settings = relaxSpecificRule(settings, ruleName, settings.EscalationStepPct)
      logger.Warn("engine: hard rule feedback — bottleneck detected, rule relaxed", {ruleName, pct, ...})
```

**`relaxSpecificRule` 매핑:**
| ruleName | 완화 방법 |
|----------|---------|
| `hard_strength_min` | `HardStrengthMin *= (1 - rate)` |
| `hard_rsi_max` | `HardRSIMax *= (1 + rate)` |
| `hard_disparity_m5` | `HardDisparityM5Max *= (1 + rate)`, `HardDisparityM5Min *= (1 + rate)` |
| `hard_high_price_diff_max` | `HardHighPriceDiffMax *= (1 - rate)` (음수, 절대값 축소) |
| `hard_ma_bearish` | 완화 불가 — 로그만 출력 |
| `hard_open_price_diff_max` | `HardOpenPriceDiffMax *= (1 + rate)` |
| `hard_macd_bearish` | `HardMACDBearishEnabled = false` |
| 기타 (high_formed_mins, vol_vs_3avg, relative_strength) | 해당 필드 비활성화 (0으로 설정) |

---

### Phase 4 — No-trade 분석 summary 개선 (optimization.go)

`noTradeSummary`에 `HardRuleStats map[string]int` 필드 추가.
`generateNoTradeReport`에서 `selLogs`의 `hard_rule_stats` JSON을 누적 집계하여 포함.

Claude `AnalyzeNoTradeDay` 프롬프트의 `selection_fail_reasons` 설명에 `hard_rule_*` 항목 해석 가이드 추가.

---

## Verification

1. `go build ./...` 성공
2. `npm run build` 성공
3. `engine_test.go` 있으면 `go test ./internal/trader/...` 통과
4. 수동 확인: 
   - `trader_selection_logs` 테이블에서 `hard_rule_stats` 컬럼 확인
   - `fail_reason`에 룰명:건수 포맷 포함 여부 확인
   - `hard_rule_feedback_enabled=true` 설정 후, 5사이클 연속 `hard_strength_min` 80% 이상 발동 시 로그에 "hard rule feedback — bottleneck detected" 출력 확인
