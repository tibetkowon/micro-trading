# 데이터 품질 개선 계획 (1~5순위)

## Goal
Claude `SelectStocks`에 전달되는 데이터 품질을 개선하여 눌림목 판단 정확도를 높인다.
추가 API 호출 없이(5순위 포함) 기존 데이터를 더 풍부하게 활용한다.

## Requirements
- 추가 KIS API 호출: 0회 (기존 호출의 응답 데이터를 더 많이 활용)
- 기존 `BidAskRatio` 호환성 유지 (engine.go 사전 점수화 로직 그대로)
- Go 빌드 통과

## Affected Files
| 파일 | 변경 내용 |
|------|----------|
| `backend/internal/agent/stock_info.go` | `StockInfo`에 신규 필드 추가 + 계산 로직 |
| `backend/internal/kis/client.go` | `GetBidAskRatio` → `GetOrderBookSnapshot` 확장 |
| `backend/internal/trader/claude.go` | `RankItem` 신규 필드 + 프롬프트 개선 |
| `backend/internal/trader/engine.go` | `StockInfo`→`RankItem` 매핑 + OrderBookSnapshot 호출 |

## Implementation Phases

### Phase 1: agent/stock_info.go — StockInfo 신규 필드
- `RecentCandles [5]CandleSnap` — 최근 5분봉 (close, volume, dir)
- `HighFormedMinsAgo int` — 고점 형성 후 경과 시간(분)
- `VolTrend3 float64` — 최근 3봉 거래량 기울기 (-1~1)
- `VolAtHigh int64` — 고점 봉의 거래량

### Phase 2: kis/client.go — OrderBookSnapshot 확장
- `OrderBookSnapshot` 타입 추가 (BidAskRatio + NearBidAskRatio + TopAskWall + TopAskWallSize)
- `GetOrderBookSnapshot` 함수 추가 (기존 `GetBidAskRatio` 유지 + 내부 위임)

### Phase 3: trader/claude.go — RankItem + 프롬프트
- `RankItem`에 Phase1/2 신규 필드 추가
- `SelectStocks` 프롬프트에 SessionPhase(장 시간대) 주입
- 프롬프트 해석 가이드 추가 (recent_candles, high_formed_mins_ago 등)

### Phase 4: trader/engine.go — 매핑 업데이트
- `StockInfo` → `RankItem` 매핑 블록에 신규 필드 추가
- `GetBidAskRatio` → `GetOrderBookSnapshot` 교체 후 필드 분리 매핑

## Verification
```bash
cd backend && go build ./...
```
