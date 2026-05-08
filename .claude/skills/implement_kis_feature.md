# Skill: KIS API 기능 구현 및 개선

## Description
KIS API 기능을 구현하거나 개선할 때, `docs/kis-api/` 의 **개별 API 명세서**를
참조하여 올바른 Request/Response 구조로 구현하도록 보장한다.

## Trigger
- KIS API 신규 엔드포인트 구현 요청 시
- 기존 KIS API 관련 코드 수정/개선 요청 시

## Instructions

### 1. 대상 API 식별
구현할 기능에 필요한 **TR_ID**를 파악한다. 모르면 `docs/kis-api/_index.md` 를 읽어 검색한다.

### 2. 개별 명세서만 읽기 (토큰 절약 핵심)
- **절대 하지 말 것:** `docs/kis-api/` 폴더 전체를 스캔하거나, 관련 없는 API 명세서를 읽는 행위
- **반드시 할 것:** 필요한 API의 개별 파일만 읽는다. 파일명 규칙: `{TR_ID}_{API명}.md`
  - 예: `TTTC0802U_매수주문.md`, `HOSTCNT0_실시간체결가.md`
- 공통 헤더가 필요하면 `docs/kis-api/_공통헤더.md` 참조
- 코드 목록(산업분류 등)이 필요하면 `docs/kis-api/_공통코드.md` 참조 (대부분 불필요)

### 3. 스펙 확인
명세서에서 다음 항목을 반드시 확인한다:
- `TR_ID` (거래 ID) — 실전/모의 구분
- `Method` / `URL`
- 요청 파라미터 (Query/Body) 필드 및 필수 여부
- 응답 필드 및 타입

### 4. 구현
기존 패턴을 따라 구현한다:
- KIS 클라이언트: `backend/internal/kis/client.go` 패턴 참고
- Rate Limiter: `c.rateLimiter.Wait(ctx)` 반드시 호출
- 에러 로깅: `c.logAPIError()` 사용하여 `kis_api_logs` 테이블에 기록
- 응답 파싱: `rt_cd == "0"` 으로 성공 여부 확인

### 5. 검증
`verify_implementation.md` 스킬로 빌드/테스트 확인

---

## KIS API 알려진 Quirks

### inquire-daily-ccld (일별주문체결조회)
- **TR_ID**: `TTTC0081R` (3개월 이내). 구버전 `TTTC8001R`은 deprecated — **절대 사용하지 말 것**.
- **필수 파라미터 주의**: 아래 파라미터들이 모두 Required(Y)이며 누락 시 API 게이트웨이가 `<h1>error</h1>` HTML을 반환함.
  - `INQR_DVSN_1` — 빈 문자열(`""`) 전달 (전체)
  - `INQR_DVSN_3` — `"00"` (전체 현금/신용)
  - `EXCG_ID_DVSN_CD` — `"ALL"` (전체 거래소, 모의투자는 `"KRX"`)
- **`CANC_YN` 금지**: 이 파라미터는 공식 스펙에 없음. 전송 시 요청이 비정상 처리될 수 있으므로 **절대 사용 금지**.
- **취소 주문 필터**: 취소 여부는 응답의 `cncl_yn` 필드(`"Y"` = 취소)를 읽어 애플리케이션 레벨에서 처리.
- **페이지네이션**: 응답 최상위의 `ctx_area_fk100` / `ctx_area_nk100` 를 다음 요청 파라미터로 전달. `strings.TrimSpace()` 후 두 값 모두 빈 문자열이면 마지막 페이지.
- **날짜 파라미터**: `INQR_STRT_DT` / `INQR_END_DT` 형식 `"20060102"` (Go `time.Format` 포맷).
- **GET Content-Type**: KIS API는 GET 요청에도 `Content-Type: application/json; charset=utf-8` 헤더를 요구함 — `get()` 헬퍼에서 공통으로 설정.

### chk-holiday (영업일 조회, CTCA0903R)
- **TR_ID**: `CTCA0903R`
- **URL**: `/uapi/domestic-stock/v1/quotations/chk-holiday`
- **주요 파라미터**: `BASS_DT=YYYYMMDD` (조회 기준일), `CTX_AREA_NK=`, `CTX_AREA_FK=` (빈값)
- **응답 구조**: `output` 배열 — 기준일 포함 이후 영업일 목록. **첫 번째 원소(`[0]`)만 사용**.
- **영업일 필드**: `bzdy_yn` — `"Y"` = 영업일, `"N"` = 휴장일(공휴일/주말).
- **주의**: `rt_cd == "1"` 체크 필수 (다른 GET API와 동일, `get()` 헬퍼가 자동 처리).
- **캐시 전략**: `agent/market.go` 의 `pkgCache` 참고 — KST 날짜 키 기준 하루 1회만 호출.

### ctx_area 페이지네이션 공통 패턴
```go
fk100, nk100 := "", ""
for {
    // 요청 파라미터에 fk100, nk100 포함
    // 응답에서 ctx_area_fk100, ctx_area_nk100 추출
    fk100 = strings.TrimSpace(result.CtxAreaFK100)
    nk100 = strings.TrimSpace(result.CtxAreaNK100)
    if fk100 == "" && nk100 == "" { break }
}
```
