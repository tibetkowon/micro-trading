# 주식기본조회

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `주식기본조회` |
| API ID | `v1_국내주식-067` |
| 실전 TR_ID | `CTPF1002R` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/quotations/search-stock-info` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> 주식기본조회 API입니다.
국내주식 종목의 종목상세정보를 확인할 수 있습니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `PRDT_TYPE_CD` | 상품유형코드 | string | Y | 300: 주식, ETF, ETN, ELW ; 301 : 선물옵션 ; 302 : 채권 ; 306 : ELS' |
| `PDNO` | 상품번호 | string | Y | 종목번호 (6자리); ETN의 경우, Q로 시작 (EX. Q500001) |

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
| `output` | 응답상세1 | object | Y |  |
| `pdno` | 상품번호 | string | Y |  |
| `prdt_type_cd` | 상품유형코드 | string | Y |  |
| `mket_id_cd` | 시장ID코드 | string | Y | AGR.농축산물파생 → [공통코드](_공통코드.md#mket_id_cd) 참조 |
| `scty_grp_id_cd` | 증권그룹ID코드 | string | Y | BC.수익증권 → [공통코드](_공통코드.md#scty_grp_id_cd) 참조 |
| `excg_dvsn_cd` | 거래소구분코드 | string | Y | 01.한국증권 → [공통코드](_공통코드.md#excg_dvsn_cd) 참조 |
| `setl_mmdd` | 결산월일 | string | Y |  |
| `lstg_stqt` | 상장주수 | string | Y |  |
| `lstg_cptl_amt` | 상장자본금액 | string | Y |  |
| `cpta` | 자본금 | string | Y |  |
| `papr` | 액면가 | string | Y |  |
| `issu_pric` | 발행가격 | string | Y |  |
| `kospi200_item_yn` | 코스피200종목여부 | string | Y |  |
| `scts_mket_lstg_dt` | 유가증권시장상장일자 | string | Y |  |
| `scts_mket_lstg_abol_dt` | 유가증권시장상장폐지일자 | string | Y |  |
| `kosdaq_mket_lstg_dt` | 코스닥시장상장일자 | string | Y |  |
| `kosdaq_mket_lstg_abol_dt` | 코스닥시장상장폐지일자 | string | Y |  |
| `frbd_mket_lstg_dt` | 프리보드시장상장일자 | string | Y |  |
| `frbd_mket_lstg_abol_dt` | 프리보드시장상장폐지일자 | string | Y |  |
| `reits_kind_cd` | 리츠종류코드 | string | Y |  |
| `etf_dvsn_cd` | ETF구분코드 | string | Y |  |
| `oilf_fund_yn` | 유전펀드여부 | string | Y |  |
| `idx_bztp_lcls_cd` | 지수업종대분류코드 | string | Y |  |
| `idx_bztp_mcls_cd` | 지수업종중분류코드 | string | Y |  |
| `idx_bztp_scls_cd` | 지수업종소분류코드 | string | Y |  |
| `stck_kind_cd` | 주식종류코드 | string | Y | 000.해당사항없음 → [공통코드](_공통코드.md#stck_kind_cd) 참조 |
| `mfnd_opng_dt` | 뮤추얼펀드개시일자 | string | Y |  |
| `mfnd_end_dt` | 뮤추얼펀드종료일자 | string | Y |  |
| `dpsi_erlm_cncl_dt` | 예탁등록취소일자 | string | Y |  |
| `etf_cu_qty` | ETFCU수량 | string | Y |  |
| `prdt_name` | 상품명 | string | Y |  |
| `prdt_name120` | 상품명120 | string | Y |  |
| `prdt_abrv_name` | 상품약어명 | string | Y |  |
| `std_pdno` | 표준상품번호 | string | Y |  |
| `prdt_eng_name` | 상품영문명 | string | Y |  |
| `prdt_eng_name120` | 상품영문명120 | string | Y |  |
| `prdt_eng_abrv_name` | 상품영문약어명 | string | Y |  |
| `dpsi_aptm_erlm_yn` | 예탁지정등록여부 | string | Y |  |
| `etf_txtn_type_cd` | ETF과세유형코드 | string | Y |  |
| `etf_type_cd` | ETF유형코드 | string | Y |  |
| `lstg_abol_dt` | 상장폐지일자 | string | Y |  |
| `nwst_odst_dvsn_cd` | 신주구주구분코드 | string | Y |  |
| `sbst_pric` | 대용가격 | string | Y |  |
| `thco_sbst_pric` | 당사대용가격 | string | Y |  |
| `thco_sbst_pric_chng_dt` | 당사대용가격변경일자 | string | Y |  |
| `tr_stop_yn` | 거래정지여부 | string | Y |  |
| `admn_item_yn` | 관리종목여부 | string | Y |  |
| `thdt_clpr` | 당일종가 | string | Y |  |
| `bfdy_clpr` | 전일종가 | string | Y |  |
| `clpr_chng_dt` | 종가변경일자 | string | Y |  |
| `std_idst_clsf_cd` | 표준산업분류코드 | string | Y |  |
| `std_idst_clsf_cd_name` | 표준산업분류코드명 | string | Y | 표준산업소분류코드 → [공통코드](_공통코드.md#std_idst_clsf_cd_name) 참조 |
| `idx_bztp_lcls_cd_name` | 지수업종대분류코드명 | string | Y | 표준산업대분류코드 → [공통코드](_공통코드.md#idx_bztp_lcls_cd_name) 참조 |
| `idx_bztp_mcls_cd_name` | 지수업종중분류코드명 | string | Y | 표준산업중분류코드 → [공통코드](_공통코드.md#idx_bztp_mcls_cd_name) 참조 |
| `idx_bztp_scls_cd_name` | 지수업종소분류코드명 | string | Y | 표준산업소분류코드 참조 |
| `ocr_no` | OCR번호 | string | Y |  |
| `crfd_item_yn` | 크라우드펀딩종목여부 | string | Y |  |
| `elec_scty_yn` | 전자증권여부 | string | Y |  |
| `issu_istt_cd` | 발행기관코드 | string | Y |  |
| `etf_chas_erng_rt_dbnb` | ETF추적수익율배수 | string | Y |  |
| `etf_etn_ivst_heed_item_yn` | ETFETN투자유의종목여부 | string | Y |  |
| `stln_int_rt_dvsn_cd` | 대주이자율구분코드 | string | Y |  |
| `frnr_psnl_lmt_rt` | 외국인개인한도비율 | string | Y |  |
| `lstg_rqsr_issu_istt_cd` | 상장신청인발행기관코드 | string | Y |  |
| `lstg_rqsr_item_cd` | 상장신청인종목코드 | string | Y |  |
| `trst_istt_issu_istt_cd` | 신탁기관발행기관코드 | string | Y |  |
| `cptt_trad_tr_psbl_yn` | NXT 거래종목여부 | string | Y | NXT 거래가능한 종목은 Y, 그 외 종목은 N |
| `nxt_tr_stop_yn` | NXT 거래정지여부 | string | Y | NXT 거래종목 중 거래정지가 된 종목은 Y, 그 외 모든 종목은 N |

## Example

### Request Example (Python)
```json
{
"PDNO":"000660",
"PRDT_TYPE_CD":"300"
}
```

### Response Example
```json
{
    "output": {
        "pdno": "00000A000660",
        "prdt_type_cd": "300",
        "mket_id_cd": "STK",
        "scty_grp_id_cd": "ST",
        "excg_dvsn_cd": "02",
        "setl_mmdd": "12",
        "lstg_stqt": "728002365",
        "lstg_cptl_amt": "0",
        "cpta": "3657652050000",
        "papr": "5000",
        "issu_pric": "5000",
        "kospi200_item_yn": "Y",
        "scts_mket_lstg_dt": "19961226",
        "scts_mket_lstg_abol_dt": "",
        "kosdaq_mket_lstg_dt": "",
        "kosdaq_mket_lstg_abol_dt": "",
        "frbd_mket_lstg_dt": "19961226",
        "frbd_mket_lstg_abol_dt": "",
        "reits_kind_cd": "",
        "etf_dvsn_cd": "0",
        "oilf_fund_yn": "N",
        "idx_bztp_lcls_cd": "002",
        "idx_bztp_mcls_cd": "013",
        "idx_bztp_scls_cd": "013",
        "stck_kind_cd": "101",
        "mfnd_opng_dt": "",
        "mfnd_end_dt": "",
        "dpsi_erlm_cncl_dt": "",
        "etf_cu_qty": "0",
        "prdt_name": "에스케이하이닉스보통주",
        "prdt_name120": "에스케이하이닉스보통주",
        "prdt_abrv_name": "SK하이닉스",
        "std_pdno": "KR7000660001",
        "prdt_eng_name": "SK hynix",
        "prdt_eng_name120": "SK hynix",
        "prdt_eng_abrv_name": "SK hynix",
        "dpsi_aptm_erlm_yn": "Y",
        "etf_txtn_type_cd": "00",
        "etf_type_cd": "",
        "lstg_abol_dt": "",
        "nwst_odst_dvsn_cd": "1",
        "sbst_pric": "115980",
        "thco_sbst_pric": "115980",
        "thco_sbst_pric_chng_dt": "20240215",
        "tr_stop_yn": "N",
        "admn_item_yn": "N",
        "thdt_clpr": "146800",
        "bfdy_clpr": "148700",
        "clpr_chng_dt": "20240216",
        "std_idst_clsf_cd": "032601",
        "std_idst_clsf_cd_name": "반도체 제조업",
        "idx_bztp_lcls_cd_name": "시가총액규모대",
        "idx_bztp_mcls_cd_name": "전기,전자",
        "idx_bztp_scls_cd_name": "전기,전자",
        "ocr_no": "1147",
        "crfd_item_yn": "N",
        "elec_scty_yn": "Y"
    },
    "rt_cd": "0",
    "msg_cd": "KIOK0530",
    "msg1": "조회되었습니다                                                                  "
}
```
