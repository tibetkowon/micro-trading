# Phase 1: 데이터 모델 설계 및 구현 - 학습 문서

## 1. Overview: 왜 데이터 모델 리팩토링이 필요했는가?

기존 코드는 Phase 1~5가 혼재되어 빠르게 구현된 상태였다. 데이터 모델에 다음과 같은 문제가 있었다:

- **Account**: 초기 자산(`initial_balance`)이 없어 수익률 계산 불가, 수수료율 필드 부재
- **StockMemo**: "메모"라는 이름이 실제 역할(관심종목 관리)과 불일치
- **Trade**: 수수료 필드 없음 → 요구사항 F-05 위반
- **FK 컬럼 인덱스 누락**: `account_id` 등 외래키에 인덱스가 없어 JOIN 성능 저하
- **PriceCache 부재**: 시세가 메모리에만 존재 → 서버 재시작 시 유실
- **TradingMode 중복**: `config.py`와 `schemas/common.py`에 동일 Enum이 2개

이런 기반이 부실하면 Phase 2(API), Phase 3(거래 로직)을 올릴 때 계속 문제가 생긴다.

---

## 2. Logic Flow: 모델 간 관계와 데이터 흐름

```
Account (1) ──┬──< Order (N)
              │       │
              │       └──< Trade (N)  ← commission, total_amount 추가
              │
              └──< Position (N)

WatchlistItem  ← StockMemo에서 리네이밍
PriceCache     ← 신규 (symbol + market unique)
```

### 데이터 흐름 (주문 → 체결)
```
1. 사용자가 주문 생성 → Order (PENDING)
2. 거래 엔진이 시장가 확인 → PriceCache 참조
3. 체결 → Trade 생성 (commission = price × qty × rate)
4. Position 업데이트 (평균단가 재계산)
5. Account 잔고 차감
```

### 인덱스 전략
| 테이블 | 인덱스 | 용도 |
|:---|:---|:---|
| `orders` | `account_id`, `status`, `trading_mode` | 계좌별/상태별 주문 조회 |
| `trades` | `account_id`, `(account_id, trading_mode)` | P&L 집계 |
| `positions` | `account_id` | 계좌별 보유종목 조회 |
| `accounts` | `name` | 계좌 이름 검색 |

---

## 3. Pythonic Point: 파이썬/SQLAlchemy 핵심 기법

### 3-1. DeclarativeBase + mapped_column (SQLAlchemy 2.0 스타일)

```python
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column

class Base(DeclarativeBase):
    pass

class Account(Base):
    __tablename__ = "accounts"
    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(100), index=True)
```

- `Mapped[int]`은 파이썬 타입 힌트와 DB 컬럼 타입을 동시에 선언
- `mapped_column()`은 SQLAlchemy 2.0의 새 방식 (기존 `Column()` 대체)
- 타입 체커(mypy)와 완벽 호환

### 3-2. TimestampMixin (Mixin 패턴)

```python
class TimestampMixin:
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
        server_default=func.now(),
    )
```

- **Mixin**: 여러 모델에 공통 필드를 주입하는 패턴
- `default=lambda: ...` → 파이썬 레벨 기본값 (INSERT 시)
- `server_default=func.now()` → DB 레벨 기본값 (SQL DEFAULT)
- 두 개를 모두 지정하는 이유: ORM 경유 / raw SQL 모두 대응

### 3-3. `__table_args__` 튜플

```python
class Trade(Base):
    __table_args__ = (
        Index("ix_trades_account_mode", "account_id", "trading_mode"),
    )
```

- 복합 인덱스나 UniqueConstraint는 `__table_args__`에 튜플로 선언
- **주의**: 마지막에 `,`(trailing comma) 필수 → 튜플이 되어야 함

---

## 4. Java vs Python: JPA Entity와의 비교

### 4-1. 모델 선언 비교

**Java (JPA/Hibernate)**
```java
@Entity
@Table(name = "accounts", indexes = {
    @Index(columnList = "name")
})
public class Account {
    @Id @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(length = 100)
    private String name;

    @Column(columnDefinition = "DOUBLE DEFAULT 0.0005")
    private double commissionRate;
}
```

**Python (SQLAlchemy 2.0)**
```python
class Account(TimestampMixin, Base):
    __tablename__ = "accounts"
    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(100), index=True)
    commission_rate: Mapped[float] = mapped_column(default=0.0005)
```

| 항목 | Java (JPA) | Python (SQLAlchemy) |
|:---|:---|:---|
| 모델 선언 | `@Entity` + `@Table` | `Base` 상속 + `__tablename__` |
| PK 자동증가 | `@GeneratedValue` | `primary_key=True` (SQLite 자동) |
| 인덱스 | `@Index` 어노테이션 | `index=True` 파라미터 |
| 복합 인덱스 | `@Table(indexes=...)` | `__table_args__ = (Index(...),)` |
| Mixin | `@MappedSuperclass` | 일반 클래스 다중 상속 |
| Nullable | `@Column(nullable=true)` | `Mapped[str \| None]` |

### 4-2. 핵심 차이: Mixin 상속

Java에서 `@MappedSuperclass`는 단일 상속만 지원하므로, 여러 Mixin을 쓰려면 인터페이스나 `@Embeddable` 조합이 필요하다. Python은 다중 상속으로 자연스럽게 해결:

```python
class Account(TimestampMixin, SoftDeleteMixin, Base):
    ...
```

### 4-3. Spring의 application.yml vs Python의 PRAGMA

```yaml
# Spring - application.yml
spring:
  jpa:
    properties:
      hibernate.jdbc.batch_size: 20
```

```python
# Python - SQLite PRAGMA 직접 설정
@event.listens_for(engine.sync_engine, "connect")
def _set_sqlite_pragmas(dbapi_conn, _):
    cursor = dbapi_conn.cursor()
    cursor.execute("PRAGMA cache_size=-4000")  # 4MB
    cursor.execute("PRAGMA mmap_size=67108864")  # 64MB
```

Python/SQLite 환경에서는 DB 레벨 튜닝을 직접 PRAGMA로 제어한다.

---

## 5. Key Concept: 초보자가 놓치기 쉬운 포인트

### 5-1. SQLite PRAGMA 최적화

| PRAGMA | 값 | 의미 |
|:---|:---|:---|
| `journal_mode=WAL` | Write-Ahead Logging | 읽기/쓰기 동시 가능 |
| `cache_size=-4000` | 4MB | 페이지 캐시 크기 (1GB RAM 고려) |
| `mmap_size=67108864` | 64MB | 파일을 가상 메모리에 매핑 (물리 RAM 소비 X) |
| `temp_store=MEMORY` | - | 임시 테이블을 메모리에 생성 |
| `foreign_keys=ON` | - | FK 제약조건 활성화 (SQLite 기본은 OFF!) |

**주의**: `mmap_size`는 가상 주소 공간만 사용하므로 물리 메모리를 직접 소비하지 않는다. OS가 필요할 때만 페이지를 물리 RAM에 로드한다.

### 5-2. FK 인덱스는 자동 생성되지 않는다

SQLite와 SQLAlchemy 모두 **외래키 선언 시 인덱스를 자동 생성하지 않는다**. MySQL의 InnoDB는 FK에 자동 인덱스를 생성하지만, SQLite는 명시적으로 `index=True`를 추가해야 한다:

```python
# 인덱스 없음 → JOIN 시 full table scan
account_id: Mapped[int] = mapped_column(ForeignKey("accounts.id"))

# 인덱스 있음 → 빠른 조회
account_id: Mapped[int] = mapped_column(ForeignKey("accounts.id"), index=True)
```

### 5-3. async session과 expire_on_commit

```python
async_session = async_sessionmaker(engine, class_=AsyncSession, expire_on_commit=False)
```

`expire_on_commit=False`가 중요한 이유:
- `commit()` 후에도 객체 속성에 접근 가능
- 기본값(`True`)이면 commit 후 속성 접근 시 DB를 다시 조회 → async에서 `MissingGreenlet` 에러 발생
- 비동기 환경에서는 거의 필수 설정

### 5-4. PriceCache: TimestampMixin을 쓰지 않은 이유

`PriceCache`는 `created_at`이 불필요하다. 시세 데이터는 "언제 생성됐는지"보다 "마지막으로 언제 갱신됐는지"만 중요하므로 `updated_at` 하나만 직접 선언했다. 불필요한 컬럼은 디스크와 메모리 모두 낭비한다.
