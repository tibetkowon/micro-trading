# Plan: AI 자동 최적화 제안 기능

## Goal
장 마감 후 일별 리포트를 기반으로 Claude가 설정값·프롬프트 파라미터·신규 기능을 분석·제안하고,
사용자가 설정한 적용 모드에 따라 자동 또는 수동으로 반영할 수 있는 시스템 구축.

---

## Requirements

### 기능 요구사항
1. 15:20 일별 리포트 생성 직후 Claude 분석 자동 실행
2. Claude가 반환하는 제안 2가지 카테고리:
   - `settings`: DB settings 테이블 키·값 변경 제안 (거래 설정 + 프롬프트 파라미터 통합, 코멘트 포함)
   - `feature`: 신규 지표·설정 필드 등 개발 요청 (코멘트 포함, 항상 수동)
   - TradingRules 파라미터도 이미 settings 테이블에 저장되므로 별도 카테고리 없이 settings로 통합
3. 제안별 `status`: `PENDING` / `APPLIED` / `REJECTED`
4. **적용 모드** (settings 테이블 키: `optimization_apply_mode`):
   - `all_auto`: settings 제안 자동 적용 (feature는 항상 수동)
   - `all_manual`: 모든 제안 수동 적용
5. 모드와 무관하게 각 제안마다 UI에서 [적용] / [무시] 버튼 제공
6. 자동 적용 시 유효성 검사 통과한 제안만 반영 (범위 이탈 시 PENDING 유지 + 경고 로그)

### 비기능 요구사항
- 거래 0건인 날은 분석 skip (API 비용 절감)
- `ANTHROPIC_API_KEY` 미설정 시 graceful skip
- NCP Micro 서버 메모리 부담 없음 (분석은 1일 1회, 비동기 goroutine)

---

## Data Structure

### Claude 응답 JSON 구조
카테고리는 `settings`와 `feature` 2가지로 단순화.
프롬프트 파라미터(TradingRules)도 이미 DB settings 테이블에 저장되므로 settings 카테고리로 통합.

```json
{
  "overall_assessment": "오늘 전체 거래 분석 요약 (2~3문장)",
  "suggestions": [
    {
      "category": "settings",
      "key": "stop_loss_pct",
      "current_value": "2.0",
      "suggested_value": "1.5",
      "comment": "오늘 3건 중 2건이 손절 직후 반등. 손절폭 축소로 불필요한 실현 손실 방지 기대."
    },
    {
      "category": "settings",
      "key": "hard_rsi_max",
      "current_value": "70.0",
      "suggested_value": "65.0",
      "comment": "RSI 68~70 진입 종목 2건이 이후 하락. 임계값을 낮춰 과매수 구간 진입 차단 권장."
    },
    {
      "category": "feature",
      "key": "",
      "name": "볼린저밴드 %B",
      "type": "indicator",
      "current_value": "",
      "suggested_value": "",
      "comment": "오늘 급등 후 눌림 종목의 볼린저 하단 터치 여부가 진입 타이밍 필터로 유효할 것으로 판단."
    }
  ]
}
```

### settings 테이블 신규 키
| 키 | 기본값 | 설명 |
|---|---|---|
| `optimization_apply_mode` | `all_manual` | 자동 적용 모드 |

### TradingRules ↔ prompt_suggestions 매핑 테이블
| rule_name | TradingRules 필드 |
|---|---|
| `HardRSIMax` | `HardRSIMax` |
| `HardDisparityM5Max` | `HardDisparityM5Max` |
| `HardDisparityM5Min` | `HardDisparityM5Min` |
| `HardHighPriceDiffMax` | `HardHighPriceDiffMax` |
| `HardHighPriceDiffMin` | `HardHighPriceDiffMin` |
| `HardPrevVolRatioMax` | `HardPrevVolRatioMax` |
| `HardStrengthMin` | `HardStrengthMin` |
| `HardOpenPriceDiffMax` | `HardOpenPriceDiffMax` |
| `VWAPDiffMin` | `VWAPDiffMin` |
| `VWAPDiffMax` | `VWAPDiffMax` |
| `RSIBuyMin` | `RSIBuyMin` |
| `RSIBuyMax` | `RSIBuyMax` |
| `BidAskRatioMin` | `BidAskRatioMin` |
| `IndexDropThreshold` | `IndexDropThreshold` |

prompt_suggestions의 rule_name은 위 키 중 하나로 강제. Claude 응답 파싱 시 미인식 rule_name은 무시.

### settings 테이블에 저장되는 TradingRules 키 (자동 적용 대상)
prompt_suggestions의 `suggested_value`는 아래 settings 키로 저장됨:
`hard_disparity_m5_min`, `hard_disparity_m5_max`, `hard_high_price_diff_max`,
`hard_high_price_diff_min`, `hard_prev_vol_ratio_max`, `hard_strength_min`,
`hard_rsi_max`, `hard_open_price_diff_max`, `vwap_diff_min`, `vwap_diff_max`,
`rsi_buy_min`, `rsi_buy_max`, `bid_ask_ratio_min`, `index_drop_threshold`

---

## Affected Files

### 신규 파일
| 파일 | 역할 |
|---|---|
| `backend/internal/report/optimization.go` | `GenerateOptimizationSuggestions()` — Claude 호출 & DB 저장 |
| `frontend/src/pages/OptimizationReports.jsx` | 제안 목록 + [적용]/[무시] UI |

### 수정 파일
| 파일 | 변경 내용 |
|---|---|
| `backend/internal/models/models.go` | `OptimizationReport` struct 추가 |
| `backend/internal/database/db.go` | `optimization_reports` 테이블 + CRUD 함수 |
| `backend/internal/trader/claude.go` | `AnalyzeDailyReport()` 메서드 추가 |
| `backend/internal/api/handlers.go` | `GetOptimizationReports`, `ApplyOptimizationSuggestion`, `RejectOptimizationSuggestion` |
| `backend/internal/api/router.go` | `/api/reports/optimization` 라우트 추가 |
| `backend/cmd/server/main.go` | 15:20 블록에 `GenerateOptimizationSuggestions()` 연계 |
| `frontend/src/App.jsx` | OptimizationReports 라우트 추가 |
| `frontend/src/pages/Settings.jsx` | `optimization_apply_mode` 셀렉트 추가 |
| `docs/db_schema.md` | 신규 테이블 반영 |

---

## Implementation Phases

### Phase 1: DB 스키마 & 모델
- `models.OptimizationReport` struct 정의
- `optimization_reports` 테이블 생성 (db.go migration)
- CRUD: `InsertOptimizationReport`, `GetOptimizationReports`, `UpdateOptimizationSuggestionStatus`
- settings 신규 키 `optimization_apply_mode` 기본값 INSERT

### Phase 2: Claude 분석 메서드
- `claude.go`: `AnalyzeDailyReport(ctx, dr DailyReport, currentSettings map[string]string) (*OptimizationResult, error)`
- 프롬프트: DailyReport 전체 데이터 + 현재 TradingRules 값 + 현재 settings 값 포함
- 응답 파싱 + 유효성 검사 (rule_name 화이트리스트, 수치 범위)

### Phase 3: 자동 적용 로직 & 스케줄러 연계
- `report/optimization.go`: `GenerateOptimizationSuggestions(ctx, db, claude, date)`
  - 거래 0건 → skip
  - Claude 분석 → DB 저장 (모든 제안 PENDING)
  - `optimization_apply_mode` 읽어서 자동 적용 대상 필터링
  - 자동 적용 대상: `ApplySuggestion()` 호출 → settings 테이블 업데이트 → status=APPLIED
- `main.go` 15:20 블록: `report.GenerateDailyReport()` 후 goroutine으로 `GenerateOptimizationSuggestions()` 호출

### Phase 4: API 핸들러 & 라우터
- `GET /api/reports/optimization?limit=10` — 제안 목록 (최신순)
- `POST /api/reports/optimization/:id/apply` — 특정 제안 수동 적용
- `POST /api/reports/optimization/:id/reject` — 특정 제안 무시
- `POST /api/reports/optimization/analyze` — 수동 분석 트리거 (일별 리포트 미생성 시 자동 생성 후 분석)

### Phase 5: 프론트엔드
- `OptimizationReports.jsx`: 날짜별 제안 카드, 카테고리별 탭, 상태 뱃지, [적용]/[무시] 버튼
- `Settings.jsx`: `optimization_apply_mode` 셀렉트 (전체자동/설정자동/전체수동)
- `App.jsx` + 네비게이션: "AI 개선 제안" 메뉴 추가

---

## Validation Rules (자동 적용 안전장치)

### settings 제안 범위 제한
| key | min | max | max_delta |
|---|---|---|---|
| `stop_loss_pct` | 0.3 | 5.0 | 1.0 |
| `take_profit_pct` | 0.5 | 15.0 | 2.0 |
| `order_amount_pct` | 10 | 99 | 10 |
| `max_positions` | 1 | 5 | 1 |
| `indicator_check_interval_min` | 1 | 60 | 10 |

### TradingRules 제안 범위 제한
| rule_name | min | max |
|---|---|---|
| `HardRSIMax` | 55 | 85 |
| `HardStrengthMin` | 80 | 130 |
| `HardDisparityM5Max` | 1.0 | 6.0 |
| `HardDisparityM5Min` | -5.0 | 0.0 |
| `RSIBuyMin` | 30 | 55 |
| `RSIBuyMax` | 50 | 75 |
| `BidAskRatioMin` | 0.8 | 2.0 |

범위 초과 시: status = PENDING 유지, 경고 로그 기록, 자동 적용 건너뜀

---

## DB Schema: optimization_reports

```sql
CREATE TABLE IF NOT EXISTS optimization_reports (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    date                TEXT NOT NULL,           -- YYYY-MM-DD (daily_reports.date 참조)
    overall_assessment  TEXT,                    -- Claude 전체 평가 요약
    suggestions         TEXT NOT NULL,           -- JSON: []OptimizationSuggestion
    apply_mode_snapshot TEXT,                    -- 생성 시점의 optimization_apply_mode
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(date)
);
```

### suggestions JSON 아이템 구조
```json
{
  "id": "uuid-or-seq",
  "category": "settings | prompt | feature",
  "key": "stop_loss_pct",          // settings/prompt 전용
  "rule_name": "HardRSIMax",       // prompt 전용
  "name": "볼린저밴드 %B",           // feature 전용
  "type": "indicator",              // feature 전용
  "current_value": "2.0",
  "suggested_value": "1.5",
  "comment": "근거 설명",
  "status": "PENDING"              // PENDING | APPLIED | REJECTED
}
```

suggestions는 단일 JSON 배열로 저장 (카테고리 혼합). API 응답 시 카테고리별 분류.

---

## Verification
1. `go build ./...` 빌드 통과
2. `npm run build` 빌드 통과
3. 수동 테스트:
   - `POST /api/reports/optimization/analyze` → optimization_reports 생성 확인
   - `GET /api/reports/optimization` → 제안 목록 반환 확인
   - `POST /api/reports/optimization/:id/apply` → settings 테이블 업데이트 + status=APPLIED 확인
   - Settings 화면에서 `optimization_apply_mode` 변경 → 저장 확인
4. 거래 0건 날: 분석 skip 로그 확인
5. `ANTHROPIC_API_KEY` 미설정: graceful skip 로그 확인
