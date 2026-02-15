# Phase 7: 웹 화면 개선 및 정리 - 학습 문서

## 1. Overview: 왜 화면 정리가 필요했는가?

Phase 4~6에서 기능을 빠르게 추가하면서 기술 부채(Technical Debt)가 쌓였다:

- **레이아웃 이원화**: `base.html`(레거시)과 `trading_base.html`(3패널 트레이딩 뷰) 두 개가 공존하며 nav 메뉴가 서로 달랐다
- **고아 템플릿**: `index.html`은 라우트가 없어 어디서도 접근 불가능한 죽은 파일
- **숨겨진 페이지**: `/orders`, `/portfolio`는 URL을 직접 입력해야만 접근 가능 (메뉴에 없음)
- **중복 기능**: `/positions`과 `/stocks`는 트레이딩 뷰에서 이미 제공하는 기능과 중복
- **일관성 부재**: footer가 한쪽에만 있고, 버전이 하드코딩됨

리팩토링은 새로운 기능 추가가 아니지만, **사용자 경험(UX)**과 **유지보수성** 측면에서 중요한 작업이다.

---

## 2. Logic Flow: 정리 과정 단계별 설명

### 2-1. 정리 전 화면 구조

```
라우트 → 템플릿 → 레이아웃
──────────────────────────────────────
/              → trading.html     → trading_base.html (nav: 전략만)
/orders        → orders.html      → base.html (nav: 트레이딩, 전략)
/positions     → positions.html   → base.html
/portfolio     → portfolio.html   → base.html
/stocks        → stocks.html      → base.html
/strategies    → strategies.html  → base.html
(없음)         → index.html       → base.html (고아 파일)
```

### 2-2. 정리 후 화면 구조

```
라우트 → 템플릿 → 레이아웃
──────────────────────────────────────
/              → trading.html     → trading_base.html (nav: 4개 메뉴 + footer)
/orders        → orders.html      → base.html (nav: 4개 메뉴 통일)
/portfolio     → portfolio.html   → base.html
/strategies    → strategies.html  → base.html
```

### 2-3. 삭제 대상 판단 기준

```
삭제 판단 흐름:

1. 라우트가 없는가? → index.html → 삭제 (고아 파일)

2. 기능이 중복되는가?
   /positions → 트레이딩 뷰 사이드바 + /portfolio에서 확인 가능 → 삭제
   /stocks    → 트레이딩 뷰 왼쪽 패널 검색/워치리스트로 대체됨 → 삭제

3. 독자적 가치가 있는가?
   /orders    → 주문 상세 내역 + 신규 주문 폼 → 유지
   /portfolio → 포트폴리오 차트 + 요약 → 유지
   /strategies → 전략 관리 → 유지
```

---

## 3. Pythonic Point: Jinja2 템플릿 상속과 정리 기법

### 3-1. Jinja2 템플릿 상속 구조

```
trading_base.html (레이아웃 A)    base.html (레이아웃 B)
├── nav (공통 메뉴)               ├── nav (공통 메뉴)
├── 3패널 레이아웃                 ├── 단일 컨테이너
│   ├── panel_left                │   └── content (블록)
│   ├── panel_main                ├── footer
│   └── panel_right               └── scripts (블록)
├── footer
└── scripts (블록)
```

Jinja2의 `{% extends %}` + `{% block %}` 패턴:

```html
{# trading.html #}
{% extends "trading_base.html" %}

{% block panel_left %}
  {# 왼쪽 패널 내용 #}
{% endblock %}

{% block panel_main %}
  {# 메인 패널 내용 #}
{% endblock %}
```

```html
{# orders.html #}
{% extends "base.html" %}

{% block content %}
  {# 주문 내역 #}
{% endblock %}
```

### 3-2. nav 메뉴에서 현재 페이지 하이라이트

`base.html`에서 Jinja2의 `request` 객체로 현재 URL을 판별:

```html
<a href="/portfolio"
   {% if request.url.path == '/portfolio' %}class="active"{% endif %}>
   포트폴리오
</a>
```

- FastAPI의 `Request` 객체가 Jinja2 템플릿에 자동 전달됨
- `request.url.path`로 현재 경로를 비교하여 `active` 클래스 추가
- CSS에서 `.active` 스타일로 현재 페이지를 시각적으로 구분

### 3-3. FastAPI 라우트 삭제의 영향 범위

```python
# 삭제된 라우트
@web_router.get("/positions", response_class=HTMLResponse)
async def positions_page(...):
    ...

@web_router.get("/stocks", response_class=HTMLResponse)
async def stocks_page(...):
    ...
```

라우트를 삭제하면:
- 해당 URL 접근 시 FastAPI가 자동으로 **404 Not Found** 반환
- 별도의 404 처리 코드가 필요 없음 (프레임워크가 처리)
- API 문서(`/api/docs`)에서도 자동 제거됨

단, HTMX 파셜 라우트(`/partials/positions` 등)는 유지:
- 트레이딩 뷰 사이드바에서 HTMX로 호출하기 때문
- 페이지 라우트와 파셜 라우트는 독립적

---

## 4. Java vs Python: 템플릿 엔진과 레이아웃 패턴 비교

### 4-1. 템플릿 상속 비교

**Java (Thymeleaf)**
```html
<!-- layout.html -->
<html xmlns:th="http://www.thymeleaf.org"
      xmlns:layout="http://www.ultraq.net.nz/thymeleaf/layout">
<body>
    <nav th:fragment="navigation">
        <a th:href="@{/}" th:classappend="${#httpServletRequest.requestURI == '/' ? 'active' : ''}">
            트레이딩
        </a>
    </nav>
    <div layout:fragment="content"></div>
    <footer th:fragment="footer">
        <small>Micro Trading</small>
    </footer>
</body>
</html>

<!-- orders.html -->
<html layout:decorate="~{layout}">
<div layout:fragment="content">
    주문 내역...
</div>
</html>
```

**Python (Jinja2)**
```html
<!-- base.html -->
<nav>
    <a href="/"
       {% if request.url.path == '/' %}class="active"{% endif %}>
       트레이딩
    </a>
</nav>
<main>{% block content %}{% endblock %}</main>
<footer>
    <small>Micro Trading</small>
</footer>

<!-- orders.html -->
{% extends "base.html" %}
{% block content %}
    주문 내역...
{% endblock %}
```

| 항목 | Java (Thymeleaf) | Python (Jinja2) |
|:---|:---|:---|
| 상속 키워드 | `layout:decorate` | `{% extends %}` |
| 블록 정의 | `layout:fragment` | `{% block %}` |
| 현재 URL 접근 | `#httpServletRequest.requestURI` | `request.url.path` |
| 조건부 클래스 | `th:classappend` | `{% if %}class="active"{% endif %}` |
| 설정 필요성 | Thymeleaf Layout Dialect 추가 | 없음 (Jinja2 기본 기능) |

### 4-2. 라우트(컨트롤러) 삭제 비교

**Java (Spring MVC)**
```java
// 삭제: PositionController.java
@Controller
public class PositionController {
    @GetMapping("/positions")
    public String positionsPage(Model model) {
        // ...
        return "positions";  // positions.html 렌더링
    }
}
```

**Python (FastAPI)**
```python
# 삭제: routes.py 내 함수
@web_router.get("/positions", response_class=HTMLResponse)
async def positions_page(request: Request, ...):
    return templates.TemplateResponse("positions.html", {...})
```

| 항목 | Java (Spring) | Python (FastAPI) |
|:---|:---|:---|
| 라우트 위치 | 컨트롤러 클래스 (파일별 분리) | 라우터 파일 (함수 단위) |
| 삭제 방법 | 클래스/메소드 삭제 | 함수 삭제 |
| 404 처리 | Spring이 자동 처리 | FastAPI가 자동 처리 |
| 영향 확인 | 컴파일 시 참조 오류 감지 | 런타임까지 감지 불가 → 테스트 중요 |

### 4-3. 미사용 코드 탐지

**Java**: IDE(IntelliJ)에서 "Find Usages"로 미사용 클래스/메소드 자동 감지. 컴파일 시점에서 참조 오류 확인.

**Python**: 동적 타입 언어이므로 IDE 감지가 불완전. 직접 코드 검색(`grep`)으로 확인 필요:

```bash
# 삭제 후 잔존 참조 확인
grep -r "positions\.html\|stocks\.html\|index\.html" app/
```

---

## 5. Key Concept: 초보자가 놓치기 쉬운 포인트

### 5-1. 기술 부채(Technical Debt)란?

빠르게 개발하면서 쌓이는 "나중에 고쳐야 할 것들":

```
Phase 4: base.html 만들어서 대시보드 구현
Phase 5: trading_base.html 새로 만들어서 3패널 구현
         → base.html의 nav와 trading_base.html의 nav가 달라짐 (부채 발생)
Phase 6: 배포에 집중, UI 부채 방치
Phase 7: 부채 청산 (이번 작업)
```

기술 부채를 방치하면:
- 새 기능 추가 시 "어느 레이아웃에 넣지?" 혼란
- nav 수정 시 두 파일을 동시에 고쳐야 함 (실수 가능성)
- 사용자가 기능을 찾지 못함 (메뉴에 없으므로)

### 5-2. 고아 파일(Orphan File) 문제

```
index.html이 고아가 된 과정:

1. Phase 4: index.html을 대시보드로 만듦
2. Phase 5: "/" 라우트를 trading.html로 변경
3. index.html의 라우트가 사라짐 → 고아 파일
```

고아 파일의 위험:
- 디스크 공간 낭비 (미미하지만)
- 코드 읽기 시 혼란: "이 파일은 어디서 쓰이지?"
- 보안: 만약 누군가 라우트를 다시 연결하면 의도치 않은 페이지가 노출될 수 있음

### 5-3. 레이아웃 이원화의 트레이드오프

두 레이아웃을 유지하는 이유:

```
trading_base.html
├── 3패널 레이아웃 (panel_left / panel_main / panel_right)
├── trading.css (추가 스타일시트)
├── chart.js (차트 라이브러리)
└── trading.js (전용 스크립트)

base.html
├── 단일 컬럼 레이아웃 (container)
├── style.css만
└── app.js (범용 스크립트)
```

통합하지 않은 이유:
- `trading_base.html`은 3패널 + 실시간 데이터 + 차트 라이브러리가 필요
- `base.html` 페이지(주문내역, 전략)는 단순 테이블/폼이므로 차트가 불필요
- 불필요한 JS/CSS 로딩은 1GB RAM 서버에서 메모리 낭비

대신 **nav 메뉴와 footer를 통일**하여 사용자에게는 하나의 사이트처럼 보이게 한다.

### 5-4. 파셜 라우트 vs 페이지 라우트

```
페이지 라우트 (삭제 대상):
  GET /positions → 전체 HTML 페이지 반환 (base.html 포함)

파셜 라우트 (유지):
  GET /partials/positions → 테이블 HTML 조각만 반환 (레이아웃 없음)
```

HTMX에서는 파셜 라우트만 사용한다:

```html
<!-- 트레이딩 뷰 사이드바에서 HTMX로 포지션 목록 갱신 -->
<div hx-get="/partials/positions-compact"
     hx-trigger="every 30s"
     hx-swap="innerHTML">
</div>
```

페이지 라우트를 삭제해도 HTMX 파셜이 정상 동작하는 이유:
- HTMX는 파셜 라우트(`/partials/...`)만 호출
- 페이지 라우트(`/positions`)는 브라우저 전체 페이지 로드용
- 두 라우트는 완전히 독립적

### 5-5. 삭제 전 검증 순서

안전한 코드 삭제를 위한 단계:

```
1. 삭제 대상 식별
   └── 라우트 없는 템플릿, 중복 기능 페이지

2. 참조 확인
   └── grep으로 다른 파일에서 참조하는지 확인
       - 템플릿에서 {% include %} 하는 곳?
       - JS에서 URL을 하드코딩한 곳?
       - 다른 템플릿에서 링크하는 곳?

3. 삭제 실행
   └── 라우트(routes.py) + 템플릿(.html) 동시 삭제

4. 테스트 실행
   └── pytest로 기존 기능이 깨지지 않았는지 확인

5. 잔존 참조 재확인
   └── grep으로 삭제된 파일명이 코드에 남아있지 않은지 확인
```
