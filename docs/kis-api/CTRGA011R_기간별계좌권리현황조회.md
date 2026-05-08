# 기간별계좌권리현황조회

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `기간별계좌권리현황조회` |
| API ID | `국내주식-211` |
| 실전 TR_ID | `CTRGA011R` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/trading/period-rights` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> 기간별계좌권리현황조회 API입니다.
한국투자 HTS(eFriend Plus) &gt; [7344] 권리유형별 현황조회 화면을 API로 개발한 사항으로, 해당 화면을 참고하시면 기능을 이해하기 쉽습니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `INQR_DVSN` | 조회구분 | string | Y | 03 입력 |
| `CUST_RNCNO25` | 고객실명확인번호25 | string | Y | 공란 |
| `HMID` | 홈넷ID | string | Y | 공란 |
| `CANO` | 종합계좌번호 | string | Y | 계좌번호 8자리 입력 (ex.12345678) |
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y | 상품계좌번호 2자리 입력(ex. 01 or 22) |
| `INQR_STRT_DT` | 조회시작일자 | string | Y | 조회시작일자(YYYYMMDD) |
| `INQR_END_DT` | 조회종료일자 | string | Y | 조회종료일자(YYYYMMDD) |
| `RGHT_TYPE_CD` | 권리유형코드 | string | Y | 공란 |
| `PDNO` | 상품번호 | string | Y | 공란 |
| `PRDT_TYPE_CD` | 상품유형코드 | string | Y | 공란 |
| `CTX_AREA_NK100` | 연속조회키100 | string | Y | 다음조회시 입력 |
| `CTX_AREA_FK100` | 연속조회검색조건100 | string | Y | 다음조회시 입력 |

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
| `acno10` | 계좌번호10 | string | Y |  |
| `rght_type_cd` | 권리유형코드 | string | Y | 1	유상 → [공통코드](_공통코드.md#rght_type_cd) 참조 |
| `bass_dt` | 기준일자 | string | Y |  |
| `rght_cblc_type_cd` | 권리잔고유형코드 | string | Y | 1	입고 → [공통코드](_공통코드.md#rght_cblc_type_cd) 참조 |
| `rptt_pdno` | 대표상품번호 | string | Y |  |
| `pdno` | 상품번호 | string | Y |  |
| `prdt_type_cd` | 상품유형코드 | string | Y |  |
| `shtn_pdno` | 단축상품번호 | string | Y |  |
| `prdt_name` | 상품명 | string | Y |  |
| `cblc_qty` | 잔고수량 | string | Y |  |
| `last_alct_qty` | 최종배정수량 | string | Y |  |
| `excs_alct_qty` | 초과배정수량 | string | Y |  |
| `tot_alct_qty` | 총배정수량 | string | Y |  |
| `last_ftsk_qty` | 최종단수주수량 | string | Y |  |
| `last_alct_amt` | 최종배정금액 | string | Y |  |
| `last_ftsk_chgs` | 최종단수주대금 | string | Y |  |
| `rdpt_prca` | 상환원금 | string | Y |  |
| `dlay_int_amt` | 지연이자금액 | string | Y |  |
| `lstg_dt` | 상장일자 | string | Y |  |
| `sbsc_end_dt` | 청약종료일자 | string | Y |  |
| `cash_dfrm_dt` | 현금지급일자 | string | Y |  |
| `rqst_qty` | 신청수량 | string | Y |  |
| `rqst_amt` | 신청금액 | string | Y |  |
| `rqst_dt` | 신청일자 | string | Y |  |
| `rfnd_dt` | 환불일자 | string | Y |  |
| `rfnd_amt` | 환불금액 | string | Y |  |
| `lstg_stqt` | 상장주수 | string | Y |  |
| `tax_amt` | 세금금액 | string | Y |  |
| `sbsc_unpr` | 청약단가 | string | Y |  |

## Example

### Request Example (Python)
```json
INQR_DVSN:03
CUST_RNCNO25:
HMID:
CANO:12345678
ACNT_PRDT_CD:01
INQR_STRT_DT:20240508
INQR_END_DT:20241106
RGHT_TYPE_CD:
PDNO:
PRDT_TYPE_CD:
CTX_AREA_NK100:
CTX_AREA_FK100:
```

### Response Example
```json
{
    "ctx_area_nk100": "                                                                                                    ",
    "ctx_area_fk100": "03!^!^!^12345678!^01!^20240508!^20241106!^!^!^                                                      ",
    "output": [
        {
            "acno10": "1234567801",
            "rght_type_cd": "01",
            "bass_dt": "20240919",
            "rght_cblc_type_cd": "01",
            "rptt_pdno": "00000A357880",
            "pdno": "00000A357880",
            "prdt_type_cd": "300",
            "shtn_pdno": "357880",
            "prdt_name": "비트나인",
            "cblc_qty": "1000",
            "last_alct_qty": "1050",
            "excs_alct_qty": "0",
            "tot_alct_qty": "1050",
            "last_ftsk_qty": "0.0000000000",
            "last_alct_amt": "0",
            "last_ftsk_chgs": "0",
            "rdpt_prca": "0",
            "dlay_int_amt": "0",
            "lstg_dt": "",
            "sbsc_end_dt": "20241011",
            "cash_dfrm_dt": "",
            "rqst_qty": "1000",
            "rqst_amt": "1865000",
            "rqst_dt": "20241011",
            "rfnd_dt": "",
            "rfnd_amt": "0",
            "lstg_stqt": "0",
            "tax_amt": "0",
            "sbsc_unpr": "1865.0000"
        }
    ],
    "rt_cd": "0",
    "msg_cd": "KIOK0460",
    "msg1": "조회 되었습니다. (마지막 자료)                                                  "
}
```
