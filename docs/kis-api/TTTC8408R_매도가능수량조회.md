# 매도가능수량조회

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `매도가능수량조회` |
| API ID | `국내주식-165` |
| 실전 TR_ID | `TTTC8408R` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/trading/inquire-psbl-sell` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

> 매도가능수량조회 API입니다. 
한국투자 HTS(eFriend Plus) &gt; [0971] 주식 매도 화면에서 종목코드 입력 후 "가능" 클릭 시 매도가능수량이 확인되는 기능을 API로 개발한 사항으로, 해당 화면을 참고하시면 기능을 이해하기 쉽습니다.

특정종목 매도가능수량 확인 시, 매도주문 내시려는 주문종목(PDNO)으로 API 호출 후 
output &gt; ord_psbl_qty(주문가능수량) 확인하실 수 있습니다.

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `CANO` | 종합계좌번호 | string | Y | 종합계좌번호 |
| `ACNT_PRDT_CD` | 계좌상품코드 | string | Y | 계좌상품코드 |
| `PDNO` | 종목번호 | string | Y | 보유종목 코드 ex)000660 |

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
| `pdno` | 상품번호 | string | Y |  |
| `prdt_name` | 상품명 | string | Y |  |
| `buy_qty` | 매수수량 | string | Y |  |
| `sll_qty` | 매도수량 | string | Y |  |
| `cblc_qty` | 잔고수량 | string | Y |  |
| `nsvg_qty` | 비저축수량 | string | Y |  |
| `ord_psbl_qty` | 주문가능수량 | string | Y |  |
| `pchs_avg_pric` | 매입평균가격 | string | Y |  |
| `pchs_amt` | 매입금액 | string | Y |  |
| `now_pric` | 현재가 | string | Y |  |
| `evlu_amt` | 평가금액 | string | Y |  |
| `evlu_pfls_amt` | 평가손익금액 | string | Y |  |
| `evlu_pfls_rt` | 평가손익율 | string | Y |  |

## Example

### Request Example (Python)
```json
CANO:12345678
ACNT_PRDT_CD:01
PDNO:005930
```

### Response Example
```json
{
    "output": {
        "pdno": "005930",
        "prdt_name": "삼성전자",
        "buy_qty": "1746",
        "sll_qty": "2",
        "cblc_qty": "1744",
        "nsvg_qty": "0",
        "ord_psbl_qty": "1744",
        "pchs_avg_pric": "54388.4874",
        "pchs_amt": "0",
        "now_pric": "75800",
        "evlu_amt": "132195200",
        "evlu_pfls_amt": "37341678",
        "evlu_pfls_rt": "39.36"
    },
    "rt_cd": "0",
    "msg_cd": "KIOK0420",
    "msg1": "정상적으로 조회되었습니다                                                       "
}
```
