# 국내주식 실시간체결가 (통합)

| 항목 | 값 |
|------|---|
| 통신방식 | `WEBSOCKET` |
| API 명 | `국내주식 실시간체결가 (통합)` |
| API ID | `국내주식 실시간체결가 (통합)` |
| 실전 TR_ID | `H0UNCNT0` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `POST` |
| URL | `/tryitout/H0UNCNT0` |

- 실전: `ws://ops.koreainvestment.com:21000`
- 모의: `모의투자 미지원`

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_type` | 거래타입 | string | N | 1 : 등록 2 : 해제 |

## Request Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `tr_id` | 거래ID | string | Y | H0UNCNT0 : 실시간 주식 체결가 통합 |
| `tr_key` | 구분값 | string | Y | 종목코드 (ex 005930 삼성전자) |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `MKSC_SHRN_ISCD` | 유가증권 단축 종목코드 | string | Y |  |
| `STCK_CNTG_HOUR` | 주식 체결 시간 | string | Y |  |
| `STCK_PRPR` | 주식 현재가 | string | Y |  |
| `PRDY_VRSS_SIGN` | 전일 대비 부호 | string | Y |  |
| `PRDY_VRSS` | 전일 대비 | string | Y |  |
| `PRDY_CTRT` | 전일 대비율 | string | Y |  |
| `WGHN_AVRG_STCK_PRC` | 가중 평균 주식 가격 | string | Y |  |
| `STCK_OPRC` | 주식 시가 | string | Y |  |
| `STCK_HGPR` | 주식 최고가 | string | Y |  |
| `STCK_LWPR` | 주식 최저가 | string | Y |  |
| `ASKP1` | 매도호가1 | string | Y |  |
| `BIDP1` | 매수호가1 | string | Y |  |
| `CNTG_VOL` | 체결 거래량 | string | Y |  |
| `ACML_VOL` | 누적 거래량 | string | Y |  |
| `ACML_TR_PBMN` | 누적 거래 대금 | string | Y |  |
| `SELN_CNTG_CSNU` | 매도 체결 건수 | string | Y |  |
| `SHNU_CNTG_CSNU` | 매수 체결 건수 | string | Y |  |
| `NTBY_CNTG_CSNU` | 순매수 체결 건수 | string | Y |  |
| `CTTR` | 체결강도 | string | Y |  |
| `SELN_CNTG_SMTN` | 총 매도 수량 | string | Y |  |
| `SHNU_CNTG_SMTN` | 총 매수 수량 | string | Y |  |
| `CNTG_CLS_CODE` | 체결구분 | string | Y |  |
| `SHNU_RATE` | 매수비율 | string | Y |  |
| `PRDY_VOL_VRSS_ACML_VOL_RATE` | 전일 거래량 대비 등락율 | string | Y |  |
| `OPRC_HOUR` | 시가 시간 | string | Y |  |
| `OPRC_VRSS_PRPR_SIGN` | 시가대비구분 | string | Y |  |
| `OPRC_VRSS_PRPR` | 시가대비 | string | Y |  |
| `HGPR_HOUR` | 최고가 시간 | string | Y |  |
| `HGPR_VRSS_PRPR_SIGN` | 고가대비구분 | string | Y |  |
| `HGPR_VRSS_PRPR` | 고가대비 | string | Y |  |
| `LWPR_HOUR` | 최저가 시간 | string | Y |  |
| `LWPR_VRSS_PRPR_SIGN` | 저가대비구분 | string | Y |  |
| `LWPR_VRSS_PRPR` | 저가대비 | string | Y |  |
| `BSOP_DATE` | 영업 일자 | string | Y |  |
| `NEW_MKOP_CLS_CODE` | 신 장운영 구분 코드 | string | Y |  |
| `TRHT_YN` | 거래정지 여부 | string | Y |  |
| `ASKP_RSQN1` | 매도호가 잔량1 | string | Y |  |
| `BIDP_RSQN1` | 매수호가 잔량1 | string | Y |  |
| `TOTAL_ASKP_RSQN` | 총 매도호가 잔량 | string | Y |  |
| `TOTAL_BIDP_RSQN` | 총 매수호가 잔량 | string | Y |  |
| `VOL_TNRT` | 거래량 회전율 | string | Y |  |
| `PRDY_SMNS_HOUR_ACML_VOL` | 전일 동시간 누적 거래량 | string | Y |  |
| `PRDY_SMNS_HOUR_ACML_VOL_RATE` | 전일 동시간 누적 거래량 비율 | string | Y |  |
| `HOUR_CLS_CODE` | 시간 구분 코드 | string | Y |  |
| `MRKT_TRTM_CLS_CODE` | 임의종료구분코드 | string | Y |  |
| `VI_STND_PRC` | 정적VI발동기준가 | string | Y |  |
