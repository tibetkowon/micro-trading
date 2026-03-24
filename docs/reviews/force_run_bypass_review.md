# 강제 실행(Force Run) 조건 우회 개선 — 코드 리뷰

## 개요

"강제 실행" 버튼을 눌러도 요일 제한, 매수 중단 시간대, 지수 하락 필터 등의 조건에 막혀 실제 매수가 진행되지 않는 문제를 수정했습니다.

`selectAndBuy` / `selectAndBuyUS` 함수에 `force bool` 파라미터를 추가하여, 강제 실행 시에는 스케줄·시장 상황 체크를 건너뛰도록 개선했습니다.

---

## Go 백엔드 해설

### 변경 전 구조

```go
func (e *Engine) selectAndBuy(ctx context.Context, settings database.TradingSettings) error {
    // 요일 체크
    if len(settings.TradingDays) > 0 { ... }
    // 매수 중단 시간대 체크
    if settings.BuyPauseStart != "" { ... }
    // 지수 하락 필터
    for _, code := range settings.IndexCodes { ... }
}

func (e *Engine) ForceRun(ctx context.Context) {
    go func() {
        e.selectAndBuy(ctx, settings) // 위 체크들에 막힘!
    }()
}
```

강제 실행(`ForceRun`)이 `selectAndBuy`를 그대로 호출하기 때문에, 일반 실행과 동일하게 모든 조건 체크를 통과해야 했습니다.

### 변경 후 구조

```go
func (e *Engine) selectAndBuy(ctx context.Context, settings database.TradingSettings, force bool) error {
    // 요일 체크 (강제 실행 시 건너뜀)
    if !force && len(settings.TradingDays) > 0 { ... }
    // 매수 중단 시간대 체크 (강제 실행 시 건너뜀)
    if !force && settings.BuyPauseStart != "" { ... }
    // 지수 하락 필터 (강제 실행 시 건너뜀)
    if !force && drop <= indexDropThreshold { ... }
}

// ForceRun: force=true 전달
func (e *Engine) ForceRun(ctx context.Context) {
    go func() {
        e.selectAndBuy(ctx, settings, true)  // 스케줄 체크 우회
    }()
}

// runCycle: force=false 전달 (기존 동작 유지)
if err := e.selectAndBuy(ctx, settings, false); err != nil { ... }
```

### Go 언어 개념 설명

**함수 시그니처 변경 (Signature)**
- `force bool`을 마지막 파라미터로 추가했습니다.
- Go에서는 파라미터를 추가하면 해당 함수를 호출하는 **모든 곳**을 수정해야 합니다 (컴파일 에러로 즉시 발견 가능).
- 이 함수는 `runCycle`(일반 사이클)과 `ForceRun`(강제 실행) 두 곳에서만 호출되므로 안전하게 수정 가능했습니다.

**`!force` 가드 패턴**
```go
if !force && len(settings.TradingDays) > 0 {
    // force=true이면 이 블록 전체를 건너뜀 (short-circuit evaluation)
}
```
- Go의 `&&` 연산자는 **단락 평가(short-circuit)**를 사용합니다.
- `!force`가 `false`이면 뒤의 조건은 평가하지 않고 즉시 다음 코드로 넘어갑니다.

**내부(unexported) 함수**
- `selectAndBuy`는 소문자로 시작하므로 `trader` 패키지 내부에서만 호출 가능합니다.
- 외부에 공개된 메서드는 `ForceRun`만이며, `force` 파라미터는 외부에 노출되지 않습니다.
- 사용자 입장에서는 "강제 실행 버튼" 하나만 누르면 되고, 내부 구현 복잡도는 숨겨집니다.

### 건너뛰는 체크 vs 유지하는 체크

| 체크 항목 | 강제 실행 시 | 이유 |
|-----------|-------------|------|
| 요일 제한 | **건너뜀** | 사용자가 의도적으로 주말에도 실행 가능하게 |
| 매수 중단 시간대 | **건너뜀** | 시간대 무관하게 즉시 실행 |
| 지수 하락 필터 | **건너뜀** | 시장 상황 무관하게 시도 |
| 자금 부족 | **유지** | 실제로 매수 불가한 실질적 사유 |
| 랭킹 결과 없음 | **유지** | 종목이 없으면 어차피 불가 |
| 이미 거래한 종목 제외 | **유지** | 중복 매수 방지 (의도적 안전장치) |

---

## 핵심 요약

1. **함수 파라미터로 동작 분기**: 조건부 로직을 별도 함수로 분리하는 것보다, `bool` 파라미터 하나로 동작을 제어하는 방식이 코드 중복 없이 깔끔합니다.

2. **호출부 책임 분리**: `ForceRun`은 `true`, `runCycle`은 `false`를 넘기는 방식으로 각 호출부가 자신의 의도를 명시적으로 표현합니다.

3. **"강제"의 범위 설정**: 모든 체크를 다 우회하는 것이 아니라, 스케줄·시장 조건만 우회하고 실질적 불가 사유(자금, 랭킹)는 유지합니다. 강제 실행이라도 돈이 없으면 매수할 수 없기 때문입니다.
