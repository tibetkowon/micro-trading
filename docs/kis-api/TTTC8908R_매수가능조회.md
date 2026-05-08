# 매수가능조회

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `매수가능조회` |
| API ID | `v1_국내주식-007` |
| 실전 TR_ID | `TTTC8908R` |
| 모의 TR_ID | `VTTC8908R` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/trading/inquire-psbl-order` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `https://openapivts.koreainvestment.com:29443`

> 매수가능 조회 API입니다. 
실전계좌/모의계좌의 경우, 한 번의 호출에 최대 1건까지 확인 가능합니다.


1) 매수가능금액 확인
 . 미수 사용 X: nrcvb_buy_amt(미수없는매수금액) 확인
 . 미수 사용 O: max_buy_amt(최대매수금액) 확인


2) 매수가능수량 확인
 . 특정 종목 전량매수 시 가능수량을 확인하실 경우 ORD_DVSN:00(지정가)는 종목증거금율이 반영되지 않습니다. 
   따라서 "반드시" ORD_DVSN:01(시장가)로 지정하여 종목증거금율이 반영된 가능수량을 확인하시기 바랍니다. 

   (다만, 조건부지정가 등 특정 주문구분(ex.IOC)으로 주문 시 가능수량을 확인할 경우 주문 시와 동일한 주문구분(ex.IOC) 입력하여 가능수량 확인)

 . 미수 사용 X: ORD_DVSN:01(시장가) or 특정 주문구분(ex.IOC)로 지정하여 nrcvb_buy_qty(미수없는매수수량) 확인
 . 미수 사용 O: ORD_DVSN:01(시장가) or 특정 주문구분(ex.IOC)로 지정하여 max_buy_qty(최대매수수량) 확인

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `CANO` | 종합계좌번호 | string | Y | 계좌번호 체계(8-2)의 앞 8자리 |
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y | 계좌번호 체계(8-2)의 뒤 2자리 |
| `PDNO` | 상품번호 | string | Y | 종목번호(6자리); * PDNO, ORD_UNPR 공란 입력 시, 매수수량 없이 매수금액만 조회됨 |
| `ORD_UNPR` | 주문단가 | string | Y | 1주당 가격; * 시장가(ORD_DVSN:01)로 조회 시, 공란으로 입력; * PDNO, ORD_UNPR 공란 입력 시, 매수수량 없이 매수금액만 조회됨 |
| `ORD_DVSN` | 주문구분 | string | Y | * 특정 종목 전량매수 시 가능수량을 확인할 경우 → [공통코드](_공통코드.md#ORD_DVSN) 참조 |
| `CMA_EVLU_AMT_ICLD_YN` | CMA평가금액포함여부 | string | Y | Y : 포함; N : 포함하지 않음 |
| `OVRS_ICLD_YN` | 해외포함여부 | string | Y | Y : 포함; N : 포함하지 않음 |

## Response Header

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `content-type` | 컨텐츠타입 | string | Y | application/json; charset=utf-8 |
| `tr_id` | 거래ID | string | Y | 요청한 tr_id |
| `tr_cont` | 연속 거래 여부 | string | Y | tr_cont를 이용한 다음조회 불가 API |
| `gt_uid` | Global UID | string | Y | [법인 전용] 거래고유번호로 사용하므로 거래별로 UNIQUE해야 함 |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `rt_cd` | 성공 실패 여부 | string | Y | 0 : 성공; 0 이외의 값 : 실패 |
| `msg_cd` | 응답코드 | string | Y | 응답코드 |
| `msg1` | 응답메세지 | string | Y | 응답메세지 |
| `output` | 응답상세 | object | Y | Single |
| `ord_psbl_cash` | 주문가능현금 | string | Y | 예수금으로 계산된 주문가능금액 |
| `ord_psbl_sbst` | 주문가능대용 | string | Y |  |
| `ruse_psbl_amt` | 재사용가능금액 | string | Y | 전일/금일 매도대금으로 계산된 주문가능금액 |
| `fund_rpch_chgs` | 펀드환매대금 | string | Y |  |
| `psbl_qty_calc_unpr` | 가능수량계산단가 | string | Y |  |
| `nrcvb_buy_amt` | 미수없는매수금액 | string | Y | 미수를 사용하지 않으실 경우 nrcvb_buy_amt(미수없는매수금액)을 확인 |
| `nrcvb_buy_qty` | 미수없는매수수량 | string | Y | 미수를 사용하지 않으실 경우 nrcvb_buy_qty(미수없는매수수량)을 확인; * 특정 종목 전량매수 시 가능수량을 확인하실 경우; 조회 시 ORD_DVSN:01(시장가)로 지정 필수; * 다만, 조건부지정가 등 특정 주문구분(ex.IOC)으로 주문 시 가능수량을 확인할 경우 주문 시와 동일한 주문구분(ex.IOC) 입력 |
| `max_buy_amt` | 최대매수금액 | string | Y | 미수를 사용하시는 경우 max_buy_amt(최대매수금액)를 확인 |
| `max_buy_qty` | 최대매수수량 | string | Y | 미수를 사용하시는 경우 max_buy_qty(최대매수수량)를 확인; * 특정 종목 전량매수 시 가능수량을 확인하실 경우; 조회 시 ORD_DVSN:01(시장가)로 지정 필수; * 다만, 조건부지정가 등 특정 주문구분(ex.IOC)으로 주문 시 가능수량을 확인할 경우 주문 시와 동일한 주문구분(ex.IOC) 입력 |
| `cma_evlu_amt` | CMA평가금액 | string | Y |  |
| `ovrs_re_use_amt_wcrc` | 해외재사용금액원화 | string | Y |  |
| `ord_psbl_frcr_amt_wcrc` | 주문가능외화금액원화 | string | Y |  |

## Example

### Request Example (Python)
```json
{
	"CANO": "810XXXXX",
	"ACNT_PRDT_CD": "01",
	"PDNO": "005930",
	"ORD_UNPR": "0",
	"ORD_DVSN": "01",
	"CMA_EVLU_AMT_ICLD_YN": "N",
	"OVRS_ICLD_YN": "N"
}
```

### Response Example
```json
{
  "output": {
    "ord_psbl_cash": "741191178",
    "ord_psbl_sbst": "0",
    "ruse_psbl_amt": "0",
    "fund_rpch_chgs": "0",
    "psbl_qty_calc_unpr": "70000",
    "nrcvb_buy_amt": "107177377",
    "nrcvb_buy_qty": "1531",
    "max_buy_amt": "1482382356",
    "max_buy_qty": "21176",
    "cma_evlu_amt": "0",
    "ovrs_re_use_amt_wcrc": "0",
    "ord_psbl_frcr_amt_wcrc": "1468797045293"
  },
  "rt_cd": "0",
  "msg_cd": "KIOK0510",
  "msg1": "조회가 완료되었습니다                                                           "
}
```
