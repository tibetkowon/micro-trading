# Phase 1.5 — 패키지명 정리 계획

## Goal

Phase 1 완료 후, Phase 2 코딩 전에 패키지 이름이 역할을 정확히 반영하도록 정리한다.
코드를 추가·삭제하지 않고 이름 변경과 import 경로 수정만 수행한다.

---

## 현황 진단

| 패키지 | 현재 이름 | 문제점 |
|--------|-----------|--------|
| `internal/agent` | `agent` | Phase 1에서 Claude AI 제거 후 역할이 "KIS API 비즈니스 오퍼레이션 레이어"가 됐음에도 AI 에이전트 뉘앙스를 유지하고 있음 |
| `internal/mst` | `mst` | 한국 증권업계 내부 약어. 코드를 처음 보는 사람은 의미 파악 불가. |
| 나머지 패키지 | - | 역할과 이름이 일치 — 변경 불필요 |

---

## 변경 계획

### 1. `internal/agent` → `internal/ops`

**이유:** 이 패키지는 자율 판단 주체가 아니라, KIS API를 호출해 결과를 DB에 저장하는 **비즈니스 오퍼레이션 계층**이다.  
Go 컨벤션상 짧고 명확한 단일 단어 `ops`가 적합하다.

**변경 파일 (소스 7개):**
```
internal/agent/balance.go     → internal/ops/balance.go
internal/agent/chart.go       → internal/ops/chart.go
internal/agent/history.go     → internal/ops/history.go
internal/agent/market.go      → internal/ops/market.go
internal/agent/order.go       → internal/ops/order.go
internal/agent/ranking.go     → internal/ops/ranking.go
internal/agent/stock_info.go  → internal/ops/stock_info.go
```

각 파일 상단 `package agent` → `package ops` 변경.

**import 수정 대상 (3개 파일):**
```
backend/cmd/server/main.go
backend/internal/api/handlers.go
backend/internal/monitor/monitor.go
```

---

### 2. `internal/mst` → `internal/stockmaster`

**이유:** `mst`는 "종목 마스터(Stock Master)"의 한국 증권 약어. `stockmaster`로 변경하면 외국어 배경 개발자도 즉시 이해 가능.

**변경 파일 (소스 3개):**
```
internal/mst/store.go      → internal/stockmaster/store.go
internal/mst/downloader.go → internal/stockmaster/downloader.go
internal/mst/parser.go     → internal/stockmaster/parser.go
```

각 파일 상단 `package mst` → `package stockmaster` 변경.

**import 수정 대상 (4개 파일):**
```
backend/cmd/server/main.go
backend/internal/api/handlers.go
backend/internal/monitor/monitor.go
backend/internal/trader/engine.go
```

---

## 변경하지 않는 패키지

| 패키지 | 이유 |
|--------|------|
| `kis` | 한국투자증권(KIS) 의미로 국내 핀테크 팀에서 표준 약어. 변경 시 오히려 혼란. |
| `database` | 추상화 레이어로서 적절. Firestore로 구현됐지만 향후 DB 교체 시 이름 유지가 유리. |
| `api`, `config`, `logger`, `models`, `monitor`, `notify`, `report`, `trader` | 역할과 이름이 일치. |

---

## 구현 단계

### Step 1 — 디렉터리 이동 + package 선언 변경
- `internal/agent/` → `internal/ops/` (7개 파일, package 선언 수정)
- `internal/mst/` → `internal/stockmaster/` (3개 파일, package 선언 수정)

### Step 2 — import 경로 수정
- 위 3+4 = 7개 파일의 import 블록에서 경로 변경

### Step 3 — 빌드 검증
- `go build ./...` 통과 확인
- `go vet ./...` 통과 확인

---

## Verification

```bash
cd backend
go build ./...
go vet ./...
# 에러 없으면 완료
```

---

## 예상 소요 시간
- 코드 로직 변경 없음 — 이름·경로 교체만
- 약 10분 이내 완료 예상
