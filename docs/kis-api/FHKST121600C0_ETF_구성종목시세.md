# ETF 구성종목시세

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `ETF 구성종목시세` |
| API ID | `국내주식-073` |
| 실전 TR_ID | `FHKST121600C0` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/etfetn/v1/quotations/inquire-component-stock-price` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> ETF 구성종목시세 API입니다. 
한국투자 HTS(eFriend Plus) &gt; [0245] ETF/ETN 구성종목시세 화면의 기능을 API로 개발한 사항으로, 해당 화면을 참고하시면 기능을 이해하기 쉽습니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `FID_COND_MRKT_DIV_CODE` | 조건시장분류코드 | string | Y | 시장구분코드 (J) |
| `FID_INPUT_ISCD` | 입력종목코드 | string | Y | 종목코드 |
| `FID_COND_SCR_DIV_CODE` | 조건화면분류코드 | string | Y | Unique key( 11216 ) |

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
| `stck_prpr` | 주식 현재가 | string | Y |  |
| `prdy_vrss` | 전일 대비 | string | Y |  |
| `prdy_vrss_sign` | 전일 대비 부호 | string | Y |  |
| `prdy_ctrt` | 전일 대비율 | string | Y |  |
| `etf_cnfg_issu_avls` | ETF구성종목시가총액 | string | Y |  |
| `nav` | NAV | string | Y |  |
| `nav_prdy_vrss_sign` | NAV 전일 대비 부호 | string | Y |  |
| `nav_prdy_vrss` | NAV 전일 대비 | string | Y |  |
| `nav_prdy_ctrt` | NAV 전일 대비율 | string | Y |  |
| `etf_ntas_ttam` | ETF 순자산 총액 | string | Y |  |
| `prdy_clpr_nav` | NAV전일종가 | string | Y |  |
| `oprc_nav` | NAV시가 | string | Y |  |
| `hprc_nav` | NAV고가 | string | Y |  |
| `lprc_nav` | NAV저가 | string | Y |  |
| `etf_cu_unit_scrt_cnt` | ETF CU 단위 증권 수 | string | Y |  |
| `etf_cnfg_issu_cnt` | ETF 구성 종목 수 | string | Y |  |
| `output2` | 응답상세 | object array | Y | array |
| `stck_shrn_iscd` | 주식 단축 종목코드 | string | Y |  |
| `hts_kor_isnm` | HTS 한글 종목명 | string | Y |  |
| `stck_prpr` | 주식 현재가 | string | Y |  |
| `prdy_vrss` | 전일 대비 | string | Y |  |
| `prdy_vrss_sign` | 전일 대비 부호 | string | Y |  |
| `prdy_ctrt` | 전일 대비율 | string | Y |  |
| `acml_vol` | 누적 거래량 | string | Y |  |
| `acml_tr_pbmn` | 누적 거래 대금 | string | Y |  |
| `tday_rsfl_rate` | 당일 등락 비율 | string | Y |  |
| `prdy_vrss_vol` | 전일 대비 거래량 | string | Y |  |
| `tr_pbmn_tnrt` | 거래대금회전율 | string | Y |  |
| `hts_avls` | HTS 시가총액 | string | Y |  |
| `etf_cnfg_issu_avls` | ETF구성종목시가총액 | string | Y |  |
| `etf_cnfg_issu_rlim` | ETF구성종목비중 | string | Y |  |
| `etf_vltn_amt` | ETF구성종목내평가금액 | string | Y |  |

## Example

### Request Example (Python)
```json
fid_cond_mrkt_div_code:J
fid_input_iscd:069500
fid_cond_scr_div_code:11216
```

### Response Example
```json
{
    "output1": {
        "stck_prpr": "37195",
        "prdy_vrss": "-365",
        "prdy_vrss_sign": "5",
        "prdy_ctrt": "-0.97",
        "etf_cnfg_issu_avls": "184153",
        "nav": "37301.11",
        "nav_prdy_vrss_sign": "5",
        "nav_prdy_vrss": "-347.36",
        "nav_prdy_ctrt": "-0.92",
        "etf_ntas_ttam": "68256",
        "prdy_clpr_nav": "37648.47",
        "oprc_nav": "37653.39",
        "hprc_nav": "37720.17",
        "lprc_nav": "37223.93",
        "etf_cu_unit_scrt_cnt": "50000",
        "etf_cnfg_issu_cnt": "201"
    },
    "output2": [
        {
            "stck_shrn_iscd": "005930",
            "hts_kor_isnm": "삼성전자",
            "stck_prpr": "83700",
            "prdy_vrss": "-400",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-0.48",
            "acml_vol": "16967184",
            "acml_tr_pbmn": "1421776834400",
            "tday_rsfl_rate": "2.02",
            "prdy_vrss_vol": "-8570824",
            "tr_pbmn_tnrt": "0.28",
            "hts_avls": "4996708",
            "etf_cnfg_issu_avls": "601300800",
            "etf_cnfg_issu_rlim": "32.65",
            "etf_vltn_amt": "604174400"
        },
        {
            "stck_shrn_iscd": "000660",
            "hts_kor_isnm": "SK하이닉스",
            "stck_prpr": "187400",
            "prdy_vrss": "-1000",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-0.53",
            "acml_vol": "3042349",
            "acml_tr_pbmn": "575151315700",
            "tday_rsfl_rate": "2.34",
            "prdy_vrss_vol": "-1055882",
            "tr_pbmn_tnrt": "0.42",
            "hts_avls": "1364276",
            "etf_cnfg_issu_avls": "160039600",
            "etf_cnfg_issu_rlim": "8.69",
            "etf_vltn_amt": "160893600"
        },
        {
            "stck_shrn_iscd": "005380",
            "hts_kor_isnm": "현대차",
            "stck_prpr": "238000",
            "prdy_vrss": "-3000",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-1.24",
            "acml_vol": "993944",
            "acml_tr_pbmn": "237608070000",
            "tday_rsfl_rate": "1.87",
            "prdy_vrss_vol": "-859847",
            "tr_pbmn_tnrt": "0.47",
            "hts_avls": "503445",
            "etf_cnfg_issu_avls": "50694000",
            "etf_cnfg_issu_rlim": "2.75",
            "etf_vltn_amt": "51333000"
        },
        {
            "stck_shrn_iscd": "068270",
            "hts_kor_isnm": "셀트리온",
            "stck_prpr": "182200",
            "prdy_vrss": "2700",
            "prdy_vrss_sign": "2",
            "prdy_ctrt": "1.50",
            "acml_vol": "473566",
            "acml_tr_pbmn": "85712403800",
            "tday_rsfl_rate": "2.90",
            "prdy_vrss_vol": "-52048",
            "tr_pbmn_tnrt": "0.22",
            "hts_avls": "397287",
            "etf_cnfg_issu_avls": "46643200",
            "etf_cnfg_issu_rlim": "2.53",
            "etf_vltn_amt": "45952000"
        },
        {
            "stck_shrn_iscd": "000270",
            "hts_kor_isnm": "기아",
            "stck_prpr": "109800",
            "prdy_vrss": "-1900",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-1.70",
            "acml_vol": "1221033",
            "acml_tr_pbmn": "134154098000",
            "tday_rsfl_rate": "3.31",
            "prdy_vrss_vol": "-996547",
            "tr_pbmn_tnrt": "0.30",
            "hts_avls": "441445",
            "etf_cnfg_issu_avls": "41724000",
            "etf_cnfg_issu_rlim": "2.27",
            "etf_vltn_amt": "42446000"
        },
        {
            "stck_shrn_iscd": "005490",
            "hts_kor_isnm": "POSCO홀딩스",
            "stck_prpr": "395000",
            "prdy_vrss": "-5000",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-1.25",
            "acml_vol": "317449",
            "acml_tr_pbmn": "126170285000",
            "tday_rsfl_rate": "2.62",
            "prdy_vrss_vol": "-155864",
            "tr_pbmn_tnrt": "0.38",
            "hts_avls": "334056",
            "etf_cnfg_issu_avls": "40685000",
            "etf_cnfg_issu_rlim": "2.21",
            "etf_vltn_amt": "41200000"
        },
        {
            "stck_shrn_iscd": "035420",
            "hts_kor_isnm": "NAVER",
            "stck_prpr": "185900",
            "prdy_vrss": "2300",
            "prdy_vrss_sign": "2",
            "prdy_ctrt": "1.25",
            "acml_vol": "831828",
            "acml_tr_pbmn": "155684174100",
            "tday_rsfl_rate": "2.56",
            "prdy_vrss_vol": "-360853",
            "tr_pbmn_tnrt": "0.52",
            "hts_avls": "301918",
            "etf_cnfg_issu_avls": "37737700",
            "etf_cnfg_issu_rlim": "2.05",
            "etf_vltn_amt": "37270800"
        },
        {
            "stck_shrn_iscd": "105560",
            "hts_kor_isnm": "KB금융",
            "stck_prpr": "66300",
            "prdy_vrss": "-2000",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-2.93",
            "acml_vol": "1756980",
            "acml_tr_pbmn": "116400995300",
            "tday_rsfl_rate": "2.93",
            "prdy_vrss_vol": "-1142245",
            "tr_pbmn_tnrt": "0.44",
            "hts_avls": "267528",
            "etf_cnfg_issu_avls": "34608600",
            "etf_cnfg_issu_rlim": "1.88",
            "etf_vltn_amt": "35652600"
        },
        {
            "stck_shrn_iscd": "006400",
            "hts_kor_isnm": "삼성SDI",
            "stck_prpr": "401000",
            "prdy_vrss": "-6500",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-1.60",
            "acml_vol": "261871",
            "acml_tr_pbmn": "105929587000",
            "tday_rsfl_rate": "2.45",
            "prdy_vrss_vol": "-54274",
            "tr_pbmn_tnrt": "0.38",
            "hts_avls": "275746",
            "etf_cnfg_issu_avls": "31679000",
            "etf_cnfg_issu_rlim": "1.72",
            "etf_vltn_amt": "32192500"
        },
        {
            "stck_shrn_iscd": "055550",
            "hts_kor_isnm": "신한지주",
            "stck_prpr": "41850",
            "prdy_vrss": "-1250",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-2.90",
            "acml_vol": "2559109",
            "acml_tr_pbmn": "107101173850",
            "tday_rsfl_rate": "4.18",
            "prdy_vrss_vol": "-822187",
            "tr_pbmn_tnrt": "0.50",
            "hts_avls": "213181",
            "etf_cnfg_issu_avls": "28123200",
            "etf_cnfg_issu_rlim": "1.53",
            "etf_vltn_amt": "28963200"
        },
        {
            "stck_shrn_iscd": "051910",
            "hts_kor_isnm": "LG화학",
            "stck_prpr": "393000",
            "prdy_vrss": "6000",
            "prdy_vrss_sign": "2",
            "prdy_ctrt": "1.55",
            "acml_vol": "259235",
            "acml_tr_pbmn": "102510288000",
            "tday_rsfl_rate": "4.01",
            "prdy_vrss_vol": "-231613",
            "tr_pbmn_tnrt": "0.37",
            "hts_avls": "277428",
            "etf_cnfg_issu_avls": "27510000",
            "etf_cnfg_issu_rlim": "1.49",
            "etf_vltn_amt": "27090000"
        },
        {
            "stck_shrn_iscd": "012330",
            "hts_kor_isnm": "현대모비스",
            "stck_prpr": "240500",
            "prdy_vrss": "-10500",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-4.18",
            "acml_vol": "198685",
            "acml_tr_pbmn": "48155091500",
            "tday_rsfl_rate": "3.19",
            "prdy_vrss_vol": "-91322",
            "tr_pbmn_tnrt": "0.21",
            "hts_avls": "225241",
            "etf_cnfg_issu_avls": "23328500",
            "etf_cnfg_issu_rlim": "1.27",
            "etf_vltn_amt": "24347000"
        },
        {
            "stck_shrn_iscd": "035720",
            "hts_kor_isnm": "카카오",
            "stck_prpr": "47850",
            "prdy_vrss": "-200",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-0.42",
            "acml_vol": "942730",
            "acml_tr_pbmn": "45251826950",
            "tday_rsfl_rate": "1.66",
            "prdy_vrss_vol": "-1062355",
            "tr_pbmn_tnrt": "0.21",
            "hts_avls": "213049",
            "etf_cnfg_issu_avls": "23015850",
            "etf_cnfg_issu_rlim": "1.25",
            "etf_vltn_amt": "23112050"
        },
        {
            "stck_shrn_iscd": "086790",
            "hts_kor_isnm": "하나금융지주",
            "stck_prpr": "55000",
            "prdy_vrss": "-3000",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-5.17",
            "acml_vol": "1816647",
            "acml_tr_pbmn": "100978675300",
            "tday_rsfl_rate": "4.66",
            "prdy_vrss_vol": "12633",
            "tr_pbmn_tnrt": "0.63",
            "hts_avls": "160796",
            "etf_cnfg_issu_avls": "22055000",
            "etf_cnfg_issu_rlim": "1.20",
            "etf_vltn_amt": "23258000"
        },
        {
            "stck_shrn_iscd": "028260",
            "hts_kor_isnm": "삼성물산",
            "stck_prpr": "140100",
            "prdy_vrss": "-6900",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-4.69",
            "acml_vol": "594987",
            "acml_tr_pbmn": "83994051400",
            "tday_rsfl_rate": "4.08",
            "prdy_vrss_vol": "-140143",
            "tr_pbmn_tnrt": "0.32",
            "hts_avls": "260014",
            "etf_cnfg_issu_avls": "21015000",
            "etf_cnfg_issu_rlim": "1.14",
            "etf_vltn_amt": "22050000"
        },
        {
            "stck_shrn_iscd": "373220",
            "hts_kor_isnm": "LG에너지솔루션",
            "stck_prpr": "371500",
            "prdy_vrss": "-8500",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-2.24",
            "acml_vol": "171804",
            "acml_tr_pbmn": "64468613500",
            "tday_rsfl_rate": "2.63",
            "prdy_vrss_vol": "-169366",
            "tr_pbmn_tnrt": "0.07",
            "hts_avls": "869310",
            "etf_cnfg_issu_avls": "19689500",
            "etf_cnfg_issu_rlim": "1.07",
            "etf_vltn_amt": "20140000"
        },
        {
            "stck_shrn_iscd": "207940",
            "hts_kor_isnm": "삼성바이오로직스",
            "stck_prpr": "790000",
            "prdy_vrss": "-5000",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-0.63",
            "acml_vol": "41087",
            "acml_tr_pbmn": "32506854000",
            "tday_rsfl_rate": "1.76",
            "prdy_vrss_vol": "-20146",
            "tr_pbmn_tnrt": "0.06",
            "hts_avls": "562275",
            "etf_cnfg_issu_avls": "18960000",
            "etf_cnfg_issu_rlim": "1.03",
            "etf_vltn_amt": "19080000"
        },
        {
            "stck_shrn_iscd": "066570",
            "hts_kor_isnm": "LG전자",
            "stck_prpr": "93500",
            "prdy_vrss": "-2900",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-3.01",
            "acml_vol": "906616",
            "acml_tr_pbmn": "85249832600",
            "tday_rsfl_rate": "2.80",
            "prdy_vrss_vol": "326913",
            "tr_pbmn_tnrt": "0.56",
            "hts_avls": "153011",
            "etf_cnfg_issu_avls": "15427500",
            "etf_cnfg_issu_rlim": "0.84",
            "etf_vltn_amt": "15906000"
        },
        {
            "stck_shrn_iscd": "316140",
            "hts_kor_isnm": "우리금융지주",
            "stck_prpr": "13410",
            "prdy_vrss": "-360",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-2.61",
            "acml_vol": "2599188",
            "acml_tr_pbmn": "34967766860",
            "tday_rsfl_rate": "2.61",
            "prdy_vrss_vol": "-646133",
            "tr_pbmn_tnrt": "0.35",
            "hts_avls": "99582",
            "etf_cnfg_issu_avls": "13973220",
            "etf_cnfg_issu_rlim": "0.76",
            "etf_vltn_amt": "14348340"
        },
        {
            "stck_shrn_iscd": "000810",
            "hts_kor_isnm": "삼성화재",
            "stck_prpr": "288500",
            "prdy_vrss": "-6500",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-2.20",
            "acml_vol": "131889",
            "acml_tr_pbmn": "37752099000",
            "tday_rsfl_rate": "3.73",
            "prdy_vrss_vol": "-67056",
            "tr_pbmn_tnrt": "0.28",
            "hts_avls": "136676",
            "etf_cnfg_issu_avls": "13848000",
            "etf_cnfg_issu_rlim": "0.75",
            "etf_vltn_amt": "14160000"
        },
        {
            "stck_shrn_iscd": "033780",
            "hts_kor_isnm": "KT&G",
            "stck_prpr": "88300",
            "prdy_vrss": "-2200",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-2.43",
            "acml_vol": "297711",
            "acml_tr_pbmn": "26378868700",
            "tday_rsfl_rate": "1.88",
            "prdy_vrss_vol": "-17817",
            "tr_pbmn_tnrt": "0.23",
            "hts_avls": "115075",
            "etf_cnfg_issu_avls": "13333300",
            "etf_cnfg_issu_rlim": "0.72",
            "etf_vltn_amt": "13665500"
        },
        {
            "stck_shrn_iscd": "009150",
            "hts_kor_isnm": "삼성전기",
            "stck_prpr": "157700",
            "prdy_vrss": "1200",
            "prdy_vrss_sign": "2",
            "prdy_ctrt": "0.77",
            "acml_vol": "418178",
            "acml_tr_pbmn": "66112119600",
            "tday_rsfl_rate": "2.81",
            "prdy_vrss_vol": "40454",
            "tr_pbmn_tnrt": "0.56",
            "hts_avls": "117792",
            "etf_cnfg_issu_avls": "13246800",
            "etf_cnfg_issu_rlim": "0.72",
            "etf_vltn_amt": "13146000"
        },
        {
            "stck_shrn_iscd": "323410",
            "hts_kor_isnm": "카카오뱅크",
            "stck_prpr": "24850",
            "prdy_vrss": "-1300",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-4.97",
            "acml_vol": "1142001",
            "acml_tr_pbmn": "28599812250",
            "tday_rsfl_rate": "4.21",
            "prdy_vrss_vol": "147850",
            "tr_pbmn_tnrt": "0.24",
            "hts_avls": "118515",
            "etf_cnfg_issu_avls": "12822600",
            "etf_cnfg_issu_rlim": "0.70",
            "etf_vltn_amt": "13493400"
        },
        {
            "stck_shrn_iscd": "138040",
            "hts_kor_isnm": "메리츠금융지주",
            "stck_prpr": "78200",
            "prdy_vrss": "-2500",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-3.10",
            "acml_vol": "718495",
            "acml_tr_pbmn": "56416800000",
            "tday_rsfl_rate": "5.95",
            "prdy_vrss_vol": "204312",
            "tr_pbmn_tnrt": "0.37",
            "hts_avls": "152233",
            "etf_cnfg_issu_avls": "11886400",
            "etf_cnfg_issu_rlim": "0.65",
            "etf_vltn_amt": "12266400"
        },
        {
            "stck_shrn_iscd": "030200",
            "hts_kor_isnm": "KT",
            "stck_prpr": "34600",
            "prdy_vrss": "-800",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-2.26",
            "acml_vol": "911564",
            "acml_tr_pbmn": "31586950850",
            "tday_rsfl_rate": "3.25",
            "prdy_vrss_vol": "-211653",
            "tr_pbmn_tnrt": "0.35",
            "hts_avls": "88979",
            "etf_cnfg_issu_avls": "11833200",
            "etf_cnfg_issu_rlim": "0.64",
            "etf_vltn_amt": "12106800"
        },
        {
            "stck_shrn_iscd": "017670",
            "hts_kor_isnm": "SK텔레콤",
            "stck_prpr": "50500",
            "prdy_vrss": "-700",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-1.37",
            "acml_vol": "583199",
            "acml_tr_pbmn": "29461595000",
            "tday_rsfl_rate": "1.95",
            "prdy_vrss_vol": "-58668",
            "tr_pbmn_tnrt": "0.27",
            "hts_avls": "108469",
            "etf_cnfg_issu_avls": "11413000",
            "etf_cnfg_issu_rlim": "0.62",
            "etf_vltn_amt": "11571200"
        },
        {
            "stck_shrn_iscd": "402340",
            "hts_kor_isnm": "SK스퀘어",
            "stck_prpr": "77900",
            "prdy_vrss": "4800",
            "prdy_vrss_sign": "2",
            "prdy_ctrt": "6.57",
            "acml_vol": "642164",
            "acml_tr_pbmn": "49780422500",
            "tday_rsfl_rate": "9.58",
            "prdy_vrss_vol": "141092",
            "tr_pbmn_tnrt": "0.46",
            "hts_avls": "108266",
            "etf_cnfg_issu_avls": "11373400",
            "etf_cnfg_issu_rlim": "0.62",
            "etf_vltn_amt": "10672600"
        },
        {
            "stck_shrn_iscd": "012450",
            "hts_kor_isnm": "한화에어로스페이스",
            "stck_prpr": "217000",
            "prdy_vrss": "3000",
            "prdy_vrss_sign": "2",
            "prdy_ctrt": "1.40",
            "acml_vol": "466379",
            "acml_tr_pbmn": "100879357000",
            "tday_rsfl_rate": "3.97",
            "prdy_vrss_vol": "-59982",
            "tr_pbmn_tnrt": "0.92",
            "hts_avls": "109867",
            "etf_cnfg_issu_avls": "11284000",
            "etf_cnfg_issu_rlim": "0.61",
            "etf_vltn_amt": "11128000"
        },
        {
            "stck_shrn_iscd": "003670",
            "hts_kor_isnm": "포스코퓨처엠",
            "stck_prpr": "268000",
            "prdy_vrss": "-14500",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-5.13",
            "acml_vol": "371604",
            "acml_tr_pbmn": "101346274000",
            "tday_rsfl_rate": "5.31",
            "prdy_vrss_vol": "106492",
            "tr_pbmn_tnrt": "0.49",
            "hts_avls": "207601",
            "etf_cnfg_issu_avls": "10988000",
            "etf_cnfg_issu_rlim": "0.60",
            "etf_vltn_amt": "11582500"
        },
        {
            "stck_shrn_iscd": "003550",
            "hts_kor_isnm": "LG",
            "stck_prpr": "77600",
            "prdy_vrss": "-2000",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-2.51",
            "acml_vol": "421568",
            "acml_tr_pbmn": "32857335000",
            "tday_rsfl_rate": "3.39",
            "prdy_vrss_vol": "62211",
            "tr_pbmn_tnrt": "0.27",
            "hts_avls": "122066",
            "etf_cnfg_issu_avls": "10941600",
            "etf_cnfg_issu_rlim": "0.59",
            "etf_vltn_amt": "11223600"
        },
        {
            "stck_shrn_iscd": "259960",
            "hts_kor_isnm": "크래프톤",
            "stck_prpr": "238500",
            "prdy_vrss": "-9000",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-3.64",
            "acml_vol": "86465",
            "acml_tr_pbmn": "20865856000",
            "tday_rsfl_rate": "4.04",
            "prdy_vrss_vol": "-22704",
            "tr_pbmn_tnrt": "0.18",
            "hts_avls": "115349",
            "etf_cnfg_issu_avls": "10732500",
            "etf_cnfg_issu_rlim": "0.58",
            "etf_vltn_amt": "11137500"
        },
        {
            "stck_shrn_iscd": "032830",
            "hts_kor_isnm": "삼성생명",
            "stck_prpr": "81100",
            "prdy_vrss": "-3900",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-4.59",
            "acml_vol": "642230",
            "acml_tr_pbmn": "52575516400",
            "tday_rsfl_rate": "5.76",
            "prdy_vrss_vol": "-59341",
            "tr_pbmn_tnrt": "0.32",
            "hts_avls": "162200",
            "etf_cnfg_issu_avls": "10380800",
            "etf_cnfg_issu_rlim": "0.56",
            "etf_vltn_amt": "10880000"
        },
        {
            "stck_shrn_iscd": "034020",
            "hts_kor_isnm": "두산에너빌리티",
            "stck_prpr": "15310",
            "prdy_vrss": "190",
            "prdy_vrss_sign": "2",
            "prdy_ctrt": "1.26",
            "acml_vol": "3313182",
            "acml_tr_pbmn": "50428459700",
            "tday_rsfl_rate": "2.91",
            "prdy_vrss_vol": "-5813097",
            "tr_pbmn_tnrt": "0.51",
            "hts_avls": "98070",
            "etf_cnfg_issu_avls": "10012740",
            "etf_cnfg_issu_rlim": "0.54",
            "etf_vltn_amt": "9888480"
        },
        {
            "stck_shrn_iscd": "015760",
            "hts_kor_isnm": "한국전력",
            "stck_prpr": "20200",
            "prdy_vrss": "-1100",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-5.16",
            "acml_vol": "4137251",
            "acml_tr_pbmn": "84685300850",
            "tday_rsfl_rate": "5.40",
            "prdy_vrss_vol": "737190",
            "tr_pbmn_tnrt": "0.65",
            "hts_avls": "129677",
            "etf_cnfg_issu_avls": "9675800",
            "etf_cnfg_issu_rlim": "0.53",
            "etf_vltn_amt": "10202700"
        },
        {
            "stck_shrn_iscd": "096770",
            "hts_kor_isnm": "SK이노베이션",
            "stck_prpr": "108400",
            "prdy_vrss": "-2200",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-1.99",
            "acml_vol": "435485",
            "acml_tr_pbmn": "47496560900",
            "tday_rsfl_rate": "2.53",
            "prdy_vrss_vol": "-416906",
            "tr_pbmn_tnrt": "0.46",
            "hts_avls": "103777",
            "etf_cnfg_issu_avls": "9647600",
            "etf_cnfg_issu_rlim": "0.52",
            "etf_vltn_amt": "9843400"
        },
        {
            "stck_shrn_iscd": "010140",
            "hts_kor_isnm": "삼성중공업",
            "stck_prpr": "8910",
            "prdy_vrss": "410",
            "prdy_vrss_sign": "2",
            "prdy_ctrt": "4.82",
            "acml_vol": "10237108",
            "acml_tr_pbmn": "90433426630",
            "tday_rsfl_rate": "6.59",
            "prdy_vrss_vol": "5960302",
            "tr_pbmn_tnrt": "1.15",
            "hts_avls": "78408",
            "etf_cnfg_issu_avls": "8954550",
            "etf_cnfg_issu_rlim": "0.49",
            "etf_vltn_amt": "8542500"
        },
        {
            "stck_shrn_iscd": "034730",
            "hts_kor_isnm": "SK",
            "stck_prpr": "161400",
            "prdy_vrss": "-1200",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-0.74",
            "acml_vol": "124326",
            "acml_tr_pbmn": "20057323800",
            "tday_rsfl_rate": "2.77",
            "prdy_vrss_vol": "-115492",
            "tr_pbmn_tnrt": "0.17",
            "hts_avls": "118142",
            "etf_cnfg_issu_avls": "8877000",
            "etf_cnfg_issu_rlim": "0.48",
            "etf_vltn_amt": "8943000"
        },
        {
            "stck_shrn_iscd": "018260",
            "hts_kor_isnm": "삼성에스디에스",
            "stck_prpr": "151400",
            "prdy_vrss": "300",
            "prdy_vrss_sign": "2",
            "prdy_ctrt": "0.20",
            "acml_vol": "99997",
            "acml_tr_pbmn": "15056548600",
            "tday_rsfl_rate": "2.51",
            "prdy_vrss_vol": "-94813",
            "tr_pbmn_tnrt": "0.13",
            "hts_avls": "117150",
            "etf_cnfg_issu_avls": "8781200",
            "etf_cnfg_issu_rlim": "0.48",
            "etf_vltn_amt": "8763800"
        },
        {
            "stck_shrn_iscd": "009540",
            "hts_kor_isnm": "HD한국조선해양",
            "stck_prpr": "117600",
            "prdy_vrss": "1300",
            "prdy_vrss_sign": "2",
            "prdy_ctrt": "1.12",
            "acml_vol": "234901",
            "acml_tr_pbmn": "27561454600",
            "tday_rsfl_rate": "4.73",
            "prdy_vrss_vol": "-38441",
            "tr_pbmn_tnrt": "0.33",
            "hts_avls": "83229",
            "etf_cnfg_issu_avls": "8349600",
            "etf_cnfg_issu_rlim": "0.45",
            "etf_vltn_amt": "8257300"
        },
        {
            "stck_shrn_iscd": "010130",
            "hts_kor_isnm": "고려아연",
            "stck_prpr": "470500",
            "prdy_vrss": "-1500",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-0.32",
            "acml_vol": "41733",
            "acml_tr_pbmn": "19584722500",
            "tday_rsfl_rate": "1.69",
            "prdy_vrss_vol": "-10286",
            "tr_pbmn_tnrt": "0.20",
            "hts_avls": "98375",
            "etf_cnfg_issu_avls": "7998500",
            "etf_cnfg_issu_rlim": "0.43",
            "etf_vltn_amt": "8024000"
        },
        {
            "stck_shrn_iscd": "003490",
            "hts_kor_isnm": "대한항공",
            "stck_prpr": "20500",
            "prdy_vrss": "-500",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-2.38",
            "acml_vol": "842914",
            "acml_tr_pbmn": "17385214850",
            "tday_rsfl_rate": "2.14",
            "prdy_vrss_vol": "-98708",
            "tr_pbmn_tnrt": "0.23",
            "hts_avls": "75485",
            "etf_cnfg_issu_avls": "7933500",
            "etf_cnfg_issu_rlim": "0.43",
            "etf_vltn_amt": "8127000"
        },
        {
            "stck_shrn_iscd": "267260",
            "hts_kor_isnm": "HD현대일렉트릭",
            "stck_prpr": "235000",
            "prdy_vrss": "3000",
            "prdy_vrss_sign": "2",
            "prdy_ctrt": "1.29",
            "acml_vol": "1262526",
            "acml_tr_pbmn": "306921768500",
            "tday_rsfl_rate": "17.89",
            "prdy_vrss_vol": "450052",
            "tr_pbmn_tnrt": "3.62",
            "hts_avls": "84711",
            "etf_cnfg_issu_avls": "7285000",
            "etf_cnfg_issu_rlim": "0.40",
            "etf_vltn_amt": "7192000"
        },
        {
            "stck_shrn_iscd": "011200",
            "hts_kor_isnm": "HMM",
            "stck_prpr": "15380",
            "prdy_vrss": "-220",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-1.41",
            "acml_vol": "1459194",
            "acml_tr_pbmn": "22389748170",
            "tday_rsfl_rate": "3.27",
            "prdy_vrss_vol": "-303696",
            "tr_pbmn_tnrt": "0.21",
            "hts_avls": "105974",
            "etf_cnfg_issu_avls": "6936380",
            "etf_cnfg_issu_rlim": "0.38",
            "etf_vltn_amt": "7035600"
        },
        {
            "stck_shrn_iscd": "000100",
            "hts_kor_isnm": "유한양행",
            "stck_prpr": "71200",
            "prdy_vrss": "2000",
            "prdy_vrss_sign": "2",
            "prdy_ctrt": "2.89",
            "acml_vol": "353495",
            "acml_tr_pbmn": "25012292500",
            "tday_rsfl_rate": "3.76",
            "prdy_vrss_vol": "-29138",
            "tr_pbmn_tnrt": "0.44",
            "hts_avls": "57109",
            "etf_cnfg_issu_avls": "6550400",
            "etf_cnfg_issu_rlim": "0.36",
            "etf_vltn_amt": "6366400"
        },
        {
            "stck_shrn_iscd": "161390",
            "hts_kor_isnm": "한국타이어앤테크놀로지",
            "stck_prpr": "59800",
            "prdy_vrss": "-700",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-1.16",
            "acml_vol": "232184",
            "acml_tr_pbmn": "13963274600",
            "tday_rsfl_rate": "3.64",
            "prdy_vrss_vol": "-318186",
            "tr_pbmn_tnrt": "0.19",
            "hts_avls": "74077",
            "etf_cnfg_issu_avls": "6518200",
            "etf_cnfg_issu_rlim": "0.35",
            "etf_vltn_amt": "6594500"
        },
        {
            "stck_shrn_iscd": "090430",
            "hts_kor_isnm": "아모레퍼시픽",
            "stck_prpr": "135000",
            "prdy_vrss": "7600",
            "prdy_vrss_sign": "2",
            "prdy_ctrt": "5.97",
            "acml_vol": "351097",
            "acml_tr_pbmn": "47076583200",
            "tday_rsfl_rate": "8.01",
            "prdy_vrss_vol": "102372",
            "tr_pbmn_tnrt": "0.60",
            "hts_avls": "78965",
            "etf_cnfg_issu_avls": "6345000",
            "etf_cnfg_issu_rlim": "0.34",
            "etf_vltn_amt": "5987800"
        },
        {
            "stck_shrn_iscd": "352820",
            "hts_kor_isnm": "하이브",
            "stck_prpr": "213000",
            "prdy_vrss": "-3500",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-1.62",
            "acml_vol": "169056",
            "acml_tr_pbmn": "36344402500",
            "tday_rsfl_rate": "3.93",
            "prdy_vrss_vol": "-37279",
            "tr_pbmn_tnrt": "0.41",
            "hts_avls": "88719",
            "etf_cnfg_issu_avls": "5964000",
            "etf_cnfg_issu_rlim": "0.32",
            "etf_vltn_amt": "6062000"
        },
        {
            "stck_shrn_iscd": "028050",
            "hts_kor_isnm": "삼성E&A",
            "stck_prpr": "24950",
            "prdy_vrss": "-50",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-0.20",
            "acml_vol": "520991",
            "acml_tr_pbmn": "13051679850",
            "tday_rsfl_rate": "2.20",
            "prdy_vrss_vol": "-398376",
            "tr_pbmn_tnrt": "0.27",
            "hts_avls": "48902",
            "etf_cnfg_issu_avls": "5963050",
            "etf_cnfg_issu_rlim": "0.32",
            "etf_vltn_amt": "5975000"
        },
        {
            "stck_shrn_iscd": "005830",
            "hts_kor_isnm": "DB손해보험",
            "stck_prpr": "88100",
            "prdy_vrss": "-7400",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-7.75",
            "acml_vol": "267078",
            "acml_tr_pbmn": "23699343200",
            "tday_rsfl_rate": "7.85",
            "prdy_vrss_vol": "26008",
            "tr_pbmn_tnrt": "0.38",
            "hts_avls": "62375",
            "etf_cnfg_issu_avls": "5902700",
            "etf_cnfg_issu_rlim": "0.32",
            "etf_vltn_amt": "6398500"
        },
        {
            "stck_shrn_iscd": "024110",
            "hts_kor_isnm": "기업은행",
            "stck_prpr": "12740",
            "prdy_vrss": "-360",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-2.75",
            "acml_vol": "1773632",
            "acml_tr_pbmn": "22730534060",
            "tday_rsfl_rate": "2.98",
            "prdy_vrss_vol": "149508",
            "tr_pbmn_tnrt": "0.22",
            "hts_avls": "101592",
            "etf_cnfg_issu_avls": "5261620",
            "etf_cnfg_issu_rlim": "0.29",
            "etf_vltn_amt": "5410300"
        },
        {
            "stck_shrn_iscd": "047810",
            "hts_kor_isnm": "한국항공우주",
            "stck_prpr": "48600",
            "prdy_vrss": "-1050",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-2.11",
            "acml_vol": "408379",
            "acml_tr_pbmn": "19958527850",
            "tday_rsfl_rate": "2.01",
            "prdy_vrss_vol": "27622",
            "tr_pbmn_tnrt": "0.42",
            "hts_avls": "47373",
            "etf_cnfg_issu_avls": "5200200",
            "etf_cnfg_issu_rlim": "0.28",
            "etf_vltn_amt": "5312550"
        },
        {
            "stck_shrn_iscd": "051900",
            "hts_kor_isnm": "LG생활건강",
            "stck_prpr": "368000",
            "prdy_vrss": "15000",
            "prdy_vrss_sign": "2",
            "prdy_ctrt": "4.25",
            "acml_vol": "83736",
            "acml_tr_pbmn": "30575342500",
            "tday_rsfl_rate": "4.96",
            "prdy_vrss_vol": "2995",
            "tr_pbmn_tnrt": "0.53",
            "hts_avls": "57475",
            "etf_cnfg_issu_avls": "5152000",
            "etf_cnfg_issu_rlim": "0.28",
            "etf_vltn_amt": "4942000"
        },
        {
            "stck_shrn_iscd": "001570",
            "hts_kor_isnm": "금양",
            "stck_prpr": "101000",
            "prdy_vrss": "-3800",
            "prdy_vrss_sign": "5",
            "prdy_ctrt": "-3.63",
            "acml_vol": "459491",
            "acml_tr_pbmn": "47017893700",
            "tday_rsfl_rate": "3.82",
            "prdy_vrss_vol": "120901",
            "tr_pbmn_tnrt": "0.80",
            "hts_avls": "58631",
            "etf_cnfg_issu_avls": "5050000",
            "etf_cnfg_issu_rlim": "0.27",
```
