# 국내주식 실시간예상체결 (NXT)

| 항목 | 값 |
|------|---|
| 통신방식 | `WEBSOCKET` |
| API 명 | `국내주식 실시간예상체결 (NXT)` |
| API ID | `국내주식 실시간예상체결 (NXT)` |
| 실전 TR_ID | `H0NXANC0` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `POST` |
| URL | `/tryitout/H0NXANC0` |

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
| `tr_id` | 거래ID | string | Y | H0NXANC0 : 국내주식 실시간예상체결 (NXT) |
| `tr_key` | 구분값 | string | Y | 종목코드 (ex 005930 삼성전자) |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `MKSC_SHRN_ISCD` | 유가증권단축종목코드 | string | Y |  |
| `STCK_CNTG_HOUR` | 주식체결시간 | string | Y |  |
| `STCK_PRPR` | 주식현재가 | string | Y |  |
| `PRDY_VRSS_SIGN` | 전일대비구분 | string | Y |  |
| `PRDY_VRSS` | 전일대비 | string | Y |  |
| `PRDY_CTRT` | 등락율 | string | Y |  |
| `WGHN_AVRG_STCK_PRC` | 가중평균주식가격 | string | Y |  |
| `STCK_OPRC` | 시가 | string | Y |  |
| `STCK_HGPR` | 고가 | string | Y |  |
| `STCK_LWPR` | 저가 | string | Y |  |
| `ASKP1` | 매도호가 | string | Y |  |
| `BIDP1` | 매수호가 | string | Y |  |
| `CNTG_VOL` | 거래량 | string | Y |  |
| `ACML_VOL` | 누적거래량 | string | Y |  |
| `ACML_TR_PBMN` | 누적거래대금 | string | Y |  |
| `SELN_CNTG_CSNU` | 매도체결건수 | string | Y |  |
| `SHNU_CNTG_CSNU` | 매수체결건수 | string | Y |  |
| `NTBY_CNTG_CSNU` | 순매수체결건수 | string | Y |  |
| `CTTR` | 체결강도 | string | Y |  |
| `SELN_CNTG_SMTN` | 총매도수량 | string | Y |  |
| `SHNU_CNTG_SMTN` | 총매수수량 | string | Y |  |
| `CNTG_CLS_CODE` | 체결구분 | string | Y |  |
| `SHNU_RATE` | 매수비율 | string | Y |  |
| `PRDY_VOL_VRSS_ACML_VOL_RATE` | 전일거래량대비등락율 | string | Y |  |
| `OPRC_HOUR` | 시가시간 | string | Y |  |
| `OPRC_VRSS_PRPR_SIGN` | 시가대비구분 | string | Y |  |
| `OPRC_VRSS_PRPR` | 시가대비 | string | Y |  |
| `HGPR_HOUR` | 최고가시간 | string | Y |  |
| `HGPR_VRSS_PRPR_SIGN` | 고가대비구분 | string | Y |  |
| `HGPR_VRSS_PRPR` | 고가대비 | string | Y |  |
| `LWPR_HOUR` | 최저가시간 | string | Y |  |
| `LWPR_VRSS_PRPR_SIGN` | 저가대비구분 | string | Y |  |
| `LWPR_VRSS_PRPR` | 저가대비 | string | Y |  |
| `BSOP_DATE` | 영업일자 | string | Y |  |
| `NEW_MKOP_CLS_CODE` | 신장운영구분코드 | string | Y |  |
| `TRHT_YN` | 거래정지여부 | string | Y |  |
| `ASKP_RSQN1` | 매도호가잔량1 | string | Y |  |
| `BIDP_RSQN1` | 매수호가잔량1 | string | Y |  |
| `TOTAL_ASKP_RSQN` | 총매도호가잔량 | string | Y |  |
| `TOTAL_BIDP_RSQN` | 총매수호가잔량 | string | Y |  |
| `VOL_TNRT` | 거래량회전율 | string | Y |  |
| `PRDY_SMNS_HOUR_ACML_VOL` | 전일동시간누적거래량 | string | Y |  |
| `PRDY_SMNS_HOUR_ACML_VOL_RATE` | 전일동시간누적거래량비율 | string | Y |  |
| `HOUR_CLS_CODE` | 시간구분코드 | string | Y |  |
| `MRKT_TRTM_CLS_CODE` | 임의종료구분코드 | string | Y |  |
| `VI_STND_PRC` | VI 상태값 | string | Y |  |
