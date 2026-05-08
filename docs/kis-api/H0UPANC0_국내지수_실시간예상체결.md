# 국내지수 실시간예상체결

| 항목 | 값 |
|------|---|
| 통신방식 | `WEBSOCKET` |
| API 명 | `국내지수 실시간예상체결` |
| API ID | `실시간-027` |
| 실전 TR_ID | `H0UPANC0` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `POST` |
| URL | `/tryitout/H0UPANC0` |

- 실전: `ws://ops.koreainvestment.com:21000`
- 모의: `모의투자 미지원`

> [참고자료]
실시간시세(웹소켓) 파이썬 샘플코드는 한국투자증권 Github 참고 부탁드립니다.
https://github.com/koreainvestment/open-trading-api/blob/main/websocket/python/ws_domestic_overseas_all.py

실시간시세(웹소켓) API 사용방법에 대한 자세한 설명은 한국투자증권 Wikidocs 참고 부탁드립니다.
https://wikidocs.net/book/7847 (국내주식 업데이트 완료, 추후 해외주식·국내선물옵션 업데이트 예정)

종목코드 마스터파일 파이썬 정제코드는 한국투자증권 Github 참고 부탁드립니다.
https://github.com/koreainvestment/open-trading-api/tree/main/stocks_info

[호출 데이터]
헤더와 바디 값을 합쳐 JSON 형태로 전송합니다.

[응답 데이터]
1. 정상 등록 여부 (JSON)
- JSON["body"]["msg1"] - 정상 응답 시, SUBSCRIBE SUCCESS
- JSON["body"]["output"]["iv"] - 실시간 결과 복호화에 필요한 AES256 IV (Initialize Vector)
- JSON["body"]["output"]["key"] - 실시간 결과 복호화에 필요한 AES256 Key

2. 실시간 결과 응답 ( | 로 구분되는 값)
ex) 0|H0STCNT0|004|005930^123929^73100^5^...
- 암호화 유무 : 0 암호화 되지 않은 데이터 / 1 암호화된 데이터
- TR_ID : 등록한 tr_id (ex. H0STCNT0)
- 데이터 건수 : (ex. 001 인 경우 데이터 건수 1건, 004인 경우 데이터 건수 4건)
- 응답 데이터 : 아래 response 데이터 참조 ( ^로 구분됨)

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_type` | 등록/해제 | string | Y | "1: 등록, 2:해제" |

## Request Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_id` | 거래ID | string | Y | H0UPANC0 |
| `tr_key` | 종목코드 | string | Y | 업종구분코드 |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `BSTP_CLS_CODE` | 업종 구분 코드 | object | Y | '각 항목사이에는 구분자로 ^ 사용,; 모든 데이터타입은 String으로 변환되어 push 처리됨' |
| `BSOP_HOUR` | 영업 시간 | string | Y |  |
| `PRPR_NMIX` | 현재가 지수 | string | Y |  |
| `PRDY_VRSS_SIGN` | 전일 대비 부호 | string | Y |  |
| `BSTP_NMIX_PRDY_VRSS` | 업종 지수 전일 대비 | string | Y |  |
| `ACML_VOL` | 누적 거래량 | string | Y |  |
| `ACML_TR_PBMN` | 누적 거래 대금 | string | Y |  |
| `PCAS_VOL` | 건별 거래량 | string | Y |  |
| `PCAS_TR_PBMN` | 건별 거래 대금 | string | Y |  |
| `PRDY_CTRT` | 전일 대비율 | string | Y |  |
| `OPRC_NMIX` | 시가 지수 | string | Y |  |
| `NMIX_HGPR` | 지수 최고가 | string | Y |  |
| `NMIX_LWPR` | 지수 최저가 | string | Y |  |
| `OPRC_VRSS_NMIX_PRPR` | 시가 대비 지수 현재가 | string | Y |  |
| `OPRC_VRSS_NMIX_SIGN` | 시가 대비 지수 부호 | string | Y |  |
| `HGPR_VRSS_NMIX_PRPR` | 최고가 대비 지수 현재가 | string | Y |  |
| `HGPR_VRSS_NMIX_SIGN` | 최고가 대비 지수 부호 | string | Y |  |
| `LWPR_VRSS_NMIX_PRPR` | 최저가 대비 지수 현재가 | string | Y |  |
| `LWPR_VRSS_NMIX_SIGN` | 최저가 대비 지수 부호 | string | Y |  |
| `PRDY_CLPR_VRSS_OPRC_RATE` | 전일 종가 대비 시가2 비율 | string | Y |  |
| `PRDY_CLPR_VRSS_HGPR_RATE` | 전일 종가 대비 최고가 비율 | string | Y |  |
| `PRDY_CLPR_VRSS_LWPR_RATE` | 전일 종가 대비 최저가 비율 | string | Y |  |
| `UPLM_ISSU_CNT` | 상한 종목 수 | string | Y |  |
| `ASCN_ISSU_CNT` | 상승 종목 수 | string | Y |  |
| `STNR_ISSU_CNT` | 보합 종목 수 | string | Y |  |
| `DOWN_ISSU_CNT` | 하락 종목 수 | string | Y |  |
| `LSLM_ISSU_CNT` | 하한 종목 수 | string | Y |  |
| `QTQT_ASCN_ISSU_CNT` | 기세 상승 종목수 | string | Y |  |
| `QTQT_DOWN_ISSU_CNT` | 기세 하락 종목수 | string | Y |  |
| `TICK_VRSS` | TICK대비 | string | Y |  |

## Example

### Request Example (Python)
```json
{
    "header": {
        "approval_key": "35xxxxxa-bxxa-4xxb-87xxx-f56xxxxxxxxxx",
        "custtype": "P",
        "tr_type": "1",
        "content-type": "utf-8"
    },
    "body": {
        "input": {
            "tr_id": "H0UPANC0",
            "tr_key": "0001"
        }
    }
}
```

### Response Example
```json
# 연결 확인
{
    "header": {
        "tr_id": "H0UPANC0", 
        "tr_key": "0001", 
        "encrypt": "N"
        }, 
    "body": {
        "rt_cd": "0", 
        "msg_cd": "OPSP0000",
        "msg1": "SUBSCRIBE SUCCESS", 
        "output": {
            "iv": "0123456789abcdef", 
            "key": "abcdefghijklmnopabcdefghijklmnop"}
        }
}

# output
0|H0UPANC0|001|0001^085910^2607.71^2^15.85^5424^192338^5424^192338^0.61^0^43
9^201^251^201
```
