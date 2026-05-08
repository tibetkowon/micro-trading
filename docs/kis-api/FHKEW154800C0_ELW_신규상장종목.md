# ELW 신규상장종목

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `ELW 신규상장종목` |
| API ID | `국내주식-181` |
| 실전 TR_ID | `FHKEW154800C0` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/elw/v1/quotations/newly-listed` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> ELW 신규상장종목 API입니다. 
한국투자 HTS(eFriend Plus) &gt; [0297] ELW 신규상장종목 화면의 기능을 API로 개발한 사항으로, 해당 화면을 참고하시면 기능을 이해하기 쉽습니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `FID_COND_MRKT_DIV_CODE` | 조건시장분류코드 | string | Y | 시장구분코드 (W) |
| `FID_COND_SCR_DIV_CODE` | 조건화면분류코드 | string | Y | Unique key(11548) |
| `FID_DIV_CLS_CODE` | 분류구분코드 | string | Y | 전체(02), 콜(00), 풋(01) |
| `FID_UNAS_INPUT_ISCD` | 기초자산입력종목코드 | string | Y | 'ex) 000000(전체), 2001(코스피200); , 3003(코스닥150), 005930(삼성전자) ' |
| `FID_INPUT_ISCD_2` | 입력종목코드2 | string | Y | '00003(한국투자증권), 00017(KB증권),; 00005(미래에셋증권)' |
| `FID_INPUT_DATE_1` | 입력날짜1 | string | Y | 날짜 (ex) 20240402) |
| `FID_BLNC_CLS_CODE` | 결재방법 | string | Y | 0(전체), 1(일반), 2(조기종료) |

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
| `output` | 응답상세 | object array | Y | array |
| `stck_lstn_date` | 주식상장일자 | string | Y |  |
| `elw_kor_isnm` | ELW한글종목명 | string | Y |  |
| `elw_shrn_iscd` | ELW단축종목코드 | string | Y |  |
| `unas_isnm` | 기초자산종목명 | string | Y |  |
| `pblc_co_name` | 발행회사명 | string | Y |  |
| `lstn_stcn` | 상장주수 | string | Y |  |
| `acpr` | 행사가 | string | Y |  |
| `stck_last_tr_date` | 주식최종거래일자 | string | Y |  |
| `elw_ko_barrier` | 조기종료발생기준가격 | string | Y |  |

## Example

### Request Example (Python)
```json
FID_COND_MRKT_DIV_CODE:W
FID_COND_SCR_DIV_CODE:11548
FID_DIV_CLS_CODE:02
FID_UNAS_INPUT_ISCD:000000
FID_INPUT_ISCD_2:00003
FID_INPUT_DATE_1:20240410
FID_BLNG_CLS_CODE:0
```

### Response Example
```json
{
    "output": [
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국K924HLB콜",
            "elw_shrn_iscd": "57K924",
            "unas_isnm": "HLB",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "7100000",
            "acpr": "78000.00",
            "stck_last_tr_date": "20240613",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국K925HMM콜",
            "elw_shrn_iscd": "57K925",
            "unas_isnm": "HMM",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "6700000",
            "acpr": "20000.00",
            "stck_last_tr_date": "20240912",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국K926HMM콜",
            "elw_shrn_iscd": "57K926",
            "unas_isnm": "HMM",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "5600000",
            "acpr": "20000.00",
            "stck_last_tr_date": "20241212",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국KB45HMM풋",
            "elw_shrn_iscd": "57KB45",
            "unas_isnm": "HMM",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "6900000",
            "acpr": "17700.00",
            "stck_last_tr_date": "20240613",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국K927KB금융콜",
            "elw_shrn_iscd": "57K927",
            "unas_isnm": "KB금융",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "24400000",
            "acpr": "73600.00",
            "stck_last_tr_date": "20240613",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국K928KB금융콜",
            "elw_shrn_iscd": "57K928",
            "unas_isnm": "KB금융",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "22300000",
            "acpr": "73600.00",
            "stck_last_tr_date": "20240912",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국K929KB금융콜",
            "elw_shrn_iscd": "57K929",
            "unas_isnm": "KB금융",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "18200000",
            "acpr": "72000.00",
            "stck_last_tr_date": "20240711",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국K930KB금융콜",
            "elw_shrn_iscd": "57K930",
            "unas_isnm": "KB금융",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "20000000",
            "acpr": "70500.00",
            "stck_last_tr_date": "20240613",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국K931KB금융콜",
            "elw_shrn_iscd": "57K931",
            "unas_isnm": "KB금융",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "15700000",
            "acpr": "69000.00",
            "stck_last_tr_date": "20240711",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국KB46KB금융풋",
            "elw_shrn_iscd": "57KB46",
            "unas_isnm": "KB금융",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "11200000",
            "acpr": "65000.00",
            "stck_last_tr_date": "20240613",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국KB47KB금융풋",
            "elw_shrn_iscd": "57KB47",
            "unas_isnm": "KB금융",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "12000000",
            "acpr": "63700.00",
            "stck_last_tr_date": "20240711",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국KB98KOSPI200풋",
            "elw_shrn_iscd": "57KB98",
            "unas_isnm": "KOSPI200",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "7000000",
            "acpr": "400.00",
            "stck_last_tr_date": "20240613",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국KB99KOSPI200풋",
            "elw_shrn_iscd": "57KB99",
            "unas_isnm": "KOSPI200",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "7000000",
            "acpr": "395.00",
            "stck_last_tr_date": "20240613",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국KC00KOSPI200풋",
            "elw_shrn_iscd": "57KC00",
            "unas_isnm": "KOSPI200",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "7000000",
            "acpr": "390.00",
            "stck_last_tr_date": "20240613",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국KC01KOSPI200풋",
            "elw_shrn_iscd": "57KC01",
            "unas_isnm": "KOSPI200",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "9000000",
            "acpr": "387.50",
            "stck_last_tr_date": "20240613",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국KC02KOSPI200풋",
            "elw_shrn_iscd": "57KC02",
            "unas_isnm": "KOSPI200",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "10000000",
            "acpr": "385.00",
            "stck_last_tr_date": "20240613",
            "elw_ko_barrier": "0.00"
        },
        {
            "stck_lstn_date": "20240320",
            "elw_kor_isnm": "한국KC03KOSPI200풋",
            "elw_shrn_iscd": "57KC03",
            "unas_isnm": "KOSPI200",
            "pblc_co_name": "한국투자증권(주)",
            "lstn_stcn": "9000000",
            "acpr": "382.50",
            "stck_last_tr_date": "20240613",
            "elw_ko_barrier": "0.00"
        },...
    ],
    "rt_cd": "0",
    "msg_cd": "MCA00000",
    "msg1": "정상처리 되었습니다."
}
```
