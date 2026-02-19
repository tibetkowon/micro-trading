# ADR-003: orders-compact 초기 렌더링을 HTMX-only로 변경

- **날짜**: 2026-02-19
- **상태**: 확정(Accepted)
- **Phase**: 20 (버그 수정)

---

## Context

`portfolio_sidebar.html`은 최초 렌더링 시 정적 `{% include %}`와 HTMX 폴링을 병행했습니다.

```html
<!-- 기존 방식 -->
<div hx-get="/partials/orders-compact" hx-trigger="load, every 15s" hx-swap="innerHTML">
    <div class="sidebar-title">최근 주문</div>
    {% include "partials/orders_compact.html" %}  ← 정적 include
</div>
```

문제 발생:
- Phase 20에서 `orders_compact.html`이 `name_map` 딕셔너리를 필요로 하게 됨
- `portfolio_sidebar.html`은 `trading.html` → `trading_view()` 컨텍스트로 렌더링됨
- `trading_view()`는 `name_map`을 전달하지 않음 → 정적 include 시 `name_map` 변수 없음
- 결과: 종목명이 표시되지 않거나 템플릿 오류 발생

---

## Decision

정적 `{% include %}`를 제거하고, HTMX `load` 트리거만으로 렌더링한다.

```html
<!-- 변경 후 -->
<div hx-get="/partials/orders-compact"
     hx-trigger="load, every 15s, refreshSidebar from:body"
     hx-swap="innerHTML">
    <div class="sidebar-title">최근 주문</div>
    <div class="wl-empty">불러오는 중...</div>  ← 로딩 플레이스홀더
</div>
```

---

## Rationale

**왜 정적 include를 제거하는가:**
- 정적 include는 페이지 컨텍스트를 그대로 공유함 → partial이 필요한 모든 변수를 부모 라우트가 제공해야 함
- HTMX partial은 전용 라우트에서 필요한 데이터를 완전히 자급함 → 컨텍스트 독립적

**왜 "불러오는 중..." 플레이스홀더를 넣는가:**
- 정적 include 제거 후 첫 로드 시 빈 영역이 생겨 레이아웃이 어색해 보임
- HTMX `load` 트리거가 발동되기 전(수ms) 짧은 플레이스홀더로 UX 보완

**`refreshSidebar` 이벤트 추가:**
- 주문 제출 후 사이드바 갱신을 위한 `refreshSidebar` 이벤트도 트리거에 추가
- 기존 `trading.js`의 이벤트 리스너와 통합

---

## Consequences

**이득:**
- orders-compact 파셜이 컨텍스트에 완전히 독립적 → 어느 페이지에서도 재사용 가능
- `name_map`, `commission_map` 등 추가 데이터가 늘어도 부모 라우트 수정 불필요

**트레이드오프:**
- 첫 페이지 로드 시 orders 영역이 ~100ms 지연 후 표시됨 (HTMX load 시간)
- 플레이스홀더 텍스트("불러오는 중...")가 순간적으로 노출됨
