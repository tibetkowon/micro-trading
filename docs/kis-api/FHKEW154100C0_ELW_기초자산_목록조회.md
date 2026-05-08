# ELW 기초자산 목록조회

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `ELW 기초자산 목록조회` |
| API ID | `국내주식-185` |
| 실전 TR_ID | `FHKEW154100C0` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/elw/v1/quotations/udrl-asset-list` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> ELW 기초자산 목록조회 API입니다.
한국투자 HTS(eFriend Plus) &gt; [0288] ELW 기초자산별 ELW 시세 화면 의 "왼쪽 기초자산 목록" 기능을 API로 개발한 사항으로, 해당 화면을 참고하시면 기능을 이해하기 쉽습니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `FID_COND_SCR_DIV_CODE` | 조건화면분류코드 | string | Y | 11541(Primary key) |
| `FID_RANK_SORT_CLS_CODE` | 순위정렬구분코드 | string | Y | 0(종목명순), 1(콜발행종목순), 2(풋발행종목순), 3(전일대비 상승율순), 4(전일대비 하락율순), 5(현재가 크기순), 6(종목코드순) |
| `FID_INPUT_ISCD` | 입력종목코드 | string | Y | 00000(전체), 00003(한국투자증권), 00017(KB증권), 00005(미래에셋) |

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
| `unas_shrn_iscd` | 기초자산단축종목코드 | string | Y |  |
| `unas_isnm` | 기초자산종목명 | string | Y |  |
| `unas_prpr` | 기초자산현재가 | string | Y |  |
| `unas_prdy_vrss` | 기초자산전일대비 | string | Y |  |
| `unas_prdy_vrss_sign` | 기초자산전일대비부호 | string | Y |  |
| `unas_prdy_ctrt` | 기초자산전일대비율 | string | Y |  |

## Example

### Request Example (Python)
```json
FID_COND_SCR_DIV_CODE:11541
FID_RANK_SORT_CLS_CODE:0
FID_INPUT_ISCD:00000
```

### Response Example
```json
{
    "output": [
        {
            "unas_shrn_iscd": "2001",
            "unas_isnm": "KOSPI200",
            "unas_prpr": "371.33",
            "unas_prdy_vrss": "0.17",
            "unas_prdy_vrss_sign": "2",
            "unas_prdy_ctrt": "0.05"
        },
        {
            "unas_shrn_iscd": "000990",
            "unas_isnm": "DB하이텍",
            "unas_prpr": "40850.00",
            "unas_prdy_vrss": "-300.00",
            "unas_prdy_vrss_sign": "5",
            "unas_prdy_ctrt": "-0.73"
        },
        {
            "unas_shrn_iscd": "009540",
            "unas_isnm": "HD한국조선해양",
            "unas_prpr": "135400.00",
            "unas_prdy_vrss": "1100.00",
            "unas_prdy_vrss_sign": "2",
            "unas_prdy_ctrt": "0.82"
        },
        {
            "unas_shrn_iscd": "267260",
            "unas_isnm": "HD현대일렉트릭",
            "unas_prpr": "302500.00",
            "unas_prdy_vrss": "9000.00",
            "unas_prdy_vrss_sign": "2",
            "unas_prdy_ctrt": "3.07"
        },
        {
            "unas_shrn_iscd": "028300",
            "unas_isnm": "HLB",
            "unas_prpr": "64700.00",
            "unas_prdy_vrss": "8500.00",
            "unas_prdy_vrss_sign": "2",
            "unas_prdy_ctrt": "15.12"
        },
        {
            "unas_shrn_iscd": "011200",
            "unas_isnm": "HMM",
            "unas_prpr": "18010.00",
            "unas_prdy_vrss": "460.00",
            "unas_prdy_vrss_sign": "2",
            "unas_prdy_ctrt": "2.62"
        },
        {
            "unas_shrn_iscd": "403870",
            "unas_isnm": "HPSP",
            "unas_prpr": "45200.00",
            "unas_prdy_vrss": "2900.00",
            "unas_prdy_vrss_sign": "2",
            "unas_prdy_ctrt": "6.86"
        },
        {
            "unas_shrn_iscd": "035900",
            "unas_isnm": "JYP Ent.",
            "unas_prpr": "58800.00",
            "unas_prdy_vrss": "-1700.00",
            "unas_prdy_vrss_sign": "5",
            "unas_prdy_ctrt": "-2.81"
        },
        {
            "unas_shrn_iscd": "105560",
            "unas_isnm": "KB금융",
            "unas_prpr": "77100.00",
            "unas_prdy_vrss": "800.00",
            "unas_prdy_vrss_sign": "2",
            "unas_prdy_ctrt": "1.05"
        },
        {
            "unas_shrn_iscd": "3003",
            "unas_isnm": "KSQ150",
            "unas_prpr": "1355.15",
            "unas_prdy_vrss": "0.44",
            "unas_prdy_vrss_sign": "2",
            "unas_prdy_ctrt": "0.03"
        },
        {
            "unas_shrn_iscd": "030200",
            "unas_isnm": "KT",
            "unas_prpr": "36150.00",
            "unas_prdy_vrss": "-450.00",
            "unas_prdy_vrss_sign": "5",
            "unas_prdy_ctrt": "-1.23"
        },
        {
            "unas_shrn_iscd": "033780",
            "unas_isnm": "KT&G",
            "unas_prpr": "86100.00",
            "unas_prdy_vrss": "-100.00",
            "unas_prdy_vrss_sign": "5",
            "unas_prdy_ctrt": "-0.12"
        },
        {
            "unas_shrn_iscd": "003550",
            "unas_isnm": "LG",
            "unas_prpr": "81400.00",
            "unas_prdy_vrss": "1500.00",
            "unas_prdy_vrss_sign": "2",
            "unas_prdy_ctrt": "1.88"
        },...
    ],
    "rt_cd": "0",
    "msg_cd": "MCA00000",
    "msg1": "정상처리 되었습니다."
}
```
