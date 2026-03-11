# 코드 리뷰: 거래 시간 설정 + 횡보 감지 자동 매도

> 작성일: 2026-03-11

---

## 1. 거래 시간 설정화면 추가

### 문제
`backend/cmd/server/main.go:252`에 거래 시작·종료 시간이 `915`(09:15), `1515`(15:15)로 하드코딩되어 있었습니다. 시간을 바꾸려면 서버 코드를 직접 수정해야 했습니다.

### 해결 방법

**Go — `parseHHMM()` 헬퍼 함수**

```go
func parseHHMM(s string, def int) int {
    if s == "" { return def }
    t, err := time.Parse("15:04", s)
    if err != nil { return def }
    return t.Hour()*100 + t.Minute()
}
```

- `"09:15"` → `915`, `"15:15"` → `1515` 형태의 정수로 변환
- `time.Parse("15:04", s)` : Go의 시간 파싱. `"15:04"`는 레퍼런스 포맷 (HH:MM 의미)
- 파싱 실패 시 `def` 기본값 반환 (안전한 폴백)

**스케줄러 루프에서 DB 조회**

```go
startHHMM := parseHHMM(db.GetSetting(ctx, "trading_start_time"), 915)
endHHMM   := parseHHMM(db.GetSetting(ctx, "trading_end_time"), 1515)
```

매 30초 틱마다 DB에서 읽으므로, **서버 재시작 없이 변경값이 즉시 반영**됩니다.

---

## 2. 횡보 감지 자동 매도

### 아이디어
모니터링 중인 종목이 진입가 기준 ±1% 이내에서 30분 이상 변동 없이 머무르면 → 횡보로 판단 → 자동 매도 후 새 종목 탐색.

### 구현 설계

**인메모리 타이머 맵** (`monitor/monitor.go`)

```go
stagnantSince map[string]*time.Time  // 종목코드 → 횡보 시작 시각
```

- 포지션별로 "언제부터 횡보 구간에 진입했는지" 시각을 기록합니다.
- 포인터(`*time.Time`)를 사용해 `nil` = 아직 횡보 아님, `non-nil` = 횡보 진행 중을 표현합니다.

**HandlePrice() — 횡보 추적**

```go
default:  // 목표가/손절가 미도달 시
    changePct := math.Abs(price - pos.FilledPrice) / pos.FilledPrice * 100
    if changePct < threshold {
        // 처음 진입 시에만 시각 기록 (이미 기록된 경우 덮어쓰지 않음)
        if _, exists := m.stagnantSince[stockCode]; !exists {
            now := time.Now()
            m.stagnantSince[stockCode] = &now
        }
    } else {
        // 임계치 초과 → 타이머 리셋
        delete(m.stagnantSince, stockCode)
    }
```

- `switch`의 `default` 케이스: 목표가/손절가에 걸리지 않았을 때만 실행됩니다.
- 진입가에서 1% 초과로 움직이면 타이머가 초기화됩니다 (횡보 탈출로 판단).

**checkIndicators() — 횡보 조건 체크**

```go
case "stagnation":
    since, hasSince := m.stagnantSince[code]
    if hasSince && since != nil && durationMin > 0 {
        elapsed := time.Since(*since)
        if elapsed >= time.Duration(durationMin)*time.Minute {
            triggered = true
        }
    }
```

- 기존 RSI/MACD 조건과 동일한 우선순위 배열에 추가됩니다.
- 지표 확인 주기(기본 5분)마다 체크하므로, 실제 매도 트리거는 설정 시간 + 최대 5분 지연이 있습니다.

**락(Lock) 분리**

```go
mu     sync.RWMutex  // positions 맵 보호
stagnMu sync.Mutex   // stagnantSince 맵 보호
```

- `positions`은 읽기가 많고(HandlePrice, List, Count), `stagnantSince`는 쓰기가 잦습니다(매 가격 업데이트).
- 별도 뮤텍스를 사용해 불필요한 경합을 줄입니다.

---

## 3. 설정 화면 재구성

기존 7개 독립 섹션 → 5개 의미 단위로 재구성:

| 전 | 후 |
|---|---|
| 거래 제어 | **거래 제어** (ON/OFF + 시작·종료 시간) |
| 순위조회 제외 종목 + 순위 조회 설정 | **종목 선정** (제외 필터 + 순위 조회 옵션) |
| 거래 파라미터 (4개 혼합) | **매수 설정** (최대 종목수, 주문 비율) |
| 매도 조건 + 지표 설정 | **매도 설정** (익절/손절 + 조건 + 지표 + 횡보) |
| Claude AI 설정 | **AI 설정** |

`stagnation` 매도 조건이 활성화된 경우에만 횡보 파라미터 입력 UI가 나타납니다 (조건부 렌더링):

```jsx
{stagnationActive && (
  <div className="...">횡보 감지 설정</div>
)}
```
