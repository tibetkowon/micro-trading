# Phase 16–22 학습 문서: 자금관리 · 주문안정성 · 수익분석 · UI개선

## Overview

이번 세션(Phase 16-22)은 기존 트레이딩 기능의 **성숙도 향상**이 핵심이었습니다.
단순 동작에서 → AI 연동 가능한 안정적 시스템으로 업그레이드했습니다.

---

## Phase 16: 자금 관리 고도화 — Orderable Amount

### Logic Flow

```
클라이언트 요청: GET /api/portfolio/orderable
      ↓
PortfolioService.get_orderable_info(trading_mode)
      ↓
모드 분기:
  PAPER → Account.paper_balance_krw + account.commission_rate(0.05%)
  REAL  → KIS API 잔고 조회 + 0.3% 안전 마진
      ↓
orderable_krw = cash_krw / (1 + commission_rate)
      ↓
OrderableResponse 반환
```

### Pythonic Point

```python
# 안전 마진을 적용한 주문가능 금액 계산
# cash = 주문가능 금액 × (1 + 수수료율)  →  역산
orderable = cash / (1 + commission_rate)

# 예: 100만원, 수수료 0.3%
# 수수료: 997,009원 × 0.003 = 2,991원
# 합계: 997,009 + 2,991 = 1,000,000원 ✓
```

### Java vs Python

| 항목 | Java (Spring) | Python (FastAPI) |
|:---|:---|:---|
| 응답 모델 | `@ResponseBody DTO` | `Pydantic BaseModel` |
| 필드 문서화 | `@Schema(description="...")` (Swagger) | `Field(..., description="...")` |
| DI | `@Autowired Service` | `Depends(get_session)` |

### Key Concept

**왜 0.3%인가?**
실제 KIS 수수료(0.15%)보다 높은 0.3%를 사용하는 이유: AI가 경계값 주문 시 잔고 부족이 나지 않도록 **보수적 안전 마진**을 적용합니다. "절대 주문 실패가 나지 않을 금액"을 보장하는 방어적 설계입니다.

---

## Phase 17: 주문 안정성 강화 — 재시도 메커니즘

### Logic Flow

```
KISClient.post(path, tr_id, body, _retry=2)
      ↓
HTTP POST 전송
      ↓
응답 코드 분기:
  401 → force_refresh_token() → 즉시 1회 재시도 (토큰 만료)
  5xx → asyncio.sleep(1) → self.post(..., _retry - 1) (서버 일시 오류)
  4xx(401 제외) → 즉시 raise_for_status() (잘못된 요청, 재시도 불필요)
  200 → 정상 반환
```

### Pythonic Point — 재귀 재시도 패턴

```python
async def post(self, path, tr_id, body, _retry=2) -> dict:
    # ...
    if resp.status_code >= 500 and _retry > 0:
        await asyncio.sleep(1)
        return await self.post(path, tr_id, body, _retry=_retry - 1)
    # ...
```

- `_retry`를 파라미터로 내려가며 감소시키는 **꼬리 재귀** 패턴
- `asyncio.sleep(1)`: 비동기 대기 → 다른 코루틴이 CPU를 사용할 수 있음 (스레드 블로킹 없음)
- 언더스코어 접두사(`_retry`)로 "내부 전용 파라미터"임을 명시

### Java vs Python

```java
// Java: 보통 RetryTemplate (Spring Retry) 사용
@Retryable(maxAttempts = 3, backoff = @Backoff(delay = 1000))
public OrderResult placeOrder(OrderRequest req) { ... }
```

```python
# Python: 재귀 + asyncio.sleep으로 직접 구현
# 라이브러리 없이 가볍게, 메모리 최소화
if resp.status_code >= 500 and _retry > 0:
    await asyncio.sleep(1)
    return await self.post(..., _retry=_retry - 1)
```

Spring Retry는 강력하지만 라이브러리 의존성이 생깁니다. 이 프로젝트는 1GB RAM 제약으로 의존성을 최소화해야 하므로 직접 구현했습니다.

### Key Concept — 401 vs 5xx 분리 처리

| 상황 | 원인 | 처리 |
|:---|:---|:---|
| 401 | 토큰 만료 | 즉시 갱신 후 재시도 (딜레이 없음) |
| 5xx | KIS 서버 일시 오류 | 1초 대기 후 재시도 (최대 2회) |
| 4xx (401 제외) | 잘못된 요청 | 재시도 없음, 즉시 실패 |

주문은 **중복 실행 위험**이 있어서, 성공 여부가 불분명한 경우(5xx)에만 재시도합니다.
잔고 부족(400)이나 권한 오류 같은 비즈니스 에러는 재시도해도 무의미합니다.

---

## Phase 18: 지정가 주문 UI — 현재가 자동 입력

### Logic Flow

```
사용자가 주문유형 드롭다운에서 "지정가" 선택
      ↓
toggleLimitPrice(select) 호출 (trading.js)
      ↓
select.value === 'LIMIT' ?
  → 입력란 활성화
  → window._currentPrice가 있으면 자동 입력
  → updateEstimate() 호출 (예상금액 재계산)
: → 입력란 비활성화, 값 초기화
```

### Pythonic Point — window._currentPrice 전역 변수

```javascript
// stock_detail.html 렌더 시 서버에서 현재가를 JS 변수로 주입
window._currentPrice = {{ price_info.price if price_info else 0 }};
```

Jinja2 템플릿이 서버 데이터를 클라이언트 JS 변수로 "브리징"합니다.
이후 HTMX로 가격이 갱신될 때마다 `_currentPrice`도 업데이트됩니다.

---

## Phase 19: 수익률 분석 시스템

### Logic Flow

```
GET /api/portfolio/pnl-analysis
      ↓
PortfolioService.get_pnl_analysis(trading_mode)
      ↓
[병렬 조회]
  1. Trade 테이블에서 종목별 realized_pnl 집계 (GROUP BY symbol, market)
  2. PortfolioSnapshot에서 일별 수익률 조회
  3. StockMaster에서 종목명 일괄 조회
      ↓
PnlAnalysisResponse 반환
```

### Pythonic Point — SQLAlchemy 집계 쿼리

```python
from sqlalchemy import func as sql_func

stmt = (
    select(
        Trade.symbol,
        Trade.market,
        sql_func.sum(Trade.realized_pnl).label("realized_pnl"),
        sql_func.count(Trade.id).label("trade_count"),
    )
    .where(Trade.trading_mode == trading_mode)
    .group_by(Trade.symbol, Trade.market)
    .order_by(sql_func.sum(Trade.realized_pnl).desc())
)
```

- `sql_func.sum()`: SQL `SUM()` 함수를 Python으로 표현
- `.label("...")`: 집계 결과에 별칭(alias) 부여 → `row.realized_pnl`로 접근 가능
- `order_by(sql_func.sum(...).desc())`: 집계 컬럼으로 정렬

### Java vs Python

```java
// Spring Data JPA: @Query + JPQL
@Query("SELECT t.symbol, SUM(t.realizedPnl) FROM Trade t GROUP BY t.symbol ORDER BY SUM(t.realizedPnl) DESC")
List<Object[]> findSymbolPnl();
```

```python
# SQLAlchemy: 파이썬 코드로 직접 쿼리 구성
stmt = select(Trade.symbol, sql_func.sum(Trade.realized_pnl)).group_by(Trade.symbol)
rows = (await session.execute(stmt)).all()
```

---

## Phase 20: 최근거래 개선 — Jinja2 필터 패턴

### Logic Flow

```
routes.py 모듈 로드 시:
  templates.env.filters["to_kst"] = _to_kst
  templates.env.filters["source_label"] = _source_label
      ↓
partial_orders_compact 호출:
  1. OrderService.get_orders() → Order 목록
  2. StockMasterService.get_names_bulk() → name_map 딕셔너리
  3. TemplateResponse("orders_compact.html", {orders, name_map})
      ↓
템플릿에서:
  {{ o.created_at | to_kst }}        → "02-19 14:30" (KST)
  {{ o.source | source_label }}      → "직접주문"
  {{ name_map.get((symbol, market)) }} → "삼성전자"
```

### Pythonic Point — Jinja2 커스텀 필터

```python
def _to_kst(dt: datetime | None) -> str:
    """UTC datetime을 KST(UTC+9) 문자열로 변환."""
    if dt is None:
        return ""
    kst = timezone(timedelta(hours=9))
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)  # naive → aware
    return dt.astimezone(kst).strftime("%m-%d %H:%M")

templates.env.filters["to_kst"] = _to_kst  # 전역 등록
```

- `dt.tzinfo is None` 체크: SQLite에서 읽은 datetime은 timezone 정보가 없는 "naive datetime"
- `replace(tzinfo=utc)`: naive → UTC aware로 변환 후 KST로 변환
- 한 번 등록하면 모든 템플릿에서 `| to_kst` 파이프 연산자로 재사용

### Java vs Python 비교

```java
// Java: DateTimeFormatter + ZoneId
LocalDateTime utc = order.getCreatedAt();
ZonedDateTime kst = utc.atZone(ZoneId.of("UTC")).withZoneSameInstant(ZoneId.of("Asia/Seoul"));
String display = kst.format(DateTimeFormatter.ofPattern("MM-dd HH:mm"));
```

Python의 Jinja2 필터는 이 변환 로직을 **템플릿 전역**에서 재사용할 수 있는 유틸리티로 등록합니다. Java의 Custom Converter와 유사한 개념입니다.

### Key Concept — name_map 딕셔너리 패턴

```python
# 종목명 일괄 조회 (N+1 쿼리 방지)
symbols = list({(o.symbol, o.market) for o in orders})  # 중복 제거
name_map = await stock_svc.get_names_bulk(symbols)       # 1번 쿼리
# 결과: {("005930", "KR"): "삼성전자", ...}

# 템플릿에서 조회
{{ name_map.get((o.symbol, o.market), o.symbol) }}
```

개별 주문마다 DB를 조회하면 **N+1 쿼리 문제** 발생. 한 번에 일괄 조회 후 딕셔너리로 O(1) 접근.

---

## Phase 21: 메뉴 통일

### 결정 사항

- 제거: `/orders` (메인 뷰에 통합됨), `/strategies` (미사용)
- 통일: `trading_base.html`과 `base.html` 모두 **트레이딩 | 포트폴리오 | 설정**

Python 프로젝트에서는 두 개의 base 템플릿이 있을 때 **메뉴를 각각 관리**해야 하는 문제가 있습니다.
이를 줄이려면 하나의 base로 통합하거나 Jinja2 macro를 사용하는 방법이 있습니다.

---

## Phase 22: 계좌정보 주문가능금액

### Logic Flow

```
GET /partials/account-info
      ↓
ConnectionService.get_paper_balance() → {cash_krw: ...}
ConnectionService.get_real_balance()  → {cash_krw: ..., total_value_krw: ...}
      ↓
paper_orderable = cash / (1 + paper_commission_rate)
real_orderable  = cash / (1 + 0.003)  # 0.3% 안전 마진
      ↓
account_info.html 렌더링:
  예수금 / 총평가금액 / 주문가능금액 표시
```

### Key Concept

주문가능금액을 계산할 때 단순히 예수금을 보여주면 안 되는 이유:
> 100만원을 가지고 100만원어치 주문하면 → 수수료 3,000원이 없어서 주문 실패

따라서 `orderable = cash / (1 + fee_rate)` 로 수수료를 **미리 제외**한 금액을 표시합니다.
