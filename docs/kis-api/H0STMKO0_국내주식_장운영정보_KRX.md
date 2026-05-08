# 국내주식 장운영정보 (KRX)

| 항목 | 값 |
|------|---|
| 통신방식 | `WEBSOCKET` |
| API 명 | `국내주식 장운영정보 (KRX)` |
| API ID | `실시간-049` |
| 실전 TR_ID | `H0STMKO0` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `POST` |
| URL | `/tryitout/H0STMKO0` |

- 실전: `ws://ops.koreainvestment.com:21000`
- 모의: `모의투자 미지원`

> 국내주식 장운영정보 연결 시, 연결종목의 VI 발동 시와 VI 해제 시에 데이터 수신됩니다. 

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
| `tr_type` | 등록/해제 | string | Y | "1: 등록, 2:해제" |

## Request Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_id` | 거래ID | string | Y | H0STMKO0 |
| `tr_key` | 종목코드 | string | Y | 종목코드 |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `MKSC_SHRN_ISCD` | 유가증권단축종목코드 | object | Y | '각 항목사이에는 구분자로 ^ 사용,; 모든 데이터타입은 String으로 변환되어 push 처리됨' |
| `TRHT_YN` | 거래정지여부 | string | Y |  |
| `TR_SUSP_REAS_CNTT` | 거래정지사유내용 | string | Y |  |
| `MKOP_CLS_CODE` | 장운영구분코드 | string | Y | 110        장전 동시호가 개시 → [공통코드](_공통코드.md#MKOP_CLS_CODE) 참조 |
| `ANTC_MKOP_CLS_CODE` | 예상장운영구분코드 | string | Y | 112 장전예상종료 ; 121 장후예상시작; 129 장후예상종료; 311 장전예상시작 |
| `MRKT_TRTM_CLS_CODE` | 임의연장구분코드 | string | Y | 1  시초동시 임의종료 지정 → [공통코드](_공통코드.md#MRKT_TRTM_CLS_CODE) 참조 |
| `DIVI_APP_CLS_CODE` | 동시호가배분처리구분코드 | string | Y | divi_app_cls_code[0] 1: 배분개시 2: 배분해제; divi_app_cls_code[1] 1: 매수상한 2: 매수하한 3: 매도상한 4: 매도하한 |
| `ISCD_STAT_CLS_CODE` | 종목상태구분코드 | string | Y | 51  관리종목 지정 종목 → [공통코드](_공통코드.md#ISCD_STAT_CLS_CODE) 참조 |
| `VI_CLS_CODE` | VI적용구분코드 | string | Y | Y VI적용된 종목; N VI적용되지 않은 종목 |
| `OVTM_VI_CLS_CODE` | 시간외단일가VI적용구분코드 | string | Y | Y 시간외단일가VI 적용된 종목; N 시간외단일가VI 적용되지 않은 종목 |
| `EXCH_CLS_CODE` | 거래소구분코드 | string | Y |  |

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
            "tr_id": "H0STMKO0",
            "tr_key": "396300"
        }
    }
}
```

### Response Example
```json
# 연결 확인
{
    "header": {
        "tr_id": "H0STMKO0", 
        "tr_key": "396300", 
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
0|H0STMKO0|001|396300^N^(null)^^311^^^55^N^N
```
