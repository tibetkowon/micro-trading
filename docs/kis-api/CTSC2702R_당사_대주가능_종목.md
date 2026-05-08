# 당사 대주가능 종목

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `당사 대주가능 종목` |
| API ID | `국내주식-195` |
| 실전 TR_ID | `CTSC2702R` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/quotations/lendable-by-company` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `미지원`

> 당사 대주가능 종목 API입니다. 
한국투자 HTS(eFriend Plus) &gt; [0490] 당사 대주가능 종목 화면의 기능을 API로 개발한 사항으로, 해당 화면을 참고하시면 기능을 이해하기 쉽습니다.

※ 본 API는 다음조회가 불가합니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `EXCG_DVSN_CD` | 거래소구분코드 | string | Y | 00(전체), 02(거래소), 03(코스닥) |
| `PDNO` | 상품번호 | string | Y | 공백 : 전체조회, 종목코드 입력 시 해당종목만 조회 |
| `THCO_STLN_PSBL_YN` | 당사대주가능여부 | string | Y | Y |
| `INQR_DVSN_1` | 조회구분1 | string | Y | 0 : 전체조회, 1: 종목코드순 정렬 |
| `CTX_AREA_FK200` | 연속조회검색조건200 | string | Y | 미입력 (다음조회 불가) |
| `CTX_AREA_NK100` | 연속조회키100 | string | Y | 미입력 (다음조회 불가) |

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
| `output1` | 응답상세 | object array | Y | array |
| `pdno` | 상품번호 | string | Y |  |
| `prdt_name` | 상품명 | string | Y |  |
| `papr` | 액면가 | string | Y |  |
| `bfdy_clpr` | 전일종가 | string | Y | 전일종가 |
| `sbst_prvs` | 대용가 | string | Y |  |
| `tr_stop_dvsn_name` | 거래정지구분명 | string | Y |  |
| `psbl_yn_name` | 가능여부명 | string | Y |  |
| `lmt_qty1` | 한도수량1 | string | Y |  |
| `use_qty1` | 사용수량1 | string | Y |  |
| `trad_psbl_qty2` | 매매가능수량2 | string | Y | 가능수량 |
| `rght_type_cd` | 권리유형코드 | string | Y |  |
| `bass_dt` | 기준일자 | string | Y |  |
| `psbl_yn` | 가능여부 | string | Y |  |
| `output2` | 응답상세 | object | Y |  |
| `tot_stup_lmt_qty` | 총설정한도수량 | string | Y |  |
| `brch_lmt_qty` | 지점한도수량 | string | Y |  |
| `rqst_psbl_qty` | 신청가능수량 | string | Y |  |

## Example

### Request Example (Python)
```json
EXCG_DVSN_CD:00
PDNO:
THCO_STLN_PSBL_YN:Y
INQR_DVSN_1:0
CTX_AREA_FK200:
CTX_AREA_NK100:
```

### Response Example
```json
{
    "ctx_area_fk200": "00!^!^Y!^0                                                                                                                                                                                              ",
    "ctx_area_nk100": "                                                                                                    ",
    "output1": [
        {
            "pdno": "130960",
            "prdt_name": "CJ E&M",
            "papr": "5000",
            "bfdy_clpr": "0",
            "sbst_prvs": "0",
            "tr_stop_dvsn_name": "거래정지",
            "psbl_yn_name": "가능",
            "lmt_qty1": "10520",
            "use_qty1": "0",
            "trad_psbl_qty2": "10520",
            "rght_type_cd": "11",
            "bass_dt": "20180629",
            "psbl_yn": "Y"
        },
        {
            "pdno": "110550",
            "prdt_name": "HIT 골드",
            "papr": "0",
            "bfdy_clpr": "0",
            "sbst_prvs": "0",
            "tr_stop_dvsn_name": "거래정지",
            "psbl_yn_name": "가능",
            "lmt_qty1": "0",
            "use_qty1": "0",
            "trad_psbl_qty2": "0",
            "rght_type_cd": "32",
            "bass_dt": "20111222",
            "psbl_yn": "Y"
        },
        {
            "pdno": "124090",
            "prdt_name": "HIT 보험",
            "papr": "0",
            "bfdy_clpr": "0",
            "sbst_prvs": "0",
            "tr_stop_dvsn_name": "거래정지",
            "psbl_yn_name": "가능",
            "lmt_qty1": "0",
            "use_qty1": "0",
            "trad_psbl_qty2": "0",
            "rght_type_cd": "32",
            "bass_dt": "20111219",
            "psbl_yn": "Y"
        },
        {
            "pdno": "002550",
            "prdt_name": "KB손해보험",
            "papr": "500",
            "bfdy_clpr": "0",
            "sbst_prvs": "0",
            "tr_stop_dvsn_name": "거래정지",
            "psbl_yn_name": "가능",
            "lmt_qty1": "0",
            "use_qty1": "0",
            "trad_psbl_qty2": "0",
            "rght_type_cd": "13",
            "bass_dt": "20170706",
            "psbl_yn": "Y"
        },
        {
            "pdno": "021960",
            "prdt_name": "KB캐피탈",
            "papr": "5000",
            "bfdy_clpr": "0",
            "sbst_prvs": "0",
            "tr_stop_dvsn_name": "거래정지",
            "psbl_yn_name": "가능",
            "lmt_qty1": "0",
            "use_qty1": "0",
            "trad_psbl_qty2": "0",
            "rght_type_cd": "13",
            "bass_dt": "20170706",
            "psbl_yn": "Y"
        },
        {
            "pdno": "105270",
            "prdt_name": "KINDEX 성장대형F15",
            "papr": "0",
            "bfdy_clpr": "0",
            "sbst_prvs": "0",
            "tr_stop_dvsn_name": "거래정지",
            "psbl_yn_name": "가능",
            "lmt_qty1": "0",
            "use_qty1": "0",
            "trad_psbl_qty2": "0",
            "rght_type_cd": "32",
            "bass_dt": "20140430",
            "psbl_yn": "Y"
        },...
        {
            "pdno": "003450",
            "prdt_name": "현대증권",
            "papr": "5000",
            "bfdy_clpr": "0",
            "sbst_prvs": "0",
            "tr_stop_dvsn_name": "거래정지",
            "psbl_yn_name": "가능",
            "lmt_qty1": "0",
            "use_qty1": "0",
            "trad_psbl_qty2": "0",
            "rght_type_cd": "13",
            "bass_dt": "20161018",
            "psbl_yn": "Y"
        }
    ],
    "output2": {
        "tot_stup_lmt_qty": "6441070",
        "brch_lmt_qty": "-1228",
        "rqst_psbl_qty": "6442095"
    },
    "rt_cd": "0",
    "msg_cd": "KIOK0460",
    "msg1": "조회 되었습니다. (마지막 자료)                                                  "
}
```
