# 주식예약주문정정취소

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `주식예약주문정정취소` |
| API ID | `v1_국내주식-018,019` |
| 실전 TR_ID | `(예약취소) CTSC0009U (예약정정) CTSC0013U` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `POST` |
| URL | `/uapi/domestic-stock/v1/trading/order-resv-rvsecncl` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> 국내주식 예약주문 정정/취소 API 입니다.
*  정정주문은 취소주문에 비해 필수 입력값이 추가 됩니다. 
   하단의 입력값을 참조하시기 바랍니다.

※ POST API의 경우 BODY값의 key값들을 대문자로 작성하셔야 합니다.
   (EX. "CANO" : "12345678", "ACNT_PRDT_CD": "01",...)

## Request Header

> 공통 헤더만 사용. [_공통헤더.md](_공통헤더.md) 참조

## Request Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `CANO` | 종합계좌번호 | string | Y | [정정/취소] 계좌번호 체계(8-2)의 앞 8자리 |
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y | [정정/취소] 계좌번호 체계(8-2)의 뒤 2자리 |
| `PDNO` | 종목코드(6자리) | string | Y | [정정] |
| `ORD_QTY` | 주문수량 | string | Y | [정정] 주문주식수 |
| `ORD_UNPR` | 주문단가 | string | Y | [정정] 1주당 가격 ; * 장전 시간외, 시장가의 경우 1주당 가격을 공란으로 비우지 않음 "0"으로 입력 권고 |
| `SLL_BUY_DVSN_CD` | 매도매수구분코드 | string | Y | [정정]; 01 : 매도; 02 : 매수 |
| `ORD_DVSN_CD` | 주문구분코드 | string | Y | [정정]; 00 : 지정가; 01 : 시장가; 02 : 조건부지정가; 05 : 장전 시간외 |
| `ORD_OBJT_CBLC_DVSN_CD` | 주문대상잔고구분코드 | string | Y | [정정] → [공통코드](_공통코드.md#ORD_OBJT_CBLC_DVSN_CD) 참조 |
| `LOAN_DT` | 대출일자 | string | N | [정정] |
| `RSVN_ORD_END_DT` | 예약주문종료일자 | string | N | [정정] |
| `CTAL_TLNO` | 연락전화번호 | string | N | [정정] |
| `RSVN_ORD_SEQ` | 예약주문순번 | string | Y | [정정/취소] |
| `RSVN_ORD_ORGNO` | 예약주문조직번호 | string | N | [정정/취소] |
| `RSVN_ORD_ORD_DT` | 예약주문주문일자 | string | N | [정정/취소] |

## Response Header

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `content-type` | 컨텐츠타입 | string | Y | application/json; charset=utf-8 |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `rt_cd` | 성공 실패 여부 | string | Y | 0 : 성공 ; 0 이외의 값 : 실패 |
| `msg_cd` | 응답코드 | string | Y |  |
| `msg` | 응답메세지 | string | Y |  |
| `output` | 응답상세 | array | Y |  |
| `nrml_prcs_yn` | 정상처리여부 | string | Y |  |

## Example

### Request Example (Python)
```json
{ 
	"_comment": "주식예약주문취소", 
	"CANO": "810XXXXX", 
	"ACNT_PRDT_CD": "01", 
	"RSVN_ORD_ORD_DT": "20220427", 
	"RSVN_ORD_SEQ": "39447", 
	"RSVN_ORD_ORGNO": "00" 
} 

{ 
	"_comment": "주식예약주문정정", 
	"CANO": "810XXXXX", 
	"ACNT_PRDT_CD": "01", 
	"PDNO": "009150", 
	"ORD_QTY": "10", 
	"ORD_UNPR": "140000", 
	"SLL_BUY_DVSN_CD":"01", 
	"ORD_DVSN_CD":"00", 
	"ORD_OBJT_CBLC_DVSN_CD":"10", 
	"LOAN_DT":"", 
	"RSVN_ORD_END_DT":"", 
	"CTAC_TLNO": "", 
	"RSVN_ORD_SEQ":"39453", 
	"RSVN_ORD_ORGNO":"", 
	"RSVN_ORD_ORD_DT":"20220427" 
}
```

### Response Example
```json
{ 
	"rt_cd": "0", 
	"msg_cd": "KIOK0430", 
	"msg1": "정상적으로 처리되었습니다", 
	"output": { 
		"NRML_PRCS_YN": "Y" 
	} 
}
```
