# 퇴직연금 예수금조회

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `퇴직연금 예수금조회` |
| API ID | `v1_국내주식-035` |
| 실전 TR_ID | `TTTC0506R` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/trading/pension/inquire-deposit` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> ​※ 55번 계좌(DC가입자계좌)의 경우 해당 API 이용이 불가합니다.
KIS Developers API의 경우 HTS ID에 반드시 연결되어있어야만 API 신청 및 앱정보 발급이 가능한 서비스로 개발되어서 실물계좌가 아닌 55번 계좌는 API 이용이 불가능한 점 양해 부탁드립니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `CANO` | 종합계좌번호 | string | Y |  |
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y | 29 |
| `ACCA_DVSN_CD` | 적립금구분코드 | string | Y | 00 |

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
| `output` | 응답상세1 | object | Y |  |
| `dnca_tota` | 예수금총액 | string | Y |  |
| `nxdy_excc_amt` | 익일정산액 | string | Y |  |
| `nxdy_sttl_amt` | 익일결제금액 | string | Y |  |
| `nx2_day_sttl_amt` | 2익일결제금액 | string | Y |  |

## Example

### Request Example (Python)
```json
{
	"CANO":"63512345",
	"ACNT_PRDT_CD":"29",
	"ACCA_DVSN_CD":"00"
}
```

### Response Example
```json
{
    "output": {
        "dnca_tota": "57622382",
        "nxdy_excc_amt": "11054042",
        "nxdy_sttl_amt": "0",
        "nx2_day_sttl_amt": "0"
    },
    "rt_cd": "0",
    "msg_cd": "KIOK0510",
    "msg1": "조회가 완료되었습니다                                                           "
}
```
