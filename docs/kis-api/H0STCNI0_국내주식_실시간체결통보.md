# 국내주식 실시간체결통보

| 항목 | 값 |
|------|---|
| 통신방식 | `WEBSOCKET` |
| API 명 | `국내주식 실시간체결통보` |
| API ID | `실시간-005` |
| 실전 TR_ID | `H0STCNI0` |
| 모의 TR_ID | `H0STCNI9` |
| Method | `POST` |
| URL | `/tryitout/H0STCNI0` |

- 실전: `ws://ops.koreainvestment.com:21000`
- 모의: `ws://ops.koreainvestment.com:31000`

> 국내주식 실시간 체결통보 수신 시에 (1) 주문·정정·취소·거부 접수 통보 와 (2) 체결 통보 가 모두 수신됩니다.
(14번째 값(CNTG_YN;체결여부)가 2이면 체결통보, 1이면 주문·정정·취소·거부 접수 통보입니다.)

※ 모의투자는 H0STCNI9 로 변경하여 사용합니다.

[참고자료]
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

체결 통보 응답 결과는 암호화되어 출력됩니다. AES256 KEY IV를 활용해 복호화하여 활용하세요. 자세한 예제는 [도구&gt;wikidocs]에 준비되어 있습니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_type` | 거래타입 | string | N | 1: 등록 2 : 해제 |

## Request Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_id` | 거래ID | string | Y | '[실전/모의투자]; H0STCNI0 : 국내주식 실시간체결통보; H0STCNI9 : 모의투자 실시간 체결통보 |
| `tr_key` | 구분값 | string | Y | HTS ID |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `CUST_ID` | 고객 ID | string | Y |  |
| `ACNT_NO` | 계좌번호 | string | Y |  |
| `ODER_NO` | 주문번호 | string | Y |  |
| `OODER_NO` | 원주문번호 | string | Y |  |
| `SELN_BYOV_CLS` | 매도매수구분 | string | Y | 01 : 매도 ; 02 : 매수 |
| `RCTF_CLS` | 정정구분 | string | Y | 0:정상 ; 1:정정 ; 2:취소 |
| `ODER_KIND` | 주문종류 | string | Y | [KRX] → [공통코드](_공통코드.md#ODER_KIND) 참조 |
| `ODER_COND` | 주문조건 | string | Y | 0:없음; 1:IOC ; 2:FOK |
| `STCK_SHRN_ISCD` | 주식 단축 종목코드 | string | Y |  |
| `CNTG_QTY` | 체결 수량 | string | Y |  |
| `CNTG_UNPR` | 체결단가 | string | Y |  |
| `STCK_CNTG_HOUR` | 주식 체결 시간 | string | Y |  |
| `RFUS_YN` | 거부여부 | string | Y | 0 : 승인 ; 1 : 거부 |
| `CNTG_YN` | 체결여부 | string | Y | 1 : 주문,정정,취소,거부; 2 : 체결 |
| `ACPT_YN` | 접수여부 | string | Y | 1 : 주문접수; 2 : 확인; 3 : 취소(FOK/IOC) |
| `BRNC_NO` | 지점번호 | string | Y |  |
| `ODER_QTY` | 주문수량 | string | Y |  |
| `ACNT_NAME` | 계좌명 | string | Y |  |
| `ORD_COND_PRC` | 호가조건가격 | string | Y | 스톱지정가 시 표시 |
| `ORD_EXG_GB` | 주문거래소 구분 | string | Y | 1:KRX, 2:NXT, 3:SOR-KRX, 4:SOR-NXT |
| `POPUP_YN` | 실시간체결창 표시여부 | string | Y | Y/N |
| `FILLER` | 필러 | string | Y |  |
| `CRDT_CLS` | 신용구분 | string | Y |  |
| `CRDT_LOAN_DATE` | 신용대출일자 | string | Y |  |
| `CNTG_ISNM40` | 체결종목명 | string | Y |  |
| `ODER_PRC` | 주문가격 | string | Y |  |

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
                           "tr_id":"H0STCNI0",
                           "tr_key":"HTS ID"
                  }
         }
}
```

### Response Example
```json
{
    "header": {
        "tr_id": "H0STCNI0", 
        "tr_key": "HTS ID", 
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

# output - 주문·정정·취소·거부 접수 통보
HTS ID^1234567801^0000002891^^02^0^01^0^136480^0000000001^000000000^094941^0
^1^1^06010^000000001^김한투^하림^10^^하림^

# output - 체결 통보
HTS ID^1234567801^0000002891^^02^0^00^0^136480^0000000001^000003190^094941^0
^2^2^06010^000000001^김한투^하림^10^^하림^000000000
```
