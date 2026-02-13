# Phase 3: 가상 거래 로직 구현 - 학습 문서

## 1. Overview: 왜 가상 거래 로직이 필요했는가?

Phase 2까지는 시세를 조회하고 주문 레코드를 만드는 틀만 있었다. 실제 "거래가 일어났을 때" 어떤 일이 벌어져야 하는지는 구현되지 않은 상태였다:

- **잔고 검증 없음**: 잔고가 부족해도 매수 주문이 통과됨
- **잔고 차감/증가 없음**: 체결 후 Account 잔고가 갱신되지 않아 항상 초기값 유지
- **수수료 미계산**: Trade 레코드에 `commission`, `total_amount`가 0 또는 null
- **통화 구분 없음**: 한국 주식(KRW)과 미국 주식(USD) 잔고를 동일하게 취급
- **테스트 부재**: 거래 로직이 올바른지 검증할 수단이 없음

요구사항 F-04(가상 거래), F-05(수수료 계산)를 충족하려면 주문 체결 시 정확한 잔고 변동과 수수료 계산이 필수였다.

---

## 2. Logic Flow: 주문 처리 전체 흐름

### 2-1. 매수 주문 흐름

```
create_order(req) 호출
    │
    ▼
[1] _get_default_account() → Account 조회
    │
    ▼
[2] _get_balance_field(market) → 사용할 잔고 필드 결정
    │  KR → "paper_balance_krw"
    │  US → "paper_balance_usd"
    │
    ▼
[3] 잔고 사전 검증 (BUY + PAPER 모드)
    │  estimated_total = price × quantity
    │  required = estimated_total + commission
    │  available = getattr(account, balance_field)
    │  if available < required → ValueError("잔고 부족")
    │
    ▼
[4] Order 레코드 생성 + session.flush()
    │
    ▼
[5] broker.place_order() 호출
    │
    ▼ 성공(result.success == True)
[6] Trade 레코드 생성
    │  total_amount = filled_price × filled_quantity
    │  commission = total_amount × account.commission_rate
    │
    ▼
[7] _update_position() 호출
    │  ├── 포지션 없음 → 신규 Position 생성
    │  ├── 포지션 있음 → 평균단가 재계산
    │  └── 잔고 차감: balance -= (total_amount + commission)
    │
    ▼
[8] session.commit() → DB 영속화
```

### 2-2. 매도 주문 흐름

```
[5] broker.place_order() 성공
    │
    ▼
[6] realized_pnl 계산
    │  realized_pnl = (filled_price - avg_price) × quantity
    │
    ▼
[7] _update_position()
    │  ├── position.quantity -= quantity
    │  ├── quantity == 0 → session.delete(position)
    │  └── 잔고 증가: balance += (total_amount - commission)
```

### 2-3. 평균단가 재계산 공식

```
새로운 avg_price = (기존 보유금액 + 추가 매수금액) / 새로운 총 수량
               = (avg_price × old_qty + price × new_qty) / (old_qty + new_qty)
```

---

## 3. Pythonic Point: 파이썬 핵심 기법

### 3-1. getattr / setattr: 동적 필드 접근

```python
def _get_balance_field(market: str) -> str:
    """시장에 따라 사용할 잔고 필드명 반환."""
    return "paper_balance_usd" if market == "US" else "paper_balance_krw"

# 사용 예시 (매수 시 잔고 조회)
balance_field = _get_balance_field(req.market.value)
available = getattr(account, balance_field)  # account.paper_balance_krw

# 잔고 변경 (매수 시 차감)
current_balance = getattr(account, balance_field)
setattr(account, balance_field, round(current_balance - total_amount - commission, 2))
```

**왜 getattr/setattr를 쓰는가?**
- `account.paper_balance_krw`와 `account.paper_balance_usd` 중 어느 필드를 써야 하는지는 런타임에 결정됨
- `if market == "KR": account.paper_balance_krw -= ...` 식으로 분기하면 코드 중복 발생
- `getattr(obj, "field_name")`은 `obj.field_name`과 완전히 동일하게 동작

### 3-2. pytest.fixture: 테스트 픽스처 패턴

```python
# conftest.py
@pytest.fixture
async def session():
    """각 테스트마다 독립적인 인메모리 DB 세션 제공."""
    engine = create_async_engine("sqlite+aiosqlite://", echo=False)  # ":memory:"와 동일

    @event.listens_for(engine.sync_engine, "connect")
    def _set_pragmas(dbapi_conn, _):
        cursor = dbapi_conn.cursor()
        cursor.execute("PRAGMA foreign_keys=ON")
        cursor.close()

    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)  # 모든 테이블 생성

    factory = async_sessionmaker(engine, class_=AsyncSession, expire_on_commit=False)
    async with factory() as sess:
        yield sess   # 테스트에 세션 전달

    await engine.dispose()  # 테스트 종료 후 정리 (인메모리 DB 소멸)
```

- `yield` 이전이 setup, 이후가 teardown
- `sqlite+aiosqlite://` (host 없음) = 인메모리 DB → 테스트마다 깨끗한 상태
- 실제 파일 DB를 건드리지 않아 안전

### 3-3. Mock 브로커 패턴

```python
def _mock_broker(fill_price: float = 50000.0):
    """성공하는 mock 브로커 생성."""
    broker = AsyncMock()
    broker.place_order.return_value = OrderResult(
        success=True,
        broker_order_id="PAPER-TEST001",
        filled_price=fill_price,
        filled_quantity=None,
    )
    broker.get_current_price.return_value = PriceInfo(
        symbol="005930", price=fill_price, market="KR",
    )
    return broker

# 테스트에서 사용
with patch("app.services.order_service.get_broker", return_value=broker):
    svc = OrderService(session)
    order = await svc.create_order(req)
```

- `AsyncMock`: 비동기 함수(async def)를 모방하는 Mock 객체
- `patch`: 특정 경로의 심볼을 테스트 중에만 교체 (컨텍스트 매니저 종료 시 복원)
- 브로커 API 호출 없이 거래 로직만 독립적으로 테스트 가능

### 3-4. pytest.approx: 부동소수점 비교

```python
# 잘못된 방법 (0.1 + 0.2 == 0.3은 False)
assert account.paper_balance_krw == 10_000_000 - 500_000 - 250

# 올바른 방법 (상대 오차 1e-6 이내면 통과)
assert account.paper_balance_krw == pytest.approx(10_000_000 - 500_000 - 250)
```

---

## 4. Java vs Python: 비교

### 4-1. 동적 필드 접근

**Java (리플렉션 - 복잡)**
```java
// Java에서는 리플렉션이 필요하고 checked exception 처리 필수
Field field = Account.class.getDeclaredField(balanceFieldName);
field.setAccessible(true);
double current = (double) field.get(account);
field.set(account, current - totalAmount - commission);
```

**Python (getattr/setattr - 단순)**
```python
# 파이썬은 문자열로 바로 속성 접근
current = getattr(account, balance_field)
setattr(account, balance_field, current - total_amount - commission)
```

| 항목 | Java | Python |
|:---|:---|:---|
| 동적 필드 접근 | `Reflection API` (복잡, 예외처리 필요) | `getattr/setattr` (직관적) |
| 타입 안전성 | 컴파일 타임 보장 | 런타임 (오타는 AttributeError) |
| 성능 | 리플렉션은 느림 | `getattr`는 일반 속성 접근과 동일 |

### 4-2. 테스트 구조 비교

**Java (JUnit 5 + Mockito)**
```java
@ExtendWith(MockitoExtension.class)
class OrderServiceTest {
    @Mock
    private BrokerFactory brokerFactory;

    @InjectMocks
    private OrderService orderService;

    @BeforeEach
    void setUp() {
        // H2 인메모리 DB 설정
        EntityManager em = Persistence.createEntityManagerFactory("test").createEntityManager();
    }

    @Test
    void testBuyDeductsBalance() {
        Mockito.when(brokerFactory.getBroker(any())).thenReturn(mockBroker);
        // ...
    }
}
```

**Python (pytest + unittest.mock)**
```python
# conftest.py에 fixture 정의 → 자동으로 주입
@pytest.mark.asyncio
async def test_buy_deducts_balance(session, account):  # fixture 자동 주입
    broker = _mock_broker(fill_price=50000.0)
    with patch("app.services.order_service.get_broker", return_value=broker):
        order = await OrderService(session).create_order(req)
    assert account.paper_balance_krw == pytest.approx(expected)
```

| 항목 | Java (JUnit+Mockito) | Python (pytest) |
|:---|:---|:---|
| 픽스처 | `@BeforeEach`, `@AfterEach` | `@pytest.fixture` (yield 기반) |
| Mock 주입 | `@Mock`, `@InjectMocks` | `with patch(...)` 컨텍스트 |
| 비동기 테스트 | `CompletableFuture` 복잡 | `@pytest.mark.asyncio` |
| 부동소수점 | `Assertions.assertEquals(a, b, delta)` | `pytest.approx(value)` |

### 4-3. 트랜잭션 처리

**Java (Spring @Transactional)**
```java
@Service
public class OrderService {
    @Transactional
    public Order createOrder(OrderRequest req) {
        // 메서드 전체가 트랜잭션
        // 예외 발생 시 자동 rollback
    }
}
```

**Python (명시적 commit)**
```python
class OrderService:
    async def create_order(self, req: OrderCreate) -> Order:
        # 트랜잭션은 session이 관리
        order = Order(...)
        self.session.add(order)
        await self.session.flush()  # DB에 반영하지만 커밋 X (ID 생성 목적)

        # ... 비즈니스 로직 ...

        await self.session.commit()  # 최종 영속화
        return order
```

- **flush vs commit**: `flush()`는 SQL을 실행하지만 트랜잭션 내에 유지 (롤백 가능), `commit()`은 영구 확정
- Python에서 예외가 발생하면 session은 rollback 상태가 됨

---

## 5. Key Concept: 초보자가 놓치기 쉬운 포인트

### 5-1. flush()가 필요한 이유

```python
order = Order(...)
self.session.add(order)
await self.session.flush()  # ← 이게 왜 필요한가?

# flush() 이후에만 order.id가 생김 (DB AUTOINCREMENT 실행됨)
trade = Trade(order_id=order.id, ...)  # flush 없으면 order.id == None
```

`flush()`는 SQL INSERT를 실행해서 DB로부터 `AUTOINCREMENT` ID를 받아온다. `commit()`을 기다리면 trade에 `order_id`를 넣을 수 없다.

### 5-2. realized_pnl vs unrealized_pnl

| 개념 | 설명 | 계산 시점 |
|:---|:---|:---|
| `realized_pnl` (실현 손익) | 실제로 팔아서 확정된 이익/손실 | 매도 체결 시 |
| `unrealized_pnl` (평가 손익) | 현재가 기준으로 계산한 이익/손실 | 조회 시 계산 |

```python
# 매도 시: realized_pnl 확정
realized_pnl = (filled_price - position.avg_price) * quantity
trade.realized_pnl = round(realized_pnl, 2)

# 조회 시: unrealized_pnl 계산 (포지션에 저장 안 함)
unrealized_pnl = (current_price - position.avg_price) * position.quantity
```

### 5-3. conftest.py의 역할

`conftest.py`는 pytest가 자동으로 찾는 특별한 파일이다:
- 같은 디렉토리 및 하위 테스트 파일에서 fixture를 `import 없이` 사용 가능
- `session`, `account` fixture가 `conftest.py`에 있으면 `test_order_service.py`에서 자동으로 주입됨
- 테스트 격리의 핵심: 각 테스트마다 새 인메모리 DB가 생성되어 테스트 간 간섭 없음

### 5-4. 수수료 계산의 정밀도

```python
commission = round(total_amount * account.commission_rate, 2)
# 500,000 * 0.0005 = 250.0 → round(250.0, 2) = 250.0

# round()를 안 하면?
# 0.1 * 3 = 0.30000000000000004 (부동소수점 오차)
```

금융 계산에서 `round(value, 2)`는 소수점 2자리 (센트/전 단위)까지 보장한다. 실무에서는 `decimal.Decimal`을 사용하기도 하지만, 이 프로젝트는 가상 거래이므로 float + round로 충분하다.

### 5-5. 전량 매도 시 포지션 삭제

```python
position.quantity -= quantity
if position.quantity == 0:
    await self.session.delete(position)  # 포지션 행 삭제
```

수량이 0인 포지션을 DB에 남기면 조회 시 노이즈가 된다. 전량 매도 후 포지션을 삭제함으로써 "보유 중인 종목"이 실제로 보유 중인 것만 표시되게 한다.
