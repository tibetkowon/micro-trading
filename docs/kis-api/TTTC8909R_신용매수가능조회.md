# 신용매수가능조회

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `신용매수가능조회` |
| API ID | `v1_국내주식-042` |
| 실전 TR_ID | `TTTC8909R` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/trading/inquire-credit-psamount` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> 신용매수가능조회 API입니다.
신용매수주문 시 주문가능수량과 금액을 확인하실 수 있습니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `CANO` | 종합계좌번호 | string | Y | 계좌번호 체계(8-2)의 앞 8자리 |
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y | 계좌번호 체계(8-2)의 뒤 2자리 |
| `PDNO` | 상품번호 | string | Y | 종목코드(6자리) |
| `ORD_UNPR` | 주문단가 | string | Y | 1주당 가격 ; * 장전 시간외, 장후 시간외, 시장가의 경우 1주당 가격을 공란으로 비우지 않음 "0"으로 입력 권고 |
| `ORD_DVSN` | 주문구분 | string | Y | 00 : 지정가 → [공통코드](_공통코드.md#ORD_DVSN) 참조 |
| `CRDT_TYPE` | 신용유형 | string | Y | 21 : 자기융자신규 → [공통코드](_공통코드.md#CRDT_TYPE) 참조 |
| `CMA_EVLU_AMT_ICLD_YN` | CMA평가금액포함여부 | string | Y | Y/N |
| `OVRS_ICLD_YN` | 해외포함여부 | string | Y | Y/N |

## Response Header

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `content-type` | 컨텐츠타입 | string | Y | application/json; charset=utf-8 |
| `tr_id` | 거래ID | string | Y | 요청한 tr_id |
| `tr_cont` | 연속 거래 여부 | string | N | tr_cont를 이용한 다음조회 불가 API |
| `gt_uid` | Global UID | string | N | [법인 전용] 거래고유번호로 사용하므로 거래별로 UNIQUE해야 함 |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `rt_cd` | 성공 실패 여부 | string | Y | 0 : 성공; 0 이외의 값 : 실패 |
| `msg_cd` | 응답코드 | string | Y | 응답코드 |
| `msg1` | 응답메세지 | string | Y | 응답메시지 |
| `output` | 응답상세 | object | Y |  |
| `ord_psbl_cash` | 주문가능현금 | string | Y |  |
| `ord_psbl_sbst` | 주문가능대용 | string | Y |  |
| `ruse_psbl_amt` | 재사용가능금액 | string | Y |  |
| `fund_rpch_chgs` | 펀드환매대금 | string | Y |  |
| `psbl_qty_calc_unpr` | 가능수량계산단가 | string | Y |  |
| `nrcvb_buy_amt` | 미수없는매수금액 | string | Y |  |
| `nrcvb_buy_qty` | 미수없는매수수량 | string | Y |  |
| `max_buy_amt` | 최대매수금액 | string | Y |  |
| `max_buy_qty` | 최대매수수량 | string | Y |  |
| `cma_evlu_amt` | CMA평가금액 | string | Y |  |
| `ovrs_re_use_amt_wcrc` | 해외재사용금액원화 | string | Y |  |
| `ord_psbl_frcr_amt_wcrc` | 주문가능외화금액원화 | string | Y |  |

## Example

### Request Example (Python)
```json
{
"CANO": "12345678",
"ACNT_PRDT_CD": "01",
"PDNO": "005930",
"ORD_UNPR" : "55000",
"ORD_DVSN": "01",
"CRDT_TYPE": "21",
"CMA_EVLU_AMT_ICLD_YN": "N",
"OVRS_ICLD_YN": "N"
}
```

### Response Example
```json
{
    "output": {
        "ord_psbl_cash": "99965177664",
        "ord_psbl_sbst": "156772560",
        "ruse_psbl_amt": "0",
        "fund_rpch_chgs": "0",
        "psbl_qty_calc_unpr": "69200",
        "nrcvb_buy_amt": "0",
        "nrcvb_buy_qty": "0",
        "max_buy_amt": "0",
        "max_buy_qty": "0",
        "cma_evlu_amt": "0",
        "ovrs_re_use_amt_wcrc": "0",
        "ord_psbl_frcr_amt_wcrc": "157998704172856"
    },
    "rt_cd": "0",
    "msg_cd": "KIOK0510",
    "msg1": "조회가 완료되었습니다                                                           "
}
```
