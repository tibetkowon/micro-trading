# Phase 9: 실계좌 연동 관리 - 학습 문서

## 1. Overview: 왜 설정 페이지가 필요한가?

KIS(한국투자증권) 브로커가 이미 구현되어 있었지만(KISBroker, PaperBroker), 사용자가 연결 상태를 확인하거나 관리할 UI가 없었다. API 키가 잘 설정되었는지, 실계좌에 접근 가능한지 알 수 없는 상태였다.

개선 후:
- **설정 페이지** (`/settings`): 거래 모드, KIS API 연결 상태, 계좌 잔고를 한 곳에서 확인
- **연결 테스트**: 버튼 클릭으로 KIS API 토큰 발급 및 잔고 조회 가능 여부 확인
- **API 키 마스킹**: 보안을 위해 앞 4자리만 표시

---

## 2. Logic Flow: 데이터 흐름 단계별 설명

### 2-1. 설정 페이지 로딩

```
GET /settings
  │
  ├─ ConnectionService.get_connection_status()
  │    ├─ settings.kis_app_key → 존재 여부 확인
  │    ├─ settings.kis_app_secret → 존재 여부 확인
  │    ├─ settings.kis_account_number → 존재 여부 확인
  │    └─ settings.kis_base_url → "vts" 포함 여부 → 모의서버 판별
  │
  ├─ ConnectionService.mask_key() → 키 마스킹 (앞 4자 + ****)
  │
  └─ settings.html 렌더링 (3개 섹션)
       ├─ 섹션 1: 거래 모드 (현재 모드 + 전환 버튼)
       ├─ 섹션 2: KIS API 연결 (상태 + 테스트 버튼)
       └─ 섹션 3: 계좌 정보 (HTMX로 lazy load)
```

### 2-2. KIS 연결 테스트

```
POST /settings/test-connection (HTMX)
  │
  ├─ ConnectionService.test_connection()
  │    ├─ API 키 3종 모두 설정? → No → "API 키가 설정되지 않았습니다"
  │    │
  │    └─ Yes → BrokerFactory.get_broker(REAL)
  │         ├─ KISBroker 생성 (또는 캐시에서)
  │         ├─ broker.get_balance() 호출
  │         │    ├─ KISClient.get() → KIS OpenAPI 잔고 조회
  │         │    └─ BalanceInfo(cash_krw=..., total_value_krw=...)
  │         │
  │         ├─ 성공 → "연결 성공" 표시
  │         └─ 실패 → 에러 메시지 표시
  │
  └─ #connection-result에 결과 HTML 삽입
```

### 2-3. 계좌 정보 조회

```
GET /partials/account-info (HTMX, 페이지 로드 시 자동)
  │
  ├─ ConnectionService.get_paper_balance()
  │    └─ PaperBroker.get_balance() → DB Account 테이블 조회
  │
  ├─ ConnectionService.get_real_balance()
  │    ├─ KIS 미설정 → {"error": "API 키가 설정되지 않았습니다"}
  │    └─ KIS 설정됨 → KISBroker.get_balance() → KIS API 호출
  │
  └─ account_info.html partial 렌더링
       ├─ 모의투자 잔고 (KRW/USD)
       ├─ 실계좌 잔고 (예수금/총평가금액)
       └─ 수수료율
```

---

## 3. 핵심 코드 상세 분석

### 3-1. ConnectionService — 연결 관리 서비스

```python
# app/services/connection_service.py

class ConnectionService:
    def get_connection_status(self) -> ConnectionStatus:
        """API 키 설정 여부 및 서버 정보를 dataclass로 반환."""
        return ConnectionStatus(
            has_app_key=bool(settings.kis_app_key),
            # ...
        )

    async def test_connection(self) -> ConnectionStatus:
        """실제 KIS API 호출로 연결 가능 여부 확인."""
        # BrokerFactory를 통해 KISBroker 획득 → get_balance() 시도
        broker = await get_broker(TradingMode.REAL)
        balance = await broker.get_balance()

    @staticmethod
    def mask_key(key: str) -> str:
        """'PSbq****...' 형태로 키 마스킹."""
        return key[:4] + "*" * (len(key) - 4)
```

**포인트**: BrokerFactory의 캐싱 메커니즘 덕분에 연결 테스트 시 KISBroker가 이미 초기화되어 있으면 재사용된다.

### 3-2. 설정 페이지 라우트

```python
@web_router.get("/settings")
async def settings_page(request: Request):
    conn_svc = ConnectionService()
    status = conn_svc.get_connection_status()
    # 마스킹된 키와 함께 템플릿 렌더링
```

**포인트**: 계좌 정보는 HTMX `hx-trigger="load"`로 lazy load한다. 페이지 자체는 빠르게 로딩되고, API 호출이 필요한 잔고 조회는 비동기로 후속 로딩된다.

---

## 4. Java 비교 분석

### 4-1. 서비스 패턴 비교

| 관점 | Python (현재) | Java (Spring) |
|:---|:---|:---|
| 서비스 클래스 | 일반 클래스, 필요 시 인스턴스화 | `@Service` 빈, DI 컨테이너 관리 |
| 설정 읽기 | `settings` 싱글턴 직접 참조 | `@Value("${kis.app-key}")` 주입 |
| 데이터 클래스 | `@dataclass` | `record` 또는 Lombok `@Data` |
| 비동기 HTTP | `httpx.AsyncClient` | `WebClient` (Spring WebFlux) |
| 키 마스킹 | `str[:4] + "*" * n` | `StringUtils.overlay(key, "****", 4, len)` |

### 4-2. HTMX vs REST API + SPA

| 관점 | HTMX (현재) | React/Vue SPA |
|:---|:---|:---|
| 연결 테스트 | `hx-post` → 서버에서 HTML 반환 | `fetch()` → JSON → 프론트에서 렌더 |
| 계좌 정보 | `hx-get` + `hx-trigger="load"` | `useEffect(() => fetch(...))` |
| 페이지 전환 | 서버 라우팅 (`GET /settings`) | 클라이언트 라우팅 (`react-router`) |
| 장점 | 서버 사이드 렌더링, JS 최소화 | 풍부한 인터랙션, 상태 관리 |

---

## 5. 주요 설계 결정 및 트레이드오프

1. **`.env` 기반 설정**: API 키는 런타임에 변경할 수 없고 `.env` 파일에서만 설정한다. 보안상 웹 UI에서 키를 입력받지 않는 것이 안전하다.

2. **Lazy Loading**: 계좌 정보를 `hx-trigger="load"`로 비동기 로딩하여 설정 페이지 초기 렌더링 속도를 높였다. KIS API 호출 실패 시에도 페이지 자체는 정상 표시된다.

3. **BrokerFactory 캐싱 활용**: ConnectionService가 직접 KISClient를 생성하지 않고, BrokerFactory를 통해 기존 브로커 인스턴스를 재사용한다. 코드 중복을 방지하고 연결 관리를 한 곳에서 한다.
