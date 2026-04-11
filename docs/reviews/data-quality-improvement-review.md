# 코드 리뷰: Claude 종목 선정 데이터 품질 개선 (1~5순위)

## 개요

Claude에게 전달하는 `RankItem` 데이터를 풍부하게 만들어 눌림목 판단 정확도를 높이는 작업입니다.  
핵심 원칙은 **추가 KIS API 호출 없이** 이미 가져오는 데이터에서 더 많은 정보를 추출하는 것입니다.

---

## Go 백엔드 해설

### 1순위: 최근 5분봉 캔들 시퀀스 (`agent/stock_info.go`)

```go
type CandleSnap struct {
    Close  float64 `json:"c"`
    Volume int64   `json:"v"`
    Dir    string  `json:"d,omitempty"` // "U", "D", "="
}
```

- Go에서 **배열 슬라이싱**으로 마지막 5개 요소 추출: `candles5m[len(candles5m)-5:]`
- `omitempty` 태그: 필드가 zero value이면 JSON 출력에서 생략됨

Claude가 "D/D/D 연속 + volume 감소" 패턴을 직접 볼 수 있어 눌림목 판별 품질이 올라갑니다.

---

### 2순위: 고점 형성 경과 시간

```go
for i := len(candles5m) - 1; i >= 0; i-- {
    if candles5m[i].High >= dayHigh*0.9999 {
        highIdx = i
        break
    }
}
minsAgo := (len(candles5m) - 1 - highIdx) * 5
```

- **역방향 탐색**: 가장 최근에 고점에 도달한 봉을 찾기 위해 뒤에서부터 탐색
- `*0.9999`: 부동소수점 비교의 정밀도 오차를 허용하는 관용구

고점 형성 후 5분 미만이면 아직 하락 중, 15~45분이면 안정적 눌림목 구간입니다.

---

### 3순위: 거래량 기울기

```go
slope := (v3 - v1) / maxV
info.VolTrend3 = math.Round(slope*100) / 100
```

- 단순 기울기 정규화: 최대값으로 나눠 -1~1 범위로 만듦
- `math.Round(x*100)/100`: Go에서 소수점 2자리 반올림 관용구 (기본 Round가 없어서 직접 계산)

---

### 4순위: 장 시간대 컨텍스트 (`trader/claude.go`)

```go
totalMin := hour*60 + min
switch {
case totalMin < 10*60:
    sessionPhase = "OPEN"
...
}
```

- 시/분을 분 단위로 통합(`hour*60 + min`)해서 범위 비교를 단순하게 처리
- `time.Now()`는 서버 KST 기준이므로 별도 timezone 변환 불필요

---

### 5순위: 호가창 분포 세분화 (`kis/client.go`)

```go
type OrderBookSnapshot struct {
    BidAskRatio     float64 // 전체 비율
    NearBidAskRatio float64 // ±2% 근거리만
    TopAskWall      float64 // 최대 매도벽 위치 (%)
    TopAskWallSize  int64   // 최대 매도벽 잔량
}
```

**NearBidAskRatio 계산 원리:**
```go
nearBand := currentPrice * 0.02
for i := 0; i < 10; i++ {
    if math.Abs(ap - currentPrice) <= nearBand {
        nearAsk += ar  // 범위 안의 매도 잔량만 합산
    }
}
```
- 전체 호가 중 현재가 근처만 필터링 → 실질적인 매수/매도 압력 측정

**TopAskWall 찾기:**
```go
snap.TopAskWall = math.Round((ap-currentPrice)/currentPrice*10000) / 100
```
- `×10000÷100` = `×100`과 같지만, 중간에 `Round`를 통해 소수점 2자리로 정규화
- 목표가보다 낮은 위치에 큰 벽이 있으면 돌파 어려움 → 진입 자제 신호

**하위 호환성 유지:**
```go
func (c *Client) GetBidAskRatio(...) (float64, error) {
    snap, err := c.GetOrderBookSnapshot(ctx, stockCode, 0)
    return snap.BidAskRatio, err
}
```
- 기존 함수는 그대로 두고 내부에서 새 함수에 **위임(delegation)**
- 다른 코드에서 `GetBidAskRatio`를 부르는 곳이 있어도 수정 불필요

---

## 핵심 요약

| 개념 | 설명 |
|------|------|
| 역방향 슬라이스 탐색 | `for i := len(s)-1; i >= 0; i--` |
| 부동소수점 범위 비교 | `*0.9999` 또는 `math.Abs(a-b) <= epsilon` |
| JSON `omitempty` | zero value 필드는 JSON 직렬화 시 생략 |
| 함수 위임 패턴 | 기존 함수 → 새 함수 내부 호출로 하위호환 유지 |
| 정규화 반올림 | `math.Round(x * 100) / 100` |
