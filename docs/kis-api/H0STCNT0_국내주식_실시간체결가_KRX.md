# 국내주식 실시간체결가 (KRX)

| 항목 | 값 |
|------|---|
| 통신방식 | `WEBSOCKET` |
| API 명 | `국내주식 실시간체결가 (KRX)` |
| API ID | `실시간-003` |
| 실전 TR_ID | `H0STCNT0` |
| 모의 TR_ID | `H0STCNT0` |
| Method | `POST` |
| URL | `/tryitout/H0STCNT0` |

- 실전: `ws://ops.koreainvestment.com:21000`
- 모의: `ws://ops.koreainvestment.com:31000`

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

※ 데이터가 많은 경우 여러 건을 페이징 처리해서 데이터를 보내는 점 참고 부탁드립니다.
ex) 0|H0STCNT0|004|... 인 경우 004가 데이터 개수를 의미하여, 뒤에 체결데이터가 4건 들어옴
→ 0|H0STCNT0|004|005930^123929...(체결데이터1)...^005930^123929...(체결데이터2)...^005930^123929...(체결데이터3)...^005930^123929...(체결데이터4)...

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_type` | 거래타입 | string | Y | 1 : 등록; 2 : 해제 |

## Request Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_id` | 거래ID | string | Y | [실전/모의투자]; H0STCNT0 : 실시간 주식 체결가 |
| `tr_key` | 구분값 | string | Y | 종목번호 (6자리); ETN의 경우, Q로 시작 (EX. Q500001) |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `MKSC_SHRN_ISCD` | 유가증권 단축 종목코드 | string | Y |  |
| `STCK_CNTG_HOUR` | 주식 체결 시간 | string | Y |  |
| `STCK_PRPR` | 주식 현재가 | number | Y | 체결가격 |
| `PRDY_VRSS_SIGN` | 전일 대비 부호 | string | Y | 1 : 상한; 2 : 상승; 3 : 보합; 4 : 하한; 5 : 하락 |
| `PRDY_VRSS` | 전일 대비 | number | Y |  |
| `PRDY_CTRT` | 전일 대비율 | number | Y |  |
| `WGHN_AVRG_STCK_PRC` | 가중 평균 주식 가격 | number | Y |  |
| `STCK_OPRC` | 주식 시가 | number | Y |  |
| `STCK_HGPR` | 주식 최고가 | number | Y |  |
| `STCK_LWPR` | 주식 최저가 | number | Y |  |
| `ASKP1` | 매도호가1 | number | Y |  |
| `BIDP1` | 매수호가1 | number | Y |  |
| `CNTG_VOL` | 체결 거래량 | number | Y |  |
| `ACML_VOL` | 누적 거래량 | number | Y |  |
| `ACML_TR_PBMN` | 누적 거래 대금 | number | Y |  |
| `SELN_CNTG_CSNU` | 매도 체결 건수 | number | Y |  |
| `SHNU_CNTG_CSNU` | 매수 체결 건수 | number | Y |  |
| `NTBY_CNTG_CSNU` | 순매수 체결 건수 | number | Y |  |
| `CTTR` | 체결강도 | number | Y |  |
| `SELN_CNTG_SMTN` | 총 매도 수량 | number | Y |  |
| `SHNU_CNTG_SMTN` | 총 매수 수량 | number | Y |  |
| `CCLD_DVSN` | 체결구분 | string | Y | 1:매수(+) ; 3:장전 ; 5:매도(-) |
| `SHNU_RATE` | 매수비율 | number | Y |  |
| `PRDY_VOL_VRSS_ACML_VOL_RATE` | 전일 거래량 대비 등락율 | number | Y |  |
| `OPRC_HOUR` | 시가 시간 | string | Y |  |
| `OPRC_VRSS_PRPR_SIGN` | 시가대비구분 | string | Y | 1 : 상한; 2 : 상승; 3 : 보합; 4 : 하한; 5 : 하락 |
| `OPRC_VRSS_PRPR` | 시가대비 | number | Y |  |
| `HGPR_HOUR` | 최고가 시간 | string | Y |  |
| `HGPR_VRSS_PRPR_SIGN` | 고가대비구분 | string | Y | 1 : 상한; 2 : 상승; 3 : 보합; 4 : 하한; 5 : 하락 |
| `HGPR_VRSS_PRPR` | 고가대비 | number | Y |  |
| `LWPR_HOUR` | 최저가 시간 | string | Y |  |
| `LWPR_VRSS_PRPR_SIGN` | 저가대비구분 | string | Y | 1 : 상한; 2 : 상승; 3 : 보합; 4 : 하한; 5 : 하락 |
| `LWPR_VRSS_PRPR` | 저가대비 | number | Y |  |
| `BSOP_DATE` | 영업 일자 | string | Y |  |
| `NEW_MKOP_CLS_CODE` | 신 장운영 구분 코드 | string | Y | (1) 첫 번째 비트 → [공통코드](_공통코드.md#NEW_MKOP_CLS_CODE) 참조 |
| `TRHT_YN` | 거래정지 여부 | string | Y | Y : 정지; N : 정상거래 |
| `ASKP_RSQN1` | 매도호가 잔량1 | number | Y |  |
| `BIDP_RSQN1` | 매수호가 잔량1 | number | Y |  |
| `TOTAL_ASKP_RSQN` | 총 매도호가 잔량 | number | Y |  |
| `TOTAL_BIDP_RSQN` | 총 매수호가 잔량 | number | Y |  |
| `VOL_TNRT` | 거래량 회전율 | number | Y |  |
| `PRDY_SMNS_HOUR_ACML_VOL` | 전일 동시간 누적 거래량 | number | Y |  |
| `PRDY_SMNS_HOUR_ACML_VOL_RATE` | 전일 동시간 누적 거래량 비율 | number | Y |  |
| `HOUR_CLS_CODE` | 시간 구분 코드 | string | Y | 0 : 장중; A : 장후예상; B : 장전예상; C : 9시이후의 예상가, VI발동; D : 시간외 단일가 예상 |
| `MRKT_TRTM_CLS_CODE` | 임의종료구분코드 | string | Y |  |
| `VI_STND_PRC` | 정적VI발동기준가 | number | Y |  |

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
                           "tr_id":"H0STCNT0",
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
        "tr_id": "H0STCNT0", 
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
005930^093354^71900^5^-100^-0.14^72023.83^72100^72400^71700^71900^71800^1^3052
507^219853241700^5105^6937^1832^84.90^1366314^1159996^1^0.39^20.28^090020^5^-2
00^090820^5^-500^092619^2^200^20230612^20^N^65945^216924^1118750^2199206^0.05^
2424142^125.92^0^^72100
```
