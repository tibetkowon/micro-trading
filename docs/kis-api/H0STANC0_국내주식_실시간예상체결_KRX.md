# 국내주식 실시간예상체결 (KRX)

| 항목 | 값 |
|------|---|
| 통신방식 | `WEBSOCKET` |
| API 명 | `국내주식 실시간예상체결 (KRX)` |
| API ID | `실시간-041` |
| 실전 TR_ID | `H0STANC0` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `POST` |
| URL | `/tryitout/H0STANC0` |

- 실전: `ws://ops.koreainvestment.com:21000`
- 모의: `모의투자 미지원`

> 국내주식 실시간예상체결 API입니다.

[참고자료]
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
| `tr_type` | 등록/해제 | string | Y | 1: 등록, 2:해제 |

## Request Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_id` | 거래ID | string | Y | H0STANC0 |
| `tr_key` | 구분값 | string | Y | 종목코드 (ex 005930 삼성전자) |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `MKSC_SHRN_ISCD` | 유가증권단축종목코드 | string | Y |  |
| `STCK_CNTG_HOUR` | 주식체결시간 | string | Y |  |
| `STCK_PRPR` | 주식현재가 | string | Y |  |
| `PRDY_VRSS_SIGN` | 전일대비구분 | string | Y |  |
| `PRDY_VRSS` | 전일대비 | string | Y |  |
| `PRDY_CTRT` | 등락율 | string | Y |  |
| `WGHN_AVRG_STCK_PRC` | 가중평균주식가격 | string | Y |  |
| `STCK_OPRC` | 시가 | string | Y |  |
| `STCK_HGPR` | 고가 | string | Y |  |
| `STCK_LWPR` | 저가 | string | Y |  |
| `ASKP1` | 매도호가 | string | Y |  |
| `BIDP1` | 매수호가 | string | Y |  |
| `CNTG_VOL` | 거래량 | string | Y |  |
| `ACML_VOL` | 누적거래량 | string | Y |  |
| `ACML_TR_PBMN` | 누적거래대금 | string | Y |  |
| `SELN_CNTG_CSNU` | 매도체결건수 | string | Y |  |
| `SHNU_CNTG_CSNU` | 매수체결건수 | string | Y |  |
| `NTBY_CNTG_CSNU` | 순매수체결건수 | string | Y |  |
| `CTTR` | 체결강도 | string | Y |  |
| `SELN_CNTG_SMTN` | 총매도수량 | string | Y |  |
| `SHNU_CNTG_SMTN` | 총매수수량 | string | Y |  |
| `CNTG_CLS_CODE` | 체결구분 | string | Y |  |
| `SHNU_RATE` | 매수비율 | string | Y |  |
| `PRDY_VOL_VRSS_ACML_VOL_RATE` | 전일거래량대비등락율 | string | Y |  |
| `OPRC_HOUR` | 시가시간 | string | Y |  |
| `OPRC_VRSS_PRPR_SIGN` | 시가대비구분 | string | Y |  |
| `OPRC_VRSS_PRPR` | 시가대비 | string | Y |  |
| `HGPR_HOUR` | 최고가시간 | string | Y |  |
| `HGPR_VRSS_PRPR_SIGN` | 고가대비구분 | string | Y |  |
| `HGPR_VRSS_PRPR` | 고가대비 | string | Y |  |
| `LWPR_HOUR` | 최저가시간 | string | Y |  |
| `LWPR_VRSS_PRPR_SIGN` | 저가대비구분 | string | Y |  |
| `LWPR_VRSS_PRPR` | 저가대비 | string | Y |  |
| `BSOP_DATE` | 영업일자 | string | Y |  |
| `NEW_MKOP_CLS_CODE` | 신장운영구분코드 | string | Y |  |
| `TRHT_YN` | 거래정지여부 | string | Y |  |
| `ASKP_RSQN1` | 매도호가잔량1 | string | Y |  |
| `BIDP_RSQN1` | 매수호가잔량1 | string | Y |  |
| `TOTAL_ASKP_RSQN` | 총매도호가잔량 | string | Y |  |
| `TOTAL_BIDP_RSQN` | 총매수호가잔량 | string | Y |  |
| `VOL_TNRT` | 거래량회전율 | string | Y |  |
| `PRDY_SMNS_HOUR_ACML_VOL` | 전일동시간누적거래량 | string | Y |  |
| `PRDY_SMNS_HOUR_ACML_VOL_RATE` | 전일동시간누적거래량비율 | string | Y |  |
| `HOUR_CLS_CODE` | 시간구분코드 | string | Y |  |
| `MRKT_TRTM_CLS_CODE` | 임의종료구분코드 | string | Y |  |

## Example

### Request Example (Python)
```json
{
         "header":
         {
                  "approval_key": "35xxxxxa-bxxa-4xxb-87xxx-f56xxxxxxxxxx",
                  "custtype":"P",
                  "tr_type":"1",
                  "content-type":"utf-8"
         },
         "body":
         {
                  "input":
                  {
                           "tr_id":"H0STANC0",
                           "tr_key":"005930"
                  }
         }
}
```

### Response Example
```json
# 연결 확인
{
    "header": {
        "tr_id": "H0STANC0", 
        "tr_key": "005930", 
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
0|H0STANC0|001|005930^084945^77600^2^1300^1.70^0.00^0^0^0^77600^77
500^64^221986^17226113600^0^0^0^0.00^0^0^1^0.01^0.00^000000^3^0^000000^3^0^000000^3^
0^20240426^00^N^11591^2878^41034^6265^0.00^0^0.00^B^
```
