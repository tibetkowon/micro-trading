# Phase 13: 미국 주식 제거 + 주문가능금액 거래 모드별 표시 학습 문서

## 1. Overview — 이 기능이 왜 필요한가?

### 미국 주식 제거

KIS API는 해외주식 거래를 지원하지만 다음 이유로 실질적 사용이 불가능했습니다.

- KIS 해외주식 전용 계좌 개설 필요 (일반 계좌와 별도)
- 해외주식 실시간 시세는 별도 요금제 가입 필요
- 환율 처리, USD/KRW 이중 잔고 관리의 복잡도

**불필요한 복잡도를 제거**하고 국내주식에 집중함으로써 코드베이스를 단순화합니다.

### 주문가능금액 버그 수정

실매매 모드로 전환해도 주문 폼의 "주문 가능" 금액이 항상 모의투자 잔고(`paper_balance_krw`)를 표시하는 버그가 있었습니다.

```
버그 전: REAL 모드 → 주문 가능 = paper_balance_krw (잘못됨)
수정 후: REAL 모드 → 주문 가능 = KIS 실계좌 예수금 (정확)
         PAPER 모드 → 주문 가능 = paper_balance_krw
```

---

## 2. Logic Flow — 데이터가 어떻게 흘러가는가?

### Feature 1: 미국 주식 제거 경로

```
app/schemas/common.py       Market enum에서 US 제거 → KR만 허용
app/web/stock_list.py       US_STOCKS 리스트 삭제
app/broker/free/provider.py _get_us_price(), _get_us_daily() 제거
app/broker/kis/broker.py    _place_us_order(), _cancel_us_order() 등 5개 메서드 제거
app/broker/kis/endpoints.py US_ORDER_PATH 등 US 상수 전체 삭제
app/services/
  order_service.py          _get_balance_field() 헬퍼 제거 → paper_balance_krw 직접 참조
  stock_master_service.py   sync_us_stocks() 제거
  portfolio_service.py      cash_usd = 0.0 고정 (USD 잔고 미사용 선언)
app/web/routes.py           US 탭·루프·임포트 제거
templates/                  US option, US 탭 버튼, USD 표시 블록 제거
```

### Feature 2: 주문가능금액 거래 모드별 분기

```
GET /partials/order-balance
  └─ routes.py: partial_order_balance()
       ├─ settings.get_trading_mode() == REAL?
       │    └─ ConnectionService().get_real_balance()   ← KIS API 예수금 조회
       │         └─ real.get("cash_krw", 0.0)
       └─ PAPER?
            └─ Account.paper_balance_krw               ← DB 모의투자 잔고
  └─ templates/partials/order_balance.html 렌더링 (원화 단일 표시)
```

---

## 3. Pythonic Point — 파이썬 특유의 기법

### `getattr` / `setattr` vs 직접 접근

이전 코드는 동적 필드 접근을 사용했습니다.

```python
# 이전: 동적으로 필드명을 결정해서 접근
balance_field = "paper_balance_usd" if market == "US" else "paper_balance_krw"
available = getattr(account, balance_field)
setattr(account, balance_field, new_value)

# 수정 후: KR만 지원하므로 직접 접근
available = account.paper_balance_krw
account.paper_balance_krw = new_value
```

`getattr(obj, "attr_name")` 은 `obj.attr_name` 과 동일합니다. 필드명이 런타임에 결정될 때 유용하지만, 필드가 고정되면 직접 접근이 더 명확합니다.

### Enum 단순화

```python
# 이전
class Market(str, Enum):
    KR = "KR"
    US = "US"

# 수정 후
class Market(str, Enum):
    KR = "KR"
```

`str, Enum` 다중 상속은 열거형 값이 문자열처럼 동작하게 합니다. `Market.KR == "KR"` 이 `True` 입니다. FastAPI는 이를 자동으로 경로 파라미터 검증에 사용합니다.

---

## 4. Java vs Python — 비교

### Java (Spring)라면?

```java
// Java: 별도 Market enum 파일, switch 문
public enum Market {
    KR, US;
}

// Controller
@GetMapping("/order-balance")
public ResponseEntity<?> getOrderBalance(@RequestParam String market) {
    if (tradingModeService.isReal()) {
        BigDecimal balance = kisService.getCashBalance(); // KIS API 호출
        return ResponseEntity.ok(Map.of("balance", balance));
    }
    Account account = accountRepository.findFirst();
    BigDecimal balance = account.getPaperBalanceKrw();
    return ResponseEntity.ok(Map.of("balance", balance));
}
```

### Python (FastAPI)라면?

```python
# Python: async/await로 I/O 비차단 처리
@web_router.get("/partials/order-balance", response_class=HTMLResponse)
async def partial_order_balance(request: Request, session: AsyncSession = Depends(get_session)):
    mode = settings.get_trading_mode()
    if mode == TradingMode.REAL:
        real = await ConnectionService().get_real_balance()  # 비동기 KIS 호출
        balance = real.get("cash_krw", 0.0) if not real.get("error") else 0.0
    else:
        account = (await session.execute(select(Account).limit(1))).scalar_one_or_none()
        balance = account.paper_balance_krw if account else 0.0
    return templates.TemplateResponse(...)
```

**차이점:**
- Java Spring은 스레드풀로 동기 I/O를 처리 (Tomcat 기본)
- Python FastAPI는 `async/await` 로 단일 이벤트 루프에서 I/O를 비차단 처리
- 1GB RAM 서버에서 스레드풀보다 이벤트루프가 메모리 효율적

---

## 5. Key Concept — 초보자가 놓치기 쉬운 포인트

### DB 마이그레이션 없이 필드 유지

`Account` 모델의 `paper_balance_usd`, `initial_balance_usd` 필드는 **삭제하지 않았습니다.**

```
이유: SQLite는 컬럼 삭제(DROP COLUMN)가 복잡하고,
      기존 운영 DB에 데이터가 있을 수 있어 마이그레이션 위험이 있음.
해결: 코드에서 더 이상 사용하지 않고 0.0으로 고정 → DB 스키마는 그대로 유지
```

이것은 **Strangler Fig 패턴**의 미니 버전입니다. 레거시 필드를 즉시 삭제하지 않고, 코드 레벨에서 먼저 무력화한 뒤 나중에 안전하게 제거합니다.

### HTMX partial 파라미터 제거의 영향

```html
<!-- 이전: market 파라미터 포함 -->
hx-get="/partials/order-balance?market={{ market }}"

<!-- 수정 후: 파라미터 없음 -->
hx-get="/partials/order-balance"
```

HTMX는 `hx-get` URL을 그대로 GET 요청으로 보냅니다. 서버 측에서 `market` 파라미터를 제거했으므로, 템플릿에서도 함께 제거해야 합니다. 파라미터가 남아 있어도 서버가 무시하지만, 불필요한 노이즈를 제거하는 것이 원칙입니다.

### `real.get("cash_krw", 0.0)` 패턴

```python
balance = real.get("cash_krw", 0.0) if not real.get("error") else 0.0
```

- `dict.get(key, default)`: 키가 없으면 기본값 반환 (KeyError 없음)
- KIS API 오류 시 `{"error": "..."}` 형태로 반환되므로 `error` 키 존재 여부를 먼저 체크
- 오류가 있으면 0.0을 표시해 사용자에게 잘못된 잔고가 표시되지 않도록 방어

---

## 6. 변경 파일 목록

| 파일 | 변경 내용 |
|------|-----------|
| `app/schemas/common.py` | `Market.US` 제거 |
| `app/web/stock_list.py` | `US_STOCKS` 리스트 삭제 |
| `app/broker/free/provider.py` | US 메서드 2개 + US 분기 제거 |
| `app/broker/kis/broker.py` | US 메서드 5개 + US 분기 제거 |
| `app/broker/kis/endpoints.py` | US 상수 전체 삭제 |
| `app/services/order_service.py` | `_get_balance_field()` 제거, 직접 참조로 교체 |
| `app/services/stock_master_service.py` | `sync_us_stocks()` 제거 |
| `app/services/portfolio_service.py` | `cash_usd = 0.0` 고정 |
| `app/web/routes.py` | US 제거 + order-balance 모드 분기 수정 |
| `templates/orders.html` | US option 제거 |
| `templates/strategies.html` | US option 제거 |
| `templates/partials/watchlist.html` | US 탭 제거 |
| `templates/partials/order_balance.html` | market 파라미터 제거, 원화 단일 표시 |
| `templates/partials/summary_cards.html` | USD 표시 블록 제거 |
| `templates/partials/account_info.html` | USD 잔고 행 제거 |
| `templates/partials/stock_detail.html` | order-balance market 파라미터 제거 |
| `tests/test_order_service.py` | `test_us_market_uses_usd_balance()` 제거 |
