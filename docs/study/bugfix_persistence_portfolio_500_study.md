# Bugfix: 설정 영속화 · 포트폴리오 잔고 조회 · 메인화면 500 오류

## 1. Overview — 무슨 문제였는가?

### 버그 1: 재기동 시 거래 모드 초기화

앱을 재기동할 때마다 거래 모드가 `.env` 기본값(`PAPER`)으로 돌아갔습니다.

```
버그 전: 실매매로 전환 → 앱 재기동 → 다시 모의투자로 초기화 (매번 수동 전환 필요)
수정 후: 실매매로 전환 → 앱 재기동 → 실매매 유지 (runtime_settings.json 영속화)
```

**근본 원인**: `_runtime_trading_mode`가 파이썬 모듈 수준의 메모리 변수였기 때문에 프로세스 재시작 시 항상 `None`으로 초기화됨.

---

### 버그 2: 포트폴리오 요약 API `cash_krw = 0`

`GET /api/portfolio/summary` 호출 시 `cash_krw`, `orderable_krw`가 항상 0으로 반환됨.

```json
{
  "total_value": 37860,
  "total_invested": 0,
  "cash_krw": 0,        ← ⚠️ 항상 0
  "orderable_krw": 0    ← ⚠️ 항상 0
}
```

**근본 원인 (1)**: 서비스 코드에서 REAL 모드일 때 잔고를 `0.0`으로 하드코딩.

```python
# 버그 코드
cash_krw = account.paper_balance_krw if is_paper else 0.0  # REAL → 무조건 0
```

**근본 원인 (2)**: API 엔드포인트의 `trading_mode` 기본값이 `"PAPER"` 하드코딩. 사용자가 REAL 모드로 전환한 상태에서 파라미터 없이 호출하면 PAPER 기준으로 응답.

```python
# 버그 코드
async def portfolio_summary(trading_mode: str = "PAPER", ...):
```

---

### 버그 3: 메인 트레이딩 화면 500 오류

`GET /` 접속 시 HTTP 500 오류가 발생.

**근본 원인 (1)**: `order_table.html` 템플릿이 `commission_map` 딕셔너리를 사용하는데, 메인 뷰 렌더링 컨텍스트에 해당 변수가 누락됨. 주문 내역이 존재하면 Jinja2가 `UndefinedError`를 발생.

```html
<!-- order_table.html -->
{{ "{:,.0f}".format(commission_map.get(o.id, 0)) }}원  ← commission_map 없으면 500
```

**근본 원인 (2)**: `dashboard_default.html`의 HTMX 폴링 경로 `/partials/orders`가 라우터에 등록되지 않아 404 반환.

```html
<div hx-get="/partials/orders" hx-trigger="load, every 30s">  ← 라우트 미존재
```

---

## 2. Logic Flow — 데이터가 어떻게 흘러가는가?

### Fix 1: 거래 모드 영속화 흐름

```
[거래 모드 전환]
POST /settings/trading-mode (mode=REAL)
  └─ routes.py: switch_trading_mode()
       └─ config.py: Settings.switch_trading_mode(TradingMode.REAL)
            ├─ _runtime_trading_mode = TradingMode.REAL  (메모리)
            └─ _save_runtime_settings()                  (파일)
                 └─ runtime_settings.json: {"trading_mode": "REAL"}

[앱 재기동 후 첫 모드 조회]
settings.get_trading_mode()
  ├─ _runtime_trading_mode is None?  → Yes
  │    └─ _load_runtime_settings()
  │         └─ runtime_settings.json 읽기 → _runtime_trading_mode = REAL
  └─ return TradingMode.REAL  ✓
```

### Fix 2: 포트폴리오 잔고 조회 흐름

```
GET /api/portfolio/summary  (파라미터 없음)
  └─ api/portfolio.py: portfolio_summary(trading_mode=None)
       ├─ mode = settings.get_trading_mode().value  ← 런타임 모드 자동 반영
       └─ PortfolioService.get_summary(mode)

PortfolioService.get_summary(mode="REAL")
  ├─ is_paper = False
  ├─ [PAPER] cash_krw = account.paper_balance_krw
  └─ [REAL]  ConnectionService.get_real_balance()
               └─ KIS API → broker.get_balance() → cash_krw (실잔고)
                  실패 시 → cash_krw = 0.0 (폴백, 경고 로그)
```

### Fix 3: 메인화면 렌더링 흐름

```
GET /
  └─ routes.py: trading_view()
       ├─ order_svc.get_orders(limit=10)           → recent_orders
       ├─ order_svc.get_trades_by_order_ids(ids)   → commission_map  ← 신규 추가
       └─ TemplateResponse("trading.html", {
               "orders": recent_orders,
               "commission_map": commission_map,   ← 컨텍스트 전달
          })

GET /partials/orders  ← 신규 라우트
  └─ routes.py: partial_orders()
       ├─ order_svc.get_orders(limit=10)
       ├─ order_svc.get_trades_by_order_ids(ids) → commission_map
       └─ TemplateResponse("partials/order_table.html", {orders, commission_map})
```

---

## 3. 변경 파일 목록

| 파일 | 변경 유형 | 내용 |
|:---|:---|:---|
| `app/config.py` | 수정 | `_load_runtime_settings()`, `_save_runtime_settings()` 추가; `get_trading_mode()`·`switch_trading_mode()` 영속화 로직 연동 |
| `app/api/portfolio.py` | 수정 | `trading_mode` 파라미터 기본값 `"PAPER"` → `None` (런타임 모드 자동 반영) |
| `app/services/portfolio_service.py` | 수정 | REAL 모드에서 `ConnectionService.get_real_balance()`로 실잔고 조회; 수익률 기준 `initial_balance_krw` 사용 |
| `app/web/routes.py` | 수정 | 메인 뷰에 `commission_map` 컨텍스트 추가; `/partials/orders` 라우트 신규 등록 |

---

## 4. 핵심 코드 Before / After

### 설정 영속화

```python
# Before (인메모리만, 재기동 시 초기화)
_runtime_trading_mode: TradingMode | None = None

def switch_trading_mode(self, mode: TradingMode) -> None:
    global _runtime_trading_mode
    _runtime_trading_mode = mode

# After (파일 영속화)
_RUNTIME_SETTINGS_PATH = pathlib.Path("runtime_settings.json")

def switch_trading_mode(self, mode: TradingMode) -> None:
    global _runtime_trading_mode
    _runtime_trading_mode = mode
    _save_runtime_settings(mode)   # → runtime_settings.json 기록

def get_trading_mode(self) -> TradingMode:
    global _runtime_trading_mode
    if _runtime_trading_mode is None:
        _load_runtime_settings()   # ← 재기동 후 첫 접근 시 파일에서 복원
    ...
```

### REAL 모드 잔고 조회

```python
# Before
cash_krw = account.paper_balance_krw if is_paper else 0.0  # REAL → 항상 0

# After
if is_paper:
    cash_krw = account.paper_balance_krw
else:
    real_bal = await ConnectionService().get_real_balance()
    cash_krw = real_bal.get("cash_krw", 0.0) if not real_bal.get("error") else 0.0
```

### API 엔드포인트 기본값

```python
# Before
async def portfolio_summary(trading_mode: str = "PAPER", ...):

# After
async def portfolio_summary(trading_mode: str | None = None, ...):
    mode = trading_mode or settings.get_trading_mode().value  # 런타임 모드 우선
```

---

## 5. Java 비교 분석

### 설정 영속화 패턴

| 관점 | Python (이번 구현) | Java Spring Boot |
|:---|:---|:---|
| 런타임 설정 저장 | JSON 파일 직접 읽기/쓰기 | `@ConfigurationProperties` + DB or Vault |
| 재기동 후 복원 | 모듈 초기화 시 지연 로드 | `@PostConstruct` Bean 초기화 |
| 저장소 | `runtime_settings.json` (로컬 파일) | Redis / DB / Spring Cloud Config |
| 동시성 | 단일 프로세스라 `global` 변수로 충분 | `@Synchronized` or `AtomicReference` 필요 |

### 템플릿 컨텍스트 패턴

| 관점 | Jinja2 (이번 구현) | Thymeleaf / FreeMarker (Java) |
|:---|:---|:---|
| 변수 누락 시 | `UndefinedError` → 500 | `null` 처리 or `#variables.containsKey()` |
| 컨텍스트 전달 | `dict`를 명시적으로 구성 | `Model.addAttribute()` or `ModelAndView` |
| 재사용 단위 | `{% include %}` partial | `th:fragment` / `th:replace` |

---

## 6. 운영 참고

- **`runtime_settings.json`**: 앱 실행 디렉토리(`./`)에 자동 생성됩니다. Docker 컨테이너 재시작 후에도 유지하려면 볼륨 마운트 경로(`./trading.db`와 동일 위치)에 포함시켜야 합니다.
- **KIS API 미설정 시 REAL 잔고**: API 키가 없거나 호출 실패 시 `cash_krw = 0`으로 폴백하며, 서버 로그에 경고(`WARNING`)가 기록됩니다.
- **수익률 계산**: PAPER 모드는 `initial_balance_krw` 기준, REAL 모드는 현재 투자원금(`cash_krw + total_invested`) 기준으로 계산됩니다.
