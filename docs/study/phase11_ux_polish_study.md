# Phase 11: KIS 발급 가이드 & 포트폴리오 차트 버그 수정 - 학습 문서

## 1. Overview: 왜 이 작업이 필요한가?

두 가지 UX 문제를 해결한다:

1. **KIS API 키 발급 가이드 부재**: 설정 페이지에서 "KIS API 키를 먼저 설정하세요"라고만 알려줄 뿐, **어디서 어떻게 발급받는지** 안내가 없었다. 신규 사용자는 `.env`에 무엇을 넣어야 하는지 모른다.

2. **포트폴리오 차트가 무한히 늘어나는 버그**: `/portfolio` 페이지의 Chart.js 캔버스가 페이지를 열 때마다 아래로 계속 커지는 현상. 화면이 사용 불가 수준으로 깨진다.

---

## 2. Logic Flow

### 2-1. KIS API 키 발급 가이드 (settings.html)

```
기존:
  <p class="settings-hint">
      API 키는 .env 파일에서 설정합니다: KIS_APP_KEY, KIS_APP_SECRET, KIS_ACCOUNT_NUMBER
  </p>

변경 후:
  <details class="settings-guide">
      <summary>KIS API 키 발급 가이드</summary>
      <div>
          - 필요한 키 3가지 목록
          - 발급 절차 6단계 (계좌 개설 → KIS Developers 가입 → 앱 등록 → ...)
          - .env 설정 예시 (pre 태그)
          - 모의투자/실거래 키 별도 발급 안내
      </div>
  </details>
```

핵심: `<p>` 한 줄짜리 힌트를 `<details>/<summary>` 접기/펼치기 블록으로 교체.
기본적으로 접힌 상태여서 기존 UI를 방해하지 않고, 필요할 때만 펼쳐서 볼 수 있다.

### 2-2. 포트폴리오 차트 버그 수정 (portfolio.html)

**버그 원인 분석:**

```
기존 HTML:
  <div class="chart-card">           ← 높이 제한 없음
      <canvas height="300"></canvas>  ← Chart.js가 이 높이를 무시
  </div>

기존 JS 옵션:
  responsive: true,           ← 부모 크기에 맞춰 리사이즈
  maintainAspectRatio: false,  ← 비율 무시, 부모 높이를 따름
```

Chart.js의 리사이즈 동작:
1. `responsive: true` → canvas 크기를 부모 컨테이너에 맞춤
2. `maintainAspectRatio: false` → 종횡비 무시, **부모의 높이**를 그대로 사용
3. 부모(`.chart-card`)에 고정 높이가 없음 → canvas가 렌더링되면서 부모 높이가 커짐
4. 부모가 커지면 Chart.js가 다시 리사이즈 → **무한 확장 루프**

**수정:**

```html
<div class="chart-card">
    <div class="card-label">포트폴리오 추이</div>
    <div style="position: relative; height: 300px;">  ← 고정 높이 wrapper
        <canvas id="portfolio-chart"></canvas>         ← height 속성 제거
    </div>
</div>
```

- `position: relative` + `height: 300px` wrapper가 Chart.js에게 **명확한 높이 경계**를 제공
- canvas의 `height="300"` HTML 속성은 제거 (wrapper가 제어)
- Chart.js가 wrapper 안에서만 리사이즈하므로 무한 확장이 발생하지 않음

---

## 3. Pythonic Point: HTML/CSS이지만 알아둘 패턴

### 3-1. `<details>/<summary>` 네이티브 접기/펼치기

```html
<details>
    <summary>클릭하면 펼쳐짐</summary>
    <div>숨겨진 내용</div>
</details>
```

- JavaScript 없이 브라우저 네이티브로 동작
- `open` 속성을 추가하면 기본 펼침 상태: `<details open>`
- 모든 모던 브라우저 지원 (IE 제외)
- React/Vue 없는 Jinja2 템플릿 환경에서 특히 유용

### 3-2. Chart.js 반응형 설정의 올바른 패턴

Chart.js 공식 문서가 권장하는 패턴:

```html
<!-- 올바른 패턴 -->
<div style="position: relative; height: 300px;">
    <canvas id="myChart"></canvas>
</div>
```

왜 `position: relative`가 필요한가?
- Chart.js는 내부적으로 canvas에 `position: absolute`를 설정
- 부모가 `position: relative`여야 absolute가 올바르게 동작
- 이 조합이 없으면 canvas가 문서 흐름(document flow)을 벗어날 수 있음

---

## 4. Java vs Python: 프론트엔드 처리 방식 비교

### Spring(Thymeleaf) vs FastAPI(Jinja2) 템플릿

| 항목 | Spring + Thymeleaf | FastAPI + Jinja2 |
|:---|:---|:---|
| 접기/펼치기 | JavaScript 라이브러리 사용 or `th:if` 조건부 렌더링 | `<details>` 네이티브 태그 활용 |
| 차트 라이브러리 | 주로 서버 사이드 렌더링 (JFreeChart) 또는 프론트에서 Chart.js | 동일하게 Chart.js CDN 사용 |
| 템플릿 상속 | `th:fragment` + `th:replace` | `{% extends %}` + `{% block %}` |
| 변수 전달 | `model.addAttribute("key", value)` | Jinja2 `{{ variable }}` + `| safe` 필터 |

Spring Boot에서 같은 작업을 한다면:

```java
// Controller
@GetMapping("/settings")
public String settings(Model model) {
    model.addAttribute("kisConfigured", kisService.isConfigured());
    model.addAttribute("maskedAppKey", maskString(kisConfig.getAppKey()));
    return "settings";
}
```

```html
<!-- Thymeleaf -->
<div th:if="${!kisConfigured}">
    <details>
        <summary>KIS API 키 발급 가이드</summary>
        <!-- 동일한 HTML 구조 -->
    </details>
</div>
```

차이점: Thymeleaf는 `th:if`로 서버 사이드에서 조건부 렌더링, Jinja2는 `{% if %}`로 동일하게 처리. HTML 네이티브 `<details>` 태그는 템플릿 엔진과 무관하게 동작하므로 두 환경 모두 동일하다.

---

## 5. Key Concept: Chart.js 리사이즈 메커니즘

### 5-1. Chart.js가 canvas 크기를 결정하는 순서

```
1. responsive: false → canvas의 width/height HTML 속성 사용 (고정)
2. responsive: true + maintainAspectRatio: true → 부모 width 기준으로 비율 유지
3. responsive: true + maintainAspectRatio: false → 부모의 width AND height를 따름
```

**3번이 위험한 이유**: 부모에 명시적 높이가 없으면 canvas 자체의 렌더링 높이가 부모 높이가 되고, 그 높이를 다시 canvas가 참조하는 순환 참조가 발생한다.

### 5-2. ResizeObserver와 리사이즈 루프

Chart.js v4는 내부적으로 `ResizeObserver` API를 사용:

```
ResizeObserver가 부모 크기 변화 감지
  → canvas 크기를 부모에 맞춤
    → canvas 크기 변화가 부모에 영향
      → ResizeObserver가 다시 감지
        → 무한 루프 (브라우저가 일정 횟수 후 중단하지만, 높이는 이미 커진 상태)
```

이 문제는 Chart.js뿐 아니라 `responsive` 설정이 있는 모든 라이브러리 (ECharts, D3 등)에서 발생할 수 있다. **고정 높이 wrapper**는 보편적인 해결 패턴이다.
