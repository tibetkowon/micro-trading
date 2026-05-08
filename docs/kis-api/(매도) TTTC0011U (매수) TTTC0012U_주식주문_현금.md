# 주식주문(현금)

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `주식주문(현금)` |
| API ID | `v1_국내주식-001` |
| 실전 TR_ID | `(매도) TTTC0011U (매수) TTTC0012U` |
| 모의 TR_ID | `(매도) VTTC0011U (매수) VTTC0012U` |
| Method | `POST` |
| URL | `/uapi/domestic-stock/v1/trading/order-cash` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `https://openapivts.koreainvestment.com:29443`

> 국내주식주문(현금) API 입니다. 

※ TTC0012U(현금매수) 사용하셔서 미수매수 가능합니다. 단, 거래하시는 계좌가 증거금40%계좌로 신청이 되어있어야 가능합니다. 
※ 신용매수는 별도의 API가 준비되어 있습니다.

※ ORD_QTY(주문수량), ORD_UNPR(주문단가) 등을 String으로 전달해야 함에 유의 부탁드립니다.

※ ORD_UNPR(주문단가)가 없는 주문은 상한가로 주문금액을 선정하고 이후 체결이되면 체결금액로 정산됩니다.

※ POST API의 경우 BODY값의 key값들을 대문자로 작성하셔야 합니다.
   (EX. "CANO" : "12345678", "ACNT_PRDT_CD": "01",...)

※ 종목코드 마스터파일 파이썬 정제코드는 한국투자증권 Github 참고 부탁드립니다.
   https://github.com/koreainvestment/open-trading-api/tree/main/stocks_info

## Request Header

> 공통 헤더만 사용. [_공통헤더.md](_공통헤더.md) 참조

## Request Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `CANO` | 종합계좌번호 | string | Y | 종합계좌번호 |
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y | 상품유형코드 |
| `PDNO` | 상품번호 | string | Y | 종목코드(6자리) , ETN의 경우 7자리 입력 |
| `SLL_TYPE` | 매도유형 (매도주문 시) | string | N | 01@일반매도; 02@임의매매; 05@대차매도; → 미입력시 01 일반매도로 진행 |
| `ORD_DVSN` | 주문구분 | string | Y | [KRX] → [공통코드](_공통코드.md#ORD_DVSN) 참조 |
| `ORD_QTY` | 주문수량 | string | Y | 주문수량 |
| `ORD_UNPR` | 주문단가 | string | Y | 주문단가; 시장가 등 주문시, "0"으로 입력 |
| `CNDT_PRIC` | 조건가격 | string | N | 스탑지정가호가 주문 (ORD_DVSN이 22) 사용 시에만 필수 |
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
| `KRX_FWDG_ORD_ORGNO` | 계좌관리점코드 | string | Y |  |
| `ODNO` | 주문번호 | string | Y |  |
| `ORD_TMD` | 주문시간 | string | Y |  |

## Example

### Request Example (Python)
```json
{
	"CANO": "810XXXXX",
	"ACNT_PRDT_CD": "01",
	"PDNO": "009150",
	"ORD_DVSN": "00",
	"ORD_QTY": "3",
	"ORD_UNPR": "150000"
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
    "ODNO": "0001569157",
    "ORD_TMD": "155211"
  }
}
```
