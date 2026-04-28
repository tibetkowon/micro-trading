# DB 마이그레이션 — SQLite → Firebase Firestore

## Goal

현재 SQLite 기반 로컬 DB를 **Firebase Firestore**(Cloud NoSQL)로 마이그레이션한다.
목적: 외부에서 웹 브라우저 또는 Firebase Console을 통해 DB 데이터를 실시간으로 열람 가능하게 한다.

---

## 결정 사항 (Before Coding)

다음 사항을 사전에 결정해야 한다:

| 항목 | 선택지 | 권장 |
|------|--------|------|
| Firebase 제품 | Firestore vs Realtime DB | **Firestore** (구조적 쿼리 지원, 더 유연) |
| 인증 | Firebase Admin SDK (서버 → Firestore) | Admin SDK with Service Account JSON |
| 마이그레이션 전략 | 완전 교체 vs Dual-write | **완전 교체** (복잡도 최소화) |
| 오프라인 지원 | 서버 재시작 시 네트워크 없으면 동작 여부 | Firebase SDK는 기본 온라인 전용 — 캐시 설정 필요 |

---

## Requirements

### 기능 요건
1. 현재 SQLite 테이블 전체를 Firestore 컬렉션으로 대응
2. Go `database/db.go`의 모든 쿼리를 Firestore SDK 호출로 교체
3. 기존 데이터 일회성 마이그레이션 스크립트 작성 (`cmd/migrate/main.go`)
4. 환경변수 `FIREBASE_CREDENTIALS_JSON` (서비스 계정 키 JSON 경로)
5. Firebase Console에서 실시간 데이터 열람 가능

### 비기능 요건
- 1GB RAM 서버 제약: Firestore Go SDK 메모리 사용량 최소화 (스트리밍 리스너 미사용)
- 읽기/쓰기 지연: 기존 SQLite 대비 네트워크 RTT 추가 (수십 ms) — 허용 가능
- 비용: Firestore 무료 티어 (읽기 50,000/일, 쓰기 20,000/일) 내 유지 예상

---

## 현재 SQLite 테이블 → Firestore 컬렉션 매핑

| SQLite 테이블 | Firestore 컬렉션 | 문서 ID |
|--------------|----------------|---------|
| `settings` | `settings` (단일 문서 `config`) | `config` |
| `orders` | `orders` | `{id}` |
| `trader_selection_logs` | `trader_selection_logs` | `{id}` |
| `trader_cycle_logs` | `trader_cycle_logs` | `{id}` |
| `tokens` | `tokens` | `{appkey}` |
| `stock_masters` | `stock_masters` | `{code}` |

---

## Affected Files

### 수정
| 파일 | 변경 내용 |
|------|---------|
| `backend/internal/database/db.go` | 전체 교체 — SQL 드라이버 제거, Firestore client 초기화, 모든 CRUD를 Firestore SDK 호출로 교체 |
| `backend/internal/models/models.go` | 구조체 유지, Firestore 태그(`firestore:"..."`) 추가 |
| `go.mod` / `go.sum` | `cloud.google.com/go/firestore` 의존성 추가 |
| `backend/.env.example` | `FIREBASE_CREDENTIALS_JSON` 키 추가 |

### 신규 생성
| 파일 | 내용 |
|------|------|
| `cmd/migrate/main.go` | SQLite → Firestore 일회성 마이그레이션 CLI |

### 제거
| 파일 | 이유 |
|------|------|
| `backend/internal/database/db.go`의 SQLite 마이그레이션 로직 | Firestore는 스키마리스 |
| `go.mod`의 `github.com/mattn/go-sqlite3` | SQLite 드라이버 불필요 |

---

## Implementation Phases

### Phase 0 — 사전 준비 (코딩 전)
1. Firebase 프로젝트 생성 (Firebase Console)
2. Firestore 데이터베이스 생성 (Native 모드, 리전: asia-northeast3 서울)
3. 서비스 계정 키 JSON 다운로드
4. `.env`에 `FIREBASE_CREDENTIALS_JSON=./firebase-credentials.json` 추가

### Phase 1 — 모델 태그 추가 (models.go)
- 모든 구조체 필드에 `firestore:"field_name"` 태그 추가
- 시간 필드: `time.Time` → `time.Time` 유지 (Firestore Timestamp 자동 변환)

### Phase 2 — database/db.go 재작성
1. `DB` 타입을 `*firestore.Client`로 교체
2. `InitDB()` → Firebase 클라이언트 초기화
3. 함수별 교체 순서:
   - `GetTradingSettings` / `SetTradingSetting` (단순 get/set)
   - `GetToken` / `SetToken`
   - `InsertOrder` / `UpdateOrder` / `GetOrders`
   - `InsertSelectionLog` / `UpdateSelectionLog`
   - `InsertCycleLog`
   - `UpsertStockMasters`

### Phase 3 — 마이그레이션 스크립트 (cmd/migrate/main.go)
```
Usage: ./migrate --sqlite=./trading.db --firebase-credentials=./creds.json
```
- SQLite에서 테이블별 전체 읽기
- Firestore에 배치 쓰기 (500건/배치 제한 준수)

### Phase 4 — 빌드 및 통합 테스트
1. `go build ./...`
2. Firestore Emulator로 로컬 테스트
3. 실제 Firebase로 마이그레이션 실행

---

## 위험 및 대응

| 위험 | 대응 |
|------|------|
| 네트워크 단절 시 트레이딩 중단 | Firestore SDK 오프라인 캐시 활성화 + 헬스체크 추가 |
| 민감 데이터(토큰, API 키)가 Firestore에 저장됨 | Firestore Security Rules로 접근 제한 (Admin SDK만 허용) |
| Firestore 비용 초과 | 고빈도 쓰기(stock_masters 대량 upsert) → 배치 단위로 줄이기 |

---

## Verification

1. `go build ./...` 성공
2. Firestore Console에서 `orders`, `settings` 컬렉션 데이터 열람 확인
3. 마이그레이션 스크립트 실행 후 기존 SQLite 데이터와 문서 수 대조
