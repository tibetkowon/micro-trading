# Phase 4: 웹 대시보드 설계 및 구현 - 학습 문서

## 1. Overview: 왜 대시보드를 개선해야 했는가?

Phase 3까지 완성된 거래 로직을 사용자가 한눈에 파악할 수 있는 UI가 필요했다. 기존 대시보드는:

- **요약 카드 부족**: 총 평가금액과 수익률만 있고, 현금/투자금액/손익 세분화가 없음
- **관심종목 시세 부재**: 대시보드에서 관심종목의 실시간 가격을 볼 수 없음
- **사이드바 정보 부족**: 거래 화면의 우측 포트폴리오 사이드바에 현금/투자금액이 없음
- **새로고침 필요**: 시세와 포트폴리오 정보를 보려면 페이지를 직접 새로고침해야 함

Phase 4에서는 6개 카드로 확장된 요약, 관심종목 실시간 시세 그리드, HTMX 자동 갱신을 구현했다.

---

## 2. Logic Flow: 화면 구성과 데이터 흐름

### 2-1. 대시보드 전체 구조

```
trading.html (3-panel layout)
    │
    ├── 좌측 패널: watchlist.html
    │       └── 관심종목 목록 (탭: 전체/KR/US)
    │
    ├── 중앙 패널: dashboard_default.html
    │       ├── summary_cards.html (6개 카드)
    │       │       hx-trigger="load, every 30s"
    │       ├── 포트폴리오 추이 차트 (Chart.js)
    │       ├── dashboard_watchlist_prices.html (시세 카드 그리드)
    │       │       hx-trigger="load, every 30s"
    │       ├── position_table.html (보유종목)
    │       │       hx-trigger="load, every 30s"
    │       └── order_table.html (최근 주문)
    │               hx-trigger="load, every 30s"
    │
    └── 우측 패널: portfolio_sidebar.html
            └── portfolio_compact.html (요약 + 현금 + 투자금액)
                    hx-trigger="load, refreshSidebar from:body"
```

### 2-2. summary_cards.html 6개 카드 구성

```
PortfolioService.get_summary() → PortfolioSummary 객체
    │
    ├── total_value    → 총 평가금액 (현금 + 보유주식 평가금)
    ├── cash_krw       → 보유 현금 (KRW)
    ├── cash_usd       → 보유 현금 (USD, 0이면 미표시)
    ├── total_invested → 투자 금액 (평균단가 × 수량 합산)
    ├── return_pct     → 수익률 (%)
    ├── unrealized_pnl → 평가 손익 (미실현)
    └── realized_pnl   → 실현 손익
```

### 2-3. HTMX 자동 갱신 흐름

```
브라우저 로드
    │
    ▼
hx-trigger="load, every 30s" 감지
    │
    ▼
GET /partials/dashboard/watchlist-prices
    │
    ▼
routes.py: partial_dashboard_watchlist_prices()
    ├── WatchlistService.list_all() → 관심종목 목록
    └── MarketService.get_price() × N → 각 종목 현재가
    │
    ▼
dashboard_watchlist_prices.html 렌더링
    │
    ▼
hx-swap="innerHTML" → 기존 내용 교체 (페이지 새로고침 없음)
```

---

## 3. Pythonic Point: 파이썬/Jinja2/HTMX 핵심 기법

### 3-1. Jinja2 조건부 CSS 클래스

```html
<!-- summary_cards.html: 수익률이 양수면 green, 음수면 red -->
<p class="big-number {{ 'positive' if summary.return_pct >= 0 else 'negative' }}">
    {{ "{:+.2f}".format(summary.return_pct) }}%
</p>
```

- `{{ 'A' if condition else 'B' }}`: 파이썬의 삼항 연산자를 Jinja2 템플릿에서 사용
- `"{:+.2f}".format(value)`: `+` 플래그로 양수에도 `+` 부호 표시 → `+3.45%`, `-1.23%`

### 3-2. 숫자 포맷팅 패턴

```html
<!-- KRW: 천 단위 콤마, 소수점 없음 -->
{{ "{:,.0f}".format(summary.total_value) }}   → "10,230,000"

<!-- USD: 천 단위 콤마, 소수점 2자리 -->
{{ "{:,.2f}".format(summary.cash_usd) }}      → "9,850.75"

<!-- 손익: 부호 + 천 단위 콤마 -->
{{ "{:+,.0f}".format(summary.unrealized_pnl) }} → "+230,000" or "-15,000"
```

| 포맷 코드 | 의미 | 예시 |
|:---|:---|:---|
| `{:,}` | 천 단위 콤마 | `1234567` → `1,234,567` |
| `{:.0f}` | 소수점 0자리 (반올림) | `1234.7` → `1235` |
| `{:.2f}` | 소수점 2자리 | `9.5` → `9.50` |
| `{:+.2f}` | 부호 강제 표시 | `3.5` → `+3.50` |
| `{:+,.0f}` | 부호 + 콤마 + 정수 | `230000` → `+230,000` |

### 3-3. Jinja2 조건부 블록

```html
<!-- portfolio_compact.html: USD 잔고가 있을 때만 표시 -->
{% if summary.cash_usd > 0 %}
<p class="sub-number">{{ "{:,.2f}".format(summary.cash_usd) }} <small>USD</small></p>
{% endif %}
```

### 3-4. HTMX hx-trigger 패턴

```html
<!-- 페이지 로드 시 + 30초마다 자동 갱신 -->
<div hx-get="/partials/dashboard/watchlist-prices"
     hx-trigger="load, every 30s"
     hx-swap="innerHTML">
    <div class="wl-empty">시세 불러오는 중...</div>
</div>
```

- `load`: 요소가 DOM에 추가될 때 즉시 요청
- `every 30s`: 30초마다 반복 요청
- `hx-swap="innerHTML"`: 응답 HTML로 요소 내부 교체 (요소 자체는 유지)
- 초기 값(`시세 불러오는 중...`)은 첫 요청이 완료될 때까지 표시

### 3-5. 카드 그리드 템플릿 (dashboard_watchlist_prices.html)

```html
{% if items %}
<div class="dash-wl-grid">
    {% for item in items %}
    <div class="dash-wl-card"
         hx-get="/partials/stock-detail/{{ item.symbol }}?market={{ item.market }}&name={{ item.name | urlencode }}"
         hx-target="#panel-main"
         hx-swap="innerHTML"
         hx-push-url="/stock/{{ item.symbol }}?market={{ item.market }}"
         style="cursor:pointer;">
        <div class="dash-wl-card-bottom">
            {% if item.price and item.price > 0 %}
            <span class="dash-wl-price">
                {{ "{:,.0f}".format(item.price) if item.market == 'KR' else "{:,.2f}".format(item.price) }}
            </span>
            <span class="dash-wl-change {{ 'positive' if item.change_pct >= 0 else 'negative' }}">
                {{ "{:+.2f}".format(item.change_pct) }}%
            </span>
            {% else %}
            <span class="dash-wl-price" style="color:var(--text-muted);">-</span>
            {% endif %}
        </div>
    </div>
    {% endfor %}
</div>
{% else %}
<div class="wl-empty">관심종목을 추가하면 실시간 시세가 표시됩니다.</div>
{% endif %}
```

- `hx-push-url`: AJAX 요청이지만 브라우저 URL도 업데이트 → 뒤로가기 지원
- `| urlencode`: 종목명에 특수문자나 공백이 있을 때 URL-safe 문자로 변환
- `item.market == 'KR'` 조건으로 KRW/USD 포맷 분기

### 3-6. 새 라우트 추가: /partials/dashboard/watchlist-prices

```python
@web_router.get("/partials/dashboard/watchlist-prices", response_class=HTMLResponse)
async def partial_dashboard_watchlist_prices(
    request: Request,
    session: AsyncSession = Depends(get_session),
):
    """대시보드 관심종목 실시간 시세 카드 그리드."""
    market_svc = MarketService(session)
    memo_svc = WatchlistService(session)
    memos = await memo_svc.list_all()
    items = []
    for m in memos:
        try:
            p = await market_svc.get_price(m.symbol, m.market)
            items.append({
                "symbol": m.symbol, "market": m.market, "name": m.name,
                "price": p.price, "change": p.change, "change_pct": p.change_pct,
            })
        except Exception:
            items.append({
                "symbol": m.symbol, "market": m.market, "name": m.name,
                "price": 0, "change": 0, "change_pct": 0,
            })
    return templates.TemplateResponse("partials/dashboard_watchlist_prices.html", {
        "request": request,
        "items": items,
    })
```

---

## 4. Java vs Python: 비교

### 4-1. 템플릿 엔진 비교

**Java (Thymeleaf)**
```html
<!-- 조건부 CSS -->
<p th:class="${summary.returnPct >= 0 ? 'positive' : 'negative'}"
   th:text="${#numbers.formatDecimal(summary.returnPct, 1, 2) + '%'}">0%</p>

<!-- 반복 -->
<div th:each="item : ${items}">
    <span th:text="${item.name}">이름</span>
    <span th:text="${#numbers.formatDecimal(item.price, 0, 0)}">0</span>
</div>
```

**Python (Jinja2)**
```html
<!-- 조건부 CSS -->
<p class="{{ 'positive' if summary.return_pct >= 0 else 'negative' }}">
    {{ "{:+.2f}".format(summary.return_pct) }}%
</p>

<!-- 반복 -->
{% for item in items %}
<div>
    <span>{{ item.name }}</span>
    <span>{{ "{:,.0f}".format(item.price) }}</span>
</div>
{% endfor %}
```

| 항목 | Java (Thymeleaf) | Python (Jinja2) |
|:---|:---|:---|
| 반복 | `th:each` | `{% for ... in ... %}` |
| 조건 | `th:if`, 삼항 연산자 | `{% if %}`, 인라인 삼항 |
| 표현식 | SpEL (`${...}`) | Python 표현식 (`{{ }}`) |
| 숫자 포맷 | `#numbers.formatDecimal()` | 파이썬 format spec |
| Null 체크 | `th:if="${val != null}"` | `{% if val %}` |

### 4-2. 자동 갱신 방식 비교

**Java (WebSocket + Spring)**
```java
// Spring WebSocket 설정 필요
@Configuration
@EnableWebSocketMessageBroker
public class WebSocketConfig implements WebSocketMessageBrokerConfigurer {
    // 복잡한 설정...
}

// SimpMessagingTemplate로 서버 → 클라이언트 push
messagingTemplate.convertAndSend("/topic/prices", priceUpdate);
```

**Python (HTMX Polling)**
```html
<!-- 클라이언트에서 polling - 서버 설정 없음 -->
<div hx-get="/partials/dashboard/watchlist-prices"
     hx-trigger="every 30s"
     hx-swap="innerHTML">
</div>
```

| 항목 | Java (WebSocket) | Python (HTMX Polling) |
|:---|:---|:---|
| 서버 설정 | 복잡 (STOMP, SockJS) | 없음 (일반 HTTP 엔드포인트) |
| 실시간성 | Push (즉시) | Pull (최대 30초 지연) |
| 구현 복잡도 | 높음 | 낮음 (HTML 속성만) |
| 서버 부하 | 연결 유지 (낮음) | 주기적 요청 (보통) |
| 1GB 서버 적합성 | WebSocket 연결 메모리 소비 | Polling이 적합 |

### 4-3. Controller (Spring) vs Router (FastAPI)

**Java (Spring MVC)**
```java
@Controller
@RequestMapping("/partials")
public class DashboardPartialController {

    @GetMapping("/dashboard/watchlist-prices")
    public String watchlistPrices(Model model) {
        List<WatchlistItem> items = watchlistService.findAll();
        // prices 조회 후 model에 추가
        model.addAttribute("items", enrichedItems);
        return "partials/dashboard_watchlist_prices";  // 템플릿 이름
    }
}
```

**Python (FastAPI)**
```python
@web_router.get("/partials/dashboard/watchlist-prices", response_class=HTMLResponse)
async def partial_dashboard_watchlist_prices(
    request: Request,
    session: AsyncSession = Depends(get_session),
):
    # ... 조회 로직 ...
    return templates.TemplateResponse("partials/dashboard_watchlist_prices.html", {
        "request": request,
        "items": items,
    })
```

---

## 5. Key Concept: 초보자가 놓치기 쉬운 포인트

### 5-1. HTMX의 핵심 철학

HTMX는 "서버가 HTML을 반환한다"는 전통 웹 방식으로 돌아간다:

```
전통 방식 (HTMX):
  브라우저 → GET /partials/summary → 서버 → HTML 반환 → 브라우저가 삽입

SPA 방식 (React):
  브라우저 → GET /api/summary → 서버 → JSON 반환 → 브라우저가 렌더링
```

- HTML은 서버에서 Jinja2로 렌더링 → 브라우저는 받아서 삽입만 함
- JavaScript 없이도 동적 UI 구현 가능
- 1GB RAM 서버에서 Node.js + React 대신 Python + HTMX가 훨씬 가볍다

### 5-2. hx-swap 옵션 차이

| 옵션 | 동작 | 사용 시점 |
|:---|:---|:---|
| `innerHTML` | 요소 내부 교체 | 컨테이너 div 유지하면서 내용 갱신 |
| `outerHTML` | 요소 자체 교체 | 요소 전체를 서버 응답으로 교체 |
| `none` | DOM 변경 없음 | 서버에 데이터 전송만 (추가 버튼 등) |
| `afterend` | 요소 뒤에 삽입 | 리스트에 항목 추가 |

### 5-3. Partial 템플릿 분리 전략

대시보드는 여러 부분이 서로 다른 주기로 갱신된다:

```
summary_cards    → 30초 (잔고/손익)
watchlist_prices → 30초 (시세)
position_table   → 30초 (보유종목)
stock_price_hero → 10초 (선택 종목 현재가)
```

각 파셜을 독립적인 URL로 분리하면:
- 특정 부분만 갱신할 수 있어 서버 부하 분산
- 갱신 주기를 부분별로 다르게 설정 가능
- 서버에서 템플릿 재사용 (초기 렌더링 + HTMX 갱신에 동일 템플릿)

### 5-4. 에러 내성 설계

```python
for m in memos:
    try:
        p = await market_svc.get_price(m.symbol, m.market)
        items.append({"price": p.price, "change_pct": p.change_pct, ...})
    except Exception:
        items.append({"price": 0, "change_pct": 0, ...})  # 시세 없어도 종목은 표시
```

시세 API 실패 시 해당 종목만 `-`로 표시하고 나머지는 정상 표시한다. 하나의 시세 오류가 전체 대시보드를 망가뜨리지 않는 방어적 설계이다.

### 5-5. CSS 변수(Custom Properties) 활용

```html
<span style="color:var(--text-muted);">-</span>
```

`--text-muted`는 CSS 파일에 정의된 변수:
```css
:root {
    --text-muted: #6e7681;
}
```

다크 모드 전환 시 변수 값만 바꾸면 전체 UI가 일관되게 변경된다. 하드코딩된 색상(`color:#6e7681`)보다 유지보수가 쉽다.
