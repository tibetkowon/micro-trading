# Phase 10: 실거래/모의투자 런타임 스위칭 - 학습 문서

## 1. Overview: 왜 런타임 스위칭이 필요한가?

기존에는 거래 모드를 바꾸려면 `.env` 파일의 `TRADING_MODE=REAL`을 수정하고 서버를 재시작해야 했다. 이 방식의 문제:
- 서버 재시작 = 다운타임 발생
- `.env` 직접 수정 = 오타 위험, SSH 접속 필요
- 빠른 모드 전환 불가 (모의투자로 테스트 → 실매매로 전환)

개선 후:
- **설정 페이지에서 버튼 클릭**으로 즉시 전환
- **확인 다이얼로그**: REAL 전환 시 경고 표시
- **안전장치**: KIS API 키 미설정 시 REAL 전환 거부
- **즉시 반영**: 전환 즉시 nav 뱃지, 포트폴리오, 주문 데이터가 해당 모드로 전환

---

## 2. Logic Flow: 모드 전환 처리

### 2-1. 모드 전환 플로우

```
사용자: "실매매로 전환" 버튼 클릭
  │
  ├─ confirmModeSwitch('REAL') → JavaScript
  │    └─ 확인 다이얼로그 표시
  │         "실제 자산으로 거래됩니다. 전환하시겠습니까?"
  │
  ├─ 사용자 "전환" 클릭
  │    └─ executeSwitch()
  │         └─ POST /settings/trading-mode {mode: 'REAL'}
  │
  ├─ 서버: switch_trading_mode 라우트
  │    ├─ mode == REAL?
  │    │    ├─ KIS API 키 미설정 → 400 에러 반환
  │    │    └─ KIS API 키 설정됨 → 계속
  │    │
  │    ├─ settings.switch_trading_mode(REAL)
  │    │    └─ _runtime_trading_mode = REAL (전역 변수 갱신)
  │    │
  │    └─ 응답: HX-Refresh: true → 전체 페이지 리로드
  │
  └─ 페이지 리로드 시
       ├─ settings.get_trading_mode() → REAL 반환
       ├─ nav 뱃지 → "실매매" (빨간색)
       ├─ 포트폴리오 → REAL 모드 데이터 표시
       └─ 주문 시 → KISBroker 사용 (실제 주문)
```

### 2-2. 모드별 데이터 분리

```
settings.get_trading_mode() == PAPER
  │
  ├─ 포지션 조회: is_paper=True → 모의 포지션만
  ├─ 주문 내역: trading_mode='PAPER' → 모의 주문만
  ├─ 잔고: PaperBroker.get_balance() → DB Account 테이블
  └─ 주문 실행: PaperBroker → PaperExecutionEngine (가상 체결)

settings.get_trading_mode() == REAL
  │
  ├─ 포지션 조회: is_paper=False → 실거래 포지션만
  ├─ 주문 내역: trading_mode='REAL' → 실거래 주문만
  ├─ 잔고: KISBroker.get_balance() → KIS API 실계좌 조회
  └─ 주문 실행: KISBroker → KIS OpenAPI (실제 주문)
```

---

## 3. 핵심 코드 상세 분석

### 3-1. 런타임 모드 변수 — Pydantic Settings 우회

```python
# app/config.py

# Settings는 pydantic BaseSettings로 frozen (불변)
# → 런타임 변경을 위해 별도 전역 변수 사용
_runtime_trading_mode: TradingMode | None = None

class Settings(BaseSettings):
    trading_mode: TradingMode = TradingMode.PAPER  # .env 기본값

    def get_trading_mode(self) -> TradingMode:
        """런타임 변경값이 있으면 우선 사용."""
        if _runtime_trading_mode is not None:
            return _runtime_trading_mode
        return self.trading_mode  # .env 값

    def switch_trading_mode(self, mode: TradingMode) -> None:
        """전역 변수를 갱신하여 런타임 모드 전환."""
        global _runtime_trading_mode
        _runtime_trading_mode = mode
```

**왜 전역 변수인가?**: Pydantic `BaseSettings`는 기본적으로 `model_config = {"frozen": True}`와 유사하게 동작한다. `settings.trading_mode = TradingMode.REAL`로 직접 변경하면 `ValidationError`가 발생한다. 따라서 별도의 전역 변수로 오버라이드하는 패턴을 사용했다.

### 3-2. BrokerFactory와의 연동

```python
# app/broker/factory.py

async def get_broker(mode=None) -> AbstractBroker:
    if mode is None:
        mode = settings.get_trading_mode()  # 런타임 모드 반영
    # 캐시에서 해당 모드의 브로커 반환 (없으면 생성)
```

**포인트**: BrokerFactory는 `TradingMode`별로 브로커를 캐싱한다. PAPER 브로커와 REAL 브로커가 동시에 캐시에 존재할 수 있으며, 모드 전환 시 기존 브로커를 재생성하지 않고 캐시에서 가져온다.

### 3-3. 전체 코드베이스 일괄 변경

```python
# 변경 전 (모든 라우트에서)
settings.trading_mode.value        # .env 고정값
settings.trading_mode == TradingMode.PAPER

# 변경 후
settings.get_trading_mode().value  # 런타임 값 반영
settings.get_trading_mode() == TradingMode.PAPER
```

영향 범위:
- `routes.py`: 8곳 (모든 trading_mode 참조)
- `factory.py`: 1곳 (기본 모드 결정)
- `api/system.py`: 1곳 (health 엔드포인트)

---

## 4. Java 비교 분석

### 4-1. 런타임 설정 변경 패턴

| 관점 | Python (현재) | Java (Spring) |
|:---|:---|:---|
| 설정 클래스 | Pydantic `BaseSettings` (frozen) | `@ConfigurationProperties` |
| 런타임 변경 | 전역 변수 오버라이드 | `@RefreshScope` + Spring Cloud Config |
| 모드 전환 | `switch_trading_mode()` 직접 호출 | `ApplicationEventPublisher` → 이벤트 기반 |
| 설정 소스 | `.env` 파일 | `application.yml` + Config Server |

### 4-2. 확인 다이얼로그 구현

| 관점 | 현재 (Vanilla JS) | React/Vue |
|:---|:---|:---|
| 다이얼로그 | DOM 직접 조작 (`style.display`) | `useState` + 조건부 렌더링 |
| 확인 액션 | `fetch()` → `window.location.reload()` | 상태 업데이트 → 리렌더 |
| 페이지 갱신 | 전체 리로드 (`HX-Refresh`) | 컴포넌트 단위 리렌더 |

---

## 5. 주요 설계 결정 및 트레이드오프

1. **전역 변수 vs Settings 직접 변경**: Pydantic의 immutable 보장을 깨지 않기 위해 전역 변수를 사용했다. `model_config`에 `frozen=False`를 추가하는 방법도 있지만, Pydantic의 validation 보장이 깨질 수 있어 안전한 방식을 선택했다.

2. **서버 재시작 불필요, 하지만 영속적이지 않음**: 런타임 전환은 서버 메모리에만 유지된다. 서버 재시작 시 `.env`의 기본값으로 복귀한다. 이는 의도된 설계 — 실매매 모드가 실수로 영구 설정되는 것을 방지한다.

3. **안전장치 계층화**:
   - 1차: KIS API 키 미설정 시 전환 버튼 비활성화
   - 2차: 전환 시 확인 다이얼로그로 의도 확인
   - 3차: 서버 측에서 API 키 유효성 재검증 후 거부 가능

4. **전체 페이지 리로드**: 모드 전환 시 `HX-Refresh: true`로 전체 리로드한다. HTMX의 부분 업데이트로도 가능하지만, nav 뱃지 + 포트폴리오 + 주문 내역 등 거의 모든 요소가 바뀌므로 전체 리로드가 더 단순하고 안전하다.
