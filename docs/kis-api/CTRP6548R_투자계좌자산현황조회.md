# 투자계좌자산현황조회

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `투자계좌자산현황조회` |
| API ID | `v1_국내주식-048` |
| 실전 TR_ID | `CTRP6548R` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/trading/inquire-account-balance` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> 투자계좌자산현황조회 API입니다.

output1은 한국투자 HTS(eFriend Plus) &gt; [0891] 계좌 자산비중(결제기준) 화면 아래 테이블의 기능을 API로 개발한 사항으로, 해당 화면을 참고하시면 기능을 이해하기 쉽습니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `CANO` | 종합계좌번호 | string | Y | 계좌번호 체계(8-2)의 앞 8자리 |
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y | 계좌번호 체계(8-2)의 뒤 2자리 |
| `INQR_DVSN_1` | 조회구분1 | string | Y | 공백입력 |
| `BSPR_BF_DT_APLY_YN` | 기준가이전일자적용여부 | string | Y | 공백입력 |

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
| `Output1` | 응답상세 | object array | Y | Array [아래 순서대로 출력 : 20항목] → [공통코드](_공통코드.md#Output1) 참조 |
| `pchs_amt` | 매입금액 | string | Y |  |
| `evlu_amt` | 평가금액 | string | Y |  |
| `evlu_pfls_amt` | 평가손익금액 | string | Y |  |
| `crdt_lnd_amt` | 신용대출금액 | string | Y |  |
| `real_nass_amt` | 실제순자산금액 | string | Y |  |
| `whol_weit_rt` | 전체비중율 | string | Y |  |
| `Output2` | 응답상세2 | object | Y |  |
| `pchs_amt_smtl` | 매입금액합계 | string | Y | 유가매입금액 |
| `nass_tot_amt` | 순자산총금액 | string | Y |  |
| `loan_amt_smtl` | 대출금액합계 | string | Y |  |
| `evlu_pfls_amt_smtl` | 평가손익금액합계 | string | Y | 평가손익금액 |
| `evlu_amt_smtl` | 평가금액합계 | string | Y | 유가평가금액 |
| `tot_asst_amt` | 총자산금액 | string | Y | 총 자산금액 |
| `tot_lnda_tot_ulst_lnda` | 총대출금액총융자대출금액 | string | Y |  |
| `cma_auto_loan_amt` | CMA자동대출금액 | string | Y |  |
| `tot_mgln_amt` | 총담보대출금액 | string | Y |  |
| `stln_evlu_amt` | 대주평가금액 | string | Y |  |
| `crdt_fncg_amt` | 신용융자금액 | string | Y |  |
| `ocl_apl_loan_amt` | OCL_APL대출금액 | string | Y |  |
| `pldg_stup_amt` | 질권설정금액 | string | Y |  |
| `frcr_evlu_tota` | 외화평가총액 | string | Y |  |
| `tot_dncl_amt` | 총예수금액 | string | Y |  |
| `cma_evlu_amt` | CMA평가금액 | string | Y |  |
| `dncl_amt` | 예수금액 | string | Y |  |
| `tot_sbst_amt` | 총대용금액 | string | Y |  |
| `thdt_rcvb_amt` | 당일미수금액 | string | Y |  |
| `ovrs_stck_evlu_amt1` | 해외주식평가금액1 | string | Y |  |
| `ovrs_bond_evlu_amt` | 해외채권평가금액 | string | Y |  |
| `mmf_cma_mgge_loan_amt` | MMFCMA담보대출금액 | string | Y |  |
| `sbsc_dncl_amt` | 청약예수금액 | string | Y |  |
| `pbst_sbsc_fnds_loan_use_amt` | 공모주청약자금대출사용금액 | string | Y |  |
| `etpr_crdt_grnt_loan_amt` | 기업신용공여대출금액 | string | Y |  |

## Example

### Request Example (Python)
```json
{
	"CANO":"12345678",
	"ACNT_PRDT_CD":"01",
	"INQR_DVSN_1":"",
	"BSPR_BF_DT_APLY_YN":"",
}
```

### Response Example
```json
{
    "output1": [
        {
            "pchs_amt": "129105",
            "evlu_amt": "406000",
            "evlu_pfls_amt": "276895",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "406000",
            "whol_weit_rt": "0.00000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "0",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "0",
            "whol_weit_rt": "0.00000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "0",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "0",
            "whol_weit_rt": "0.00000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "0",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "0",
            "whol_weit_rt": "0.00000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "0",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "0",
            "whol_weit_rt": "0.00000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "0",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "0",
            "whol_weit_rt": "0.00000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "0",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "0",
            "whol_weit_rt": "0.00000000"
        },
        {
            "pchs_amt": "161026228",
            "evlu_amt": "185144504",
            "evlu_pfls_amt": "24118276",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "185144504",
            "whol_weit_rt": "0.01000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "0",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "0",
            "whol_weit_rt": "0.00000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "0",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "0",
            "whol_weit_rt": "0.00000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "0",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "0",
            "whol_weit_rt": "0.00000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "0",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "0",
            "whol_weit_rt": "0.00000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "0",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "0",
            "whol_weit_rt": "0.00000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "0",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "0",
            "whol_weit_rt": "0.00000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "0",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "0",
            "whol_weit_rt": "0.00000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "1651434483743",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "1651434483743",
            "whol_weit_rt": "99.97000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "249855300",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "249855300",
            "whol_weit_rt": "0.01000000"
        },
        {
            "pchs_amt": "0",
            "evlu_amt": "0",
            "evlu_pfls_amt": "0",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "0",
            "whol_weit_rt": "0.00000000"
        },
        {
            "pchs_amt": "161155333",
            "evlu_amt": "1651869889547",
            "evlu_pfls_amt": "24395171",
            "crdt_lnd_amt": "0",
            "real_nass_amt": "1651869889547",
            "whol_weit_rt": "100.00000000"
        }
    ],
    "output2": {
        "pchs_amt_smtl": "161155333",
        "nass_tot_amt": "185550504",
        "loan_amt_smtl": "0",
        "evlu_pfls_amt_smtl": "24395171",
        "evlu_amt_smtl": "185550504",
        "tot_asst_amt": "1651869889547",
        "tot_lnda_tot_ulst_lnda": "0",
        "cma_auto_loan_amt": "0",
        "tot_mgln_amt": "0",
        "stln_evlu_amt": "0",
        "crdt_fncg_amt": "0",
        "ocl_apl_loan_amt": "0",
        "pldg_stup_amt": "0",
        "frcr_evlu_tota": "1651434483743",
        "tot_dncl_amt": "249855300",
        "cma_evlu_amt": "0",
        "dncl_amt": "249855300",
        "tot_sbst_amt": "0",
        "thdt_rcvb_amt": "0",
        "ovrs_stck_evlu_amt1": "185144504.000000",
        "ovrs_bond_evlu_amt": "0.000000",
        "mmf_cma_mgge_loan_amt": "0",
        "sbsc_dncl_amt": "0",
        "pbst_sbsc_fnds_loan_use_amt": "0",
        "etpr_crdt_grnt_loan_amt": "0"
    },
    "rt_cd": "0",
    "msg_cd": "KIOK0530",
    "msg1": "조회되었습니다                                                                  "
}
```
