# Micro-Trading (Python)

## Core Context
- **Goal**: Lightweight trading bot for NCP Micro server.
- **Requirements**: Refer to `requirements.md` for functional specs.
- **Resources**: 1GB Physical RAM + 2GB Swap Memory.
- **Stack**: Python 3.11+, FastAPI, SQLite, CCXT.
- **Constraints**: Optimize memory usage. Refer to `.claude/skills/memory.md`.

## Essential Commands
- **Env**: `source .venv/bin/activate`
- **Install**: `pip install -r requirements.txt`
- **Run**: `uvicorn main:app --reload`
- **Test**: `pytest`

## Critical Rules
- Use Korean for docstrings and comments.
- Always follow `memory.md` for any code changes.

## Web Pages (현행 구조)

메인 UI는 `/` (3-패널 트레이딩 뷰). 나머지는 보조 페이지.

| 경로 | 목적 | 비고 |
| :--- | :--- | :--- |
| `/` | 메인 트레이딩 뷰 (관심종목·주문·포트폴리오) | 주요 진입점 |
| `/portfolio` | 포트폴리오 차트 (수익 히스토리) | 고유 기능 |
| `/settings` | KIS API 키·거래모드 설정 | |

- **제거된 페이지**: `/orders` — 메인 뷰에서 완전히 대체됨 (2025-02 제거)
- **메뉴 미노출**: `/strategies` — 백엔드 라우트 유지, 네비게이션에서 제거 (2026-02, Phase 21)
- 레거시 페이지를 새로 만들지 말 것. 기능은 메인 뷰 패널에 통합한다.

## 네비게이션 메뉴 (통일 기준)

모든 페이지(`base.html`, `trading_base.html`)의 nav는 동일하게 유지:
**트레이딩 | 포트폴리오 | 설정**

## Jinja2 Template Filters

`web/routes.py`에 등록된 커스텀 필터:

| 필터 | 설명 | 예시 |
| :--- | :--- | :--- |
| `to_kst` | UTC datetime → KST `MM-DD HH:MM` 문자열 | `{{ o.created_at \| to_kst }}` |
| `source_label` | 주문 출처 한국어 변환 | `{{ o.source \| source_label }}` → `직접주문`/`자동전략` |

## AI용 API 엔드포인트

| 경로 | 목적 |
| :--- | :--- |
| `GET /api/portfolio/orderable` | 주문가능 금액 (수수료 0.3% 반영) |
| `GET /api/portfolio/pnl-analysis` | 종목별 실현손익 + 일별 수익률 |

## Web Route 규칙

### HTMX Partial 뮤테이션 패턴
DB를 변경하는 HTMX 라우트는 변경 후 **갱신된 HTML partial을 그대로 반환**한다.

```python
# 예: 관심종목 추가/삭제 → watchlist_items.html 반환
@web_router.post("/watchlist/item")
async def add_watchlist_item(...):
    await svc.add(symbol, market, name)
    items = await _build_watchlist_items(svc, market_svc)
    return templates.TemplateResponse("partials/watchlist_items.html", {...})
```

- 뮤테이션 전용 라우트는 `/watchlist/item` 처럼 리소스 중심 경로 사용
- 기존 `/stocks/memo` 엔드포인트는 레거시 메모 테이블 전용으로 유지

## Skills
Custom validation and maintenance skills are defined in `.claude/skills/`.

| Skill | Purpose |
| :--- | :--- |
| [`verify-implementation`](./.claude/skills/verify-implementation.md) | 통합 검증 보고서 생성 및 실행 |
| [`manage-skills`](./.claude/skills/manage-skills.md) | 세션 변경사항 분석 및 스킬/CLAUDE.md 관리 |
| [`study-mentor`](./.claude/skills/study-mentor.md) | 구현 로직 설명 및 자바 비교 분석 학습 문서 생성 |
| [`adr`](./.claude/skills/adr.md) | 설계 결정 사항 및 배경 기록 (Architecture Decision Record) |
| [`context-slimming`](./.claude/skills/context-slimming.md) | 토큰 절약 및 코드 최적화 관리 |
| [`kis-api`](./.claude/skills/kis-api.md) | KIS OpenAPI 연동 필수 헤더, tr_id 매핑, 에러 처리 패턴, 토큰 세션 관리 |
