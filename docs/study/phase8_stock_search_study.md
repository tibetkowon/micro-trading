# Phase 8: 종목 검색 기능 개선 - 학습 문서

## 1. Overview: 왜 검색 기능을 개선했는가?

기존 검색은 `stock_list.py`에 하드코딩된 **30개 종목**(KR 20 + US 10)에서만 동작했다. "한화에어로스페이스", "카카오뱅크" 같은 종목을 검색하면 아무 결과도 나오지 않는 심각한 UX 문제가 있었다.

개선 후:
- **KR**: pykrx로 KRX 전체 상장 종목(KOSPI+KOSDAQ, 2,500+개)을 DB에 동기화
- **US**: S&P500 전종목(502개)을 하드코딩하여 DB에 저장
- **초성 검색**: "ㅅㅅ" → 삼성전자, 삼성SDI 등 한글 초성 매칭 지원

---

## 2. Logic Flow: 데이터 흐름 단계별 설명

### 2-1. 종목 마스터 동기화 (앱 시작 시)

```
앱 시작 (lifespan)
  │
  ├─ DB 테이블 생성 (create_all)
  ├─ 마이그레이션 실행
  ├─ 기본 계정 생성
  │
  └─ 종목 마스터 동기화 ← NEW
       │
       ├─ DB에 데이터 있고 7일 이내? → 스킵
       │
       ├─ KR 종목: pykrx API 호출
       │    ├─ get_market_ticker_list("KOSPI")
       │    ├─ get_market_ticker_list("KOSDAQ")
       │    └─ 각 ticker → get_market_ticker_name() → DB upsert
       │
       └─ US 종목: US_STOCKS 리스트(502개) → DB upsert
```

### 2-2. 검색 요청 처리 흐름

```
사용자 입력 ("삼성" or "ㅅㅅ" or "AAPL")
  │
  ├─ HTMX: keyup → 300ms debounce → GET /partials/watchlist/search?q=...
  │
  └─ StockMasterService.search(query)
       │
       ├─ 초성 전용 쿼리인가? (is_chosung_only)
       │    ├─ Yes → KR 종목 전체 로드 → Python에서 초성 매칭
       │    └─ No  → DB LIKE 쿼리 (symbol OR name)
       │
       └─ 결과 정렬: exact(0) > prefix(1) > contains(2) > chosung
            │
            └─ 상위 15건 반환 → HTML 렌더링 (match_type 포함)
```

### 2-3. 초성 검색 알고리즘

```
"ㅅㅅ" 검색 시:
  1. is_chosung_only("ㅅㅅ") → True (모든 문자가 자음)
  2. DB에서 KR 종목 전체 로드 (symbol, name만 SELECT)
  3. 각 종목에 대해:
     - extract_chosung("삼성전자") → "ㅅㅅㅈㅈ"
     - "ㅅㅅ" in "ㅅㅅㅈㅈ" → True → 매칭!
  4. 15건 채워지면 조기 종료
```

---

## 3. Pythonic Point: 파이썬 특유의 기법

### 3-1. 유니코드 연산으로 초성 추출

```python
# 한글 유니코드 구조: 가(0xAC00) ~ 힣(0xD7A3)
# 각 글자 = 초성(19) × 중성(21) × 종성(28)
def extract_chosung(text: str) -> str:
    for ch in text:
        code = ord(ch)
        if 0xAC00 <= code <= 0xD7A3:
            idx = (code - 0xAC00) // (21 * 28)  # 초성 인덱스
            result.append(CHOSUNG_LIST[idx])
```

자바에서는 `Character.getType()`이나 ICU 라이브러리가 필요하지만, 파이썬에서는 `ord()` 하나로 유니코드 코드포인트를 직접 다룰 수 있어 외부 라이브러리 없이 구현이 가능하다.

### 3-2. SQLAlchemy Upsert 패턴

```python
async def _upsert(self, *, symbol, market, name, sector):
    result = await self.session.execute(
        select(StockMaster).where(
            StockMaster.symbol == symbol,
            StockMaster.market == market,
        )
    )
    existing = result.scalar_one_or_none()

    if existing:
        existing.name = name  # ORM이 변경 감지 (dirty tracking)
    else:
        self.session.add(StockMaster(...))
```

SQLAlchemy ORM의 **Unit of Work** 패턴: 객체 속성만 수정하면 `session.commit()` 시 자동으로 UPDATE SQL이 생성된다. JPA/Hibernate의 dirty checking과 동일한 개념이다.

### 3-3. Jinja2 필터 체이닝으로 하이라이트

```html
{{ item.name | replace(q, '<b>' ~ q ~ '</b>') | safe }}
```

`replace` 필터로 검색어 부분을 `<b>` 태그로 감싸고, `safe` 필터로 HTML 이스케이프를 비활성화한다. Django 템플릿에서는 커스텀 태그가 필요하지만 Jinja2는 파이프라인으로 간결하게 처리할 수 있다.

---

## 4. Java vs Python: Spring이라면 어떻게 짰을까?

### 4-1. 종목 마스터 모델

| 항목 | Python (SQLAlchemy) | Java (Spring JPA) |
|:---|:---|:---|
| 모델 정의 | `class StockMaster(TimestampMixin, Base)` | `@Entity class StockMaster extends BaseEntity` |
| 유니크 제약 | `__table_args__ = (UniqueConstraint(...),)` | `@Table(uniqueConstraints = @UniqueConstraint(...))` |
| Mixin | 다중 상속 `(TimestampMixin, Base)` | `@MappedSuperclass` 또는 `@Embeddable` |
| 타입 힌트 | `Mapped[str \| None]` | `@Column(nullable = true) String sector` |

**차이점**: Python은 다중 상속으로 Mixin을 자연스럽게 합성한다. Java는 단일 상속이므로 `@MappedSuperclass`나 `@EntityListeners`를 사용해야 한다.

### 4-2. 동기화 서비스

```java
// Spring에서의 동기화 패턴
@Service
public class StockMasterService {

    @Scheduled(cron = "0 0 6 * * MON")  // 매주 월요일 6시
    @Transactional
    public void syncIfNeeded() {
        // Spring Scheduler로 주기적 실행
    }
}
```

```python
# Python에서의 동기화 패턴
async def sync_if_needed(self) -> None:
    # 앱 시작 시 lifespan에서 호출
    # 별도 스케줄러 없이 "마지막 동기화 시점" 체크
```

**차이점**: Spring은 `@Scheduled`로 별도 스레드에서 주기적 실행이 자연스럽다. FastAPI의 lifespan은 시작/종료 시 1회 실행이므로, 주기적 동기화가 필요하면 APScheduler를 사용해야 한다. 이 프로젝트에서는 "7일 이상 경과 시 재수집" 방식으로 간단히 처리했다.

### 4-3. 검색 쿼리

```java
// Spring Data JPA - 메서드 이름으로 쿼리 생성
List<StockMaster> findBySymbolContainingOrNameContaining(String symbol, String name);

// 초성 검색은? → DB에서 불가능, 별도 로직 필요
@Query("SELECT s FROM StockMaster s WHERE s.market = 'KR'")
List<StockMaster> findAllKr();  // 가져와서 Java에서 필터링
```

```python
# SQLAlchemy - 명시적 쿼리 빌더
result = await self.session.execute(
    select(StockMaster).where(
        (StockMaster.symbol.ilike(q_like))
        | (StockMaster.name.ilike(q_like))
    ).limit(200)
)
```

**공통점**: 초성 검색은 어느 언어든 DB에서 직접 처리할 수 없어 애플리케이션 레벨에서 필터링해야 한다. Elasticsearch를 도입하면 한글 형태소 분석기(nori)로 해결할 수 있지만, 이 프로젝트의 메모리 제약(1GB)에서는 과도한 해결책이다.

---

## 5. Key Concept: 초보자가 놓치기 쉬운 포인트

### 5-1. pykrx는 동기(sync) 라이브러리

pykrx는 내부적으로 `requests`를 사용하는 **동기 라이브러리**다. FastAPI의 async 환경에서 직접 호출하면 이벤트 루프를 블로킹한다.

```python
# 현재 코드: lifespan에서 호출 (앱 시작 전이므로 영향 적음)
await stock_svc.sync_if_needed()
# → 내부에서 pykrx 동기 호출이 발생하지만,
#   앱이 아직 요청을 받지 않는 시점이라 문제 없음

# 만약 API 요청 중에 동기화를 해야 한다면?
# → run_in_executor()로 별도 스레드에서 실행해야 함
import asyncio
loop = asyncio.get_event_loop()
await loop.run_in_executor(None, pykrx_stock.get_market_ticker_list, today, "KOSPI")
```

### 5-2. 검색 정렬 전략: 사용자 의도 추론

검색 결과를 단순 ABC순으로 보여주면 사용자 경험이 나쁘다. "삼성"을 검색했을 때 "삼성전자"가 아닌 "대한항공(삼성 포함 아님)"이 먼저 나오면 안 된다.

```
정렬 우선순위:
  0: exact   → "삼성전자" == "삼성전자"  (정확 일치)
  1: prefix  → "삼성전자".startswith("삼성")  (접두사 일치)
  2: contains → "OO삼성OO" contains "삼성"  (부분 일치)
```

이 패턴은 IDE의 파일 검색, 브라우저 주소창 자동완성 등에서도 동일하게 사용된다.

### 5-3. DB 동기화의 멱등성(Idempotency)

`sync_kr_stocks()`를 여러 번 실행해도 결과가 동일해야 한다. upsert 패턴(있으면 UPDATE, 없으면 INSERT)이 이를 보장한다.

```python
existing = result.scalar_one_or_none()
if existing:
    existing.name = name       # UPDATE
else:
    self.session.add(StockMaster(...))  # INSERT
```

JPA의 `saveOrUpdate()`와 동일한 개념이다. `(symbol, market)` 유니크 제약이 데이터 무결성을 보장한다.

### 5-4. 메모리 최적화: 초성 검색 시 필요한 컬럼만 로드

```python
# 전체 객체 로드 (비효율)
select(StockMaster).where(StockMaster.market == "KR")

# 필요한 컬럼만 로드 (메모리 절약)
select(StockMaster.symbol, StockMaster.market, StockMaster.name)
    .where(StockMaster.market == "KR")
```

2,500개 종목의 모든 컬럼(id, sector, created_at, updated_at 등)을 로드하면 불필요한 메모리를 사용한다. 1GB 서버에서는 이런 작은 최적화가 중요하다. JPA에서는 `@Query`에 `SELECT new DTO(...)` 프로젝션을 사용하는 것과 같은 개념이다.
