# Phase 2: 실시간 시장가 API 연동 - 학습 문서

## 1. Overview: 왜 시장가 API 연동을 개선해야 했는가?

기존 `MarketService`는 메모리 TTL 캐시만 사용하여:
- 서버 재시작 시 모든 시세 데이터 유실
- DB에 시세를 저장하지 않으므로 오프라인/API 장애 시 대응 불가
- 스케줄러에 주기적 시세 갱신이 없어 F-03 요구사항(30초 간격) 미충족
- `TradingMode`이 `config.py`에서 제거되었는데 참조가 남아있음

Phase 2에서는 **3계층 캐시 전략**(메모리 → API → DB)을 구현하고, 스케줄러로 자동 갱신한다.

---

## 2. Logic Flow: 시세 데이터 흐름

### 2-1. 시세 조회 흐름 (get_price)

```
클라이언트 요청
    │
    ▼
[1] 메모리 캐시 확인 (TTL 15초)
    ├── HIT → 즉시 반환
    │
    ▼ MISS
[2] API 호출 (pykrx/yfinance via FreeMarketProvider)
    ├── 성공 → 메모리 캐시 저장 + DB 저장 → 반환
    │
    ▼ 실패
[3] DB 캐시에서 복원 (PriceCache 테이블)
    ├── 있음 → 반환
    │
    ▼ 없음
[4] 만료된 메모리 캐시 반환
    │
    ▼ 없음
[5] PriceInfo(price=0.0) 반환
```

### 2-2. 스케줄러 시세 갱신 흐름

```
APScheduler (30초 간격)
    │
    ▼
refresh_prices() job
    │
    ▼
MarketService.refresh_watchlist_prices()
    ├── WatchlistItem 전체 조회
    ├── Position (수량 > 0) 전체 조회
    ├── 중복 제거 (set)
    └── 각 종목에 get_price() 호출 → 자동으로 DB 저장
```

### 2-3. 브로커 계층 구조

```
MarketService
    │
    ▼
BrokerFactory.get_broker()
    ├── KIS 키 있음 → KISBroker (실제 증권사 API)
    └── KIS 키 없음 → PaperBroker
                            └── FreeMarketProvider
                                  ├── KR → pykrx (한국거래소 무료 데이터)
                                  └── US → yfinance (Yahoo Finance)
```

---

## 3. Pythonic Point: 핵심 기법

### 3-1. Optional 의존성 주입 (session: AsyncSession | None)

```python
class MarketService:
    def __init__(self, session: AsyncSession | None = None):
        self.session = session
```

- session이 있으면 DB 캐시 활용, 없으면 메모리 캐시만 사용
- **하위 호환성 유지**: 기존 `MarketService()` 호출도 동작
- 파이썬의 `Union` 타입(`X | None`)으로 간결하게 표현

### 3-2. asyncio.to_thread (동기 라이브러리 비동기 래핑)

```python
async def _get_kr_price(self, symbol: str) -> PriceInfo:
    def _fetch():
        from pykrx import stock as pykrx_stock
        df = pykrx_stock.get_market_ohlcv_by_date(...)
        return PriceInfo(...)

    return await asyncio.to_thread(_fetch)
```

- `pykrx`, `yfinance`는 동기 라이브러리 → 비동기 이벤트 루프를 블로킹
- `asyncio.to_thread()`로 별도 스레드에서 실행 → 이벤트 루프 차단 방지
- **주의**: 스레드 안에서 async 함수를 호출할 수 없음

### 3-3. Lazy Import 패턴

```python
async def _save_to_db(self, info: PriceInfo) -> None:
    try:
        from app.services.price_cache_service import PriceCacheService
        svc = PriceCacheService(self.session)
        await svc.upsert(info)
    except Exception as e:
        logger.debug("DB 캐시 저장 실패: %s", e)
```

- 함수 내부에서 import → 순환 참조(circular import) 방지
- 모듈 로드 시점이 아닌 실행 시점에 의존성 해결
- try/except로 감싸서 DB 저장 실패가 시세 조회를 중단시키지 않음

### 3-4. APScheduler의 interval vs cron

```python
# interval: 정확히 N초마다 실행
scheduler.add_job(refresh_prices, "interval", seconds=30)

# cron: 특정 시간/분에 실행 (리눅스 crontab과 동일)
scheduler.add_job(run_strategy_tick, "cron", hour="9-15", minute="*/1")
```

- **interval**: 시세 갱신처럼 일정 간격 반복에 적합
- **cron**: 장 시간대에만 실행해야 하는 전략 틱에 적합

---

## 4. Java vs Python: 캐시 및 스케줄링 비교

### 4-1. 캐시 전략

**Java (Spring Cache)**
```java
@Cacheable(value = "prices", key = "#symbol + '_' + #market")
public PriceInfo getPrice(String symbol, String market) {
    return broker.getCurrentPrice(symbol, market);
}
```

**Python (수동 dict 캐시)**
```python
_price_cache: dict[tuple[str, str], tuple[PriceInfo, float]] = {}

async def get_price(self, symbol, market):
    key = (symbol, market)
    cached = _price_cache.get(key)
    if cached and (now - cached[1]) < CACHE_TTL:
        return cached[0]
```

| 항목 | Java (Spring) | Python (이 프로젝트) |
|:---|:---|:---|
| 캐시 방식 | `@Cacheable` + Redis/Caffeine | dict + monotonic time |
| TTL 설정 | `@CacheConfig(cacheNames=..., ttl=...)` | 상수 `CACHE_TTL = 15` |
| 캐시 저장소 | Redis, EhCache, Caffeine | 모듈 레벨 dict (싱글턴) |
| 메모리 부담 | Redis 별도 프로세스 | 프로세스 내 dict (초경량) |

1GB RAM 환경에서는 Redis를 띄울 여유가 없으므로, dict + SQLite 조합이 적절하다.

### 4-2. 스케줄링

**Java (Spring Scheduler)**
```java
@Scheduled(fixedRate = 30000)  // 30초
public void refreshPrices() { ... }

@Scheduled(cron = "0 */1 9-15 * * MON-FRI")  // 장중 1분
public void runStrategy() { ... }
```

**Python (APScheduler)**
```python
scheduler.add_job(refresh_prices, "interval", seconds=30)
scheduler.add_job(run_strategy_tick, "cron", hour="9-15", minute="*/1")
```

| 항목 | Java (Spring) | Python (APScheduler) |
|:---|:---|:---|
| 어노테이션 | `@Scheduled` | `scheduler.add_job()` |
| 비동기 지원 | `@Async` + `@Scheduled` 조합 | 네이티브 async 지원 |
| 동적 등록 | `TaskScheduler` 인터페이스 | `add_job()` / `remove_job()` |
| 영속성 | DB 기반 (Quartz) | 메모리 (기본), DB (선택) |

---

## 5. Key Concept: 초보자가 놓치기 쉬운 포인트

### 5-1. 모듈 레벨 dict는 싱글턴처럼 동작한다

```python
# market_service.py 모듈 레벨
_price_cache: dict[...] = {}
```

파이썬 모듈은 첫 import 시 한 번만 실행된다. 따라서 `_price_cache`는 프로세스 내에서 하나만 존재한다. `MarketService`를 여러 번 인스턴스화해도 같은 캐시를 공유한다.

### 5-2. time.monotonic() vs time.time()

```python
now = time.monotonic()  # ← 사용 중
```

- `time.time()`: 시스템 시계 기반 → NTP 동기화로 시간이 뒤로 갈 수 있음
- `time.monotonic()`: 단조 증가 보장 → 경과 시간 측정에 안전
- TTL 캐시처럼 "N초 지났는지"를 판단할 때는 반드시 `monotonic()` 사용

### 5-3. Graceful Degradation (점진적 성능 저하)

```python
try:
    price_info = await broker.get_current_price(symbol, market)
except Exception:
    db_price = await self._load_from_db(symbol, market)  # DB 캐시 복원
    if not db_price and cached:
        return cached[0]  # 만료된 메모리 캐시라도 반환
    return PriceInfo(price=0.0)  # 최후 수단
```

API 장애 시 서비스가 완전히 죽지 않도록 **다단계 폴백** 전략을 사용한다:
1. DB 캐시 (마지막으로 성공한 시세)
2. 만료된 메모리 캐시
3. 기본값 (price=0.0)

이는 "실패해도 괜찮은" 시세 데이터의 특성을 활용한 설계이다.

### 5-4. pykrx와 yfinance의 제한사항

| 라이브러리 | 제한 | 대응 |
|:---|:---|:---|
| pykrx | 한국거래소 OHLCV만 제공, 실시간이 아닌 종가 기반 | 장 종료 후 데이터가 가장 정확 |
| yfinance | Yahoo API 비공식, 과도한 호출 시 차단 가능 | 30초 간격 + TTL 캐시로 호출 최소화 |
| 공통 | 동기 라이브러리 | `asyncio.to_thread()`로 래핑 |

무료 API의 한계를 인지하고, 캐시 전략으로 호출 빈도를 최소화하는 것이 핵심이다.
