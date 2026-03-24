# 코드 리뷰: 트레이딩 중지 사유 표시 / 강제 실행 / 필터 누적 기록

> 작성일: 2026-03-24
> 대상 파일: `backend/internal/trader/engine.go`, `backend/internal/api/handlers.go`, `backend/internal/api/router.go`, `frontend/src/pages/Dashboard.jsx`

---

## 개요

이번 업데이트는 세 가지 사용자 불편을 해결한다:

1. **트레이딩이 멈췄을 때 이유를 화면에 표시** — 매수 중단 시간대, 지수 하락, 일일 손실 한도 등 엔진이 스스로 멈춘 이유를 대시보드에 보여준다.
2. **강제 실행 버튼** — 멈춘 상태에서도 버튼 하나로 즉시 매수 사이클을 시작할 수 있다.
3. **종목 수 불일치 수정** — "9종목 조회 → 2종목 하드필터 제거" 표시인데 LLM에 1종목만 전달되는 이유를 화면에서 알 수 없었던 문제. 숨겨진 중간 필터들(현금 부족, 거래대금 미달)도 이제 필터 제거 목록에 포함된다.

---

## Go 백엔드 해설

### 1. Engine 구조체에 필드 추가

```go
type Engine struct {
    // ... 기존 필드들
    mu         sync.RWMutex
    state      EngineState
    haltReason string     // ← 새로 추가
    soldCh     chan string
    stopCh     chan struct{}
}
```

**Go 개념 — 구조체(struct)와 뮤텍스(sync.RWMutex):**
- Go의 `struct`는 여러 타입의 데이터를 묶는 컨테이너다.
- `haltReason`은 여러 goroutine에서 동시에 읽거나 쓸 수 있으므로 `sync.RWMutex`로 보호한다.
- `mu.Lock()` / `mu.Unlock()`: 쓰기 전용 잠금. 오직 하나의 goroutine만 접근.
- `mu.RLock()` / `mu.RUnlock()`: 읽기 전용 잠금. 여러 goroutine이 동시에 읽기 가능.

### 2. GetHaltReason() 메서드

```go
func (e *Engine) GetHaltReason() string {
    e.mu.RLock()
    defer e.mu.RUnlock()
    return e.haltReason
}
```

**Go 개념 — 메서드 리시버와 defer:**
- `(e *Engine)` 부분이 "메서드 리시버"다. Engine 인스턴스에 메서드를 붙이는 방법.
- `*Engine`처럼 포인터 리시버를 쓰면 원본 Engine을 직접 참조한다.
- `defer`는 함수가 끝날 때 실행된다. Lock을 건 직후 defer로 Unlock을 예약하면 return이나 panic 상황에서도 잠금 해제가 보장된다.

### 3. ForceRun() — goroutine으로 비동기 실행

```go
func (e *Engine) ForceRun(ctx context.Context) {
    go func() {
        settings, err := e.db.GetTradingSettings(ctx)
        if err != nil {
            return
        }
        if err := e.selectAndBuy(ctx, settings); err != nil {
            e.mu.Lock()
            e.haltReason = err.Error()
            e.mu.Unlock()
        } else {
            e.mu.Lock()
            e.haltReason = ""
            e.mu.Unlock()
        }
    }()
}
```

**Go 개념 — goroutine과 익명 함수:**
- `go func() { ... }()`: `go` 키워드를 붙이면 새 goroutine(경량 스레드)으로 실행된다.
- 이렇게 하면 HTTP 요청이 즉시 응답을 반환하고, 매수 사이클은 백그라운드에서 돌아간다.
- 만약 `go`를 붙이지 않으면 selectAndBuy가 끝날 때까지 HTTP 응답이 차단된다(최대 수 분).

### 4. runCycle()에서 haltReason 저장

```go
if err := e.selectAndBuy(ctx, settings); err != nil {
    consecutiveFailures++
    e.mu.Lock()
    e.haltReason = err.Error()  // 에러 메시지 저장
    e.mu.Unlock()
    // ...
} else {
    e.mu.Lock()
    e.haltReason = ""           // 성공 시 초기화
    e.mu.Unlock()
    consecutiveFailures = 0
}
```

**Go 개념 — error 타입:**
- Go에서 에러는 `error` 인터페이스를 구현한 값이다.
- `.Error()` 메서드를 호출하면 에러의 텍스트 메시지를 `string`으로 가져온다.
- 예: `fmt.Errorf("매수 중단 시간대 (11:00~14:00)")`가 에러로 반환되면, `.Error()`는 `"매수 중단 시간대 (11:00~14:00)"` 문자열을 돌려준다.

### 5. allFilteredOut — 모든 필터 단계 누적 기록

**기존 문제:**
```
9종목 (합집합) → [가격 필터로 6종목 제거, 로그 없음] → 3종목 → [하드필터 2종목 제거, 로그 있음] → 1종목 LLM 전달
화면: "합집합 9종목, 하드필터 2종목 제거, LLM 1종목" → 사용자: "왜 9-2=7이 아니고 1이야?"
```

**수정 후:**
```go
var allFilteredOut []filteredStockEntry  // 단일 슬라이스 선언

// 이미 거래된 종목 제외 시
allFilteredOut = append(allFilteredOut, filteredStockEntry{
    StockCode: r.StockCode, StockName: r.StockName,
    FilterReason: "오늘 이미 거래된 종목",
})

// 가격 필터 시
allFilteredOut = append(allFilteredOut, filteredStockEntry{
    StockCode:    item.StockCode,
    FilterReason: fmt.Sprintf("현금 부족 (주가 %.0f원 > 가용 %.0f원)", price, availableCash),
})

// 거래대금 필터 시
allFilteredOut = append(allFilteredOut, filteredStockEntry{
    FilterReason: fmt.Sprintf("거래대금 미달 (%.0f억 < %.0f억)", ...),
})

// 하드 필터 후 한 번에 DB 기록
if rankingLogID > 0 && len(allFilteredOut) > 0 {
    filteredJSON, _ := json.Marshal(allFilteredOut)
    e.db.ExecContext(ctx, `UPDATE trader_ranking_logs SET filtered_stocks=? WHERE id=?`, ...)
}
```

**Go 개념 — slice와 append:**
- `[]filteredStockEntry`는 구조체의 동적 배열(slice)이다.
- `append(slice, item)`은 항목을 추가하고 새 슬라이스를 반환한다. Go에서 slice는 길이가 자동으로 늘어난다.
- `json.Marshal()`은 Go 구조체를 JSON 바이트로 변환한다. `json:"filter_reason"` 태그로 JSON 키 이름을 지정한다.

### 6. ForceRunTrader HTTP 핸들러

```go
func (h *Handler) ForceRunTrader(c *gin.Context) {
    market := c.DefaultQuery("market", "KR")
    if market == "US" {
        if h.usEngine == nil {
            c.JSON(http.StatusServiceUnavailable, gin.H{"error": "US 엔진이 설정되지 않았습니다"})
            return
        }
        h.usEngine.ForceRun(c.Request.Context())
    } else {
        if h.engine == nil {
            c.JSON(http.StatusServiceUnavailable, gin.H{"error": "KR 엔진이 설정되지 않았습니다"})
            return
        }
        h.engine.ForceRun(c.Request.Context())
    }
    c.JSON(http.StatusOK, gin.H{"ok": true, "market": market})
}
```

**Go 개념 — nil 체크:**
- Go에서 포인터나 인터페이스는 초기화 전에 `nil` 상태다.
- `h.engine == nil`을 확인하지 않고 `h.engine.ForceRun()`을 호출하면 런타임 패닉이 발생한다.
- 항상 nil 가능성이 있는 값은 사용 전에 체크해야 한다.

---

## React 프론트엔드 해설

### Dashboard.jsx 변경점

```jsx
const [forceRunLoading, setForceRunLoading] = useState(false)

async function handleForceRun(market) {
    setForceRunLoading(true)
    try {
        await fetch(`/api/trader/force-run?market=${market}`, { method: 'POST' })
        setTimeout(refetchStatus, 2000)
    } finally {
        setForceRunLoading(false)
    }
}
```

**React 개념 — useState와 비동기 처리:**
- `useState(false)`: 버튼 로딩 상태를 관리한다. `true`이면 버튼이 비활성화된다.
- `async/await`: JavaScript의 비동기 처리 문법. `fetch`는 HTTP 요청을 보내고 응답을 기다린다.
- `try/finally`: 성공하든 실패하든 `finally`는 반드시 실행된다. 로딩 상태를 항상 해제하기 위해 사용.
- `setTimeout(refetchStatus, 2000)`: 2초 후에 서버 상태를 다시 불러온다. ForceRun은 goroutine으로 실행되므로, 즉시 조회하면 상태 변화가 반영되지 않을 수 있다.

```jsx
{/* 강제 실행 버튼 */}
<button
    onClick={() => handleForceRun('KR')}
    disabled={forceRunLoading}
    className="text-xs px-2 py-0.5 rounded-full bg-th-surface-high hover:text-emerald-400 disabled:opacity-40 transition-colors text-th-on-muted"
>
    강제 실행
</button>

{/* 중지 사유 텍스트 */}
{status.halt_reason && (
    <p className="text-xs text-amber-400 mt-1 leading-snug">{status.halt_reason}</p>
)}
```

**React 개념 — 조건부 렌더링:**
- `{status.halt_reason && <p>...</p>}`: `halt_reason`이 빈 문자열(`""`)이면 `false`로 평가되어 아무것도 렌더링되지 않는다. 값이 있을 때만 표시된다.

**Tailwind CSS 클래스 해설:**
- `text-amber-400`: 주황색 계열 텍스트. 경고 메시지에 사용.
- `disabled:opacity-40`: 버튼이 `disabled` 상태일 때 투명도 40%로 흐리게 표시.
- `hover:text-emerald-400`: 마우스를 올리면 초록색으로 변한다.
- `transition-colors`: 색상 변화에 부드러운 애니메이션 적용.
- `leading-snug`: 줄 간격을 좁게 설정. 여러 줄 텍스트가 촘촘하게 표시됨.

---

## 핵심 요약

| 개념 | 설명 |
|------|------|
| `sync.RWMutex` | goroutine 간 공유 데이터 보호. 읽기는 RLock, 쓰기는 Lock |
| `defer` | 함수 종료 시 반드시 실행. Lock 해제에 관용적으로 사용 |
| `go func() { ... }()` | 비동기 실행. HTTP 요청을 블로킹하지 않고 백그라운드 처리 |
| `append(slice, item)` | Go slice에 항목 추가. 동적으로 크기가 늘어남 |
| `nil` 체크 | 포인터/인터페이스 사용 전 항상 nil 여부 확인 |
| `useState` + `disabled` | 버튼 로딩 상태를 React state로 관리하여 중복 클릭 방지 |
| 조건부 렌더링 `{val && <...>}` | 값이 있을 때만 UI 요소를 표시하는 React 패턴 |
