# Plan: 지수 필터 — 하락 거래소 순위 조회 제외

## Goal

설정된 지수가 하락 임계값 이하로 내려갈 때, 기존처럼 "전체 매수 중단"하는 대신
**해당 지수에 대응하는 거래소만 순위 조회에서 제외**한다.

- 지수 정상 → 해당 거래소 포함해서 순위 조회
- 지수 하락 → 해당 거래소 제외하고 나머지 거래소만 순위 조회
- 모든 지수 하락 → 조회할 거래소 없음 → 기존과 동일하게 매수 중단

## Requirements

- 지수 코드 = 거래소 코드 (`0001`=KOSPI, `1001`=KOSDAQ) → 별도 매핑 불필요
- `force=true` (강제 실행) 시 지수 필터 완전 무시 — 기존 동작 유지
- `marketIndexDrop` 값은 여전히 Claude 컨텍스트(rules)에 전달 — 기존 유지
- 로그: 제외된 거래소와 남은 거래소를 INFO 레벨로 기록
- UI/DB 변경 없음 — 백엔드 engine.go 단일 파일만 수정

## Affected Files

| 파일 | 변경 내용 |
|------|----------|
| `backend/internal/trader/engine.go` | `runSelectCycle` 내 지수 필터 로직 수정 |

## Implementation

### 현재 로직 (lines 277-298)
```
for each index code:
    if drop <= threshold AND !force:
        return error (매수 전체 중단)
```

### 변경 후 로직
```
droppedExchanges = {}
for each index code:
    get index price
    update marketIndexDrop
    if drop <= threshold AND !force:
        droppedExchanges.add(code)  // 지수코드 = 거래소코드

if len(droppedExchanges) > 0:
    activeExchanges = settings.RankingExchanges - droppedExchanges
    if len(activeExchanges) == 0:
        return error "모든 지수 하락 — 순위 조회 중단"
    log INFO "지수 하락으로 N거래소 제외, M거래소로 조회"
    settings.RankingExchanges = activeExchanges  // 로컬 복사본 수정
```

### 엣지 케이스
- `settings.RankingExchanges`가 비어 있는 경우 → `getRankings` 내부에서 기본값 `["0001","1001"]` 사용 중이므로, 기본값 적용 전에 필터를 먼저 계산해야 함
  → 비어 있으면 `["0001","1001"]`로 초기화 후 필터 적용

## Verification

1. KOSPI만 하락 설정 → KOSDAQ 거래소만으로 순위 조회 로그 확인
2. KOSDAQ만 하락 설정 → KOSPI 거래소만으로 순위 조회 로그 확인
3. 둘 다 하락 → "모든 지수 하락" 에러로 중단 확인
4. `force=true` → 지수 무관하게 전체 거래소 조회 확인
5. `index_codes` 미설정 → 기존 동작 그대로 (전체 거래소 조회)
