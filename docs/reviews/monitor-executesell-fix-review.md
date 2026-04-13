# 코드 리뷰: 자동 매도 실패 시 포지션 보존 버그 수정 + Hard Rejection 룰 추가

## 개요

두 가지 개선이 하나의 릴리즈에 포함됩니다.

1. **버그 수정**: `executeSell` 실패 시에도 `Remove`가 호출되어 실제 잔고가 남아있는데 모니터링이 해제되는 문제
2. **기능 추가**: Claude 종목 선정 시 MACD bearish 진입 차단 및 고점 경과 시간 상한 Hard Rejection 룰

---

## Go 백엔드 해설

### 버그 원인 — 반환값 미사용 패턴

기존 코드의 문제는 함수 반환값을 버리는 패턴에 있었습니다.

```go
// 기존: 실패 여부 관계없이 항상 Remove
case price >= pos.TargetPrice:
    m.executeSell(stockCode, pos, "목표가 도달")  // 반환값 무시!
    m.Remove(context.Background(), stockCode)    // 항상 실행
```

`executeSell`이 KIS API 500 에러로 실패해도 `Remove`가 호출되어:
- 모니터 `positions` 맵에서 종목 제거
- 실제 계좌에는 잔고가 남아있음
- 이후 목표가/손절가 도달해도 자동 매도 불가

### 수정 — 반환값 3단계 구분

```go
// 수정 후: executeSell 반환값 의미
//  > 0  → 매도 주문 성공 (qty 반환) → Remove OK
//  == 0 → 잔고 없음               → Remove OK (이미 매도됨)
//  -1   → 주문 실패 (잔고 있음)   → Remove 차단, 다음 틱 재시도
func (m *Monitor) executeSell(...) int {
    ...
    if err 발생 {
        return -1  // 기존: return 0 → Remove 허용 ❌
    }
    return qty
}
```

**Go 언어 핵심 개념**: 함수가 여러 의미의 "0"을 반환하면 호출자가 구분할 수 없습니다. 이 패턴은 "sentinel value"라고 하며, `-1`을 오류 신호로 사용하는 C/Unix 관용구와 같습니다.

### 호출 부분 수정

```go
// 수정 후: 반환값 검사 후 조건부 Remove
case price >= pos.TargetPrice:
    if !isTest {
        if soldQty := m.executeSell(stockCode, pos, "목표가 도달"); soldQty >= 0 {
            m.Remove(context.Background(), stockCode) // 성공(>0) 또는 잔고없음(0)만 Remove
        }
        // soldQty == -1: 주문 실패 → 포지션 유지, 다음 WebSocket 틱에서 재시도
    } else {
        m.Remove(context.Background(), stockCode)
    }
```

**단축 변수 선언 (`:=` in if)**: Go에서 `if x := foo(); x > 0 { ... }` 패턴은 스코프를 if 블록으로 제한합니다. 불필요한 변수 노출을 막는 관용구입니다.

### 동일 버그가 있던 4곳 모두 수정

| 위치 | 트리거 | 수정 |
|------|--------|------|
| `HandlePrice` 목표가 | price ≥ TargetPrice | `soldQty >= 0`만 Remove |
| `HandlePrice` 손절가 | price ≤ StopPrice | `soldQty >= 0`만 Remove |
| `HandlePrice` 트레일링 스탑 | price < PeakPrice × (1-stopPct) | 실패 시 `return`만 |
| `checkIndicators` | RSI/MACD/횡보 조건 | `soldQty >= 0`만 Remove |
| `LiquidateAll` | 15:15 장마감 청산 | 실패 시 `continue` (Remove 건너뜀) |

---

### Hard Rejection 룰 추가 — 설정 키 → 프롬프트 파이프라인

새로운 필터 2개가 `settings DB → TradingSettings → TradingRules → 프롬프트` 파이프라인을 따릅니다.

```
DB settings 테이블
    hard_macd_bearish_enabled = "true"
    hard_high_formed_mins_max = "60"
        ↓ GetTradingSettings()
TradingSettings 구조체
    HardMACDBearishEnabled bool
    HardHighFormedMinsMax  float64
        ↓ engine.go 매핑
TradingRules 구조체
    HardMACDBearishEnabled bool
    HardHighFormedMinsMax  float64
        ↓ SelectStocks() 프롬프트
"9. macd_line < macd_signal → skip"
"10. high_formed_mins_ago > 60 → skip"
```

**조건부 프롬프트 생성**: 비활성(`false` / `0`)이면 프롬프트에 룰 자체가 추가되지 않습니다.

```go
macdBearishRule := ""
if rules.HardMACDBearishEnabled {
    macdBearishRule = "9. macd_line < macd_signal → ..."
}
```

이는 프롬프트 토큰을 낭비하지 않고, 미설정 시 Claude에게 혼란을 주지 않기 위한 설계입니다.

---

## 핵심 요약

| 개념 | 설명 |
|------|------|
| Sentinel value `-1` | Go에서 에러를 숫자로 표현하는 관용구 (C/Unix 전통) |
| `if x := f(); x > 0` | if 조건절 내 단축 선언으로 스코프 제한 |
| `continue` in for loop | 현재 반복 건너뛰기 (LiquidateAll에서 Remove 우회) |
| 조건부 프롬프트 빌드 | 비활성 기능은 fmt.Sprintf 문자열에서 아예 제외 |
| 설정 → 규칙 파이프라인 | DB → TradingSettings → TradingRules → Prompt 단방향 흐름 |
