# Plan: TPS 초과 및 Claude 토큰 부족 수정

## Goal
종목 선정 과정에서 발생하는 KIS API TPS 초과(EGW00201)와 Claude 응답 토큰 부족으로 인한 JSON 파싱 실패를 해결한다.

## 문제 요약

### 문제 1: KIS API TPS 초과
- `GetStockInfo` 루프에서 종목당 3회 KIS API 호출 (GetStockPrice + 분봉 + GetBidAskRatio)
- 40~50개 종목 × 3회 = 120~150회 순차 호출 → TPS 초과
- 거래대금 필터에서 `GetStockInfo` 중복 호출 (+40회)
- `GetBidAskRatio`는 LLM 랭킹용 보조 지표인데 서버 필터 전 전체에 호출

### 문제 2: Claude MaxTokens 부족
- `MaxTokens: 2048` 설정으로 40개 종목 추론 텍스트가 응답 중간에 잘림
- JSON 배열이 생성되기 전에 응답 truncated → `claude response has no JSON array` 오류

## 수정 방향

### 문제 1 수정
1. **`GetBidAskRatio` 분리**: `GetStockInfo`에서 제거, 최종 후보에만 별도 호출
2. **병렬 호출**: `GetStockInfo` 루프를 세마포어(동시 3개) 기반 goroutine으로 변경
3. **중복 제거**: 거래대금 필터에서 `GetStockInfo` 재호출 → `item.TradingValue` 직접 사용

### 문제 2 수정
1. **MaxTokens 증가**: 2048 → 4096
2. **프롬프트 추론 금지**: Chain-of-Thought 억제 문구 추가
3. **서버 사전 점수화**: Claude 랭킹 기준(MA배열, MACD, RSI, VWAPDiff, PrevVolumeRatio)으로 점수 계산 후 상위 N개만 전달
4. **`max_claude_candidates` 설정키**: 기본값 15, DB settings 테이블로 관리

## 점수화 기준 (scoreCandidate)

| 지표 | 조건 | 점수 |
|------|------|------|
| MA 배열 | ma5 > ma20 | +2.0 |
| MACD | macd_line > macd_signal | +1.0 |
| 거래량 감소 | prev_volume_ratio < 0.8 | +1.0 |
| RSI | 최적구간 중간에 가까울수록 | +0~1.0 |
| VWAPDiff | 최적구간 중간에 가까울수록 | +0~1.0 |

## 영향 파일

| 파일 | 변경 내용 |
|------|----------|
| `backend/internal/agent/stock_info.go` | `GetBidAskRatio` 호출 제거 |
| `backend/internal/trader/engine.go` | 병렬화, 중복 제거, 점수화, 후보 제한, BidAskRatio 후처리 |
| `backend/internal/trader/claude.go` | MaxTokens 4096, 추론 금지 프롬프트 |
| `backend/internal/database/db.go` | `max_claude_candidates` 설정키 추가 |

## 예상 효과

| 항목 | 수정 전 | 수정 후 |
|------|---------|---------|
| KIS API 호출 횟수 | ~264회 순차 | ~106회 병렬(3개씩) |
| 종목 선정 소요 시간 | ~30초 | ~4~6초 |
| Claude 입력 종목 수 | 40개+ | 최대 15개 |
| Claude MaxTokens | 2048 | 4096 |
