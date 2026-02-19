# ADR-002: KIS API 5xx 오류 재시도 — 재귀 패턴 + asyncio.sleep

- **날짜**: 2026-02-19
- **상태**: 확정(Accepted)
- **Phase**: 17

---

## Context

KIS API 서버가 일시적으로 503/500 오류를 반환하는 경우가 발생합니다.
특히 장 개시 시간(09:00 KST) 전후 트래픽 집중 시 빈도가 높습니다.

현황:
- 기존: 401(토큰 만료) 재시도만 구현됨
- 5xx 오류 시: 즉시 `HTTPStatusError` 발생 → 주문 REJECTED 처리
- AI 자동매매가 활성화되면 5xx 오류로 인한 주문 실패가 누적될 수 있음

요구사항:
- 5xx 일시 오류 시 자동으로 재시도해야 함
- 주문 중복 전송 금지 (5xx는 서버가 주문을 처리하지 않은 상태임)
- 메모리·의존성 추가 없이 구현

---

## Decision

`KISClient.post()` 메서드에 **재귀 재시도 + asyncio.sleep(1)** 패턴을 적용한다.

```python
async def post(self, ..., _retry: int = 2) -> dict:
    # ... (기존 로직)
    if resp.status_code >= 500 and _retry > 0:
        await asyncio.sleep(1)
        return await self.post(path, tr_id, body, use_hashkey=use_hashkey, _retry=_retry - 1)
```

- 최대 재시도 2회 (총 3회 시도)
- 재시도 간 1초 대기
- 401은 기존 즉시 재시도 유지, 5xx만 backoff 적용

---

## Rationale

**왜 재귀인가:**
- 반복문(while) 대비 상태 관리가 단순함
- `_retry` 값이 자연스럽게 감소하는 tail-recursive 구조
- 코드 가독성이 높음

**왜 외부 라이브러리(tenacity 등)를 쓰지 않았는가:**
- 1GB RAM 제약으로 의존성 최소화 원칙
- 요구사항이 단순함 (5xx, 최대 2회, 1초 딜레이) → 직접 구현이 더 가벼움

**왜 5xx만 재시도하는가:**
- 5xx: 서버가 요청을 처리하지 못함 → 재시도 안전
- 4xx(401 제외): 잘못된 요청 → 재시도해도 동일 실패 (잔고 부족 등)
- 401: 토큰 갱신 후 즉시 재시도 (딜레이 불필요)

대안:
- 옵션 A: `tenacity` 라이브러리 → 강력하지만 의존성 추가
- 옵션 B: `asyncio.wait_for` + while 루프 → 코드가 더 복잡
- **선택 옵션 C**: 재귀 재시도 → 단순, 의존성 없음

---

## Consequences

**이득:**
- 장 개시 시간대 KIS API 일시 오류로 인한 주문 실패율 감소
- 외부 라이브러리 없이 구현

**트레이드오프:**
- 재시도 총 시간: 최대 2초 추가 지연 (사용자 체감 가능)
- 5xx가 서버 장애로 이어지면 3회 시도 후 최종 실패 (적절한 한계)
- `GET` 요청은 재시도 미적용 (읽기는 멱등성 보장으로 추후 추가 검토 가능)
