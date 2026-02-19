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
