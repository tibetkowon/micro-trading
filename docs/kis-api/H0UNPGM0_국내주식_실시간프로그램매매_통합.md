# 국내주식 실시간프로그램매매 (통합)

| 항목 | 값 |
|------|---|
| 통신방식 | `WEBSOCKET` |
| API 명 | `국내주식 실시간프로그램매매 (통합)` |
| API ID | `국내주식 실시간프로그램매매 (통합)` |
| 실전 TR_ID | `H0UNPGM0` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `POST` |
| URL | `/tryitout/H0UNPGM0` |

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
| `tr_id` | 거래ID | string | Y | H0UNPGM0 : 실시간 주식종목프로그램매매 통합 |
| `tr_key` | 구분값 | string | Y | 종목코드 (ex 005930 삼성전자) |

## Response Body

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `MKSC_SHRN_ISCD` | 유가증권 단축 종목코드 | string | Y |  |
| `STCK_CNTG_HOUR` | 주식 체결 시간 | string | Y |  |
| `SELN_CNQN` | 매도 체결량 | string | Y |  |
| `SELN_TR_PBMN` | 매도 거래 대금 | string | Y |  |
| `SHNU_CNQN` | 매수2 체결량 | string | Y |  |
| `SHNU_TR_PBMN` | 매수2 거래 대금 | string | Y |  |
| `NTBY_CNQN` | 순매수 체결량 | string | Y |  |
| `NTBY_TR_PBMN` | 순매수 거래 대금 | string | Y |  |
| `SELN_RSQN` | 매도호가잔량 | string | Y |  |
| `SHNU_RSQN` | 매수호가잔량 | string | Y |  |
| `WHOL_NTBY_QTY` | 전체순매수호가잔량 | string | Y |  |
