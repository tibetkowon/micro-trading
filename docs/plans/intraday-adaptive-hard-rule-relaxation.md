# Intraday Adaptive Hard Rule Relaxation — 점진적 룰별 완화 개선

## Goal

현재 `hard_rule_feedback_enabled`(기본값 `false`)는 구현은 되어 있으나 비활성 상태이며,
`hard_high_formed_mins`·`hard_vol_vs_3avg` 두 룰은 bottleneck 탐지 시 임계값을 단계적으로 완화하는 대신
완전 비활성화(값=0)로 처리된다.

이를 개선하여:
1. 두 룰의 완화 방식을 **점진적 수치 완화**(+15분 / -10%)로 변경한다.
2. 피드백 루프 기본 파라미터를 실전 유용성 기준으로 조정한다 (window=10, threshold=70%).
3. 프론트엔드 설정 UI에서 `hard_rule_feedback_*` 키를 노출하여 사용자가 직접 제어할 수 있도록 한다.

---

## Requirements

### 기능 요건
1. `relaxSpecificRule`의 `hard_high_formed_mins` 케이스: `HardHighFormedMinsMax += 15` (분 단위 고정 증가)
2. `relaxSpecificRule`의`hard_vol_vs_3avg` 케이스: `HardVolVs3AvgRatioMin *= (1 - rate)` (비율 감소, 0 도달 시 0.1 floor)
3. DB 기본값 변경:
   - `hard_rule_feedback_window`: `"5"` → `"10"`
   - `hard_rule_feedback_threshold_pct`: `"80"` → `"70"`
   - `hard_rule_feedback_enabled`: 기본값 유지 (`"false"`), UI에서 켤 수 있도록
4. 완화 로그에 이전값·이후값 모두 출력:
   ```
   "engine: hard rule feedback — rule relaxed" {rule: "hard_high_formed_mins", before: 60, after: 75, window: 10}
   ```

### 비기능 요건
- 기존 Escalation 완화와 독립적으로 적용 (max-wins 원칙 유지)
- `hard_high_formed_mins` 상한: 최대 180분 (하루 3시간 초과 시 비활성화)
- `hard_vol_vs_3avg` 하한: 0.1 (0으로 완전 비활성화 방지)

---

## Affected Files

| 파일 | 변경 내용 |
|------|---------|
| `backend/internal/trader/engine.go` | `relaxSpecificRule` 내 2개 케이스 수정, 완화 로그 before/after 추가 |
| `backend/internal/database/db.go` | 기본값 `hard_rule_feedback_window`→10, `threshold_pct`→70 변경 |
| `frontend/src/components/Settings*.tsx` (또는 관련 설정 컴포넌트) | `hard_rule_feedback_*` 3개 키 UI 노출 |

---

## Implementation Phases

### Phase 1 — engine.go: relaxSpecificRule 수정
```go
case "hard_high_formed_mins":
    before := s.HardHighFormedMinsMax
    s.HardHighFormedMinsMax += 15
    if s.HardHighFormedMinsMax > 180 {
        s.HardHighFormedMinsMax = 0 // 180분 초과 시 비활성화
    }
    // 로그: before → after

case "hard_vol_vs_3avg":
    before := s.HardVolVs3AvgRatioMin
    s.HardVolVs3AvgRatioMin *= (1 - rate)
    if s.HardVolVs3AvgRatioMin < 0.1 {
        s.HardVolVs3AvgRatioMin = 0.1
    }
    // 로그: before → after
```

완화 로그 구조에 `before`·`after` 필드 추가:
```go
logger.Warn("engine: hard rule feedback — rule relaxed", map[string]any{
    "rule":   bottleneck,
    "before": beforeVal,
    "after":  afterVal,
    "pct":    settings.EscalationStepPct,
    "window": settings.HardRuleFeedbackWindow,
})
```

### Phase 2 — db.go: 기본값 변경
`INSERT OR IGNORE` 기본값 변경:
```go
{"hard_rule_feedback_window", "10"},
{"hard_rule_feedback_threshold_pct", "70"},
```
> 기존 DB는 `INSERT OR IGNORE`라 자동 변경되지 않음. 마이그레이션 불필요 — 사용자가 UI에서 직접 조정.

### Phase 3 — 프론트엔드 설정 UI 노출
현재 Settings 페이지에서 `hard_rule_feedback_*` 3개 키를 노출:
- `hard_rule_feedback_enabled`: Toggle (기본 OFF)
- `hard_rule_feedback_window`: Number input (1~20, step=1)
- `hard_rule_feedback_threshold_pct`: Number input (50~100, step=5)

---

## Verification

1. `go build ./...` 성공
2. `npm run build` 성공
3. `relaxSpecificRule` 단위 테스트 (있으면): `hard_high_formed_mins` 60→75, `hard_vol_vs_3avg` 1.5→1.35(rate=10%) 검증
4. 로그에서 before/after 값 확인
