# Phase 5: 웹 거래화면 설계 및 구현 - 학습 문서

## 1. Overview: 왜 거래화면 개선이 필요했는가?

Phase 4에서 대시보드를 완성했지만, 실제로 거래를 실행하는 화면은 불편했다:

- **검색 결과에서 관심종목 추가 불가**: 검색 결과를 클릭해서 주식 상세 화면으로 이동한 후, 별도로 관심종목 추가 버튼을 눌러야 했음
- **주문 가능 잔고 미표시**: 주문 폼에 현재 잔고가 보이지 않아 얼마나 살 수 있는지 알 수 없었음
- **예상 주문금액 계산 없음**: 수량을 입력해도 총 비용(수수료 포함)이 자동 계산되지 않았음
- **주문 후 자동 갱신 없음**: 주문 완료 후 보유종목/잔고를 보려면 수동으로 새로고침 필요

Phase 5는 이 불편함을 모두 해결하여 실제 증권사 HTS와 유사한 사용 흐름을 구현했다.

---

## 2. Logic Flow: 거래화면 주요 흐름

### 2-1. 검색 → 관심종목 추가 흐름

```
사용자: 검색창에 종목명 입력
    │
    ▼
GET /partials/watchlist/search?q=삼성
    │
    ▼
watchlist_search_results.html 렌더링
    │  (각 항목에 "+" 버튼 포함)
    │
사용자: "+" 버튼 클릭
    │
    ▼
hx-post="/stocks/memo"
hx-vals='{"symbol":"005930","market":"KR","name":"삼성전자"}'
hx-swap="none"   ← DOM 변경 없음 (서버에만 저장)
    │
    ▼
버튼 텍스트: "+" → "✓" (즉각 피드백)
버튼 비활성화: disabled=true
```

### 2-2. 주문 폼 흐름

```
사용자: 종목 클릭 → stock_detail.html 로드
    │
    ▼
[자동] hx-get="/partials/order-balance?market=KR"
    │   hx-trigger="load, refreshSidebar from:body"
    │
    ▼
order_balance.html → "주문 가능: 10,230,000 원"
    │
사용자: 수량 입력 (oninput="updateEstimate()")
    │
    ▼
JavaScript updateEstimate()
    │  price = window._currentPrice (10초마다 갱신됨)
    │  qty = 수량 입력값
    │  total = price × qty
    │  commission = total × 0.0005
    │  → "예상 주문금액: 500,250 원 (수수료: 250원)"
    │
사용자: "매수 주문" 클릭
    │
    ▼
hx-post="/orders/submit"
hx-target="#order-result"
hx-on::after-request="onOrderComplete(event)"
    │
    ▼
서버: OrderService.create_order()
    ├── 성공 → "005930 매수 10주 주문이 접수되었습니다."
    │           + HX-Trigger: refreshSidebar 헤더 전송
    └── 실패 → "주문 실패: 잔고 부족: 필요 500,250, 보유 100,000"
    │
    ▼
HX-Trigger: refreshSidebar
    │
    ▼
hx-trigger="refreshSidebar from:body" 감지
    ├── #order-balance 갱신 (잔고)
    └── #stock-position 갱신 (보유종목)
```

### 2-3. 실시간 가격 갱신과 예상금액 연동

```
10초마다:
GET /partials/stock-price/{symbol}
    │
    ▼
htmx:afterSwap 이벤트 발생
    │
    ▼
JavaScript event listener:
    priceEl = target.querySelector('.price-hero-value')
    window._currentPrice = parseFloat(priceEl.textContent)
    updateEstimate()  ← 새 가격으로 예상금액 자동 재계산
```

---

## 3. Pythonic Point: 핵심 기법

### 3-1. hx-vals: JSON으로 추가 데이터 전송

```html
<!-- watchlist_search_results.html -->
<button class="wl-add-btn"
        hx-post="/stocks/memo"
        hx-vals='{"symbol":"{{ item.symbol }}","market":"{{ item.market }}","name":"{{ item.name }}"}'
        hx-swap="none"
        hx-on::after-request="this.textContent='✓'; this.disabled=true;"
        title="관심종목에 추가">+</button>
```

- `hx-vals`: 폼 없이 데이터를 서버로 전송하는 방법 (JSON 형식)
- 버튼이 폼 안에 없어도 POST 요청과 함께 데이터 전달 가능
- `hx-swap="none"`: 응답을 DOM에 반영하지 않음 → 버튼 상태만 JS로 변경

### 3-2. hx-on::after-request: 인라인 이벤트 핸들러

```html
hx-on::after-request="this.textContent='✓'; this.disabled=true;"
```

- `hx-on::{이벤트명}`: HTMX 이벤트에 대한 인라인 핸들러
- `after-request`: 서버 응답 수신 후 실행 (성공/실패 무관)
- `this`: 해당 버튼 요소를 가리킴
- 별도 JavaScript 함수 없이 간단한 UI 반응 구현

### 3-3. HX-Trigger 응답 헤더: 서버에서 클라이언트 이벤트 발생

```python
# routes.py: 주문 완료 후 사이드바 갱신 트리거
@web_router.post("/orders/submit", response_class=HTMLResponse)
async def submit_order(request: Request, ...):
    # ... 주문 처리 ...
    hx_target = request.headers.get("hx-target", "")
    if hx_target == "order-result":
        from fastapi.responses import HTMLResponse as HR
        resp = HR(result_html)
        resp.headers["HX-Trigger"] = "refreshSidebar"  # ← 이벤트 발생
        return resp
```

```html
<!-- stock_detail.html: 이벤트 수신하면 갱신 -->
<div id="order-balance"
     hx-get="/partials/order-balance?market={{ market }}"
     hx-trigger="load, refreshSidebar from:body"
     hx-swap="innerHTML">
</div>

<div id="stock-position"
     hx-get="/partials/stock-position/{{ symbol }}?market={{ market }}"
     hx-trigger="load, every 15s, refreshSidebar from:body"
     hx-swap="innerHTML">
</div>
```

- 서버가 `HX-Trigger: refreshSidebar` 헤더를 보내면
- HTMX가 `body`에서 `refreshSidebar` 커스텀 이벤트를 발생시킴
- `refreshSidebar from:body`를 감지하는 모든 요소가 자동으로 갱신됨

### 3-4. JavaScript updateEstimate 함수

```javascript
// stock_detail.html <script> 섹션
window._currentPrice = {{ price_info.price if price_info and price_info.price else 0 }};
window._currentMarket = '{{ market }}';
updateEstimate();

// 예상금액 계산 함수 (전역)
function updateEstimate() {
    const qty = parseInt(document.getElementById('order-quantity')?.value) || 0;
    const limitPrice = parseFloat(document.getElementById('inline-limit-price')?.value) || 0;
    const orderType = document.getElementById('inline-order-type')?.value;

    const price = (orderType === 'LIMIT' && limitPrice > 0)
        ? limitPrice
        : window._currentPrice;

    if (!price || !qty) {
        document.getElementById('order-estimate').textContent = '';
        return;
    }

    const total = price * qty;
    const commission = total * 0.0005;
    const isUS = window._currentMarket === 'US';

    document.getElementById('order-estimate').textContent = isUS
        ? `예상: $${(total + commission).toFixed(2)} (수수료: $${commission.toFixed(2)})`
        : `예상: ${Math.round(total + commission).toLocaleString()}원 (수수료: ${Math.round(commission).toLocaleString()}원)`;
}
```

- `window._currentPrice`: 전역 변수로 현재가 저장 → 10초 갱신 시 업데이트
- 지정가 주문이면 입력값 사용, 시장가면 현재가 사용
- KR/US 시장에 따라 원화/달러 포맷 분기

### 3-5. htmx:afterSwap 이벤트로 현재가 동기화

```javascript
// stock_detail.html
document.addEventListener('htmx:afterSwap', function(evt) {
    if (evt.detail.target && evt.detail.target.id === 'price-hero') {
        const priceEl = evt.detail.target.querySelector('.price-hero-value');
        if (priceEl) {
            const text = priceEl.textContent.replace(/[^0-9.]/g, '');  // 숫자/점만 추출
            const price = parseFloat(text);
            if (price > 0) {
                window._currentPrice = price;
                updateEstimate();  // 새 가격으로 예상금액 재계산
            }
        }
    }
});
```

- `htmx:afterSwap`: HTMX가 DOM을 교체한 후 발생하는 이벤트
- 10초마다 price-hero가 갱신될 때마다 현재가를 추출해서 updateEstimate 재호출
- 사용자가 수량을 그대로 두어도 가격 변동에 따라 예상금액이 자동 업데이트

### 3-6. 주문 가능 잔고 라우트

```python
@web_router.get("/partials/order-balance", response_class=HTMLResponse)
async def partial_order_balance(
    request: Request,
    market: str = "KR",
    session: AsyncSession = Depends(get_session),
):
    """주문 폼에 표시할 주문 가능 잔고."""
    from app.models.account import Account
    from sqlalchemy import select
    result = await session.execute(select(Account).limit(1))
    account = result.scalar_one_or_none()
    balance = 0.0
    if account:
        balance = account.paper_balance_usd if market == "US" else account.paper_balance_krw
    return templates.TemplateResponse("partials/order_balance.html", {
        "request": request,
        "balance": balance,
        "market": market,
    })
```

```html
<!-- partials/order_balance.html -->
<div class="order-balance-info">
    <span class="order-balance-label">주문 가능</span>
    <span class="order-balance-value">
        {% if market == 'US' %}
        ${{ "{:,.2f}".format(balance) }}
        {% else %}
        {{ "{:,.0f}".format(balance) }} <small>원</small>
        {% endif %}
    </span>
</div>
```

---

## 4. Java vs Python: 비교

### 4-1. 폼 데이터 처리 비교

**Java (Spring MVC)**
```java
@PostMapping("/orders/submit")
public ResponseEntity<String> submitOrder(
    @RequestParam String symbol,
    @RequestParam String market,
    @RequestParam String side,
    @RequestParam int quantity,
    @RequestParam(required = false) Double price,
    HttpServletRequest request
) {
    OrderResult result = orderService.createOrder(symbol, market, side, quantity, price);
    HttpHeaders headers = new HttpHeaders();
    headers.set("HX-Trigger", "refreshSidebar");
    return ResponseEntity.ok().headers(headers).body(resultHtml);
}
```

**Python (FastAPI)**
```python
@web_router.post("/orders/submit", response_class=HTMLResponse)
async def submit_order(
    request: Request,
    symbol: str = Form(...),
    market: str = Form("KR"),
    side: str = Form(...),
    quantity: int = Form(...),
    price: float | None = Form(None),
    session: AsyncSession = Depends(get_session),
):
    # ... 주문 처리 ...
    resp = HTMLResponse(result_html)
    resp.headers["HX-Trigger"] = "refreshSidebar"
    return resp
```

| 항목 | Java (Spring) | Python (FastAPI) |
|:---|:---|:---|
| 폼 바인딩 | `@RequestParam`, `@ModelAttribute` | `Form(...)` 파라미터 |
| 선택적 파라미터 | `required = false` | `= Form(None)` |
| 응답 헤더 | `HttpHeaders` + `ResponseEntity` | `response.headers["..."] = ...` |
| 의존성 주입 | `@Autowired`, `@Bean` | `Depends(get_session)` |

### 4-2. 실시간 UI 업데이트 패턴

**Java (Thymeleaf + Alpine.js 조합)**
```javascript
// 주문 완료 후 수동으로 이벤트 발생
Alpine.store('trading').refreshBalance()
```

**Python (HTMX + 응답 헤더)**
```python
# 서버 응답 헤더로 클라이언트 이벤트 발생
resp.headers["HX-Trigger"] = "refreshSidebar"
```

```html
<!-- 수신부 -->
<div hx-trigger="refreshSidebar from:body" hx-get="/partials/order-balance">
```

서버가 `HX-Trigger` 헤더를 보내면 HTMX가 자동으로 클라이언트 이벤트를 발생시킨다. 별도 JavaScript 없이 서버 → 클라이언트 이벤트 전파가 가능하다.

### 4-3. Enum 기반 폼 데이터 변환

**Java (Spring + @Enum 자동변환)**
```java
public Order createOrder(@RequestParam OrderSide side) {
    // Spring이 "BUY" 문자열을 OrderSide.BUY로 자동 변환
}
```

**Python (수동 변환)**
```python
async def submit_order(
    side: str = Form(...),  # "BUY" 문자열로 받음
    ...
):
    req = OrderCreate(
        side=OrderSide(side),  # 명시적으로 Enum 변환
        market=Market(market),
        order_type=OrderType(order_type),
        trading_mode=TradingMode(settings.trading_mode.value),
    )
```

FastAPI는 `Form(...)` 파라미터를 str로 받고, Pydantic 스키마에서 Enum 변환이 이루어진다.

---

## 5. Key Concept: 초보자가 놓치기 쉬운 포인트

### 5-1. hx-target vs hx-swap의 조합

```html
<form hx-post="/orders/submit"
      hx-target="#order-result"
      hx-swap="innerHTML">
```

| 속성 | 역할 | 예시 |
|:---|:---|:---|
| `hx-target` | 응답을 삽입할 대상 요소 | `#order-result` (CSS 선택자) |
| `hx-swap` | 삽입 방식 | `innerHTML` (내부 교체) |

폼 자체가 아닌 다른 요소에 결과를 표시할 때 `hx-target`을 사용한다. 주문 폼은 그대로 유지하면서 `#order-result` div에만 결과가 나타난다.

### 5-2. `from:body` 이벤트 버블링

```html
hx-trigger="refreshSidebar from:body"
```

HTMX 이벤트는 기본적으로 요소 자신에서 발생해야 감지된다. `from:body`를 추가하면 `body` 요소에서 발생한 이벤트도 감지한다. `HX-Trigger` 헤더로 발생하는 이벤트는 `body`에서 발생하므로, 페이지 어디서든 감지하려면 `from:body`가 필요하다.

### 5-3. Form(...) vs Query 파라미터

FastAPI에서 데이터를 받는 방식:

```python
# URL 쿼리 파라미터: GET /order-balance?market=KR
async def partial_order_balance(market: str = "KR"):
    ...

# 폼 데이터: POST /orders/submit (Content-Type: application/x-www-form-urlencoded)
async def submit_order(symbol: str = Form(...), quantity: int = Form(...)):
    ...

# JSON 바디: POST /api/orders (Content-Type: application/json)
async def create_order(req: OrderCreate):  # Pydantic 모델
    ...
```

HTML form은 `application/x-www-form-urlencoded`로 전송하므로 FastAPI에서 `Form(...)`을 써야 한다.

### 5-4. window._currentPrice 전역 변수 패턴

```javascript
// 서버 렌더링 시 현재가 초기화
window._currentPrice = {{ price_info.price if price_info and price_info.price else 0 }};
```

Jinja2가 서버에서 렌더링할 때 현재가를 JavaScript 전역 변수로 주입한다. 이후 10초마다 DOM을 교체할 때 JS로 추출해서 갱신한다. 이 패턴은 서버 렌더링과 클라이언트 동적 업데이트를 연결하는 가교 역할을 한다.

### 5-5. 주문 결과 HTML 직접 생성

```python
result_html = (
    f'<div class="order-result success">'
    f'{symbol} {side_label} {quantity}주 주문이 접수되었습니다.'
    f'</div>'
)
```

별도 템플릿 파일 없이 Python f-string으로 간단한 결과 HTML을 생성했다. 재사용이 필요한 복잡한 HTML은 파셜 템플릿으로 분리하고, 단순한 성공/실패 메시지는 인라인으로 처리하는 것이 실용적이다.

### 5-6. 검색 결과에서 바로 관심종목 추가의 UX 의미

기존: 검색 → 상세 화면 이동 → 별도 버튼 클릭 (3단계)
개선: 검색 → "+" 버튼 클릭 (2단계)

```html
<button class="wl-add-btn"
        hx-post="/stocks/memo"
        hx-vals='{"symbol":"{{ item.symbol }}","market":"{{ item.market }}","name":"{{ item.name }}"}'
        hx-swap="none"
        hx-on::after-request="this.textContent='✓'; this.disabled=true;">+</button>
```

`hx-swap="none"`이 핵심이다. 서버 응답을 DOM에 반영하지 않으므로 검색 결과 목록이 그대로 유지된다. 사용자는 여러 종목을 연속으로 추가할 수 있다.
