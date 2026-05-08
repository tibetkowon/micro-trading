# 주식정정취소가능주문조회

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `주식정정취소가능주문조회` |
| API ID | `v1_국내주식-004` |
| 실전 TR_ID | `TTTC0084R` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/trading/inquire-psbl-rvsecncl` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> 주식정정취소가능주문조회 API입니다. 한 번의 호출에 최대 50건까지 확인 가능하며, 이후의 값은 연속조회를 통해 확인하실 수 있습니다.

※ 주식주문(정정취소) 호출 전에 반드시 주식정정취소가능주문조회 호출을 통해 정정취소가능수량(output &gt; psbl_qty)을 확인하신 후 정정취소주문 내시기 바랍니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `CANO` | 종합계좌번호 | string | Y | 계좌번호 체계(8-2)의 앞 8자리 |
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y | 계좌번호 체계(8-2)의 뒤 2자리 |
| `CTX_AREA_FK100` | 연속조회검색조건100 | string | Y | '공란 : 최초 조회시는 ; 이전 조회 Output CTX_AREA_FK100 값 : 다음페이지 조회시(2번째부터)' |
| `CTX_AREA_NK100` | 연속조회키100 | string | Y | '공란 : 최초 조회시 ; 이전 조회 Output CTX_AREA_NK100 값 : 다음페이지 조회시(2번째부터)' |
| `INQR_DVSN_1` | 조회구분1 | string | Y | '0 주문; 1 종목' |
| `INQR_DVSN_2` | 조회구분2 | string | Y | '0 전체; 1 매도; 2 매수' |

## Response Header

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `content-type` | 컨텐츠타입 | string | Y | application/json; charset=utf-8 |
| `tr_id` | 거래ID | string | Y | 요청한 tr_id |
| `tr_cont` | 연속 거래 여부 | string | N | F or M : 다음 데이터 있음; D or E : 마지막 데이터 |
| `gt_uid` | Global UID | string | N | [법인 전용] 거래고유번호로 사용하므로 거래별로 UNIQUE해야 함 |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `rt_cd` | 성공 실패 여부 | string | Y |  |
| `msg_cd` | 응답코드 | string | Y |  |
| `msg1` | 응답메세지 | string | Y |  |
| `output` | 응답상세 | object array | Y | array |
| `ord_gno_brno` | 주문채번지점번호 | string | Y | 주문시 한국투자증권 시스템에서 지정된 영업점코드 |
| `odno` | 주문번호 | string | Y | 주문시 한국투자증권 시스템에서 채번된 주문번호 |
| `orgn_odno` | 원주문번호 | string | Y | 정정/취소주문 인경우 원주문번호 |
| `ord_dvsn_name` | 주문구분명 | string | Y |  |
| `pdno` | 상품번호 | string | Y | 종목번호(뒤 6자리만 해당) |
| `prdt_name` | 상품명 | string | Y | 종목명 |
| `rvse_cncl_dvsn_name` | 정정취소구분명 | string | Y | 정정 또는 취소 여부 표시 |
| `ord_qty` | 주문수량 | string | Y |  |
| `ord_unpr` | 주문단가 | string | Y | 1주당 주문가격 |
| `ord_tmd` | 주문시각 | string | Y | 주문시각(시분초HHMMSS) |
| `tot_ccld_qty` | 총체결수량 | string | Y | 주문 수량 중 체결된 수량 |
| `tot_ccld_amt` | 총체결금액 | string | Y | 주문금액 중 체결금액 |
| `psbl_qty` | 가능수량 | string | Y | 정정/취소 주문 가능 수량 |
| `sll_buy_dvsn_cd` | 매도매수구분코드 | string | Y | 01 : 매도 / 02 : 매수 |
| `ord_dvsn_cd` | 주문구분코드 | string | Y | [KRX] → [공통코드](_공통코드.md#ord_dvsn_cd) 참조 |
| `mgco_aptm_odno` | 운용사지정주문번호 | string | Y |  |
| `excg_dvsn_cd` | 거래소구분코드 | string | Y |  |
| `excg_id_dvsn_cd` | 거래소ID구분코드 | string | Y |  |
| `excg_id_dvsn_name` | 거래소ID구분명 | string | Y |  |
| `stpm_cndt_pric` | 스톱지정가조건가격 | string | Y |  |
| `stpm_efct_occr_yn` | 스톱지정가효력발생여부 | string | Y |  |

## Example

### Request Example (Python)
```json
{
	"ACNT_PRDT_CD": "01",
	"CANO": "810XXXXX",
	"CTX_AREA_FK100": "",
	"CTX_AREA_NK100": "",
	"INQR_DVSN_1": "0",
	"INQR_DVSN_2": "0"
}
```

### Response Example
```json
{
  "ctx_area_fk100": "81055689^01^                                                                                        ",
  "ctx_area_nk100": "                                                                                                    ",
  "output": [
    {
      "ord_gno_brno": "06010",
      "odno": "0001569139",
      "orgn_odno": "0001569136",
      "ord_dvsn_name": "지정가",
      "pdno": "009150",
      "prdt_name": "SamsungElecMech",
      "rvse_cncl_dvsn_name": "BUY AMEND*",
      "ord_qty": "1",
      "ord_unpr": "140000",
      "ord_tmd": "131438",
      "tot_ccld_qty": "0",
      "tot_ccld_amt": "0",
      "psbl_qty": "1",
      "sll_buy_dvsn_cd": "02",
      "ord_dvsn_cd": "00",
      "mgco_aptm_odno": ""
    },
    {
      "ord_gno_brno": "06010",
      "odno": "0001569138",
      "orgn_odno": "",
      "ord_dvsn_name": "지정가",
      "pdno": "009150",
      "prdt_name": "SamsungElecMech",
      "rvse_cncl_dvsn_name": "",
      "ord_qty": "1",
      "ord_unpr": "200000",
      "ord_tmd": "131421",
      "tot_ccld_qty": "0",
      "tot_ccld_amt": "0",
      "psbl_qty": "1",
      "sll_buy_dvsn_cd": "02",
      "ord_dvsn_cd": "00",
      "mgco_aptm_odno": ""
    }
	],
  "rt_cd": "0",
  "msg_cd": "KIOK0510",
  "msg1": "조회가 완료되었습니다                                                           "
}
```
