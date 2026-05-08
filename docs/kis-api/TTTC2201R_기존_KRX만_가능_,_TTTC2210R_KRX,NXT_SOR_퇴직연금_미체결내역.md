# 퇴직연금 미체결내역

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `퇴직연금 미체결내역` |
| API ID | `v1_국내주식-033` |
| 실전 TR_ID | `TTTC2201R(기존 KRX만 가능), TTTC2210R (KRX,NXT/SOR)` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/trading/pension/inquire-daily-ccld` |

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
| `USER_DVSN_CD` | 사용자구분코드 | string | Y | %% |
| `SLL_BUY_DVSN_CD` | 매도매수구분코드 | string | Y | 00 : 전체 / 01 : 매도 / 02 : 매수 |
| `CCLD_NCCS_DVSN` | 체결미체결구분 | string | Y | %% : 전체 / 01 : 체결 / 02 : 미체결 |
| `INQR_DVSN_3` | 조회구분3 | string | Y | 00 : 전체 |
| `CTX_AREA_FK100` | 연속조회검색조건100 | string | Y |  |
| `CTX_AREA_NK100` | 연속조회키100 | string | Y |  |

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
| `output` | 응답상세1 | object array | Y | Array |
| `ord_gno_brno` | 주문채번지점번호 | string | Y |  |
| `sll_buy_dvsn_cd` | 매도매수구분코드 | string | Y |  |
| `trad_dvsn_name` | 매매구분명 | string | Y |  |
| `odno` | 주문번호 | string | Y |  |
| `pdno` | 상품번호 | string | Y |  |
| `prdt_name` | 상품명 | string | Y |  |
| `ord_unpr` | 주문단가 | string | Y |  |
| `ord_qty` | 주문수량 | string | Y |  |
| `tot_ccld_qty` | 총체결수량 | string | Y |  |
| `nccs_qty` | 미체결수량 | string | Y |  |
| `ord_dvsn_cd` | 주문구분코드 | string | Y |  |
| `ord_dvsn_name` | 주문구분명 | string | Y |  |
| `orgn_odno` | 원주문번호 | string | Y |  |
| `ord_tmd` | 주문시각 | string | Y |  |
| `objt_cust_dvsn_name` | 대상고객구분명 | string | Y |  |
| `pchs_avg_pric` | 매입평균가격 | string | Y |  |
| `stpm_cndt_pric` | 스톱지정가조건가격 | string | Y | 신규 API용 필드 |
| `stpm_efct_occr_dtmd` | 스톱지정가효력발생상세시각 | string | Y | 신규 API용 필드 |
| `stpm_efct_occr_yn` | 스톱지정가효력발생여부 | string | Y | 신규 API용 필드 |
| `excg_id_dvsn_cd` | 거래소ID구분코드 | string | Y | 신규 API용 필드 |

## Example

### Request Example (Python)
```json
{
	"CANO":"63512345",
	"ACNT_PRDT_CD":"29",
	"USER_DVSN_CD":"%%",
	"SLL_BUY_DVSN_CD":"00",
	"CCLD_NCCS_DVSN":"%%",
	"INQR_DVSN_3":"00",
	"CTX_AREA_FK100":"",
	"CTX_AREA_NK100":""
}
```

### Response Example
```json
{
    "ctx_area_fk100": "63512345^29^%%^00^%%^00^                                                                            ",
    "ctx_area_nk100": "^^                                                                                                  ",
    "output": [],
    "rt_cd": "0",
    "msg_cd": "KIOK0490",
    "msg1": "조회가 계속됩니다                                                               "
}
```
