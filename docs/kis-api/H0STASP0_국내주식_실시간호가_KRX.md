# 국내주식 실시간호가 (KRX)

| 항목 | 값 |
|------|---|
| 통신방식 | `WEBSOCKET` |
| API 명 | `국내주식 실시간호가 (KRX)` |
| API ID | `실시간-004` |
| 실전 TR_ID | `H0STASP0` |
| 모의 TR_ID | `H0STASP0` |
| Method | `POST` |
| URL | `/tryitout/H0STASP0` |

- 실전: `ws://ops.koreainvestment.com:21000`
- 모의: `ws://ops.koreainvestment.com:31000`

> [참고자료]
실시간시세(웹소켓) 파이썬 샘플코드는 한국투자증권 Github 참고 부탁드립니다.
https://github.com/koreainvestment/open-trading-api/blob/main/websocket/python/ws_domestic_overseas_all.py

실시간시세(웹소켓) API 사용방법에 대한 자세한 설명은 한국투자증권 Wikidocs 참고 부탁드립니다.
https://wikidocs.net/book/7847 (국내주식 업데이트 완료, 추후 해외주식·국내선물옵션 업데이트 예정)

[호출 데이터]
헤더와 바디 값을 합쳐 JSON 형태로 전송합니다.

[응답 데이터]
1. 정상 등록 여부 (JSON)
- JSON["body"]["msg1"] - 정상 응답 시, SUBSCRIBE SUCCESS
- JSON["body"]["output"]["iv"] - 실시간 결과 복호화에 필요한 AES256 IV (Initialize Vector)
- JSON["body"]["output"]["key"] - 실시간 결과 복호화에 필요한 AES256 Key

2. 실시간 결과 응답 ( | 로 구분되는 값)
- 암호화 유무 : 0 암호화 되지 않은 데이터 / 1 암호화된 데이터
- TR_ID : 등록한 tr_id
- 데이터 건수 : (ex. 001 데이터 건수를 참조하여 활용)
- 응답 데이터 : 아래 response 데이터 참조 ( ^로 구분됨)

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_type` | 거래타입 | string | Y | 1 : 등록; 2 : 해제 |

## Request Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_id` | 거래ID | string | Y | [실전/모의투자]; H0STASP0 : 주식호가 |
| `tr_key` | 구분값 | string | Y | 종목번호 (6자리); ETN의 경우, Q로 시작 (EX. Q500001) |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `MKSC_SHRN_ISCD` | 유가증권 단축 종목코드 | string | Y |  |
| `BSOP_HOUR` | 영업 시간 | string | Y |  |
| `HOUR_CLS_CODE` | 시간 구분 코드 | string | Y | 0 : 장중; A : 장후예상; B : 장전예상; C : 9시이후의 예상가, VI발동; D : 시간외 단일가 예상 |
| `ASKP1` | 매도호가1 | number | Y |  |
| `ASKP2` | 매도호가2 | number | Y |  |
| `ASKP3` | 매도호가3 | number | Y |  |
| `ASKP4` | 매도호가4 | number | Y |  |
| `ASKP5` | 매도호가5 | number | Y |  |
| `ASKP6` | 매도호가6 | number | Y |  |
| `ASKP7` | 매도호가7 | number | Y |  |
| `ASKP8` | 매도호가8 | number | Y |  |
| `ASKP9` | 매도호가9 | number | Y |  |
| `ASKP10` | 매도호가10 | number | Y |  |
| `BIDP1` | 매수호가1 | number | Y |  |
| `BIDP2` | 매수호가2 | number | Y |  |
| `BIDP3` | 매수호가3 | number | Y |  |
| `BIDP4` | 매수호가4 | number | Y |  |
| `BIDP5` | 매수호가5 | number | Y |  |
| `BIDP6` | 매수호가6 | number | Y |  |
| `BIDP7` | 매수호가7 | number | Y |  |
| `BIDP8` | 매수호가8 | number | Y |  |
| `BIDP9` | 매수호가9 | number | Y |  |
| `BIDP10` | 매수호가10 | number | Y |  |
| `ASKP_RSQN1` | 매도호가 잔량1 | number | Y |  |
| `ASKP_RSQN2` | 매도호가 잔량2 | number | Y |  |
| `ASKP_RSQN3` | 매도호가 잔량3 | number | Y |  |
| `ASKP_RSQN4` | 매도호가 잔량4 | number | Y |  |
| `ASKP_RSQN5` | 매도호가 잔량5 | number | Y |  |
| `ASKP_RSQN6` | 매도호가 잔량6 | number | Y |  |
| `ASKP_RSQN7` | 매도호가 잔량7 | number | Y |  |
| `ASKP_RSQN8` | 매도호가 잔량8 | number | Y |  |
| `ASKP_RSQN9` | 매도호가 잔량9 | number | Y |  |
| `ASKP_RSQN10` | 매도호가 잔량10 | number | Y |  |
| `BIDP_RSQN1` | 매수호가 잔량1 | number | Y |  |
| `BIDP_RSQN2` | 매수호가 잔량2 | number | Y |  |
| `BIDP_RSQN3` | 매수호가 잔량3 | number | Y |  |
| `BIDP_RSQN4` | 매수호가 잔량4 | number | Y |  |
| `BIDP_RSQN5` | 매수호가 잔량5 | number | Y |  |
| `BIDP_RSQN6` | 매수호가 잔량6 | number | Y |  |
| `BIDP_RSQN7` | 매수호가 잔량7 | number | Y |  |
| `BIDP_RSQN8` | 매수호가 잔량8 | number | Y |  |
| `BIDP_RSQN9` | 매수호가 잔량9 | number | Y |  |
| `BIDP_RSQN10` | 매수호가 잔량10 | number | Y |  |
| `TOTAL_ASKP_RSQN` | 총 매도호가 잔량 | number | Y |  |
| `TOTAL_BIDP_RSQN` | 총 매수호가 잔량 | number | Y |  |
| `OVTM_TOTAL_ASKP_RSQN` | 시간외 총 매도호가 잔량 | number | Y |  |
| `OVTM_TOTAL_BIDP_RSQN` | 시간외 총 매수호가 잔량 | number | Y |  |
| `ANTC_CNPR` | 예상 체결가 | number | Y | 동시호가 등 특정 조건하에서만 발생 |
| `ANTC_CNQN` | 예상 체결량 | number | Y | 동시호가 등 특정 조건하에서만 발생 |
| `ANTC_VOL` | 예상 거래량 | number | Y | 동시호가 등 특정 조건하에서만 발생 |
| `ANTC_CNTG_VRSS` | 예상 체결 대비 | number | Y | 동시호가 등 특정 조건하에서만 발생 |
| `ANTC_CNTG_VRSS_SIGN` | 예상 체결 대비 부호 | string | Y | 동시호가 등 특정 조건하에서만 발생 → [공통코드](_공통코드.md#ANTC_CNTG_VRSS_SIGN) 참조 |
| `ANTC_CNTG_PRDY_CTRT` | 예상 체결 전일 대비율 | number | Y |  |
| `ACML_VOL` | 누적 거래량 | number | Y |  |
| `TOTAL_ASKP_RSQN_ICDC` | 총 매도호가 잔량 증감 | number | Y |  |
| `TOTAL_BIDP_RSQN_ICDC` | 총 매수호가 잔량 증감 | number | Y |  |
| `OVTM_TOTAL_ASKP_ICDC` | 시간외 총 매도호가 증감 | number | Y |  |
| `OVTM_TOTAL_BIDP_ICDC` | 시간외 총 매수호가 증감 | number | Y |  |
| `STCK_DEAL_CLS_CODE` | 주식 매매 구분 코드 | string | Y | 사용 X (삭제된 값) |

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
                           "tr_id":"H0STASP0",
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
        "tr_id": "H0STASP0", 
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
005930^093730^0^71900^72000^72100^72200^72300^72400^72500^72600^72700^72800^71
800^71700^71600^71500^71400^71300^71200^71100^71000^70900^91918^117942^92673^7
9708^106729^141988^176192^113906^134077^104229^95221^159371^220746^284657^2127
42^195370^182710^209747^376432^158171^1159362^2095167^0^0^0^0^525579^-72000^5^
-100.00^3159115^0^8^0^0^0
```
