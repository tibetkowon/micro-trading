# 국내주식 장운영정보 (통합)

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `국내주식 장운영정보 (통합)` |
| API ID | `국내주식 장운영정보 (통합)` |
| 실전 TR_ID | `H0UNMKO0` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `POST` |
| URL | `/tryitout/H0UNMKO0` |

- 실전: `ws://ops.koreainvestment.com:21000`
- 모의: `모의투자 미지원`

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_type` | 거래타입 | string | N | 1 : 등록; 2 : 해제 |

## Request Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_id` | 거래ID | string | Y | H0UNMKO0 : 국내주식 장운영정보 (통합) |
| `tr_key` | 구분값 | string | Y | 종목코드 (ex 005930 삼성전자) |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `TRHT_YN` | 거래정지 여부 | string | Y |  |
| `TR_SUSP_REAS_CNTT` | 거래 정지 사유 내용 | string | Y |  |
| `MKOP_CLS_CODE` | 장운영 구분 코드 | string | Y |  |
| `ANTC_MKOP_CLS_CODE` | 예상 장운영 구분 코드 | string | Y |  |
| `MRKT_TRTM_CLS_CODE` | 임의연장구분코드 | string | Y |  |
| `DIVI_APP_CLS_CODE` | 동시호가배분처리구분코드 | string | Y |  |
| `ISCD_STAT_CLS_CODE` | 종목상태구분코드 | string | Y |  |
| `VI_CLS_CODE` | VI적용구분코드 | string | Y |  |
| `OVTM_VI_CLS_CODE` | 시간외단일가VI적용구분코드 | string | Y |  |
| `EXCH_CLS_CODE` | 거래소 구분코드 | string | Y |  |
