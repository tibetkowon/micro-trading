# 국내주식 실시간회원사 (NXT)

| 항목 | 값 |
|------|---|
| 통신방식 | `WEBSOCKET` |
| API 명 | `국내주식 실시간회원사 (NXT)` |
| API ID | `국내주식 실시간회원사 (NXT)` |
| 실전 TR_ID | `H0NXMBC0` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `POST` |
| URL | `/tryitout/H0NXMBC0` |

- 실전: `ws://ops.koreainvestment.com:21000`
- 모의: `모의투자 미지원`

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_type` | 거래타입 | string | N | '1 : 등록; 2 : 해제' |

## Request Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_id` | 거래ID | string | Y | H0NXMBC0 : 국내주식 주식종목회원사 (NXT) |
| `tr_key` | 구분값 | string | Y | 종목코드 (ex 005930 삼성전자) |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `MKSC_SHRN_ISCD` | 유가증권 단축 종목코드 | string | Y |  |
| `SELN2_MBCR_NAME1` | 매도2 회원사명1 | string | Y |  |
| `SELN2_MBCR_NAME2` | 매도2 회원사명2 | string | Y |  |
| `SELN2_MBCR_NAME3` | 매도2 회원사명3 | string | Y |  |
| `SELN2_MBCR_NAME4` | 매도2 회원사명4 | string | Y |  |
| `SELN2_MBCR_NAME5` | 매도2 회원사명5 | string | Y |  |
| `BYOV_MBCR_NAME1` | 매수 회원사명1 | string | Y |  |
| `BYOV_MBCR_NAME2` | 매수 회원사명2 | string | Y |  |
| `BYOV_MBCR_NAME3` | 매수 회원사명3 | string | Y |  |
| `BYOV_MBCR_NAME4` | 매수 회원사명4 | string | Y |  |
| `BYOV_MBCR_NAME5` | 매수 회원사명5 | string | Y |  |
| `TOTAL_SELN_QTY1` | 총 매도 수량1 | string | Y |  |
| `TOTAL_SELN_QTY2` | 총 매도 수량2 | string | Y |  |
| `TOTAL_SELN_QTY3` | 총 매도 수량3 | string | Y |  |
| `TOTAL_SELN_QTY4` | 총 매도 수량4 | string | Y |  |
| `TOTAL_SELN_QTY5` | 총 매도 수량5 | string | Y |  |
| `TOTAL_SHNU_QTY1` | 총 매수2 수량1 | string | Y |  |
| `TOTAL_SHNU_QTY2` | 총 매수2 수량2 | string | Y |  |
| `TOTAL_SHNU_QTY3` | 총 매수2 수량3 | string | Y |  |
| `TOTAL_SHNU_QTY4` | 총 매수2 수량4 | string | Y |  |
| `TOTAL_SHNU_QTY5` | 총 매수2 수량5 | string | Y |  |
| `SELN_MBCR_GLOB_YN_1` | 매도거래원구분1 | string | Y |  |
| `SELN_MBCR_GLOB_YN_2` | 매도거래원구분2 | string | Y |  |
| `SELN_MBCR_GLOB_YN_3` | 매도거래원구분3 | string | Y |  |
| `SELN_MBCR_GLOB_YN_4` | 매도거래원구분4 | string | Y |  |
| `SELN_MBCR_GLOB_YN_5` | 매도거래원구분5 | string | Y |  |
| `SHNU_MBCR_GLOB_YN_1` | 매수거래원구분1 | string | Y |  |
| `SHNU_MBCR_GLOB_YN_2` | 매수거래원구분2 | string | Y |  |
| `SHNU_MBCR_GLOB_YN_3` | 매수거래원구분3 | string | Y |  |
| `SHNU_MBCR_GLOB_YN_4` | 매수거래원구분4 | string | Y |  |
| `SHNU_MBCR_GLOB_YN_5` | 매수거래원구분5 | string | Y |  |
| `SELN_MBCR_NO1` | 매도거래원코드1 | string | Y |  |
| `SELN_MBCR_NO2` | 매도거래원코드2 | string | Y |  |
| `SELN_MBCR_NO3` | 매도거래원코드3 | string | Y |  |
| `SELN_MBCR_NO4` | 매도거래원코드4 | string | Y |  |
| `SELN_MBCR_NO5` | 매도거래원코드5 | string | Y |  |
| `SHNU_MBCR_NO1` | 매수거래원코드1 | string | Y |  |
| `SHNU_MBCR_NO2` | 매수거래원코드2 | string | Y |  |
| `SHNU_MBCR_NO3` | 매수거래원코드3 | string | Y |  |
| `SHNU_MBCR_NO4` | 매수거래원코드4 | string | Y |  |
| `SHNU_MBCR_NO5` | 매수거래원코드5 | string | Y |  |
| `SELN_MBCR_RLIM1` | 매도 회원사 비중1 | string | Y |  |
| `SELN_MBCR_RLIM2` | 매도 회원사 비중2 | string | Y |  |
| `SELN_MBCR_RLIM3` | 매도 회원사 비중3 | string | Y |  |
| `SELN_MBCR_RLIM4` | 매도 회원사 비중4 | string | Y |  |
| `SELN_MBCR_RLIM5` | 매도 회원사 비중5 | string | Y |  |
| `SHNU_MBCR_RLIM1` | 매수2 회원사 비중1 | string | Y |  |
| `SHNU_MBCR_RLIM2` | 매수2 회원사 비중2 | string | Y |  |
| `SHNU_MBCR_RLIM3` | 매수2 회원사 비중3 | string | Y |  |
| `SHNU_MBCR_RLIM4` | 매수2 회원사 비중4 | string | Y |  |
| `SHNU_MBCR_RLIM5` | 매수2 회원사 비중5 | string | Y |  |
| `SELN_QTY_ICDC1` | 매도 수량 증감1 | string | Y |  |
| `SELN_QTY_ICDC2` | 매도 수량 증감2 | string | Y |  |
| `SELN_QTY_ICDC3` | 매도 수량 증감3 | string | Y |  |
| `SELN_QTY_ICDC4` | 매도 수량 증감4 | string | Y |  |
| `SELN_QTY_ICDC5` | 매도 수량 증감5 | string | Y |  |
| `SHNU_QTY_ICDC1` | 매수2 수량 증감1 | string | Y |  |
| `SHNU_QTY_ICDC2` | 매수2 수량 증감2 | string | Y |  |
| `SHNU_QTY_ICDC3` | 매수2 수량 증감3 | string | Y |  |
| `SHNU_QTY_ICDC4` | 매수2 수량 증감4 | string | Y |  |
| `SHNU_QTY_ICDC5` | 매수2 수량 증감5 | string | Y |  |
| `GLOB_TOTAL_SELN_QTY` | 외국계 총 매도 수량 | string | Y |  |
| `GLOB_TOTAL_SHNU_QTY` | 외국계 총 매수2 수량 | string | Y |  |
| `GLOB_TOTAL_SELN_QTY_ICDC` | 외국계 총 매도 수량 증감 | string | Y |  |
| `GLOB_TOTAL_SHNU_QTY_ICDC` | 외국계 총 매수2 수량 증감 | string | Y |  |
| `GLOB_NTBY_QTY` | 외국계 순매수 수량 | string | Y |  |
| `GLOB_SELN_RLIM` | 외국계 매도 비중 | string | Y |  |
| `GLOB_SHNU_RLIM` | 외국계 매수2 비중 | string | Y |  |
| `SELN2_MBCR_ENG_NAME1` | 매도2 영문회원사명1 | string | Y |  |
| `SELN2_MBCR_ENG_NAME2` | 매도2 영문회원사명2 | string | Y |  |
| `SELN2_MBCR_ENG_NAME3` | 매도2 영문회원사명3 | string | Y |  |
| `SELN2_MBCR_ENG_NAME4` | 매도2 영문회원사명4 | string | Y |  |
| `SELN2_MBCR_ENG_NAME5` | 매도2 영문회원사명5 | string | Y |  |
| `BYOV_MBCR_ENG_NAME1` | 매수 영문회원사명1 | string | Y |  |
| `BYOV_MBCR_ENG_NAME2` | 매수 영문회원사명2 | string | Y |  |
| `BYOV_MBCR_ENG_NAME3` | 매수 영문회원사명3 | string | Y |  |
| `BYOV_MBCR_ENG_NAME4` | 매수 영문회원사명4 | string | Y |  |
| `BYOV_MBCR_ENG_NAME5` | 매수 영문회원사명5 | string | Y |  |
