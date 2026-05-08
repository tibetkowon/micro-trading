# 주식주문(신용)

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `주식주문(신용)` |
| API ID | `v1_국내주식-002` |
| 실전 TR_ID | `(매도) TTTC0051U (매수) TTTC0052U` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `POST` |
| URL | `/uapi/domestic-stock/v1/trading/order-credit` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> 국내주식주문(신용) API입니다. 
※ 모의투자는 사용 불가합니다.

※ POST API의 경우 BODY값의 key값들을 대문자로 작성하셔야 합니다.
   (EX. "CANO" : "12345678", "ACNT_PRDT_CD": "01",...)

## Request Header

> 공통 헤더만 사용. [_공통헤더.md](_공통헤더.md) 참조

## Request Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `CANO` | 종합계좌번호 | string | Y | 계좌번호 체계(8-2)의 앞 8자리 |
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y | 계좌번호 체계(8-2)의 뒤 2자리 |
| `PDNO` | 상품번호 | string | Y | 종목코드(6자리) |
| `SLL_TYPE` | 매도유형 | string | N | 공란 입력 |
| `CRDT_TYPE` | 신용유형 | string | Y | [매도] 22 : 유통대주신규, 24 : 자기대주신규, 25 : 자기융자상환, 27 : 유통융자상환; [매수] 21 : 자기융자신규, 23 : 유통융자신규 , 26 : 유통대주상환, 28 : 자기대주상환 |
| `LOAN_DT` | 대출일자 | string | Y | [신용매수] ; 신규 대출로, 오늘날짜(yyyyMMdd)) 입력 ; [신용매도] ; 매도할 종목의 대출일자(yyyyMMdd)) 입력 |
| `ORD_DVSN` | 주문구분 | string | Y | [KRX] → [공통코드](_공통코드.md#ORD_DVSN) 참조 |
| `ORD_QTY` | 주문수량 | string | Y |  |
| `ORD_UNPR` | 주문단가 | string | Y | 1주당 가격 ; * 장전 시간외, 장후 시간외, 시장가의 경우 1주당 가격을 공란으로 비우지 않음 "0"으로 입력 권고 |
| `RSVN_ORD_YN` | 예약주문여부 | string | N | 정규 증권시장이 열리지 않는 시간 (15:10분 ~ 익일 7:30분) 에 주문을 미리 설정 하여 다음 영업일 또는 설정한 기간 동안 아침 동시 호가에 주문하는 것 ; Y : 예약주문 ; N : 신용주문 |
| `EMGC_ORD_YN` | 비상주문여부 | string | N |  |
| `PGTR_DVSN` | 프로그램매매구분 | string | N |  |
| `MGCO_APTM_ODNO` | 운용사지정주문번호 | string | N |  |
| `LQTY_TR_NGTN_DTL_NO` | 대량거래협상상세번호 | string | N |  |
| `LQTY_TR_AGMT_NO` | 대량거래협정번호 | string | N |  |
| `LQTY_TR_NGTN_ID` | 대량거래협상자Id | string | N |  |
| `LP_ORD_YN` | LP주문여부 | string | N |  |
| `MDIA_ODNO` | 매체주문번호 | string | N |  |
| `ORD_SVR_DVSN_CD` | 주문서버구분코드 | string | N |  |
| `PGM_NMPR_STMT_DVSN_CD` | 프로그램호가신고구분코드 | string | N |  |
| `CVRG_SLCT_RSON_CD` | 반대매매선정사유코드 | string | N |  |
| `CVRG_SEQ` | 반대매매순번 | string | N |  |
| `EXCG_ID_DVSN_CD` | 거래소ID구분코드 | string | N | 한국거래소 : KRX; 대체거래소 (넥스트레이드) : NXT; SOR (Smart Order Routing) : SOR; → 미입력시 KRX로 진행되며, 모의투자는 KRX만 가능 |
| `CNDT_PRIC` | 조건가격 | string | N | 스탑지정가호가에서 사용 |

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
| `output` | 응답상세 | object | Y | single |
| `krx_fwdg_ord_orgno` | 한국거래소전송주문조직번호 | string | Y |  |
| `odno` | 주문번호 | string | Y |  |
| `ord_tmd` | 주문시간 | string | Y |  |

## Example

### Request Example (Python)
```json
{
    "CANO": "810XXXXX",
    "ACNT_PRDT_CD": "01",
    "PDNO": "009150",
    "CRDT_TYPE": "21",
    "LOAN_DT": "20211103",
    "ORD_DVSN": "00",
    "ORD_QTY": "1",
    "ORD_UNPR": "130000",
    "RSVN_ORD_YN": "N"
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
    "ODNO": "0001569138",
    "ORD_TMD": "131421"
  }
}
```
