# 주식예약주문조회

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `주식예약주문조회` |
| API ID | `v1_국내주식-020` |
| 실전 TR_ID | `CTSC0004R` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/trading/order-resv-ccnl` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> 국내예약주문 처리내역 조회 API 입니다.
실전계좌/모의계좌의 경우, 한 번의 호출에 최대 20건까지 확인 가능하며, 이후의 값은 연속조회를 통해 확인하실 수 있습니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `RSVN_ORD_ORD_DT` | 예약주문시작일자 | string | Y |  |
| `RSVN_ORD_END_DT` | 예약주문종료일자 | string | Y |  |
| `RSVN_ORD_SEQ` | 예약주문순번 | string | Y |  |
| `TMNL_MDIA_KIND_CD` | 단말매체종류코드 | string | Y | "00" 입력 |
| `CANO` | 종합계좌번호 | string | Y | 계좌번호 체계(8-2)의 앞 8자리 |
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y | 계좌번호 체계(8-2)의 뒤 2자리 |
| `PRCS_DVSN_CD` | 처리구분코드 | string | Y | 0: 전체; 1: 처리내역; 2: 미처리내역 |
| `CNCL_YN` | 취소여부 | string | Y | "Y" 유효한 주문만 조회 |
| `PDNO` | 상품번호 | string | Y | 종목코드(6자리) (공백 입력 시 전체 조회) |
| `SLL_BUY_DVSN_CD` | 매도매수구분코드 | string | Y |  |
| `CTX_AREA_FK200` | 연속조회검색조건200 | string | Y | 다음 페이지 조회시 사용 |
| `CTX_AREA_NK200` | 연속조회키200 | string | Y | 다음 페이지 조회시 사용 |

## Response Header

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `content-type` | 컨텐츠타입 | string | Y | application/json; charset=utf-8 |
| `tr_id` | 거래ID | string | Y | 요청한 tr_id |
| `tr_cont` | 연속 거래 여부 | string | Y | F or M : 다음 데이터 있음; D or E : 마지막 데이터 |
| `gt_uid` | Global UID | string | Y | [법인 전용] 거래고유번호로 사용하므로 거래별로 UNIQUE해야 함 |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `rt_cd` | 성공 실패 여부 | string | Y | 0 : 성공 ; 0 이외의 값 : 실패 |
| `msg_cd` | 응답코드 | string | Y |  |
| `msg1` | 응답메세지 | string | Y |  |
| `output` | 응답상세 | array | Y |  |
| `rsvn_ord_seq` | 예약주문 순번 | string | N |  |
| `rsvn_ord_ord_dt` | 예약주문주문일자 | string | N |  |
| `rsvn_ord_rcit_dt` | 예약주문접수일자 | string | N |  |
| `pdno` | 상품번호 | string | N |  |
| `ord_dvsn_cd` | 주문구분코드 | string | N |  |
| `ord_rsvn_qty` | 주문예약수량 | string | N |  |
| `tot_ccld_qty` | 총체결수량 | string | N |  |
| `cncl_ord_dt` | 취소주문일자 | string | N |  |
| `ord_tmd` | 주문시각 | string | N |  |
| `ctac_tlno` | 연락전화번호 | string | N |  |
| `rjct_rson2` | 거부사유2 | string | N |  |
| `odno` | 주문번호 | string | N |  |
| `rsvn_ord_rcit_tmd` | 예약주문접수시각 | string | N |  |
| `kor_item_shtn_name` | 한글종목단축명 | string | N |  |
| `sll_buy_dvsn_cd` | 매도매수구분코드 | string | N |  |
| `ord_rsvn_unpr` | 주문예약단가 | string | N |  |
| `tot_ccld_amt` | 총체결금액 | string | N |  |
| `loan_dt` | 대출일자 | string | N |  |
| `cncl_rcit_tmd` | 취소접수시각 | string | N |  |
| `prcs_rslt` | 처리결과 | string | N |  |
| `ord_dvsn_name` | 주문구분명 | string | N |  |
| `tmnl_mdia_kind_cd` | 단말매체종류코드 | string | N |  |
| `rsvn_end_dt` | 예약종료일자 | string | N |  |

## Example

### Request Example (Python)
```json
{
	"RSVN_ORD_ORD_DT":"20220520",
	"RSVN_ORD_END_DT":"20220523",
	"RSVN_ORD_SEQ":"",
	"TMNL_MDIA_KIND_CD":"00",
	"CANO":"81019970",
	"ACNT_PRDT_CD":"01",
	
	"PRCS_DVSN_CD":"0",
	"CNCL_YN":"Y",
	"PDNO":"",
	"SLL_BUY_DVSN_CD":"",
	"CTX_AREA_FK200":"",
	"CTX_AREA_NK200":""
}
```

### Response Example
```json
{
    "ctx_area_fk200": "20220520!^null!^0!^Y!^!^                                                                                                                                                                                ",
    "ctx_area_nk200": " !^ !^                                                                                                                                                                                                  ",
    "output": [
        {
            "rsvn_ord_seq": "42401",
            "rsvn_ord_ord_dt": "20220523",
            "rsvn_ord_rcit_dt": "20220520",
            "pdno": "005940",
            "ord_dvsn_cd": "01",
            "ord_rsvn_qty": "1",
            "tot_ccld_qty": "0",
            "cncl_ord_dt": "",
            "ord_tmd": "",
            "ctac_tlno": "0",
            "rjct_rson2": "",
            "odno": "",
            "rsvn_ord_rcit_tmd": "165318",
            "kor_item_shtn_name": "NH투자증권",
            "sll_buy_dvsn_cd": "02",
            "ord_rsvn_unpr": "6000",
            "tot_ccld_amt": "0",
            "loan_dt": "",
            "cncl_rcit_tmd": "",
            "prcs_rslt": "미처리",
            "ord_dvsn_name": "현금매수",
            "tmnl_mdia_kind_cd": "31",
            "rsvn_end_dt": "20220523"
        },
        {
            "rsvn_ord_seq": "42405",
            "rsvn_ord_ord_dt": "20220523",
            "rsvn_ord_rcit_dt": "20220520",
            "pdno": "005940",
            "ord_dvsn_cd": "01",
            "ord_rsvn_qty": "1",
            "tot_ccld_qty": "0",
            "cncl_ord_dt": "",
            "ord_tmd": "",
            "ctac_tlno": "0",
            "rjct_rson2": "",
            "odno": "",
            "rsvn_ord_rcit_tmd": "170422",
            "kor_item_shtn_name": "NH투자증권",
            "sll_buy_dvsn_cd": "02",
            "ord_rsvn_unpr": "6000",
            "tot_ccld_amt": "0",
            "loan_dt": "",
            "cncl_rcit_tmd": "",
            "prcs_rslt": "미처리",
            "ord_dvsn_name": "현금매수",
            "tmnl_mdia_kind_cd": "31",
            "rsvn_end_dt": ""
        },
        {
            "rsvn_ord_seq": "42406",
            "rsvn_ord_ord_dt": "20220523",
            "rsvn_ord_rcit_dt": "20220520",
            "pdno": "005940",
            "ord_dvsn_cd": "01",
            "ord_rsvn_qty": "1",
            "tot_ccld_qty": "0",
            "cncl_ord_dt": "",
            "ord_tmd": "",
            "ctac_tlno": "0",
            "rjct_rson2": "",
            "odno": "",
            "rsvn_ord_rcit_tmd": "170453",
            "kor_item_shtn_name": "NH투자증권",
            "sll_buy_dvsn_cd": "02",
            "ord_rsvn_unpr": "6000",
            "tot_ccld_amt": "0",
            "loan_dt": "",
            "cncl_rcit_tmd": "",
            "prcs_rslt": "미처리",
            "ord_dvsn_name": "현금매수",
            "tmnl_mdia_kind_cd": "31",
            "rsvn_end_dt": "20220523"
        }
    ],
    "rt_cd": "0",
    "msg_cd": "KIOK0460",
    "msg1": "조회 되었습니다. (마지막 자료)                                                  "
}
```
