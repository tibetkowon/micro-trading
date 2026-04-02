# KIS 종목마스터(MST) 파일 명세

## 개요

KIS Developers에서 매일 오전 제공하는 전체 종목 마스터 파일.
장 시작 전(08:40) 자동 다운로드하여 SQLite `stock_masters` 테이블에 UPSERT.

---

## 다운로드 URL

| 시장 | URL |
|------|-----|
| KOSPI | `https://new.real.download.dws.co.kr/common/master/kospi_code.mst.zip` |
| KOSDAQ | `https://new.real.download.dws.co.kr/common/master/kosdaq_code.mst.zip` |

- **인증 불필요** (공개 URL, HTTP GET)
- ZIP 파일 내 단일 `.mst` 파일 포함
- 2026-04-02 기준: KOSPI 2,501건(288B/레코드), KOSDAQ 1,825건(282B/레코드)

---

## 파일 포맷

### 레코드 구조 (바이너리 고정폭, 인코딩: EUC-KR)

각 레코드는 `\n` (0x0A) 으로 구분됨.

#### KOSPI (288 bytes/레코드)

| 오프셋 | 길이 | 인코딩 | 필드명 | 설명 |
|--------|------|--------|--------|------|
| 0 | 6 | ASCII | 단축코드 | 종목코드 6자리 (예: `005930`) |
| 6 | 3 | ASCII | 공백패딩 | 스페이스 |
| 9 | 12 | ASCII | 표준코드 | ISIN 코드 (예: `KR7005930003`) |
| 21 | 40 | EUC-KR | 한글종목명 | 종목 한글명 (공백 패딩) |
| 61 | 2 | ASCII | 소속부코드 | 시장 분류 코드 (아래 참조) |
| 63+ | 나머지 | 혼합 | 기타 필드 | 상장주식수 등 (offset ~180) |

#### KOSDAQ (282 bytes/레코드)

| 오프셋 | 길이 | 인코딩 | 필드명 | 설명 |
|--------|------|--------|--------|------|
| 0 | 6 | ASCII | 단축코드 | 종목코드 6자리 |
| 6 | 3 | ASCII | 공백패딩 | 스페이스 |
| 9 | 12 | ASCII | 표준코드 | ISIN 코드 |
| 21 | 40 | EUC-KR | 한글종목명 | 종목 한글명 |
| 61 | 2 | ASCII | 소속부코드 | 시장 분류 코드 |
| 63+ | 나머지 | 혼합 | 기타 필드 | |

### 소속부코드 (offset 61-62)

| 코드 | 의미 |
|------|------|
| `ST` | 일반 주식 (보통주) |
| `EF` | ETF (Exchange Traded Fund) |
| `BC` | 투자신탁 (뮤추얼펀드 등) |
| `FS` | 외국주식 |
| `MF` | 매매거래정지 |
| `RT` | 리츠 (REITs) |

---

## ETF 세금 분류 규칙

`소속부코드 == "EF"` 인 종목이 ETF이나, **비과세 여부**는 국내주식형 여부로 결정됨.

### 국내주식형 ETF (비과세, applicable_tax_rate = 0.0)
- 기초자산이 국내 주식/지수인 ETF
- 종목명 키워드로 식별:
  - 포함: `200`, `코스닥`, `코스피`, `KRX`, `KTOP`, `KQ`
  - 예: `TIGER 200`, `KODEX 코스닥150`, `TIGER KTOP30`

### 비국내주식형 ETF (과세, applicable_tax_rate = stock_tax_rate)
- 해외 지수/원자재/채권/파생형 ETF
- 종목명 키워드로 식별:
  - 포함: `미국`, `나스닥`, `S&P`, `달러`, `채권`, `금`, `원유`, `(합성)`, `커버드콜`, `배당`, `리츠`
  - 예: `TIGER 미국나스닥100`, `KODEX 골드선물`, `TIGER 26-04 회사채`

> **주의**: 키워드 기반 분류는 완벽하지 않음. MST 파싱 후 `stock_masters` 테이블에서 `is_domestic_equity_etf` 수동 수정 가능.

---

## Go 파싱 코드 예시

```go
// backend/internal/mst/parser.go

const (
    kospiRecordLen  = 288
    kosdaqRecordLen = 282
    codeOffset      = 0
    codeLen         = 6
    isinOffset      = 9
    isinLen         = 12
    nameOffset      = 21
    nameLen         = 40  // EUC-KR bytes
    groupOffset     = 61
    groupLen        = 2
)

type StockMaster struct {
    StockCode           string
    StockName           string
    ISIN                string
    MarketType          string // KOSPI / KOSDAQ
    GroupCode           string // ST / EF / BC / FS ...
    IsETF               bool
    IsDomesticEquityETF bool   // 비과세 여부
    ListedShares        int64
    UpdatedAt           time.Time
}

func parseRecord(rec []byte, market string) (*StockMaster, error) {
    if len(rec) < 63 {
        return nil, fmt.Errorf("record too short: %d", len(rec))
    }
    code := strings.TrimSpace(string(rec[codeOffset : codeOffset+codeLen]))
    isin := strings.TrimSpace(string(rec[isinOffset : isinOffset+isinLen]))
    nameEUCKR := rec[nameOffset : nameOffset+nameLen]
    nameUTF8, _ := charmap.ToUTF8(nameEUCKR)  // golang.org/x/text/encoding/korean
    name := strings.TrimSpace(nameUTF8)
    group := strings.TrimSpace(string(rec[groupOffset : groupOffset+groupLen]))

    isETF := group == "EF"
    isDomesticEquityETF := isETF && classifyDomesticEquityETF(name)

    return &StockMaster{
        StockCode:           code,
        StockName:           name,
        ISIN:                isin,
        MarketType:          market,
        GroupCode:           group,
        IsETF:               isETF,
        IsDomesticEquityETF: isDomesticEquityETF,
    }, nil
}

func classifyDomesticEquityETF(name string) bool {
    domestic := []string{"200", "코스닥", "코스피", "KRX", "KTOP", "KQ150"}
    foreign := []string{"미국", "나스닥", "S&P", "달러", "채권", "금", "원유", "합성", "커버드콜"}
    for _, kw := range foreign {
        if strings.Contains(name, kw) { return false }
    }
    for _, kw := range domestic {
        if strings.Contains(name, kw) { return true }
    }
    return false  // 불명확한 경우 false (과세 적용)
}
```

---

## 의존성

```
golang.org/x/text v0.x.x  // EUC-KR → UTF-8 변환
```

프로젝트 `go.mod`에 이미 포함 여부 확인 필요. 없으면 추가:
```bash
go get golang.org/x/text
```

---

## 재시도 및 폴백 로직

```
08:40 다운로드 시도
  ├── 성공 → stock_masters UPSERT
  └── 실패 → 5분 후 재시도 (최대 3회)
              └── 모두 실패 → 전일 데이터 유지 + 경고 로그 + 알림
```

장중 거래정지 동기화:
- 5분마다 순위 API 응답의 `iscd_stat_cls_code` 확인
- 거래정지(`iscd_stat_cls_code != ""`) 종목은 hard_watch_symbols 후보에서 제외
