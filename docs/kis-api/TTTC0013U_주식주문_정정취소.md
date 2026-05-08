# 주식주문(정정취소)

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `주식주문(정정취소)` |
| API ID | `v1_국내주식-003` |
| 실전 TR_ID | `TTTC0013U` |
| 모의 TR_ID | `VTTC0013U` |
| Method | `POST` |
| URL | `/uapi/domestic-stock/v1/trading/order-rvsecncl` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `https://openapivts.koreainvestment.com:29443`

> 주문 건에 대하여 정정 및 취소하는 API입니다. 단, 이미 체결된 건은 정정 및 취소가 불가합니다.

※ 정정은 원주문에 대한 주문단가 혹은 주문구분을 변경하는 사항으로, 정정이 가능한 수량은 원주문수량을 초과 할 수 없습니다.

※ 주식주문(정정취소) 호출 전에 반드시 주식정정취소가능주문조회 호출을 통해 정정취소가능수량(output &gt; psbl_qty)을 확인하신 후 정정취소주문 내시기 바랍니다.

※ POST API의 경우 BODY값의 key값들을 대문자로 작성하셔야 합니다.
   (EX. "CANO" : "12345678", "ACNT_PRDT_CD": "01",...)

## Request Header

> 공통 헤더만 사용. [_공통헤더.md](_공통헤더.md) 참조

## Request Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `CANO` | 종합계좌번호 | string | Y | 종합계좌번호 |
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y | 상품유형코드 |
| `KRX_FWDG_ORD_ORGNO` | 한국거래소전송주문조직번호 | string | Y |  |
| `ORGN_ODNO` | 원주문번호 | string | Y | 원주문번호 |
| `ORD_DVSN` | 주문구분 | string | Y | [KRX] → [공통코드](_공통코드.md#ORD_DVSN) 참조 |
| `RVSE_CNCL_DVSN_CD` | 정정취소구분코드 | string | Y | 01@정정; 02@취소 |
| `ORD_QTY` | 주문수량 | string | Y | 주문수량 |
| `ORD_UNPR` | 주문단가 | string | Y | 주문단가 |
| `QTY_ALL_ORD_YN` | 잔량전부주문여부 | string | Y | 'Y@전량; N@일부' |
| `CNDT_PRIC` | 조건가격 | string | N | 스탑지정가호가에서 사용 |
| `EXCG_ID_DVSN_CD` | 거래소ID구분코드 | string | N | 한국거래소 : KRX; 대체거래소 (넥스트레이드) : NXT; SOR (Smart Order Routing) : SOR; → 미입력시 KRX로 진행되며, 모의투자는 KRX만 가능 |

## Response Header

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `content-type` | 컨텐츠타입 | string | Y | application/json; charset=utf-8 |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `rt_cd` | 성공 실패 여부 | string | Y |  |
| `msg_cd` | 응답코드 | string | Y |  |
| `msg1` | 응답메세지 | string | Y |  |
| `output` | 응답상세 | object array | Y | single |
| `krx_fwdg_ord_orgno` | 한국거래소전송주문조직번호 | string | Y |  |
| `odno` | 주문번호 | string | Y |  |
| `ord_tmd` | 주문시각 | string | Y |  |

## Example

### Request Example (Python)
```json
{
"CANO": "810XXXXX",
"ACNT_PRDT_CD": "01",
"KRX_FWDG_ORD_ORGNO": "",
"ORGN_ODNO": "0001566017",
"ORD_DVSN": "00",
"RVSE_CNCL_DVSN_CD": "01",
"ORD_QTY": "1",
"ORD_UNPR": "180000",
"QTY_ALL_ORD_YN": "N"
}
```

### Response Example
```json
{
  "rt_cd": "0",
  "msg_cd": "APBK0013",
  "msg1": "주문 전송 완료 되었습니다.",
  "output": {
    "KRX_FWDG_ORD_ORGNO": "06010",
    "ODNO": "0001569139",
    "ORD_TMD": "131438"
  }
}
```
