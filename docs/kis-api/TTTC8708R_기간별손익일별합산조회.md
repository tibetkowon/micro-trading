# 기간별손익일별합산조회

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `기간별손익일별합산조회` |
| API ID | `v1_국내주식-052` |
| 실전 TR_ID | `TTTC8708R` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/trading/inquire-period-profit` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> 기간별손익일별합산조회 API입니다.
한국투자 HTS(eFriend Plus) &gt; [0856] 기간별 매매손익 화면 에서 "일별" 클릭 시의 기능을 API로 개발한 사항으로, 해당 화면을 참고하시면 기능을 이해하기 쉽습니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y |  |
| `CANO` | 종합계좌번호 | string | Y |  |
| `INQR_STRT_DT` | 조회시작일자 | string | Y |  |
| `PDNO` | 상품번호 | string | Y | ""공란입력 시, 전체 |
| `CTX_AREA_NK100` | 연속조회키100 | string | Y |  |
| `INQR_END_DT` | 조회종료일자 | string | Y |  |
| `SORT_DVSN` | 정렬구분 | string | Y | 00: 최근 순, 01: 과거 순, 02: 최근 순 |
| `INQR_DVSN` | 조회구분 | string | Y | 00 입력 |
| `CBLC_DVSN` | 잔고구분 | string | Y | 00: 전체 |
| `CTX_AREA_FK100` | 연속조회검색조건100 | string | Y |  |

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
| `output1` | 응답상세 | object array | Y | array |
| `trad_dt` | 매매일자 | string | Y |  |
| `buy_amt` | 매수금액 | string | Y |  |
| `sll_amt` | 매도금액 | string | Y |  |
| `rlzt_pfls` | 실현손익 | string | Y |  |
| `fee` | 수수료 | string | Y |  |
| `loan_int` | 대출이자 | string | Y |  |
| `tl_tax` | 제세금 | string | Y |  |
| `pfls_rt` | 손익률 | string | Y |  |
| `sll_qty1` | 매도수량1 | string | Y |  |
| `buy_qty1` | 매수수량1 | string | Y |  |
| `output2` | 응답상세2 | object | Y |  |
| `sll_qty_smtl` | 매도수량합계 | string | Y |  |
| `sll_tr_amt_smtl` | 매도거래금액합계 | string | Y |  |
| `sll_fee_smtl` | 매도수수료합계 | string | Y |  |
| `sll_tltx_smtl` | 매도제세금합계 | string | Y |  |
| `sll_excc_amt_smtl` | 매도정산금액합계 | string | Y |  |
| `buy_qty_smtl` | 매수수량합계 | string | Y |  |
| `buy_tr_amt_smtl` | 매수거래금액합계 | string | Y |  |
| `buy_fee_smtl` | 매수수수료합계 | string | Y |  |
| `buy_tax_smtl` | 매수제세금합계 | string | Y |  |
| `buy_excc_amt_smtl` | 매수정산금액합계 | string | Y |  |
| `tot_qty` | 총수량 | string | Y |  |
| `tot_tr_amt` | 총거래금액 | string | Y |  |
| `tot_fee` | 총수수료 | string | Y |  |
| `tot_tltx` | 총제세금 | string | Y |  |
| `tot_excc_amt` | 총정산금액 | string | Y |  |
| `tot_rlzt_pfls` | 총실현손익 | string | Y | ※ HTS[0856] 기간별 매매손익 '일별' 화면의 우측 하단 '총손익률' 항목은 ; 기간별매매손익현황조회(TTTC8715R) > output2 > tot_pftrt(총수익률) 으로 확인 가능 |
| `loan_int` | 대출이자 | string | Y |  |

## Example

### Request Example (Python)
```json
{
"CANO":"12345678",
"ACNT_PRDT_CD":"01",
"PDNO":"",
"INQR_STRT_DT":"20230101",
"INQR_END_DT":"20240220",
"SORT_DVSN":"00",
"INQR_DVSN":"00",
"CBLC_DVSN":"00",
"CTX_AREA_FK100":"",
"CTX_AREA_NK100":""
}
```

### Response Example
```json
{
    "ctx_area_fk100": "                                                                                                    ",
    "ctx_area_nk100": "                                                                                                    ",
    "output1": [
        {
            "trad_dt": "20240220",
            "buy_amt": "116697331",
            "sll_amt": "96455",
            "rlzt_pfls": "22991",
            "fee": "0",
            "loan_int": "0",
            "tl_tax": "0",
            "pfls_rt": "31.29560057",
            "sll_qty1": "8",
            "buy_qty1": "2003"
        }
    ],
    "output2": {
        "sll_qty_smtl": "8",
        "sll_tr_amt_smtl": "96455",
        "sll_fee_smtl": "0",
        "sll_tltx_smtl": "0",
        "sll_excc_amt_smtl": "96455",
        "buy_qty_smtl": "2003",
        "buy_tr_amt_smtl": "116697331",
        "buy_fee_smtl": "0",
        "buy_tax_smtl": "0",
        "buy_excc_amt_smtl": "116697331",
        "tot_qty": "2011",
        "tot_tr_amt": "116793786",
        "tot_fee": "0",
        "tot_tltx": "0",
        "tot_excc_amt": "116793786",
        "tot_rlzt_pfls": "22991",
        "loan_int": "0"
    },
    "rt_cd": "0",
    "msg_cd": "KIOK0510",
    "msg1": "조회가 완료되었습니다                                                           "
}
```
