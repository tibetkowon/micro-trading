# 주식잔고조회

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `주식잔고조회` |
| API ID | `v1_국내주식-006` |
| 실전 TR_ID | `TTTC8434R` |
| 모의 TR_ID | `VTTC8434R` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/trading/inquire-balance` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `https://openapivts.koreainvestment.com:29443`

> 주식 잔고조회 API입니다. 
실전계좌의 경우, 한 번의 호출에 최대 50건까지 확인 가능하며, 이후의 값은 연속조회를 통해 확인하실 수 있습니다. 
모의계좌의 경우, 한 번의 호출에 최대 20건까지 확인 가능하며, 이후의 값은 연속조회를 통해 확인하실 수 있습니다. 

* 당일 전량매도한 잔고도 보유수량 0으로 보여질 수 있으나, 해당 보유수량 0인 잔고는 최종 D-2일 이후에는 잔고에서 사라집니다.

※ 중요 : 해당 API는 제공 정보량이 많아 조회속도가 느린 API입니다. 주문 준비를 위해서는 주식매수/매도가능수량 조회 TR 사용을 권장 드립니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `CANO` | 종합계좌번호 | string | Y | 계좌번호 체계(8-2)의 앞 8자리 |
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y | 계좌번호 체계(8-2)의 뒤 2자리 |
| `AFHR_FLPR_YN` | 시간외단일가, 거래소여부 | string | Y | N : 기본값,; Y : 시간외단일가,; X : NXT 정규장 (프리마켓, 메인, 애프터마켓); ※ NXT 선택 시 : NXT 거래종목만 시세 등 정보가 NXT 기준으로 변동됩니다. KRX 종목들은 그대로 유지 |
| `OFL_YN` | 오프라인여부 | string | N | 공란(Default) |
| `INQR_DVSN` | 조회구분 | string | Y | 01 : 대출일별 |
| `UNPR_DVSN` | 단가구분 | string | Y | 01 : 기본값 |
| `FUND_STTL_ICLD_YN` | 펀드결제분포함여부 | string | Y | N : 포함하지 않음; Y : 포함 |
| `FNCG_AMT_AUTO_RDPT_YN` | 융자금액자동상환여부 | string | Y | N : 기본값 |
| `PRCS_DVSN` | 처리구분 | string | Y | 00 : 전일매매포함; 01 : 전일매매미포함 |
| `CTX_AREA_FK100` | 연속조회검색조건100 | string | N | 공란 : 최초 조회시; 이전 조회 Output CTX_AREA_FK100 값 : 다음페이지 조회시(2번째부터) |
| `CTX_AREA_NK100` | 연속조회키100 | string | N | 공란 : 최초 조회시; 이전 조회 Output CTX_AREA_NK100 값 : 다음페이지 조회시(2번째부터) |

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
| `rt_cd` | 성공 실패 여부 | string | Y | 0 : 성공; 0 이외의 값 : 실패 |
| `msg_cd` | 응답코드 | string | Y | 응답코드 |
| `msg1` | 응답메세지 | string | Y | 응답메세지 |
| `ctx_area_fk100` | 연속조회검색조건100 | string | Y |  |
| `ctx_area_nk100` | 연속조회키100 | string | Y |  |
| `output1` | 응답상세1 | object array | Y | Array |
| `pdno` | 상품번호 | string | Y | 종목번호(뒷 6자리) |
| `prdt_name` | 상품명 | string | Y | 종목명 |
| `trad_dvsn_name` | 매매구분명 | string | Y | 매수매도구분 |
| `bfdy_buy_qty` | 전일매수수량 | string | Y |  |
| `bfdy_sll_qty` | 전일매도수량 | string | Y |  |
| `thdt_buyqty` | 금일매수수량 | string | Y |  |
| `thdt_sll_qty` | 금일매도수량 | string | Y |  |
| `hldg_qty` | 보유수량 | string | Y |  |
| `ord_psbl_qty` | 주문가능수량 | string | Y |  |
| `pchs_avg_pric` | 매입평균가격 | string | Y | 매입금액 / 보유수량 |
| `pchs_amt` | 매입금액 | string | Y |  |
| `prpr` | 현재가 | string | Y |  |
| `evlu_amt` | 평가금액 | string | Y |  |
| `evlu_pfls_amt` | 평가손익금액 | string | Y | 평가금액 - 매입금액 |
| `evlu_pfls_rt` | 평가손익율 | string | Y |  |
| `evlu_erng_rt` | 평가수익율 | string | Y | 미사용항목(0으로 출력) |
| `loan_dt` | 대출일자 | string | Y | INQR_DVSN(조회구분)을 01(대출일별)로 설정해야 값이 나옴 |
| `loan_amt` | 대출금액 | string | Y |  |
| `stln_slng_chgs` | 대주매각대금 | string | Y |  |
| `expd_dt` | 만기일자 | string | Y |  |
| `fltt_rt` | 등락율 | string | Y |  |
| `bfdy_cprs_icdc` | 전일대비증감 | string | Y |  |
| `item_mgna_rt_name` | 종목증거금율명 | string | Y |  |
| `grta_rt_name` | 보증금율명 | string | Y |  |
| `sbst_pric` | 대용가격 | string | Y | 증권매매의 위탁보증금으로서 현금 대신에 사용되는 유가증권 가격 |
| `stck_loan_unpr` | 주식대출단가 | string | Y |  |
| `output2` | 응답상세2 | object array | Y | Array |
| `dnca_tot_amt` | 예수금총금액 | string | Y | 예수금 |
| `nxdy_excc_amt` | 익일정산금액 | string | Y | D+1 예수금 |
| `prvs_rcdl_excc_amt` | 가수도정산금액 | string | Y | D+2 예수금 |
| `cma_evlu_amt` | CMA평가금액 | string | Y |  |
| `bfdy_buy_amt` | 전일매수금액 | string | Y |  |
| `thdt_buy_amt` | 금일매수금액 | string | Y |  |
| `nxdy_auto_rdpt_amt` | 익일자동상환금액 | string | Y |  |
| `bfdy_sll_amt` | 전일매도금액 | string | Y |  |
| `thdt_sll_amt` | 금일매도금액 | string | Y |  |
| `d2_auto_rdpt_amt` | D+2자동상환금액 | string | Y |  |
| `bfdy_tlex_amt` | 전일제비용금액 | string | Y |  |
| `thdt_tlex_amt` | 금일제비용금액 | string | Y |  |
| `tot_loan_amt` | 총대출금액 | string | Y |  |
| `scts_evlu_amt` | 유가평가금액 | string | Y |  |
| `tot_evlu_amt` | 총평가금액 | string | Y | 유가증권 평가금액 합계금액 + D+2 예수금 |
| `nass_amt` | 순자산금액 | string | Y |  |
| `fncg_gld_auto_rdpt_yn` | 융자금자동상환여부 | string | Y | 보유현금에 대한 융자금만 차감여부; 신용융자 매수체결 시점에서는 융자비율을 매매대금 100%로 계산 하였다가 수도결제일에 보증금에 해당하는 금액을 고객의 현금으로 충당하여 융자금을 감소시키는 업무 |
| `pchs_amt_smtl_amt` | 매입금액합계금액 | string | Y |  |
| `evlu_amt_smtl_amt` | 평가금액합계금액 | string | Y | 유가증권 평가금액 합계금액 |
| `evlu_pfls_smtl_amt` | 평가손익합계금액 | string | Y |  |
| `tot_stln_slng_chgs` | 총대주매각대금 | string | Y |  |
| `bfdy_tot_asst_evlu_amt` | 전일총자산평가금액 | string | Y |  |
| `asst_icdc_amt` | 자산증감액 | string | Y |  |
| `asst_icdc_erng_rt` | 자산증감수익율 | string | Y | 데이터 미제공 |

## Example

### Request Example (Python)
```json
{
	"CANO": "810XXXXX",
	"ACNT_PRDT_CD": "01",
	"AFHR_FLPR_YN": "N",
	"OFL_YN": "",
	"INQR_DVSN": "01",
	"UNPR_DVSN": "01",
	"FUND_STTL_ICLD_YN": "N",
	"FNCG_AMT_AUTO_RDPT_YN": "N",
	"PRCS_DVSN": "01",
	"CTX_AREA_FK100": "",
	"CTX_AREA_NK100": ""
}
```

### Response Example
```json
{
  "ctx_area_fk100": "81055689^01^N^N^01^01^N^                                                                            ",
  "ctx_area_nk100": "                                                                                                    ",
  "output1": [
    {
      "pdno": "009150",
      "prdt_name": "삼성전기",
      "trad_dvsn_name": "현금",
      "bfdy_buy_qty": "12",
      "bfdy_sll_qty": "0",
      "thdt_buyqty": "1686",
      "thdt_sll_qty": "41",
      "hldg_qty": "1657",
      "ord_psbl_qty": "1611",
      "pchs_avg_pric": "135440.2517",
      "pchs_amt": "224424497",
      "prpr": "0",
      "evlu_amt": "0",
      "evlu_pfls_amt": "0",
      "evlu_pfls_rt": "0.00",
      "evlu_erng_rt": "0.00000000",
      "loan_dt": "",
      "loan_amt": "0",
      "stln_slng_chgs": "0",
      "expd_dt": "",
      "fltt_rt": "-100.00000000",
      "bfdy_cprs_icdc": "-184500",
      "item_mgna_rt_name": "",
      "grta_rt_name": "",
      "sbst_pric": "140220",
      "stck_loan_unpr": "0.0000"
    },
    {
      "pdno": "009150",
      "prdt_name": "삼성전기",
      "trad_dvsn_name": "자기융자",
      "bfdy_buy_qty": "3",
      "bfdy_sll_qty": "0",
      "thdt_buyqty": "0",
      "thdt_sll_qty": "0",
      "hldg_qty": "3",
      "ord_psbl_qty": "3",
      "pchs_avg_pric": "123000.0000",
      "pchs_amt": "369000",
      "prpr": "0",
      "evlu_amt": "0",
      "evlu_pfls_amt": "0",
      "evlu_pfls_rt": "0.00",
      "evlu_erng_rt": "0.00000000",
      "loan_dt": "20211223",
      "loan_amt": "369000",
      "stln_slng_chgs": "0",
      "expd_dt": "",
      "fltt_rt": "-100.00000000",
      "bfdy_cprs_icdc": "-184500",
      "item_mgna_rt_name": "",
      "grta_rt_name": "",
      "sbst_pric": "140220",
      "stck_loan_unpr": "123000.0000"
    }
	  ],
  "output2": [
        {
            "dnca_tot_amt": "346455",
            "nxdy_excc_amt": "346455",
            "prvs_rcdl_excc_amt": "346455",
            "cma_evlu_amt": "0",
            "bfdy_buy_amt": "0",
            "thdt_buy_amt": "0",
            "nxdy_auto_rdpt_amt": "0",
            "bfdy_sll_amt": "0",
            "thdt_sll_amt": "0",
            "d2_auto_rdpt_amt": "0",
            "bfdy_tlex_amt": "0",
            "thdt_tlex_amt": "0",
            "tot_loan_amt": "0",
            "scts_evlu_amt": "1759600",
            "tot_evlu_amt": "2106055",
            "nass_amt": "2106055",
            "fncg_gld_auto_rdpt_yn": "",
            "pchs_amt_smtl_amt": "2516522",
            "evlu_amt_smtl_amt": "1759600",
            "evlu_pfls_smtl_amt": "-756922",
            "tot_stln_slng_chgs": "0",
            "bfdy_tot_asst_evlu_amt": "2142945",
            "asst_icdc_amt": "-36890",
            "asst_icdc_erng_rt": "0.00000000"
        }
    ],
  "rt_cd": "0",
  "msg_cd": "KIOK0510",
  "msg1": "조회가 완료되었습니다                                                           "
}
```
