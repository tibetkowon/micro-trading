# 트레이딩 고도화 계획 — 2026-03-22

## Goal

단타(데이트레이딩) 정확도 향상을 위한 5가지 개선:
1. **요일별 트레이딩 스케줄** — 특정 요일(예: 금요일) 거래 비활성화
2. **신규 기술 지표 수집** — VWAP, M5_MA10, PrevVolumeRatio, BidAskRatio
3. **프롬프트 로직 강화** — 새로운 하드 리젝션 룰 + 랭킹 기준 개선
4. **프롬프트 파라미터화** — 하드코딩 수치 → DB 설정값으로 대체
5. **트레이딩 프리셋** — 공격적/방어적 등 세팅 세트를 DB에 저장하고 즉시 전환

---

## 현황 분석 (Already Done)

| 항목 | 현재 상태 |
|------|----------|
| 지수 킬스위치 | ✅ `IndexCodes` + `IndexDropThresholdPct` 설정 존재, `GetIndexPrice` 구현됨 |
| RSI14 | ✅ `StockInfo.RSI14`, `RankItem.RSI14` 이미 계산됨 |
| MACD | ✅ `MACDLine`, `MACDSignal`, `MACDHisto` 이미 계산됨 |
| DisparityM5 | ✅ `DisparityM5` (5분봉 MA5 이격도) 이미 계산됨 |
| HighPriceDiff | ✅ 이미 계산됨 |
| OpenPriceDiff | ✅ 이미 계산됨 |
| 요일 스케줄 | ❌ 미구현 |
| VWAP | ❌ 미계산 (분봉 cntg_vol은 이미 수집 중 → 계산만 추가하면 됨) |
| M5_MA10 | ❌ 미계산 (MA5, MA20만 있음) |
| PrevVolumeRatio | ❌ 미계산 (분봉 데이터 있음 → 계산만 추가) |
| BidAskRatio | ❌ 별도 KIS API 필요 (`inquire-asking-price-exp-ccn`) |
| 지수 등락률 → Claude 전달 | ❌ 현재 엔진 내부 킬스위치로만 사용, 프롬프트에 미전달 |
| 프롬프트 파라미터화 | ❌ 수치 하드코딩 상태 |
| 프리셋 | ❌ 미구현 |

---

## 영향 파일

### Backend
| 파일 | 변경 내용 |
|------|----------|
| `internal/database/db.go` | 신규 설정키 INSERT OR IGNORE 추가 |
| `internal/models/models.go` | 신규 설정 필드 추가 (TradingDays, 하드룰 수치들, 프리셋) |
| `internal/agent/stock_info.go` | VWAP, M5_MA10, PrevVolumeRatio 계산 추가 |
| `internal/kis/client.go` | `GetBidAskRatio()` 신규 함수 (호가잔량 API) |
| `internal/trader/claude.go` | `RankItem` 신규 필드 추가, 프롬프트 파라미터화 |
| `internal/trader/engine.go` | 요일 체크 로직, 지수 등락률 프롬프트 전달, SelectStocks 파라미터 확장 |
| `internal/api/handlers.go` | UpdateSettings 요일 파싱, 프리셋 CRUD |
| `internal/api/router.go` | 프리셋 라우트 추가 |

### Frontend
| 파일 | 변경 내용 |
|------|----------|
| `src/pages/Settings.jsx` | 요일 선택 UI, AI 기준값 섹션, 프리셋 선택기 |
| `src/mocks/handlers.js` | 신규 mock 데이터 |

---

## Implementation Phases

### Phase 1 — 요일별 트레이딩 스케줄 (DB + Backend + Frontend)

**DB 설정키 추가** (`db.go` — `INSERT OR IGNORE` 블록):
```
trading_days  = "[1,2,3,4]"    // 1=월 2=화 3=수 4=목 5=금 (0=일 6=토)
```

**Backend** (`engine.go`):
- `scheduleReady()` 함수에서 `time.Now().In(kst).Weekday()` 확인
- `trading_days` 설정에 현재 요일이 없으면 엔진 시작 스킵
- 15:15 종료 시에도 해당 요일 아닐 경우 아예 시작 안 함

**Frontend** (`Settings.jsx`):
```
거래 제어 섹션에:
☑ 월  ☑ 화  ☑ 수  ☑ 목  ☐ 금  ☐ 토  ☐ 일
```

---

### Phase 2 — 신규 기술 지표 수집 (agent + kis)

#### 2a. VWAP (분봉 데이터 재활용 — 신규 API 불필요)

`GetStockInfo()` 내부에서 기존 1분봉 배열(`output2`)을 순회:
```go
// 이미 받은 분봉 배열에서 계산
var sumPV, sumV float64
for _, bar := range minBars {
    p := parseFloat(bar.stck_prpr)
    v := parseFloat(bar.cntg_vol)
    sumPV += p * v
    sumV += v
}
if sumV > 0 { info.VWAP = sumPV / sumV }
// VWAPDiff = (CurrentPrice - VWAP) / VWAP * 100
info.VWAPDiff = (currentPrice - info.VWAP) / info.VWAP * 100
```

#### 2b. M5_MA10 (5분봉 배열 재활용)

이미 5분봉 배열(`m5Bars`)을 계산하고 있으므로:
```go
if len(m5Closes) >= 10 {
    info.M5MA10 = mean(m5Closes[len(m5Closes)-10:])
}
```

#### 2c. PrevVolumeRatio (5분봉 마지막 2개 캔들)

```go
if len(m5Volumes) >= 2 {
    cur := m5Volumes[len(m5Volumes)-1]
    prev := m5Volumes[len(m5Volumes)-2]
    if prev > 0 { info.PrevVolumeRatio = cur / prev }
}
```

#### 2d. BidAskRatio (신규 KIS API 호출)

- **API**: `GET /uapi/domestic-stock/v1/quotations/inquire-asking-price-exp-ccn`
- **TR_ID**: `FHKST01010200`
- **응답 필드**: `total_askp_rsqn` (총 매도잔량), `total_bidp_rsqn` (총 매수잔량)
- **계산**: `BidAskRatio = total_bidp_rsqn / total_askp_rsqn`
- **주의**: 별도 API 콜 1회 추가. TPS 초과 방지를 위해 기존 rate limiter 적용

```go
func (c *Client) GetBidAskRatio(ctx context.Context, stockCode string) (float64, error)
```

#### 2e. 지수 등락률 → Claude 프롬프트 전달

현재 `engine.go`에서 인덱스 체크는 내부 킬스위치용으로만 사용.
개선: 체크 결과(`marketIndexDrop float64`)를 `SelectStocks()`에 파라미터로 전달 → 프롬프트에 포함.

```go
// engine.go
type TradingContext struct {
    MarketIndexDrop float64  // KOSPI/KOSDAQ 시가 대비 현재 등락률
    AvailableCash   float64
    HardRules       HardRuleParams  // Phase 3에서 추가
}
```

---

### Phase 3 — 프롬프트 파라미터화 + 하드룰 강화 (claude.go)

#### 신규 DB 설정키

```
hard_disparity_m5_min     = "-1.5"    // 5분봉 MA5 이격도 하한 (이하 → 칼날 하락)
hard_disparity_m5_max     = "3.0"     // 5분봉 MA5 이격도 상한 (이상 → 과열)
hard_high_price_diff_max  = "-0.5"    // 고점 대비 최대 허용 (이상 → 고점권)
hard_high_price_diff_min  = "-5.0"    // 고점 대비 최소 (이하 + 거래량 급증 → 추세이탈)
hard_prev_vol_ratio_max   = "1.2"     // 하락 시 전 캔들 대비 거래량 비율 상한
hard_strength_min         = "100.0"   // 최소 체결강도 (이하 → 매수세 소멸)
hard_rsi_max              = "70.0"    // RSI 상한 (이상 → 과매수에서 꺾임)
hard_open_price_diff_max  = "15.0"    // 시가 대비 상승률 상한
vwap_diff_min             = "0.0"     // VWAP 이격도 하한 (VWAP 지지선 위)
vwap_diff_max             = "1.5"     // VWAP 이격도 상한
rsi_buy_min               = "40.0"    // 이상적 RSI 매수 구간 하한
rsi_buy_max               = "60.0"    // 이상적 RSI 매수 구간 상한
bid_ask_ratio_min         = "1.2"     // 최소 매수호가 우세 비율
```

#### 개선된 KR 프롬프트 구조 (`claude.go`)

```
## Hard Rejection Rules — skip if ANY apply:
1. market_index_drop < {{MARKET_DROP_THRESHOLD}}% → 시장 급락 킬스위치
2. disparity_m5 > {{MAX_DISPARITY_M5}}% OR disparity_m5 < {{MIN_DISPARITY_M5}}% → 이탈
3. high_price_diff > {{MAX_HIGH_DIFF}}% → 고점 추격 위험
4. high_price_diff < {{MIN_HIGH_DIFF}}% AND prev_volume_ratio > {{MAX_PREV_VOL}}% → 추세이탈
5. ma5 < ma20 → 5분봉 역배열(하락추세)
6. strength < {{MIN_STRENGTH}} → 매수세 소멸
7. rsi14 > {{MAX_RSI}} → 과매수 꺾임
8. open_price_diff > {{MAX_OPEN_DIFF}}% → 상한가 영역

## Ranking Criteria:
- vwap_diff: {{VWAP_DIFF_MIN}}% ~ {{VWAP_DIFF_MAX}}% (VWAP 지지선 근처)
- prev_volume_ratio < 0.8 (눌림 시 거래량 감소)
- bid_ask_ratio > {{MIN_BID_ASK}} (매수 호가 우세)
- rsi14: {{RSI_BUY_MIN}} ~ {{RSI_BUY_MAX}} (반등 구간)
```

---

### Phase 4 — 프리셋 시스템 (DB + API + Frontend)

**DB**: `settings_presets` 테이블 신규 생성
```sql
CREATE TABLE settings_presets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,           -- "공격적", "방어적", "금요일"
    description TEXT,
    settings_json TEXT NOT NULL,         -- JSON snapshot of relevant settings
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME
);
```

**API 엔드포인트**:
- `GET /api/presets` — 목록 조회
- `POST /api/presets` — 현재 설정을 새 프리셋으로 저장
- `POST /api/presets/:id/apply` — 프리셋 설정값을 현재 settings에 적용
- `DELETE /api/presets/:id` — 삭제

**Frontend**: Settings 상단에 프리셋 드롭다운 + "저장" / "적용" 버튼

---

## 구현 우선순위 (권장 순서)

| 우선순위 | Phase | 이유 |
|---------|-------|------|
| 1 | Phase 2a,2b,2c (VWAP/MA10/PrevVol) | 신규 API 불필요, 즉시 효과 |
| 2 | Phase 3 (프롬프트 파라미터화 + 강화) | 정확도 직접 향상 |
| 3 | Phase 1 (요일 스케줄) | 사용자 요청 #1 |
| 4 | Phase 2e (지수등락률 → Claude) | 이미 API 있음 |
| 5 | Phase 2d (BidAskRatio) | 신규 API 콜 필요 |
| 6 | Phase 4 (프리셋) | 편의 기능 |

---

## 단계별 검증 방법

- **Phase 1**: Settings에서 요일 체크 후 해당 요일에 엔진이 시작 안 되는지 로그 확인
- **Phase 2**: 서비스 로그 또는 selection log에 VWAP, VWAPDiff, PrevVolumeRatio 값 출력 확인
- **Phase 3**: Claude 호출 직전 프롬프트를 DEBUG 로그로 출력하여 파라미터 치환 확인
- **Phase 4**: 프리셋 저장 → 다른 설정값으로 변경 → 프리셋 적용 → 설정값 복원 확인

---

## 기술 노트

- **BidAsk API TPS**: 별도 호출이므로 종목당 +1 req. 현 rate limiter(`rateLimiter`)에 자동 포함됨
- **VWAP 정확도**: 장 초반(9:00~9:15) 분봉 수가 적을 경우 VWAP 신뢰도 낮음 → `vwap_bar_count < 5`이면 VWAP 필드 0으로 세팅하고 프롬프트에서 N/A 처리
- **메모리**: 지표 필드 추가로 RankItem 구조체 크기 증가하나 SQLite + 1GB RAM 환경에서 무시 가능 수준
