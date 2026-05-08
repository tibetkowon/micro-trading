# 국내주식 실시간호가 (통합)

| 항목 | 값 |
|------|---|
| 통신방식 | `WEBSOCKET` |
| API 명 | `국내주식 실시간호가 (통합)` |
| API ID | `국내주식 실시간호가 (통합)` |
| 실전 TR_ID | `H0UNASP0` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `POST` |
| URL | `/tryitout/H0UNASP0` |

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
| `tr_id` | 거래ID | string | Y | H0UNASP0 : 실시간 주식 체결가 통합 |
| `tr_key` | 구분값 | string | Y | 종목코드 (ex 005930 삼성전자) |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `MKSC_SHRN_ISCD` | 유가증권 단축 종목코드 | string | Y |  |
| `BSOP_HOUR` | 영업 시간 | string | Y |  |
| `HOUR_CLS_CODE` | 시간 구분 코드 | string | Y |  |
| `ASKP1` | 매도호가1 | string | Y |  |
| `ASKP2` | 매도호가2 | string | Y |  |
| `ASKP3` | 매도호가3 | string | Y |  |
| `ASKP4` | 매도호가4 | string | Y |  |
| `ASKP5` | 매도호가5 | string | Y |  |
| `ASKP6` | 매도호가6 | string | Y |  |
| `ASKP7` | 매도호가7 | string | Y |  |
| `ASKP8` | 매도호가8 | string | Y |  |
| `ASKP9` | 매도호가9 | string | Y |  |
| `ASKP10` | 매도호가10 | string | Y |  |
| `BIDP1` | 매수호가1 | string | Y |  |
| `BIDP2` | 매수호가2 | string | Y |  |
| `BIDP3` | 매수호가3 | string | Y |  |
| `BIDP4` | 매수호가4 | string | Y |  |
| `BIDP5` | 매수호가5 | string | Y |  |
| `BIDP6` | 매수호가6 | string | Y |  |
| `BIDP7` | 매수호가7 | string | Y |  |
| `BIDP8` | 매수호가8 | string | Y |  |
| `BIDP9` | 매수호가9 | string | Y |  |
| `BIDP10` | 매수호가10 | string | Y |  |
| `ASKP_RSQN1` | 매도호가 잔량1 | string | Y |  |
| `ASKP_RSQN2` | 매도호가 잔량2 | string | Y |  |
| `ASKP_RSQN3` | 매도호가 잔량3 | string | Y |  |
| `ASKP_RSQN4` | 매도호가 잔량4 | string | Y |  |
| `ASKP_RSQN5` | 매도호가 잔량5 | string | Y |  |
| `ASKP_RSQN6` | 매도호가 잔량6 | string | Y |  |
| `ASKP_RSQN7` | 매도호가 잔량7 | string | Y |  |
| `ASKP_RSQN8` | 매도호가 잔량8 | string | Y |  |
| `ASKP_RSQN9` | 매도호가 잔량9 | string | Y |  |
| `ASKP_RSQN10` | 매도호가 잔량10 | string | Y |  |
| `BIDP_RSQN1` | 매수호가 잔량1 | string | Y |  |
| `BIDP_RSQN2` | 매수호가 잔량2 | string | Y |  |
| `BIDP_RSQN3` | 매수호가 잔량3 | string | Y |  |
| `BIDP_RSQN4` | 매수호가 잔량4 | string | Y |  |
| `BIDP_RSQN5` | 매수호가 잔량5 | string | Y |  |
| `BIDP_RSQN6` | 매수호가 잔량6 | string | Y |  |
| `BIDP_RSQN7` | 매수호가 잔량7 | string | Y |  |
| `BIDP_RSQN8` | 매수호가 잔량8 | string | Y |  |
| `BIDP_RSQN9` | 매수호가 잔량9 | string | Y |  |
| `BIDP_RSQN10` | 매수호가 잔량10 | string | Y |  |
| `TOTAL_ASKP_RSQN` | 총 매도호가 잔량 | string | Y |  |
| `TOTAL_BIDP_RSQN` | 총 매수호가 잔량 | string | Y |  |
| `OVTM_TOTAL_ASKP_RSQN` | 시간외 총 매도호가 잔량 | string | Y |  |
| `OVTM_TOTAL_BIDP_RSQN` | 시간외 총 매수호가 잔량 | string | Y |  |
| `ANTC_CNPR` | 예상 체결가 | string | Y |  |
| `ANTC_CNQN` | 예상 체결량 | string | Y |  |
| `ANTC_VOL` | 예상 거래량 | string | Y |  |
| `ANTC_CNTG_VRSS` | 예상 체결 대비 | string | Y |  |
| `ANTC_CNTG_VRSS_SIGN` | 예상 체결 대비 부호 | string | Y |  |
| `ANTC_CNTG_PRDY_CTRT` | 예상 체결 전일 대비율 | string | Y |  |
| `ACML_VOL` | 누적 거래량 | string | Y |  |
| `TOTAL_ASKP_RSQN_ICDC` | 총 매도호가 잔량 증감 | string | Y |  |
| `TOTAL_BIDP_RSQN_ICDC` | 총 매수호가 잔량 증감 | string | Y |  |
| `OVTM_TOTAL_ASKP_ICDC` | 시간외 총 매도호가 증감 | string | Y |  |
| `OVTM_TOTAL_BIDP_ICDC` | 시간외 총 매수호가 증감 | string | Y |  |
| `STCK_DEAL_CLS_CODE` | 주식 매매 구분 코드 | string | Y |  |
| `KMID_PRC` | KRX 중간가 | string | Y |  |
| `KMID_TOTAL_RSQN` | KRX 중간가잔량합계수량 | string | Y |  |
| `KMID_CLS_CODE` | KRX 중간가 매수매도 구분 | string | Y |  |
| `NMID_PRC` | NXT 중간가 | string | Y |  |
| `NMID_TOTAL_RSQN` | NXT 중간가잔량합계수량 | string | Y |  |
| `NMID_CLS_CODE` | NXT 중간가 매수매도 구분 | string | Y |  |
