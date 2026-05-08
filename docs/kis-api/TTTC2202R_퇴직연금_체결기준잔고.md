# 퇴직연금 체결기준잔고

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `퇴직연금 체결기준잔고` |
| API ID | `v1_국내주식-032` |
| 실전 TR_ID | `TTTC2202R` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/trading/pension/inquire-present-balance` |

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
| `USER_DVSN_CD` | 사용자구분코드 | string | Y | 00 |
| `CTX_AREA_FK100` | 연속조회검색조건100 | string | Y |  |
| `CTX_AREA_NK100` | 연속조회키100 | string | Y |  |
| `PRCS_DVSN_CD` | 처리구분코드 | string | N | 00 : 보유 주식 전체 조회; 01 : 보유 주식 중 0주 주식 숨김 |

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
| `output1` | 응답상세1 | object array | Y | Array |
| `cblc_dvsn` | 잔고구분 | string | Y |  |
| `cblc_dvsn_name` | 잔고구분명 | string | Y |  |
| `pdno` | 상품번호 | string | Y |  |
| `prdt_name` | 상품명 | string | Y |  |
| `hldg_qty` | 보유수량 | string | Y |  |
| `slpsb_qty` | 매도가능수량 | string | Y |  |
| `pchs_avg_pric` | 매입평균가격 | string | Y |  |
| `evlu_pfls_amt` | 평가손익금액 | string | Y |  |
| `evlu_pfls_rt` | 평가손익율 | string | Y |  |
| `prpr` | 현재가 | string | Y |  |
| `evlu_amt` | 평가금액 | string | Y |  |
| `pchs_amt` | 매입금액 | string | Y |  |
| `cblc_weit` | 잔고비중 | string | Y |  |
| `output2` | 응답상세2 | object array | Y | Array |
| `pchs_amt_smtl_amt` | 매입금액합계금액 | string | Y |  |
| `evlu_amt_smtl_amt` | 평가금액합계금액 | string | Y |  |
| `evlu_pfls_smtl_amt` | 평가손익합계금액 | string | Y |  |
| `trad_pfls_smtl` | 매매손익합계 | string | Y |  |
| `thdt_tot_pfls_amt` | 당일총손익금액 | string | Y |  |
| `pftrt` | 수익률 | string | Y |  |

## Example

### Request Example (Python)
```json
{
	"CANO":"63512345",
	"ACNT_PRDT_CD":"29",
	"USER_DVSN_CD":"00",
	"CTX_AREA_FK100":"",
	"CTX_AREA_NK100":""
}
```

### Response Example
```json
{
    "ctx_area_fk100": "63512345^29^00^                                                                                     ",
    "ctx_area_nk100": "                                                                                                    ",
    "output1": [
        {
            "cblc_dvsn": "01",
            "cblc_dvsn_name": "사용자",
            "pdno": "069500",
            "prdt_name": "KODEX 200",
            "hldg_qty": "6",
            "slpsb_qty": "6",
            "pchs_avg_pric": "35670.0000",
            "evlu_pfls_amt": "-3330",
            "evlu_pfls_rt": "-1.56",
            "prpr": "35115",
            "evlu_amt": "210690",
            "pchs_amt": "214020",
            "cblc_weit": "53.06651890"
        },
        {
            "cblc_dvsn": "01",
            "cblc_dvsn_name": "사용자",
            "pdno": "091160",
            "prdt_name": "KODEX 반도체",
            "hldg_qty": "7",
            "slpsb_qty": "7",
            "pchs_avg_pric": "35820.0000",
            "evlu_pfls_amt": "-64400",
            "evlu_pfls_rt": "-25.68",
            "prpr": "26620",
            "evlu_amt": "186340",
            "pchs_amt": "250740",
            "cblc_weit": "46.93348110"
        }
    ],
    "output2": [
        {
            "pchs_amt_smtl_amt": "464760",
            "evlu_amt_smtl_amt": "397030",
            "evlu_pfls_smtl_amt": "-67730",
            "trad_pfls_smtl": "0",
            "thdt_tot_pfls_amt": "-67730",
            "pftrt": "-14.57311300"
        }
    ],
    "rt_cd": "0",
    "msg_cd": "KIOK0510",
    "msg1": "조회가 완료되었습니다                                                           "
}
```
