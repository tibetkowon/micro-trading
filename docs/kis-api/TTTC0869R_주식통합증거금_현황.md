# 주식통합증거금 현황

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `주식통합증거금 현황` |
| API ID | `국내주식-191` |
| 실전 TR_ID | `TTTC0869R` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/trading/intgr-margin` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> 주식통합증거금 현황 API입니다.
한국투자 HTS(eFriend Plus) &gt; [0867] 통합증거금조회 화면 의 기능을 API로 개발한 사항으로, 해당 화면을 참고하시면 기능을 이해하기 쉽습니다.

※ 해당 화면은 일반계좌와 통합증거금 신청계좌에 대해서 국내 및 해외 주문가능금액을 간단하게 조회하는 화면입니다.
※ 해외 국가별 상세한 증거금현황을 원하시면 [해외주식] 주문/계좌 &gt; 해외증거금 통화별조회 API를 이용하여 주시기 바랍니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `CANO` | 종합계좌번호 | string | Y | 계좌번호 체계(8-2)의 앞 8자리 |
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y | 계좌번호 체계(8-2)의 뒤 2자리 |
| `CMA_EVLU_AMT_ICLD_YN` | CMA평가금액포함여부 | string | Y | N 입력 |
| `WCRC_FRCR_DVSN_CD` | 원화외화구분코드 | string | Y | 01(외화기준),02(원화기준) |
| `FWEX_CTRT_FRCR_DVSN_CD` | 선도환계약외화구분코드 | string | Y | 01(외화기준),02(원화기준) |

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
| `rt_cd` | 성공 실패 여부 | string | Y |  |
| `msg_cd` | 응답코드 | string | Y |  |
| `msg1` | 응답메세지 | string | Y |  |
| `output` | 응답상세 | object | Y |  |
| `acmga_rt` | 계좌증거금율 | string | Y |  |
| `acmga_pct100_aptm_rson` | 계좌증거금100퍼센트지정사유 | string | Y |  |
| `stck_cash_objt_amt` | 주식현금대상금액 | string | Y |  |
| `stck_sbst_objt_amt` | 주식대용대상금액 | string | Y |  |
| `stck_evlu_objt_amt` | 주식평가대상금액 | string | Y |  |
| `stck_ruse_psbl_objt_amt` | 주식재사용가능대상금액 | string | Y |  |
| `stck_fund_rpch_chgs_objt_amt` | 주식펀드환매대금대상금액 | string | Y |  |
| `stck_fncg_rdpt_objt_atm` | 주식융자상환금대상금액 | string | Y |  |
| `bond_ruse_psbl_objt_amt` | 채권재사용가능대상금액 | string | Y |  |
| `stck_cash_use_amt` | 주식현금사용금액 | string | Y |  |
| `stck_sbst_use_amt` | 주식대용사용금액 | string | Y |  |
| `stck_evlu_use_amt` | 주식평가사용금액 | string | Y |  |
| `stck_ruse_psbl_amt_use_amt` | 주식재사용가능금사용금액 | string | Y |  |
| `stck_fund_rpch_chgs_use_amt` | 주식펀드환매대금사용금액 | string | Y |  |
| `stck_fncg_rdpt_amt_use_amt` | 주식융자상환금사용금액 | string | Y |  |
| `bond_ruse_psbl_amt_use_amt` | 채권재사용가능금사용금액 | string | Y |  |
| `stck_cash_ord_psbl_amt` | 주식현금주문가능금액 | string | Y |  |
| `stck_sbst_ord_psbl_amt` | 주식대용주문가능금액 | string | Y |  |
| `stck_evlu_ord_psbl_amt` | 주식평가주문가능금액 | string | Y |  |
| `stck_ruse_psbl_ord_psbl_amt` | 주식재사용가능주문가능금액 | string | Y |  |
| `stck_fund_rpch_ord_psbl_amt` | 주식펀드환매주문가능금액 | string | Y |  |
| `bond_ruse_psbl_ord_psbl_amt` | 채권재사용가능주문가능금액 | string | Y |  |
| `rcvb_amt` | 미수금액 | string | Y |  |
| `stck_loan_grta_ruse_psbl_amt` | 주식대출보증금재사용가능금액 | string | Y |  |
| `stck_cash20_max_ord_psbl_amt` | 주식현금20최대주문가능금액 | string | Y |  |
| `stck_cash30_max_ord_psbl_amt` | 주식현금30최대주문가능금액 | string | Y |  |
| `stck_cash40_max_ord_psbl_amt` | 주식현금40최대주문가능금액 | string | Y |  |
| `stck_cash50_max_ord_psbl_amt` | 주식현금50최대주문가능금액 | string | Y |  |
| `stck_cash60_max_ord_psbl_amt` | 주식현금60최대주문가능금액 | string | Y |  |
| `stck_cash100_max_ord_psbl_amt` | 주식현금100최대주문가능금액 | string | Y |  |
| `stck_rsip100_max_ord_psbl_amt` | 주식재사용불가100최대주문가능 | string | Y |  |
| `bond_max_ord_psbl_amt` | 채권최대주문가능금액 | string | Y |  |
| `stck_fncg45_max_ord_psbl_amt` | 주식융자45최대주문가능금액 | string | Y |  |
| `stck_fncg50_max_ord_psbl_amt` | 주식융자50최대주문가능금액 | string | Y |  |
| `stck_fncg60_max_ord_psbl_amt` | 주식융자60최대주문가능금액 | string | Y |  |
| `stck_fncg70_max_ord_psbl_amt` | 주식융자70최대주문가능금액 | string | Y |  |
| `stck_stln_max_ord_psbl_amt` | 주식대주최대주문가능금액 | string | Y |  |
| `lmt_amt` | 한도금액 | string | Y |  |
| `ovrs_stck_itgr_mgna_dvsn_name` | 해외주식통합증거금구분명 | string | Y |  |
| `usd_objt_amt` | 미화대상금액 | string | Y |  |
| `usd_use_amt` | 미화사용금액 | string | Y |  |
| `usd_ord_psbl_amt` | 미화주문가능금액 | string | Y |  |
| `hkd_objt_amt` | 홍콩달러대상금액 | string | Y |  |
| `hkd_use_amt` | 홍콩달러사용금액 | string | Y |  |
| `hkd_ord_psbl_amt` | 홍콩달러주문가능금액 | string | Y |  |
| `jpy_objt_amt` | 엔화대상금액 | string | Y |  |
| `jpy_use_amt` | 엔화사용금액 | string | Y |  |
| `jpy_ord_psbl_amt` | 엔화주문가능금액 | string | Y |  |
| `cny_objt_amt` | 위안화대상금액 | string | Y |  |
| `cny_use_amt` | 위안화사용금액 | string | Y |  |
| `cny_ord_psbl_amt` | 위안화주문가능금액 | string | Y |  |
| `usd_ruse_objt_amt` | 미화재사용대상금액 | string | Y |  |
| `usd_ruse_amt` | 미화재사용금액 | string | Y |  |
| `usd_ruse_ord_psbl_amt` | 미화재사용주문가능금액 | string | Y |  |
| `hkd_ruse_objt_amt` | 홍콩달러재사용대상금액 | string | Y |  |
| `hkd_ruse_amt` | 홍콩달러재사용금액 | string | Y |  |
| `hkd_ruse_ord_psbl_amt` | 홍콩달러재사용주문가능금액 | string | Y |  |
| `jpy_ruse_objt_amt` | 엔화재사용대상금액 | string | Y |  |
| `jpy_ruse_amt` | 엔화재사용금액 | string | Y |  |
| `jpy_ruse_ord_psbl_amt` | 엔화재사용주문가능금액 | string | Y |  |
| `cny_ruse_objt_amt` | 위안화재사용대상금액 | string | Y |  |
| `cny_ruse_amt` | 위안화재사용금액 | string | Y |  |
| `cny_ruse_ord_psbl_amt` | 위안화재사용주문가능금액 | string | Y |  |
| `usd_gnrl_ord_psbl_amt` | 미화일반주문가능금액 | string | Y |  |
| `usd_itgr_ord_psbl_amt` | 미화통합주문가능금액 | string | Y |  |
| `hkd_gnrl_ord_psbl_amt` | 홍콩달러일반주문가능금액 | string | Y |  |
| `hkd_itgr_ord_psbl_amt` | 홍콩달러통합주문가능금액 | string | Y |  |
| `jpy_gnrl_ord_psbl_amt` | 엔화일반주문가능금액 | string | Y |  |
| `jpy_itgr_ord_psbl_amt` | 엔화통합주문가능금액 | string | Y |  |
| `cny_gnrl_ord_psbl_amt` | 위안화일반주문가능금액 | string | Y |  |
| `cny_itgr_ord_psbl_amt` | 위안화통합주문가능금액 | string | Y |  |
| `stck_itgr_cash20_ord_psbl_amt` | 주식통합현금20주문가능금액 | string | Y |  |
| `stck_itgr_cash30_ord_psbl_amt` | 주식통합현금30주문가능금액 | string | Y |  |
| `stck_itgr_cash40_ord_psbl_amt` | 주식통합현금40주문가능금액 | string | Y |  |
| `stck_itgr_cash50_ord_psbl_amt` | 주식통합현금50주문가능금액 | string | Y |  |
| `stck_itgr_cash60_ord_psbl_amt` | 주식통합현금60주문가능금액 | string | Y |  |
| `stck_itgr_cash100_ord_psbl_amt` | 주식통합현금100주문가능금액 | string | Y |  |
| `stck_itgr_100_ord_psbl_amt` | 주식통합100주문가능금액 | string | Y |  |
| `stck_itgr_fncg45_ord_psbl_amt` | 주식통합융자45주문가능금액 | string | Y |  |
| `stck_itgr_fncg50_ord_psbl_amt` | 주식통합융자50주문가능금액 | string | Y |  |
| `stck_itgr_fncg60_ord_psbl_amt` | 주식통합융자60주문가능금액 | string | Y |  |
| `stck_itgr_fncg70_ord_psbl_amt` | 주식통합융자70주문가능금액 | string | Y |  |
| `stck_itgr_stln_ord_psbl_amt` | 주식통합대주주문가능금액 | string | Y |  |
| `bond_itgr_ord_psbl_amt` | 채권통합주문가능금액 | string | Y |  |
| `stck_cash_ovrs_use_amt` | 주식현금해외사용금액 | string | Y |  |
| `stck_sbst_ovrs_use_amt` | 주식대용해외사용금액 | string | Y |  |
| `stck_evlu_ovrs_use_amt` | 주식평가해외사용금액 | string | Y |  |
| `stck_re_use_amt_ovrs_use_amt` | 주식재사용금액해외사용금액 | string | Y |  |
| `stck_fund_rpch_ovrs_use_amt` | 주식펀드환매해외사용금액 | string | Y |  |
| `stck_fncg_rdpt_ovrs_use_amt` | 주식융자상환해외사용금액 | string | Y |  |
| `bond_re_use_ovrs_use_amt` | 채권재사용해외사용금액 | string | Y |  |
| `usd_oth_mket_use_amt` | 미화타시장사용금액 | string | Y |  |
| `jpy_oth_mket_use_amt` | 엔화타시장사용금액 | string | Y |  |
| `cny_oth_mket_use_amt` | 위안화타시장사용금액 | string | Y |  |
| `hkd_oth_mket_use_amt` | 홍콩달러타시장사용금액 | string | Y |  |
| `usd_re_use_oth_mket_use_amt` | 미화재사용타시장사용금액 | string | Y |  |
| `jpy_re_use_oth_mket_use_amt` | 엔화재사용타시장사용금액 | string | Y |  |
| `cny_re_use_oth_mket_use_amt` | 위안화재사용타시장사용금액 | string | Y |  |
| `hkd_re_use_oth_mket_use_amt` | 홍콩달러재사용타시장사용금액 | string | Y |  |
| `hgkg_cny_re_use_amt` | 홍콩위안화재사용금액 | string | Y |  |
| `usd_frst_bltn_exrt` | 미국달러최초고시환율 | string | Y |  |
| `hkd_frst_bltn_exrt` | 홍콩달러최초고시환율 | string | Y |  |
| `jpy_frst_bltn_exrt` | 일본엔화최초고시환율 | string | Y |  |
| `cny_frst_bltn_exrt` | 중국위안화최초고시환율 | string | Y |  |

## Example

### Request Example (Python)
```json
CANO:12345678
ACNT_PRDT_CD:01
CMA_EVLU_AMT_ICLD_YN:N
WCRC_FRCR_DVSN_CD:01
FWEX_CTRT_FRCR_DVSN_CD:01
```

### Response Example
```json
{
    "output": {
        "acmga_rt": "100.0000",
        "acmga_pct100_aptm_rson": "고객100%신청",
        "stck_cash_objt_amt": "249855306.0000",
        "stck_sbst_objt_amt": "137816.0000",
        "stck_evlu_objt_amt": "176966.0000",
        "stck_ruse_psbl_objt_amt": "261213.0000",
        "stck_fund_rpch_chgs_objt_amt": "0.0000",
        "stck_fncg_rdpt_objt_atm": "0.0000",
        "bond_ruse_psbl_objt_amt": "1024.0000",
        "stck_cash_use_amt": "240482730.0000",
        "stck_sbst_use_amt": "20295.0000",
        "stck_evlu_use_amt": "20295.0000",
        "stck_ruse_psbl_amt_use_amt": "261213.0000",
        "stck_fund_rpch_chgs_use_amt": "0.0000",
        "stck_fncg_rdpt_amt_use_amt": "0.0000",
        "bond_ruse_psbl_amt_use_amt": "1024.0000",
        "stck_cash_ord_psbl_amt": "9372576.0000",
        "stck_sbst_ord_psbl_amt": "117521.0000",
        "stck_evlu_ord_psbl_amt": "156671.0000",
        "stck_ruse_psbl_ord_psbl_amt": "0.0000",
        "stck_fund_rpch_ord_psbl_amt": "0.0000",
        "bond_ruse_psbl_ord_psbl_amt": "0.0000",
        "rcvb_amt": "0",
        "stck_loan_grta_ruse_psbl_amt": "0.0000",
        "stck_cash20_max_ord_psbl_amt": "8128560.1990",
        "stck_cash30_max_ord_psbl_amt": "8128560.1990",
        "stck_cash40_max_ord_psbl_amt": "8128560.1990",
        "stck_cash50_max_ord_psbl_amt": "8128560.1990",
        "stck_cash60_max_ord_psbl_amt": "8128560.1990",
        "stck_cash100_max_ord_psbl_amt": "8128560.1990",
        "stck_rsip100_max_ord_psbl_amt": "8128560.1990",
        "bond_max_ord_psbl_amt": "9316675.9443",
        "stck_fncg45_max_ord_psbl_amt": "20942905.49",
        "stck_fncg50_max_ord_psbl_amt": "18869350.4950",
        "stck_fncg60_max_ord_psbl_amt": "15750449.5868",
        "stck_fncg70_max_ord_psbl_amt": "13516343.26",
        "stck_stln_max_ord_psbl_amt": "9307424.0318",
        "lmt_amt": "0",
        "ovrs_stck_itgr_mgna_dvsn_name": "",
        "usd_objt_amt": "0.00",
        "usd_use_amt": "0.00",
        "usd_ord_psbl_amt": "0.00",
        "hkd_objt_amt": "0.00",
        "hkd_use_amt": "0.00",
        "hkd_ord_psbl_amt": "0.00",
        "jpy_objt_amt": "0.00",
        "jpy_use_amt": "0.00",
        "jpy_ord_psbl_amt": "0.00",
        "cny_objt_amt": "0.00",
        "cny_use_amt": "0.00",
        "cny_ord_psbl_amt": "0.00",
        "usd_ruse_objt_amt": "0.00",
        "usd_ruse_amt": "0.00",
        "usd_ruse_ord_psbl_amt": "0.00",
        "hkd_ruse_objt_amt": "0.00",
        "hkd_ruse_amt": "0.00",
        "hkd_ruse_ord_psbl_amt": "0.00",
        "jpy_ruse_objt_amt": "0.00",
        "jpy_ruse_amt": "0.00",
        "jpy_ruse_ord_psbl_amt": "0.00",
        "cny_ruse_objt_amt": "0.00",
        "cny_ruse_amt": "0.00",
        "cny_ruse_ord_psbl_amt": "0.00",
        "usd_gnrl_ord_psbl_amt": "0.00",
        "usd_itgr_ord_psbl_amt": "0.00",
        "hkd_gnrl_ord_psbl_amt": "0.00",
        "hkd_itgr_ord_psbl_amt": "0.00",
        "jpy_gnrl_ord_psbl_amt": "0.00",
        "jpy_itgr_ord_psbl_amt": "0.00",
        "cny_gnrl_ord_psbl_amt": "0.00",
        "cny_itgr_ord_psbl_amt": "0.00",
        "stck_itgr_cash20_ord_psbl_amt": "0.00",
        "stck_itgr_cash30_ord_psbl_amt": "0.00",
        "stck_itgr_cash40_ord_psbl_amt": "0.00",
        "stck_itgr_cash50_ord_psbl_amt": "0.00",
        "stck_itgr_cash60_ord_psbl_amt": "0.00",
        "stck_itgr_cash100_ord_psbl_amt": "0.00",
        "stck_itgr_100_ord_psbl_amt": "0.00",
        "stck_itgr_fncg45_ord_psbl_amt": "0.00",
        "stck_itgr_fncg50_ord_psbl_amt": "0.00",
        "stck_itgr_fncg60_ord_psbl_amt": "0.00",
        "stck_itgr_fncg70_ord_psbl_amt": "0.00",
        "stck_itgr_stln_ord_psbl_amt": "0.00",
        "bond_itgr_ord_psbl_amt": "0.00",
        "stck_cash_ovrs_use_amt": "0.00",
        "stck_sbst_ovrs_use_amt": "0.00",
        "stck_evlu_ovrs_use_amt": "0.00",
        "stck_re_use_amt_ovrs_use_amt": "0.00",
        "stck_fund_rpch_ovrs_use_amt": "0.00",
        "stck_fncg_rdpt_ovrs_use_amt": "0.00",
        "bond_re_use_ovrs_use_amt": "0.00",
        "usd_oth_mket_use_amt": "0.00",
        "jpy_oth_mket_use_amt": "0.00",
        "cny_oth_mket_use_amt": "0.00",
        "hkd_oth_mket_use_amt": "0.00",
        "usd_re_use_oth_mket_use_amt": "0.00",
        "jpy_re_use_oth_mket_use_amt": "0.00",
        "cny_re_use_oth_mket_use_amt": "0.00",
        "hkd_re_use_oth_mket_use_amt": "0.00",
        "hgkg_cny_re_use_amt": "0.00",
        "hgkg_cny_re_use_objt_amt": "0.00",
        "hgkg_cny_re_use_ord_psbl_amt": "0.00",
        "hgkg_cny_re_use_oth_use_amt": "0.00"
        "hgkg_cny_re_use_oth_use_amt": "0.00",
        "usd_frst_bltn_exrt": "1467.00000000",
        "hkd_frst_bltn_exrt": "188.61000000",
        "jpy_frst_bltn_exrt": "10.06000000",
        "cny_frst_bltn_exrt": "200.70000000"
    },
    "rt_cd": "0",
    "msg_cd": "KIOK0510",
    "msg1": "조회가 완료되었습니다                                                           "
}
```
