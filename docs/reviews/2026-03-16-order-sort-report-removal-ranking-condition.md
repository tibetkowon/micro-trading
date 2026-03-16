# 코드 리뷰: 주문 정렬 변경 + 리포트 제거 + 순위 AND/OR 조건

> 작성일: 2026-03-16

---

## 1. 주문 내역 정렬 변경 (`agent/history.go`)

### 변경 내용

```go
// 변경 전
FROM orders ORDER BY created_at DESC, id DESC

// 변경 후
FROM orders ORDER BY created_at ASC, id ASC
```

### 설명

- **DESC (내림차순)**: 가장 최근 주문이 맨 위에 표시됨
- **ASC (오름차순)**: 가장 오래된 주문이 맨 위에 표시됨
- 주문 내역을 시간순으로 읽으면 매수→매도 흐름을 위에서 아래로 자연스럽게 파악할 수 있어 오름차순으로 변경

---

## 2. 리포트 기능 완전 제거

### 제거한 이유

리포트 기능(일일 AI 분석 리포트)은 사용되지 않아 코드 복잡도만 높이는 상태였음. DB 테이블(`reports`)은 기존 데이터 보존을 위해 삭제하지 않음.

### Go 제거 대상

#### `models/models.go` — `Report` 구조체 제거

```go
// 제거된 코드
type Report struct {
    ID         int64     `json:"id"`
    ReportDate string    `json:"report_date"`
    Content    string    `json:"content"`
    CreatedAt  time.Time `json:"created_at"`
}
```

모델은 DB 테이블과 1:1로 대응하는 구조체. 핸들러에서 더 이상 사용하지 않으므로 제거.

#### `trader/claude.go` — `ReportSummary`, `GenerateReport()` 제거

Claude API를 호출해 한국어 분석 텍스트를 생성하던 함수. `SelectStocks()`만 남김.

#### `trader/engine.go` — `GenerateDailyReport()` 제거

매일 15:20에 호출되던 함수. 오늘 체결된 주문을 DB에서 조회해 매수→매도 쌍을 매칭하고, 손익 계산 후 마크다운 테이블 생성 + Claude 분석 섹션을 붙이는 로직이었음.

#### `database/db.go` — `SaveReport()` 제거

`INSERT OR CONFLICT UPDATE` 패턴으로 리포트를 저장하던 함수. 제거.

#### `api/handlers.go` — `GetReports()`, `GetReport()` 제거

- `GET /api/reports`: 최근 30개 리포트 날짜 목록 반환
- `GET /api/reports/:date`: 특정 날짜 리포트 전문 반환

#### `api/router.go` — 라우트 그룹 제거

```go
// 제거된 코드
reports := api.Group("/reports")
{
    reports.GET("", h.GetReports)
    reports.GET("/:date", h.GetReport)
}
```

#### `cmd/server/main.go` — 15:20 스케줄러 케이스 제거

```go
// 제거된 코드
case hhmm == 1520:
    report, err := eng.GenerateDailyReport(ctx)
    ...
    db.SaveReport(ctx, kstDate, report)
```

### React 제거 대상

- `App.jsx`: `import Reports from './pages/Reports'` 제거, `navItems`에서 `{ to: '/reports', label: '리포트' }` 제거, `<Route path="/reports" element={<Reports />} />` 제거
- `pages/Reports.jsx`: 파일 전체 삭제

---

## 3. 순위 AND/OR 조건 추가

### 개요

기존에는 여러 순위 유형을 선택하면 **무조건 AND(교집합)** — 모든 순위에 공통으로 포함된 종목만 선정. 이제 **OR(합집합)** 도 선택 가능하여 하나 이상의 순위에 포함된 종목을 모두 수집할 수 있음.

### Go 변경: `database/db.go`

**`TradingSettings` 구조체에 필드 추가:**

```go
RankingCondition string // "AND" | "OR"
```

**기본값 INSERT:**

```go
{"ranking_condition", "AND"},
```

**파싱 로직:**

```go
rankingCondition := vals["ranking_condition"]
if rankingCondition != "AND" && rankingCondition != "OR" {
    rankingCondition = "AND"
}
```

유효하지 않은 값이 들어오면 `"AND"`로 안전하게 폴백.

### Go 변경: `trader/engine.go` — `getRankings()`

```go
if settings.RankingCondition == "OR" {
    // 합집합: 어느 순위에든 있으면 수집
    seen := map[string]RankItem{}
    for _, m := range byType {
        for code, item := range m {
            if _, exists := seen[code]; !exists {
                item.RankingType = strings.Join(settings.RankingTypes, "|")
                seen[code] = item
            } else {
                // 이미 있으면 추가 필드 병합 (VolIncrRate, Strength 등)
                ...
            }
        }
    }
} else {
    // AND 교집합: 모든 순위에 공통 포함된 종목만
    ...
}
```

**핵심 차이:**
| 조건 | 동작 | 종목 수 |
|------|------|---------|
| AND  | 교집합 — 모든 순위 타입에 포함된 종목만 | 적음 (엄격) |
| OR   | 합집합 — 하나라도 포함된 종목 모두 | 많음 (넓음) |

### React 변경: `Settings.jsx`

**상태 추가:**

```jsx
const [rankingCondition, setRankingCondition] = useState('AND')
```

**초기화 (useEffect):**

```jsx
if (data.ranking_condition === 'AND' || data.ranking_condition === 'OR')
    setRankingCondition(data.ranking_condition)
```

**저장 시 body에 포함:**

```jsx
ranking_condition: rankingCondition,
```

**UI — AND/OR 토글 버튼:**

```jsx
{['AND', 'OR'].map((cond) => (
    <button
        key={cond}
        type="button"
        onClick={() => setRankingCondition(cond)}
        className={`... ${rankingCondition === cond ? 'bg-blue-600 text-white' : 'bg-gray-800 text-gray-400'}`}
    >
        {cond}
    </button>
))}
```

현재 선택된 값은 파란색(`bg-blue-600`)으로 강조. 순위 유형 체크박스 섹션 하단에 배치.

---

## 요약

| 작업 | 파일 수 | 방향 |
|------|---------|------|
| 주문 정렬 ASC 변경 | 1 | 시간순 읽기 개선 |
| 리포트 기능 제거 | 8 + 1 삭제 | 미사용 코드 정리 |
| 순위 AND/OR 조건 | 4 | 종목 선정 유연성 향상 |
