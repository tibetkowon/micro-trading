# KIS OpenAPI Integration Skill

## 필수 헤더 규칙

### 인증 요청 헤더 (POST /uapi/*)
KIS OpenAPI의 모든 인증 POST 요청에는 아래 헤더가 **반드시** 포함되어야 한다.
누락 시 KIS 서버가 HTTP 500을 반환한다.

```python
{
    "content-type": "application/json; charset=utf-8",
    "authorization": f"Bearer {access_token}",
    "appkey": app_key,
    "appsecret": app_secret,
    "tr_id": tr_id,
    "custtype": "P",   # ← 필수: 개인투자자(P) / 법인투자자(B)
    "hashkey": hash,   # ← POST 요청 시 필수 (해시키 발급 후 첨부)
}
```

### 해시키 발급 헤더 (POST /uapi/hashkey)
```python
{
    "content-type": "application/json; charset=utf-8",  # charset 포함 필수
    "appKey": app_key,
    "appSecret": app_secret,
}
```
- `custtype` 불필요
- `content-type`에 반드시 `charset=utf-8` 포함

## Transaction ID 매핑

| 구분 | 실전 | 모의투자 |
|:---|:---|:---|
| 국내 매수 | `TTTC0802U` | `VTTC0802U` |
| 국내 매도 | `TTTC0801U` | `VTTC0801U` |
| 국내 취소 | `TTTC0803U` | `VTTC0803U` |
| 국내 잔고 | `TTTC8434R` | `VTTC8434R` |
| 국내 현재가 | `FHKST01010100` | (동일) |
| 국내 일별가격 | `FHKST01010400` | (동일) |

- `is_mock` 판별: `"vts" in kis_base_url.lower()`
- 실전 URL: `https://openapi.koreainvestment.com:9443`
- 모의 URL: `https://openapivts.koreainvestment.com:9443`

## 에러 처리 패턴

KIS API는 에러 시 HTTP 4xx/5xx와 함께 JSON 바디에 실제 에러코드를 반환한다.
`raise_for_status()` 호출 전에 반드시 응답 바디를 로그에 남겨야 한다.

```python
resp = await self._client.post(path, headers=headers, json=body)
if not resp.is_success:
    logger.error("KIS POST 오류 [%s] %s (tr_id=%s): %s",
                 resp.status_code, path, tr_id, resp.text)
resp.raise_for_status()
```

- `resp.text`에 KIS 에러코드(`msg_cd`)와 메시지(`msg1`)가 담겨 있음
- 로깅 없이 `raise_for_status()`만 호출하면 원인 파악 불가

## 토큰 세션 관리

### 토큰 수명 및 갱신 전략

KIS 토큰은 발급 후 **23시간** 유효하다. 세션 끊김을 방지하기 위해 3단계 방어 전략을 사용한다.

| 단계 | 방법 | 파일 |
|:---|:---|:---|
| 1. Lazy 갱신 | API 호출 전 `_ensure_token()` — 만료 시 자동 갱신 | `client.py` |
| 2. 401 자동 재시도 | 401 응답 시 강제 갱신 후 1회 재시도 | `client.py` |
| 3. 선제 갱신 | 30분 간격 스케줄러 — 만료 1시간 이내이면 미리 갱신 | `jobs.py` |

### `KISToken` 속성

```python
@property
def is_expired(self) -> bool:
    return datetime.now() >= self.expires_at

@property
def is_expiring_soon(self) -> bool:
    """만료 1시간 이내인 경우 True — 선제 갱신 판단에 사용."""
    return datetime.now() >= self.expires_at - timedelta(hours=1)
```

### `force_refresh_token()` — 강제 갱신

```python
async def force_refresh_token(self) -> None:
    """토큰 강제 갱신 — 선제 갱신 스케줄러 및 401 재시도에 사용."""
    self._token = KISToken()  # 만료 상태로 초기화
    await self._ensure_token()
```

### 401 자동 재시도 패턴

```python
resp = await self._client.get(path, headers=headers, params=params)
if resp.status_code == 401:
    logger.warning("KIS 401 응답 — 토큰 강제 갱신 후 재시도: %s", path)
    await self.force_refresh_token()
    headers = self._base_headers(tr_id)
    resp = await self._client.get(path, headers=headers, params=params)
```

- GET, POST 모두 동일 패턴 적용
- 재시도는 **1회만** 수행 (무한루프 방지)

### 5xx 서버 오류 재시도 패턴 (Phase 17)

KIS API 일시적 서버 장애(500/503/504) 시 최대 2회 재시도한다.

```python
# KISClient.post() 내부 — 5xx 오류 재시도
if resp.status_code >= 500 and _retry > 0:
    logger.warning(
        "KIS POST 5xx 오류 [%s] — %d회 재시도 예정: %s",
        resp.status_code, _retry, path,
    )
    await asyncio.sleep(1)
    return await self.post(path, tr_id, body, use_hashkey=use_hashkey, _retry=_retry - 1)
```

- `_retry=2` (기본값): 최대 2회, 1초 딜레이
- **401은 별도 처리** (토큰 갱신 후 즉시 재시도), **5xx만 backoff 적용**
- 잔고 부족·검증 오류(4xx/비즈니스 에러)는 재시도 없음 — 즉시 실패 반환

### 스케줄러 선제 갱신 (`jobs.py`)

```python
async def refresh_kis_token():
    from app.broker.factory import _broker_cache
    from app.schemas.common import TradingMode
    broker = _broker_cache.get(TradingMode.REAL)
    if broker is None:
        return  # 실매매 브로커 미초기화 시 스킵
    client = getattr(broker, "_client", None)
    if client and (client._token.is_expired or client._token.is_expiring_soon):
        await client.force_refresh_token()
```

- `_broker_cache`에 캐싱된 브로커만 체크 (새 연결 생성 없음)
- 30분 간격 실행, KIS 키 미설정 시 자동 스킵

## 주문 바디 필드

```python
{
    "CANO": account_number,       # 계좌번호 앞 8자리
    "ACNT_PRDT_CD": product_code, # 계좌상품코드 (기본값: "01")
    "PDNO": symbol,               # 종목코드 (예: "005930", "360750")
    "ORD_DVSN": "01",             # 주문유형: "01"=시장가, "00"=지정가
    "ORD_QTY": str(quantity),     # 주문수량 (문자열)
    "ORD_UNPR": "0",              # 주문단가: 시장가=0, 지정가=정수 문자열
}
```

- `ORD_QTY`, `ORD_UNPR`은 반드시 **문자열** 타입으로 전송
- 시장가 주문 시 `ORD_UNPR`은 `"0"` 고정
