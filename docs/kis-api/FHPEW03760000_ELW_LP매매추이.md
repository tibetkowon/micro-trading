# ELW LP매매추이

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `ELW LP매매추이` |
| API ID | `국내주식-182` |
| 실전 TR_ID | `FHPEW03760000` |
| Method | `GET` |
| URL | `/uapi/elw/v1/quotations/lp-trade-trend` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `미지원`

> ELW LP매매추이 API입니다.
한국투자 HTS(eFriend Plus) &gt; [0376] ELW LP매매추이 화면 의 기능을 API로 개발한 사항으로, 해당 화면을 참고하시면 기능을 이해하기 쉽습니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `FID_COND_MRKT_DIV_CODE` | 조건시장분류코드 | string | Y | 시장구분(W) |
| `FID_INPUT_ISCD` | 입력종목코드 | string | Y | 입력종목코드(ex 52K577(미래 K577KOSDAQ150콜) |

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
| `output1` | 응답상세 | object | Y |  |
| `elw_prpr` | ELW현재가 | string | Y |  |
| `prdy_vrss_sign` | 전일대비부호 | string | Y |  |
| `prdy_vrss` | 전일대비 | string | Y |  |
| `prdy_ctrt` | 전일대비율 | string | Y |  |
| `acml_vol` | 누적거래량 | string | Y |  |
| `prdy_vol` | 전일거래량 | string | Y |  |
| `stck_cnvr_rate` | 주식전환비율 | string | Y |  |
| `prit` | 패리티 | string | Y |  |
| `lvrg_val` | 레버리지값 | string | Y |  |
| `gear` | 기어링 | string | Y |  |
| `prls_qryr_rate` | 손익분기비율 | string | Y |  |
| `cfp` | 자본지지점 | string | Y |  |
| `invl_val` | 내재가치값 | string | Y |  |
| `tmvl_val` | 시간가치값 | string | Y |  |
| `acpr` | 행사가 | string | Y |  |
| `elw_ko_barrier` | 조기종료발생기준가격 | string | Y |  |
| `output2` | 응답상세 | object array | Y | array |
| `stck_bsop_date` | 주식영업일자 | string | Y |  |
| `elw_prpr` | ELW현재가 | string | Y |  |
| `prdy_vrss_sign` | 전일대비부호 | string | Y |  |
| `prdy_vrss` | 전일대비 | string | Y |  |
| `prdy_ctrt` | 전일대비율 | string | Y |  |
| `lp_seln_qty` | LP매도수량 | string | Y |  |
| `lp_seln_avrg_unpr` | LP매도평균단가 | string | Y |  |
| `lp_shnu_qty` | LP매수수량 | string | Y |  |
| `lp_shnu_avrg_unpr` | LP매수평균단가 | string | Y |  |
| `lp_hvol` | LP보유량 | string | Y |  |
| `lp_hldn_rate` | LP보유비율 | string | Y |  |
| `prsn_deal_qty` | 개인매매수량 | string | Y |  |
| `apprch_rate` | 접근도 | string | Y |  |

## Example

### Request Example (Python)
```json
FID_COND_MRKT_DIV_CODE:W
FID_INPUT_ISCD:57K281
```

### Response Example
```json
{
    "output1": {
        "elw_prpr": "40",
        "prdy_vrss_sign": "2",
        "prdy_vrss": "5",
        "prdy_ctrt": "14.29",
        "acml_vol": "320750",
        "prdy_vol": "114850",
        "stck_cnvr_rate": "0.010000",
        "prit": "103.35",
        "lvrg_val": "-12.130651",
        "gear": "19.3500",
        "prls_qryr_rate": "-1.8000",
        "cfp": "-1.7100",
        "invl_val": "27.00",
        "tmvl_val": "13.00",
        "acpr": "80000.00",
        "elw_ko_barrier": "0.00"
    },
    "output2": [
        {
            "stck_bsop_date": "20240516",
            "elw_prpr": "35",
            "prdy_vrss_sign": "3",
            "prdy_vrss": "0",
            "prdy_ctrt": "0.00",
            "lp_seln_qty": "30030",
            "lp_seln_avrg_unpr": "30",
            "lp_shnu_qty": "84810",
            "lp_shnu_avrg_unpr": "34",
            "lp_hvol": "7999900",
            "lp_hldn_rate": "99.99",
            "prsn_deal_qty": "10",
            "apprch_rate": "0.00"
        },
        {
            "stck_bsop_date": "20240514",
            "elw_prpr": "35",
            "prdy_vrss_sign": "5",
            "prdy_vrss": "-5",
            "prdy_ctrt": "-12.50",
            "lp_seln_qty": "73510",
            "lp_seln_avrg_unpr": "35",
            "lp_shnu_qty": "74440",
            "lp_shnu_avrg_unpr": "35",
            "lp_hvol": "7945120",
            "lp_hldn_rate": "99.31",
            "prsn_deal_qty": "1260",
            "apprch_rate": "0.00"
        },
        {
            "stck_bsop_date": "20240513",
            "elw_prpr": "40",
            "prdy_vrss_sign": "2",
            "prdy_vrss": "10",
            "prdy_ctrt": "33.33",
            "lp_seln_qty": "282010",
            "lp_seln_avrg_unpr": "36",
            "lp_shnu_qty": "277980",
            "lp_shnu_avrg_unpr": "36",
            "lp_hvol": "7944190",
            "lp_hldn_rate": "99.30",
            "prsn_deal_qty": "11140",
            "apprch_rate": "0.00"
        },
        {
            "stck_bsop_date": "20240510",
            "elw_prpr": "30",
            "prdy_vrss_sign": "2",
            "prdy_vrss": "5",
            "prdy_ctrt": "20.00",
            "lp_seln_qty": "137480",
            "lp_seln_avrg_unpr": "27",
            "lp_shnu_qty": "209950",
            "lp_shnu_avrg_unpr": "25",
            "lp_hvol": "7948220",
            "lp_hldn_rate": "99.35",
            "prsn_deal_qty": "2040",
            "apprch_rate": "0.00"
        },
        {
            "stck_bsop_date": "20240509",
            "elw_prpr": "25",
            "prdy_vrss_sign": "3",
            "prdy_vrss": "0",
            "prdy_ctrt": "0.00",
            "lp_seln_qty": "280020",
            "lp_seln_avrg_unpr": "25",
            "lp_shnu_qty": "209910",
            "lp_shnu_avrg_unpr": "25",
            "lp_hvol": "7875750",
            "lp_hldn_rate": "98.44",
            "prsn_deal_qty": "120",
            "apprch_rate": "0.00"
        },
        {
            "stck_bsop_date": "20240508",
            "elw_prpr": "25",
            "prdy_vrss_sign": "3",
            "prdy_vrss": "0",
            "prdy_ctrt": "0.00",
            "lp_seln_qty": "630000",
            "lp_seln_avrg_unpr": "25",
            "lp_shnu_qty": "630000",
            "lp_shnu_avrg_unpr": "25",
            "lp_hvol": "7945860",
            "lp_hldn_rate": "99.32",
            "prsn_deal_qty": "10000",
            "apprch_rate": "0.00"
        },
        {
            "stck_bsop_date": "20240507",
            "elw_prpr": "25",
            "prdy_vrss_sign": "5",
            "prdy_vrss": "-20",
            "prdy_ctrt": "-44.44",
            "lp_seln_qty": "98550",
            "lp_seln_avrg_unpr": "27",
            "lp_shnu_qty": "160420",
            "lp_shnu_avrg_unpr": "28",
            "lp_hvol": "7945860",
            "lp_hldn_rate": "99.32",
            "prsn_deal_qty": "26200",
            "apprch_rate": "0.00"
        },
        {
            "stck_bsop_date": "20240503",
            "elw_prpr": "45",
            "prdy_vrss_sign": "3",
            "prdy_vrss": "0",
            "prdy_ctrt": "0.00",
            "lp_seln_qty": "501890",
            "lp_seln_avrg_unpr": "40",
            "lp_shnu_qty": "491690",
            "lp_shnu_avrg_unpr": "40",
            "lp_hvol": "7883990",
            "lp_hldn_rate": "98.55",
            "prsn_deal_qty": "8440",
            "apprch_rate": "0.00"
        },
        {
            "stck_bsop_date": "20240502",
            "elw_prpr": "45",
            "prdy_vrss_sign": "3",
            "prdy_vrss": "0",
            "prdy_ctrt": "0.00",
            "lp_seln_qty": "402940",
            "lp_seln_avrg_unpr": "40",
            "lp_shnu_qty": "332240",
            "lp_shnu_avrg_unpr": "40",
            "lp_hvol": "7894190",
            "lp_hldn_rate": "98.67",
            "prsn_deal_qty": "54100",
            "apprch_rate": "0.00"
        },
        {
            "stck_bsop_date": "20240430",
            "elw_prpr": "45",
            "prdy_vrss_sign": "5",
            "prdy_vrss": "-5",
            "prdy_ctrt": "-10.00",
            "lp_seln_qty": "27840",
            "lp_seln_avrg_unpr": "48",
            "lp_shnu_qty": "33540",
            "lp_shnu_avrg_unpr": "45",
            "lp_hvol": "7964890",
            "lp_hldn_rate": "99.56",
            "prsn_deal_qty": "710",
            "apprch_rate": "0.00"
        },
        {
            "stck_bsop_date": "20240429",
            "elw_prpr": "50",
            "prdy_vrss_sign": "3",
            "prdy_vrss": "0",
            "prdy_ctrt": "0.00",
            "lp_seln_qty": "211510",
            "lp_seln_avrg_unpr": "50",
            "lp_shnu_qty": "175810",
            "lp_shnu_avrg_unpr": "50",
            "lp_hvol": "7959190",
            "lp_hldn_rate": "99.49",
            "prsn_deal_qty": "15700",
            "apprch_rate": "0.00"
        },
        {
            "stck_bsop_date": "20240426",
            "elw_prpr": "50",
            "prdy_vrss_sign": "3",
            "prdy_vrss": "0",
            "prdy_ctrt": "0.00",
            "lp_seln_qty": "35700",
            "lp_seln_avrg_unpr": "50",
            "lp_shnu_qty": "91400",
            "lp_shnu_avrg_unpr": "48",
            "lp_hvol": "7994890",
            "lp_hldn_rate": "99.93",
            "prsn_deal_qty": "60",
            "apprch_rate": "0.00"
        },...
    ],
    "rt_cd": "0",
    "msg_cd": "MCA00000",
    "msg1": "정상처리 되었습니다."
}
```
