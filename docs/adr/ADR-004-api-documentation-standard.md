# ADR-004: FastAPI 엔드포인트 한글 명세 의무화 (AI 연동 기준)

- **날짜**: 2026-02-19
- **상태**: 확정(Accepted)
- **Phase**: Development Rules 적용

---

## Context

프로젝트가 Phase 9 이후 실계좌 연동을 지원하고, Phase 16-19에서 AI 자동매매용 API가 추가되었습니다.

외부 AI(Claude, GPT 등)가 이 API를 직접 호출하여 트레이딩 결정을 내릴 때:
- 엔드포인트 목적이 불명확하면 AI가 잘못된 API를 호출하거나 파라미터를 오해할 수 있음
- Pydantic 필드에 설명이 없으면 AI가 각 필드의 의미를 추론해야 함 → 오류 가능성 증가
- FastAPI의 자동 생성 Swagger UI(`/api/docs`)도 설명 없이는 개발자에게 불친절함

---

## Decision

모든 FastAPI 엔드포인트와 Pydantic 모델 필드에 한글 문서를 의무화한다.

**엔드포인트:**
```python
@router.get(
    "/orderable",
    response_model=OrderableResponse,
    summary="주문가능 금액 조회 (AI용)",
    description="수수료를 반영한 실질 주문가능 금액을 반환합니다. ...",
)
```

**Pydantic 필드:**
```python
class OrderCreate(BaseModel):
    symbol: str = Field(..., description="종목 코드 (예: '005930', 'AAPL')")
    side: OrderSide = Field(..., description="주문 방향 (BUY: 매수, SELL: 매도)")
```

---

## Rationale

1. **AI Function Calling 호환성**: Claude/GPT의 함수 호출(Function Calling) 기능은 파라미터 `description`을 읽어 어떤 값을 넣을지 판단함. 설명이 없으면 AI가 임의의 값을 사용할 수 있음
2. **Swagger UI 품질**: `/api/docs`에서 개발자가 즉시 이해할 수 있는 API 문서 자동 생성
3. **온보딩 비용 감소**: 새 세션에서 AI가 코드를 읽지 않아도 API 문서만으로 기능 파악 가능

---

## Consequences

**이득:**
- AI 자동매매 연동 시 오류율 감소
- 신규 개발자(또는 새 AI 세션) 온보딩 시간 단축
- FastAPI Swagger 문서 품질 향상

**트레이드오프:**
- 신규 엔드포인트 작성 시 summary/description 추가 필요 → 약간의 작성 부담
- 기능 변경 시 설명도 함께 업데이트해야 함 (문서 drift 주의)
