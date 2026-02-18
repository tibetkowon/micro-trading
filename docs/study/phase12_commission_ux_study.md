# Phase 12: 수수료 UX 개선 & 실거래 예수금 자동 갱신 학습 문서

## 1. Overview — 이 기능이 왜 필요한가?

이번 Phase는 세 가지 실용적인 문제를 동시에 해결합니다.

| 문제 | 원인 | 해결 |
|------|------|------|
| 실거래 예수금이 오래된 값 표시 | 페이지 로드 시 1회만 조회 | HTMX 30초 폴링 추가 |
| 체결 수수료가 DB에만 있고 UI에 없음 | `Trade.commission` 미노출 | 주문 내역 테이블에 컬럼 추가 |
| 모의·실거래 수수료율이 동일 | 하드코딩된 단일 설정값 | `real_commission_rate` 필드 분리 |

---

## 2. Logic Flow — 데이터가 어떻게 흘러가는가?

### Feature 1: 예수금 30초 자동 갱신

```
브라우저 → settings.html 렌더링
  └─ HTMX: hx-trigger="load, every 30s"
       ├─ (최초) GET /partials/account-info  ← 즉시 1회
       └─ (30초마다) GET /partials/account-info  ← 반복 폴링
            └─ partial_account_info()
                 ├─ ConnectionService.get_paper_balance()  ← DB 조회
                 └─ ConnectionService.get_real_balance()   ← KIS API 또는 에러 반환
```

HTMX `every 30s`는 JavaScript `setInterval()`과 동일한 효과를 선언적으로 표현합니다.
`load` 이벤트와 함께 콤마로 연결하면 "페이지 진입 즉시 + 이후 30초마다" 가 됩니다.

### Feature 2: 주문 내역 수수료 컬럼

```
GET /partials/orders (or /orders)
  └─ OrderService.get_orders(limit=50)          ← Order 목록
  └─ OrderService.get_trades_by_order_ids(ids)  ← {order_id: commission} 맵
       └─ SELECT * FROM trade WHERE order_id IN (...)
            └─ {t.order_id: t.commission for t in rows}  ← dict comprehension
  └─ TemplateResponse("order_table.html", {commission_map: ...})
       └─ {{ commission_map.get(o.id, 0) }}      ← Jinja2에서 dict 조회
```

### Feature 3: 실거래 수수료율 분기

```
주문 접수 (create_order)
  ├─ req.trading_mode == "REAL"?
  │    ├─ YES → commission_rate = app_settings.real_commission_rate  (0.15%)
  │    └─ NO  → commission_rate = account.commission_rate            (0.05%)
  └─ commission = total_amount * commission_rate
       └─ Trade(commission=commission) 저장
```

설정 우선순위:
```
.env: REAL_COMMISSION_RATE=0.002  (최우선, 있을 때만)
  → 없으면 config.py 기본값: 0.0015
```

---

## 3. Pythonic Point — 파이썬 특유의 기법

### Dict Comprehension으로 N+1 쿼리 방지

```python
# ❌ N+1 방식 (루프마다 DB 조회)
for order in orders:
    trade = await session.get(Trade, order_id=order.id)
    commission = trade.commission

# ✅ 한 번에 조회 후 dict으로 매핑
stmt = select(Trade).where(Trade.order_id.in_(order_ids))
result = await session.execute(stmt)
return {t.order_id: t.commission for t in result.scalars().all()}
```

`{key: value for item in iterable}` 형태가 파이썬 dict comprehension입니다.
Java의 `stream().collect(Collectors.toMap(...))` 과 동일한 역할입니다.

### `.get(key, default)` — 안전한 dict 접근

```python
# ❌ KeyError 위험
commission_map[o.id]

# ✅ 기본값 반환
commission_map.get(o.id, 0)
```

`order.id`에 해당하는 Trade가 없으면(CANCELLED, REJECTED 주문) `0`을 반환합니다.

### Pydantic BaseSettings — 환경변수 자동 바인딩

```python
class Settings(BaseSettings):
    real_commission_rate: float = 0.0015
```

이 한 줄로:
1. `.env`에 `REAL_COMMISSION_RATE=0.002` 가 있으면 자동으로 `0.002` 로 오버라이드
2. 없으면 기본값 `0.0015` 사용
3. 타입 변환(`str` → `float`)도 자동

---

## 4. Java vs Python 비교

### HTMX 폴링 vs WebSocket / SSE (Java/Spring)

```java
// Java - SSE 방식 (복잡)
@GetMapping(value = "/account-stream", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
public Flux<ServerSentEvent<String>> streamBalance() {
    return Flux.interval(Duration.ofSeconds(30))
        .map(seq -> ServerSentEvent.builder(fetchBalance()).build());
}
// + 프론트엔드에서 EventSource 연결 필요
```

```html
<!-- HTMX - 선언적, 3단어로 동일한 효과 -->
hx-trigger="load, every 30s"
```

HTMX는 서버사이드 렌더링 + 주기적 HTTP 폴링으로 실시간성을 모사합니다.
WebSocket/SSE보다 단순하지만, 1초 미만 갱신이 필요한 경우엔 부적합합니다.

### 수수료 맵핑: Stream Collector vs Dict Comprehension

```java
// Java - Stream API
Map<Integer, Double> commissionMap = trades.stream()
    .collect(Collectors.toMap(Trade::getOrderId, Trade::getCommission));
```

```python
# Python - Dict Comprehension
commission_map = {t.order_id: t.commission for t in trades}
```

파이썬이 더 간결하지만 의미는 동일합니다.

### 설정값 분리: @Value vs BaseSettings

```java
// Java - @Value 또는 @ConfigurationProperties
@Value("${kis.real-commission-rate:0.0015}")
private double realCommissionRate;
```

```python
# Python - Pydantic BaseSettings (타입 안전, 자동 파싱)
real_commission_rate: float = 0.0015  # env: REAL_COMMISSION_RATE
```

---

## 5. Key Concept — 놓치기 쉬운 포인트

### `IN` 쿼리와 빈 리스트 처리

```python
async def get_trades_by_order_ids(self, order_ids: list[int]) -> dict[int, float]:
    if not order_ids:      # ← 이게 없으면 "WHERE order_id IN ()" → SQL 오류!
        return {}
    stmt = select(Trade).where(Trade.order_id.in_(order_ids))
```

빈 리스트를 `IN ()` 절에 넣으면 DB 종류에 따라 문법 오류가 발생합니다.
SQLAlchemy도 동일하므로 방어 코드가 필수입니다.

### 사전 검증 블록의 Dead Code 이해

```python
if req.side.value == "BUY" and is_paper:   # is_paper == True 일 때만 진입
    commission_rate = (
        app_settings.real_commission_rate
        if req.trading_mode.value == "REAL"  # ← 이 분기는 is_paper=True와 모순
        else account.commission_rate         # ← 항상 여기로 실행됨
    )
```

`is_paper`가 True이면 `trading_mode`는 절대 "REAL"이 될 수 없습니다.
즉 사전 검증에서 `real_commission_rate` 분기는 현재 실행되지 않습니다.
실제 수수료 분리가 적용되는 곳은 체결 후 `Trade` 저장 시점(line 107)입니다.

### Jinja2 템플릿에서 dict 접근

```jinja
{# ❌ 존재하지 않는 key면 UndefinedError #}
{{ commission_map[o.id] }}

{# ✅ Python dict.get()과 동일하게 작동 #}
{{ commission_map.get(o.id, 0) }}
```

Jinja2는 Python dict의 `.get()` 메서드를 그대로 지원합니다.

### `{:.3%}` 포맷 — 소수점 3자리 백분율

```python
>>> "{:.3%}".format(0.0015)
'0.150%'

>>> "{:.2%}".format(0.0015)  # 기존
'0.15%'
```

수수료처럼 소수점 아래 자릿수가 중요한 값은 `.3%`가 더 정확합니다.
