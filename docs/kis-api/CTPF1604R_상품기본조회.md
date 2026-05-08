# 상품기본조회

| 항목 | 값 |
|------|---|
| 통신방식 | `REST` |
| API 명 | `상품기본조회` |
| API ID | `v1_국내주식-029` |
| 실전 TR_ID | `CTPF1604R` |
| 모의 TR_ID | `모의투자 미지원` |
| Method | `GET` |
| URL | `/uapi/domestic-stock/v1/quotations/search-info` |

- 실전: `https://openapi.koreainvestment.com:9443`
- 모의: `모의투자 미지원`

## Request Header

> 공통 헤더는 [_공통헤더.md](_공통헤더.md) 참조

| Element | 한글명 | Type | Required | Description |
|---------|--------|------|----------|-------------|
| `PDNO` | 상품번호 | string | Y | '주식(하이닉스) : 000660 (코드 : 300); 선물(101S12) : KR4101SC0009 (코드 : 301); 미국(AAPL) : AAPL (코드 : 512)' |
| `PRDT_TYPE_CD` | 상품유형코드 | string | Y | '300 주식 → [공통코드](_공통코드.md#PRDT_TYPE_CD) 참조 |

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
| `pdno` | 상품번호 | string | Y |  |
| `prdt_type_cd` | 상품유형코드 | string | Y |  |
| `prdt_name` | 상품명 | string | Y |  |
| `prdt_name120` | 상품명120 | string | Y |  |
| `prdt_abrv_name` | 상품약어명 | string | Y |  |
| `prdt_eng_name` | 상품영문명 | string | Y |  |
| `prdt_eng_name120` | 상품영문명120 | string | Y |  |
| `prdt_eng_abrv_name` | 상품영문약어명 | string | Y |  |
| `std_pdno` | 표준상품번호 | string | Y |  |
| `shtn_pdno` | 단축상품번호 | string | Y |  |
| `prdt_sale_stat_cd` | 상품판매상태코드 | string | Y |  |
| `prdt_risk_grad_cd` | 상품위험등급코드 | string | Y |  |
| `prdt_clsf_cd` | 상품분류코드 | string | Y |  |
| `prdt_clsf_name` | 상품분류명 | string | Y |  |
| `sale_strt_dt` | 판매시작일자 | string | Y |  |
| `sale_end_dt` | 판매종료일자 | string | Y |  |
| `wrap_asst_type_cd` | 랩어카운트자산유형코드 | string | Y |  |
| `ivst_prdt_type_cd` | 투자상품유형코드 | string | Y |  |
| `ivst_prdt_type_cd_name` | 투자상품유형코드명 | string | Y |  |
| `frst_erlm_dt` | 최초등록일자 | string | Y |  |

## Example

### Request Example (Python)
```json
{
	"PDNO":"AAPL",
	"PRDT_TYPE_CD":"512"
}
```

### Response Example
```json
{
    "output": {
        "pdno": "AAPL",
        "prdt_type_cd": "512",
        "prdt_name": "애플",
        "prdt_name120": "애플",
        "prdt_abrv_name": "애플",
        "prdt_eng_name": "APPLE INC",
        "prdt_eng_name120": "APPLE INC",
        "prdt_eng_abrv_name": "APPLE INC",
        "std_pdno": "US0378331005",
        "shtn_pdno": "AAPL",
        "prdt_sale_stat_cd": "",
        "prdt_risk_grad_cd": "",
        "prdt_clsf_cd": "101210",
        "prdt_clsf_name": "해외주식",
        "sale_strt_dt": "",
        "sale_end_dt": "",
        "wrap_asst_type_cd": "06",
        "ivst_prdt_type_cd": "1012",
        "ivst_prdt_type_cd_name": "해외주식",
        "frst_erlm_dt": ""
    },
    "rt_cd": "0",
    "msg_cd": "KIOK0530",
    "msg1": "조회되었습니다                                                                  "
}
```
