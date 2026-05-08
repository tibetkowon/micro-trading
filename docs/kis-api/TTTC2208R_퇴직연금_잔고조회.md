# 퇴직연금 잔고조회

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `퇴직연금 잔고조회` |
| API ID | `v1_국내주식-036` |
| 실전 TR_ID | `TTTC2208R` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/trading/pension/inquire-balance` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> 주식, ETF, ETN만 조회 가능하며 펀드는 조회 불가합니다.

​※ 55번 계좌(DC가입자계좌)의 경우 해당 API 이용이 불가합니다.
KIS Developers API의 경우 HTS ID에 반드시 연결되어있어야만 API 신청 및 앱정보 발급이 가능한 서비스로 개발되어서 실물계좌가 아닌 55번 계좌는 API 이용이 불가능한 점 양해 부탁드립니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `CANO` | 종합계좌번호 | string | Y |  |
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y | 29 |
| `ACCA_DVSN_CD` | 적립금구분코드 | string | Y | 00 |
| `INQR_DVSN` | 조회구분 | string | Y | 00 : 전체 |
| `CTX_AREA_FK100` | 연속조회검색조건100 | string | Y |  |
| `CTX_AREA_NK100` | 연속조회키100 | string | Y |  |

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
| `output1` | 응답상세 | object array | Y | Array |
| `cblc_dvsn_name` | 잔고구분명 | string | Y |  |
| `prdt_name` | 상품명 | string | Y |  |
| `pdno` | 상품번호 | string | Y |  |
| `item_dvsn_name` | 종목구분명 | string | Y |  |
| `thdt_buyqty` | 금일매수수량 | string | Y |  |
| `thdt_sll_qty` | 금일매도수량 | string | Y |  |
| `hldg_qty` | 보유수량 | string | Y |  |
| `ord_psbl_qty` | 주문가능수량 | string | Y |  |
| `pchs_avg_pric` | 매입평균가격 | string | Y |  |
| `pchs_amt` | 매입금액 | string | Y |  |
| `prpr` | 현재가 | string | Y |  |
| `evlu_amt` | 평가금액 | string | Y |  |
| `evlu_pfls_amt` | 평가손익금액 | string | Y |  |
| `evlu_erng_rt` | 평가수익율 | string | Y |  |
| `output2` | 응답상세2 | object | Y |  |
| `dnca_tot_amt` | 예수금총금액 | string | Y |  |
| `nxdy_excc_amt` | 익일정산금액 | string | Y |  |
| `prvs_rcdl_excc_amt` | 가수도정산금액 | string | Y |  |
| `thdt_buy_amt` | 금일매수금액 | string | Y |  |
| `thdt_sll_amt` | 금일매도금액 | string | Y |  |
| `thdt_tlex_amt` | 금일제비용금액 | string | Y |  |
| `scts_evlu_amt` | 유가평가금액 | string | Y |  |
| `tot_evlu_amt` | 총평가금액 | string | Y |  |

## Example

### Request Example (Python)
```json
{
	"CANO":"12345678",
	"ACNT_PRDT_CD":"29",
	"ACCA_DVSN_CD":"00",
	"INQR_DVSN":"00",
	"CTX_AREA_FK100":"",
	"CTX_AREA_NK100":""
}
```

### Response Example
```json
{
    "ctx_area_fk100": "12345678^29^00^00^                                                                                  ",
    "ctx_area_nk100": "                                                                                                    ",
    "output1": [
        {
            "cblc_dvsn_name": "사용자",
            "prdt_name": "ACE 미국S&P500",
            "pdno": "360200",
            "item_dvsn_name": "현금",
            "thdt_buyqty": "5",
            "thdt_sll_qty": "0",
            "hldg_qty": "5",
            "ord_psbl_qty": "5",
            "pchs_avg_pric": "13235.0000",
            "pchs_amt": "66175",
            "prpr": "13235",
            "evlu_amt": "66175",
            "evlu_pfls_amt": "0",
            "evlu_erng_rt": "0.00000000"
        }
    ],
    "output2": {
        "dnca_tot_amt": "100000",
        "nxdy_excc_amt": "100000",
        "prvs_rcdl_excc_amt": "33825",
        "thdt_buy_amt": "66175",
        "thdt_sll_amt": "0",
        "thdt_tlex_amt": "0",
        "scts_evlu_amt": "66175",
        "tot_evlu_amt": "100000"
    },
    "rt_cd": "0",
    "msg_cd": "KIOK0510",
    "msg1": "조회가 완료되었습니다                                                           "
}
```
