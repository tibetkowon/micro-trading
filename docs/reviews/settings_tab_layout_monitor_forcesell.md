# 코드 리뷰: 설정 페이지 탭 분할 + 모니터링 강제매도 + 국장·미장 프리셋 분리

> 작성일: 2026-03-25

---

## 개요 (Overview)

이번 업데이트에서 세 가지 주요 기능을 구현했습니다.

1. **강제매도 버튼** — 모니터링 페이지에서 보유 종목을 즉시 시장가 매도하고 모니터링 해제
2. **국장·미장 프리셋 분리** — 설정 프리셋을 KR / US로 완전히 분리하여 서로 독립 관리
3. **설정 페이지 탭 분할** — 단일 스크롤 페이지를 "국장 / 미장 / AI·서버" 3개 탭으로 재편

---

## Go 백엔드 해설

### 1. `ForceSell` 메서드 (`monitor/monitor.go`)

```go
func (m *Monitor) ForceSell(ctx context.Context, stockCode string) (int, error) {
    m.mu.RLock()
    pos, ok := m.positions[stockCode]
    m.mu.RUnlock()
    if !ok {
        return 0, fmt.Errorf("모니터링 중인 포지션을 찾을 수 없습니다: %s", stockCode)
    }
    qty := m.executeSell(stockCode, pos, "강제매도")
    m.Remove(ctx, stockCode)
    if qty <= 0 {
        return 0, fmt.Errorf("매도 주문 실패 (보유수량 없음 또는 API 오류)")
    }
    return qty, nil
}
```

**핵심 개념:**
- **`m.mu.RLock()` / `m.mu.RUnlock()`**: 읽기 전용 뮤텍스 잠금. 여러 고루틴이 동시에 `positions` 맵을 읽어도 안전합니다. 쓰기(`Lock`)와 구별하는 것이 중요합니다.
- **기존 `executeSell` 재활용**: 내부 매도 로직을 그대로 사용하여 코드 중복을 방지합니다. 새 메서드는 퍼블릭 진입점(wrapper) 역할만 합니다.
- **실패 처리**: 매도 수량이 0 이하면 실패로 처리하여 API 오류 상황을 사용자에게 명확히 알립니다.

### 2. 프리셋 시장 분리 (`api/handlers.go`)

```go
for k, v := range allSettings {
    isUS := strings.HasPrefix(k, "us_")
    if req.Market == "US" && isUS {
        snapshot[k] = v
    } else if req.Market == "KR" && !isUS {
        snapshot[k] = v
    }
}
```

**핵심 개념:**
- **`strings.HasPrefix`**: 문자열이 특정 접두사로 시작하는지 확인. `us_` 접두사 규칙을 이용해 별도 키 목록 없이 KR/US 설정을 자동 분류합니다.
- **스냅샷 필터링**: 프리셋 저장 시 해당 시장과 관련 없는 키를 제외하므로, KR 프리셋 적용이 US 설정을 덮어쓰지 않습니다.

### 3. DB 마이그레이션 (`database/db.go`)

```go
ALTER TABLE settings_presets ADD COLUMN market TEXT NOT NULL DEFAULT 'KR'
```

**핵심 개념:**
- SQLite는 기존 테이블에 컬럼을 추가할 때 `ADD COLUMN`만 지원합니다 (테이블 삭제·재생성 불필요).
- `DEFAULT 'KR'`를 지정하면 기존 레코드에도 값이 자동으로 채워져 데이터 무결성을 유지합니다.

---

## React 프론트엔드 해설

### 1. 강제매도 버튼 (`Monitor.jsx`)

```jsx
const [sellingCodes, setSellingCodes] = useState(new Set())

async function handleForceSell(code, name) {
    if (!confirm(`[강제매도] ${name || code}...`)) return
    setSellingCodes((prev) => new Set(prev).add(code))
    try {
        const res = await fetch(`/api/monitor/positions/${code}/sell`, { method: 'POST' })
        // ...
    } finally {
        setSellingCodes((prev) => {
            const next = new Set(prev); next.delete(code); return next
        })
    }
}
```

**핵심 개념:**
- **`useState(new Set())`**: Set을 상태로 사용하면 여러 종목이 동시에 로딩 중임을 효율적으로 추적할 수 있습니다. (배열보다 `.has()` 체크가 빠릅니다.)
- **`new Set(prev).add(code)`**: React 상태는 불변(immutable)이어야 합니다. 기존 Set을 직접 수정하지 않고 복사본을 만들어 변경합니다.
- **`try/finally`**: 성공·실패 여부와 관계없이 로딩 상태를 반드시 제거합니다. `finally` 없이 `catch`만 쓰면 성공 시 상태가 남아 버튼이 영구적으로 비활성화됩니다.

### 2. 탭 레이아웃 (`Settings.jsx`)

```jsx
const [activeTab, setActiveTab] = useState('KR')

// 탭 패널 표시/숨김 — 컴포넌트를 unmount하지 않고 hidden 클래스만 사용
<div className={activeTab !== 'KR' ? 'hidden' : 'space-y-5'}>
    {/* 국장 설정 */}
</div>
<div className={activeTab !== 'US' ? 'hidden' : 'space-y-5'}>
    {/* 미장 설정 */}
</div>
```

**핵심 개념:**
- **`hidden` vs 조건부 렌더링**: `{activeTab === 'KR' && <div>...</div>}` 방식은 탭 전환 시 컴포넌트가 unmount되어 입력값이 초기화됩니다. `hidden` 클래스를 사용하면 DOM은 유지되어 사용자가 입력한 값이 탭 전환 후에도 보존됩니다.
- **단일 `<form>`**: KR 탭과 US 탭이 모두 하나의 `<form>` 안에 있습니다. "설정 저장" 버튼을 누르면 어느 탭이 활성화되어 있든 모든 필드값이 함께 전송됩니다.
- **`PresetPanel` 중첩 함수 컴포넌트**: `Settings()` 함수 내부에 `PresetPanel`을 정의하면 부모의 상태(`presets`, `loading` 등)를 클로저(closure)로 자연스럽게 공유할 수 있습니다. 별도 파일로 분리하지 않아도 됩니다.

### 3. INFO 탭 별도 저장 버튼

```jsx
// INFO 탭은 form 밖에 위치하므로 별도 버튼 사용
<button
    onClick={() => handleSave({ preventDefault: () => {} })}
>
    설정 저장
</button>
```

**핵심 개념:**
- INFO 탭은 `<form>` 외부에 있어 `type="submit"` 버튼이 동작하지 않습니다.
- `{ preventDefault: () => {} }` 가짜 이벤트 객체를 전달하여 `handleSave`의 `e.preventDefault()` 호출이 에러 없이 실행되도록 합니다.

---

## 핵심 요약 (Key Takeaways)

| 개념 | 내용 |
|------|------|
| **뮤텍스 읽기/쓰기 잠금** | `RLock()`은 읽기 전용, `Lock()`은 쓰기 전용 — 동시성 제어의 기본 |
| **strings.HasPrefix** | 접두사 기반 설정 분류 — 별도 키 목록 없이도 KR/US 자동 분리 |
| **hidden vs unmount** | 입력값 유지가 필요한 탭 UI에서는 `hidden` 클래스 사용 |
| **Set 상태** | 여러 항목의 로딩 상태를 동시에 추적할 때 `useState(new Set())` 활용 |
| **try/finally** | 비동기 작업의 로딩 상태는 반드시 `finally`에서 해제 |
| **SQLite ADD COLUMN** | 기존 테이블 컬럼 추가는 `ADD COLUMN` + `DEFAULT` 값 지정으로 데이터 무결성 유지 |
