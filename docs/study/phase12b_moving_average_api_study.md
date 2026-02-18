# Phase 12b: 이동평균(MA5/MA20) API 응답 내장 학습 문서

## 1. Overview — 이 기능이 왜 필요한가?

외부 AI 에이전트가 매수 판단 시 이동평균선을 기준으로 삼으려면 종가 데이터를 받은 뒤 직접 MA를 계산해야 합니다.
그러면 에이전트가 소비하는 토큰과 연산 시간이 늘어납니다.

**서버가 미리 계산해서 응답에 포함**하면:
- 에이전트는 `ma5`, `ma20` 필드만 읽으면 되므로 처리가 단순해짐
- 계산 로직을 에이전트 프롬프트에 넣지 않아도 됨
- 모든 클라이언트(UI, 봇, 외부 AI)가 동일한 값을 공유

### 변경 전/후 응답 비교

```json
// 변경 전 — 클라이언트가 직접 계산해야 함
{ "date": "2025-01-25", "close": 75400, "volume": 12000000 }

// 변경 후 — 서버가 미리 계산해서 제공
{ "date": "2025-01-25", "close": 75400, "volume": 12000000,
  "ma5": 74820.0, "ma20": 73150.0 }
```

---

## 2. Logic Flow — 데이터가 어떻게 흘러가는가?

```
GET /api/market/daily-prices/{symbol}?days=60
  └─ api/market.py: get_daily_prices()
       └─ MarketService.get_daily_prices()
            ├─ broker.get_daily_prices()     ← raw OHLCV 반환 (날짜 정렬 순서 불일치)
            │    ├─ FreeProvider (모의):  pykrx → 오름차순 (과거→현재)
            │    └─ KisBroker  (실거래):  KIS API → 내림차순 (현재→과거)
            └─ _add_moving_averages(prices)  ← 정렬 정규화 + MA 계산
                 ├─ sorted(prices, key=lambda x: x["date"])  오름차순 통일
                 ├─ closes = [p["close"] for p in sorted_prices]
                 ├─ i >= 4  → ma5  = sum(closes[i-4:i+1]) / 5
                 ├─ i >= 19 → ma20 = sum(closes[i-19:i+1]) / 20
                 └─ return sorted_prices  (날짜 오름차순)
```

### 핵심 설계 선택: "서비스 레이어에서 후처리"

```
브로커(데이터 수집)  →  서비스(비즈니스 로직)  →  API(응답 직렬화)
   raw OHLCV                 + MA 계산                + JSON 반환
```

broker는 데이터 수집만 담당합니다.
MA는 비즈니스 로직이므로 **서비스 레이어**(`market_service.py`)에서 처리합니다.
덕분에 broker를 교체(pykrx → KIS → 다른 API)해도 MA 계산 코드는 그대로입니다.

---

## 3. Pythonic Point — 파이썬 특유의 기법

### 슬라이싱으로 이동합계 계산

```python
# MA5: 현재 행 포함 과거 5개 종가의 평균
ma5 = sum(closes[i - 4 : i + 1]) / 5
```

- `closes[i-4 : i+1]`는 인덱스 `i-4`부터 `i`까지 5개의 원소
- 파이썬 슬라이싱은 음수 인덱스가 아닌 이상 범위를 벗어나도 오류 없이 빈 리스트 반환
- 단, 앞 가드 조건(`if i >= 4`)으로 부족한 구간을 명시적으로 `None` 처리

```python
for i, p in enumerate(sorted_prices):
    p["ma5"]  = round(sum(closes[i - 4:i + 1]) / 5, 2)  if i >= 4  else None
    p["ma20"] = round(sum(closes[i - 19:i + 1]) / 20, 2) if i >= 19 else None
```

Java 스타일로 for 두 개를 쓰지 않고, `enumerate()`로 인덱스와 원소를 동시에 순회합니다.

### `sorted()` — 원본 보존 정렬

```python
sorted_prices = sorted(prices, key=lambda x: x["date"])
```

- `list.sort()`: **원본 in-place 변경** (원본 훼손)
- `sorted()`: **새 리스트 반환** (원본 보존)

broker에서 받은 `prices`를 훼손하지 않기 위해 `sorted()`를 씁니다.
`key=lambda x: x["date"]`는 문자열 날짜(`"2025-01-25"`)의 사전순 비교가 날짜순 비교와 동일하다는 점을 활용합니다.

### 인라인 조건 표현식 (삼항 연산자)

```python
# Java
ma5 = (i >= 4) ? Math.round(sum / 5.0 * 100) / 100.0 : null;

# Python
ma5 = round(sum(closes[i - 4:i + 1]) / 5, 2) if i >= 4 else None
```

파이썬은 `값_참 if 조건 else 값_거짓` 순서입니다.
Java의 `조건 ? 참 : 거짓`과 순서가 다르니 주의합니다.

---

## 4. Java vs Python 비교

### 이동평균 계산: Stream vs 슬라이싱

```java
// Java — Stream + subList
OptionalDouble ma5 = (i >= 4)
    ? OptionalDouble.of(
        closes.subList(i - 4, i + 1).stream()
              .mapToDouble(Double::doubleValue)
              .average()
              .orElse(0.0))
    : OptionalDouble.empty();
```

```python
# Python — 슬라이싱 + 내장 sum()
ma5 = round(sum(closes[i - 4:i + 1]) / 5, 2) if i >= 4 else None
```

파이썬의 내장 `sum()`은 iterable(리스트, 슬라이스 등)을 직접 받습니다.
`mapToDouble()` 같은 변환 단계가 필요 없습니다.

### 날짜 문자열 정렬: Comparator vs lambda

```java
// Java
prices.sort(Comparator.comparing(p -> p.get("date").toString()));
```

```python
# Python
sorted_prices = sorted(prices, key=lambda x: x["date"])
```

파이썬의 `lambda`는 Java의 메서드 참조(`p -> p.get("date")`)와 동일한 역할입니다.
단, 파이썬은 `sorted()`가 새 리스트를 반환하고 `list.sort()`가 in-place입니다.
Java는 `List.sort()`가 in-place입니다.

### 후처리 레이어 위치: @Component(Bean) vs 모듈 함수

```java
// Java — 별도 @Component로 분리
@Component
public class MovingAverageCalculator {
    public List<PriceDto> enrich(List<PriceDto> prices) { ... }
}

// OrderService에 @Autowired로 주입
```

```python
# Python — 같은 모듈의 모듈 수준 함수 (private convention)
def _add_moving_averages(prices: list[dict]) -> list[dict]:
    ...

class MarketService:
    async def get_daily_prices(...):
        prices = await broker.get_daily_prices(...)
        return _add_moving_averages(prices)  # 직접 호출
```

파이썬에서는 DI 컨테이너 없이 모듈 함수로 유틸리티를 구성합니다.
`_` 접두사는 "이 파일 내부에서만 쓰는 함수"라는 파이썬 관례입니다.

---

## 5. Key Concept — 놓치기 쉬운 포인트

### Broker별 날짜 정렬 순서 불일치

| Broker | 정렬 순서 | 이유 |
|--------|-----------|------|
| FreeProvider (pykrx) | 오름차순 (과거→현재) | `df.iterrows()` 가 날짜 오름차순 |
| KisBroker (KIS API) | 내림차순 (현재→과거) | KIS API 기본 응답이 최신일 우선 |

이 불일치를 무시하고 MA를 계산하면 broker에 따라 결과가 뒤집힙니다.
`sorted(prices, key=lambda x: x["date"])`로 **항상 오름차순으로 정규화**한 뒤 계산합니다.

### 문자열 날짜의 사전순 = 날짜순

```python
"2025-01-09" < "2025-01-10"  # True ← 사전순이 날짜순과 일치
```

`YYYY-MM-DD` 형식은 자릿수가 고정되어 있어 문자열 사전순 비교가 날짜 비교와 동일합니다.
`datetime.strptime()` 변환 없이도 `key=lambda x: x["date"]`만으로 올바르게 정렬됩니다.

### `round(value, 2)` — 부동소수점 오차 제거

```python
>>> 74200 * 0.1 + 74400 * 0.1 + 74600 * 0.1 + 74800 * 0.1 + 75000 * 0.1
37300.000000000004  # 부동소수점 오차

>>> round(sum([74200, 74400, 74600, 74800, 75000]) / 5, 2)
74600.0  # 깔끔
```

주가처럼 정수 기반 데이터라도 나눗셈 후 부동소수점 오차가 생길 수 있습니다.
`round(..., 2)`로 소수점 2자리로 제한합니다.

### 전략 코드와의 관계 — MA 이중 계산 없음

기존 전략(`moving_average.py`, `rsi_rebalance.py`)은 내부적으로 `closes`를 직접 계산합니다.
이번 변경으로 API 응답에 MA가 추가됐지만, **전략은 API를 호출하지 않고 `daily_prices` 리스트를 직접 받습니다**.

```
전략 실행 흐름:
  runner.py → market_svc.get_daily_prices()  ← MA 포함 반환
            → strategy.should_buy(daily_prices=prices)  ← MA 필드는 무시, closes만 사용
```

전략이 `ma5`, `ma20` 필드를 활용하도록 개선하는 것은 향후 리팩토링 포인트입니다.
현재는 API 클라이언트(외부 AI, UI)를 위한 편의 필드입니다.
