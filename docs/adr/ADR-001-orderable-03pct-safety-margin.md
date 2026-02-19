# ADR-001: 실계좌 주문가능 금액 계산에 0.3% 안전 마진 적용

- **날짜**: 2026-02-19
- **상태**: 확정(Accepted)
- **Phase**: 16

---

## Context

주문 가능 금액을 계산할 때 어떤 수수료율을 기준으로 할지 결정이 필요했습니다.

현황:
- 모의투자 수수료: 0.05% (`PAPER_COMMISSION_RATE`)
- 실거래 수수료: 0.15% (`REAL_COMMISSION_RATE`, `.env` 설정값)
- AI 자동매매가 주문가능 금액을 조회하여 최대 주문 수량을 계산함

문제:
- `.env`의 `REAL_COMMISSION_RATE=0.0015`를 그대로 orderable 계산에 쓰면,
  환경에 따라 실제 KIS 수수료가 더 높을 경우 AI 주문이 잔고 부족으로 실패할 수 있음
- AI는 경계값 근처에서 주문하므로 작은 차이가 주문 실패로 이어짐

---

## Decision

실계좌 orderable 계산 전용으로 **0.3%** 고정값을 사용한다.

```python
# portfolio_service.py
commission_rate = account.commission_rate if is_paper else 0.003
orderable_krw = round(cash_krw / (1 + commission_rate), 2)
```

- `.env`의 `REAL_COMMISSION_RATE`(실제 거래 수수료)와는 별개로 관리
- orderable 계산은 항상 0.3%로 고정 (환경 변수 무관)

---

## Rationale

1. **보수적 안전 마진**: 0.3%는 KIS 실거래 기본 수수료(0.015% 위탁 + 세금 + 기타)를 충분히 커버함
2. **AI 주문 실패 방지**: 자동매매 AI가 반환된 orderable_krw로 주문하면 절대 잔고 부족이 발생하지 않아야 함
3. **단순성**: 설정 파일 의존 없이 코드에 고정값으로 명확하게 표현

대안:
- 옵션 A: `REAL_COMMISSION_RATE` 그대로 사용 → 환경에 따라 주문 실패 가능성 있음
- 옵션 B: 별도 `ORDERABLE_SAFETY_RATE` 환경변수 추가 → 설정 파일 복잡성 증가
- **선택 옵션 C**: 코드 내 0.003 고정 → 단순하고 안전

---

## Consequences

**이득:**
- AI 자동매매의 주문 실패율 감소
- 간단하고 예측 가능한 동작

**트레이드오프:**
- 실제 수수료보다 보수적으로 계산하므로 orderable_krw가 실제보다 약간 낮게 표시됨
  (예: 예수금 1,000,000원 → 실제 997,009원이지만 표시는 997,009원보다 낮은 997,009원 - 사실상 동일)
- 0.3%가 실제 수수료와 다름을 문서화로 명확히 해야 함
