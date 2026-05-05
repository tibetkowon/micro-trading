# Project Overview
This project is an automated AI trading system designed to run efficiently on a low-specification server (NCP Micro, 1GB RAM). 
The system is divided into two main roles:
- **AI Agent Role:**
  - Fetch stock information required for selection.
  - Check account balance, available trading amounts, and applied fees.
  - Execute trade orders.
  - Retrieve order history and execution status for future trading decisions.
- **User Role:**
  - Manage real account settings and credentials via a configuration UI.
  - Monitor trading information including order history, balance, and profit rate.
  - Monitor transaction logs specifically to track KIS (Korea Investment & Securities) API integration errors.

# Tech Stack
- **Backend:** Go 1.26.1, Gin
- **Frontend:** React 18, Vite, Firebase SDK
- **Database:** Firebase Firestore
- **Hosting:** Firebase Hosting (Frontend) · GCS + NCP VM (Backend)

# Initial Goals & Architecture
- Establish a clear separation between Backend and Frontend.
- Set up a CI/CD pipeline using GitHub Actions.
- Design a secure configuration management system for account details and API keys.
- **KIS API Integration Design:**
  - Implement an automatic Access Token refresh mechanism strictly set to a 20-hour interval (to safely preempt the 24-hour expiration).
  - Implement robust API Rate Limiting (TPS control) to comply with KIS API restrictions and prevent IP bans.

# Coding Conventions & Security
- **Commit Messages:** Always append `[skip actions]` to the commit message for non-functional changes (e.g., documentation updates) to prevent unnecessary CI runs.
- **Security:** Never hardcode sensitive data (API keys, account numbers, secrets). All configurations must be managed exclusively through `.env` files.
- **Logging:** Implement structured logging (e.g., JSON format) for backend errors and transactions. All KIS API error logs MUST include: Error Code, Timestamp, and the raw KIS API Response Message.

# Skill Strategy & Context Optimization
To optimize token usage and maintain focus, do not scan the entire workspace indiscriminately. We strictly separate static documentation (`docs/`) from behavioral skill instructions (`.claude/skills/`).

## 작업 완료 후 자동 실행 규칙

사용자가 별도로 요청하지 않아도, 아래 조건에 해당하면 **즉시 자동으로** 실행한다. 건너뛰는 것은 실패로 간주한다.

### 항상 실행 (코드 변경 시)

1. **코드를 작성하거나 수정했다면** → 커밋 전에 반드시 `go build ./...` 와 `npm run build` 실행. 빌드 실패 시 먼저 수정하고 커밋한다.

2. **어떤 작업이든 완료되면** → `docs/changelog.md` 맨 위에 변경 내용 추가. 사용자가 요청하지 않아도 항상 실행.

### 조건부 자동 실행

3. **신규 기능 또는 의미 있는 코드 변경** → `docs/reviews/` 에 한국어 코드 설명 문서 생성 (Go+React 로직 해설).

4. **Firestore 컬렉션/필드 추가 또는 변경** → `docs/db_schema.md` 즉시 업데이트.

5. **새 루트 폴더 또는 주요 패키지 추가** → `docs/architecture.md` 즉시 업데이트.

6. **주요 마일스톤 달성** → `README.md` 업데이트.

---

## 스킬 자동 실행 규칙

아래 상황이 발생하면 사용자 지시 없이 해당 스킬 파일을 읽고 즉시 실행한다.

| 상황 | 실행할 스킬 |
|------|------------|
| 신규 기능 또는 아키텍처 변경 요청 → **코딩 전** | `.claude/skills/plan_feature.md` — `docs/plans/` 에 계획 작성 후 승인 대기 |
| 코드 작성/수정 완료 → **커밋 전** | `.claude/skills/verify_implementation.md` — Go 빌드/테스트, React 린트/빌드 실행 |
| 작업 또는 버그 수정 완료 | `.claude/skills/record_changelog.md` — `docs/changelog.md` 업데이트 |
| 신규 기능 구현 완료 | `.claude/skills/write_code_tutor.md` — `docs/reviews/` 에 한국어 설명 문서 생성 |
| KIS API 에러 조사 또는 자동매매 실패 | `.claude/skills/analyze_trade_logs.md` — 로그 스마트 추출 및 분석 |
| Firestore 컬렉션/필드 생성 또는 변경 | `.claude/skills/update_db_schema.md` — `docs/db_schema.md` 업데이트 |
| 새 루트 폴더 또는 주요 패키지 추가 | `.claude/skills/update_architecture.md` — `docs/architecture.md` 업데이트 |
| 주요 마일스톤 달성 | `.claude/skills/update_readme.md` — `README.md` 업데이트 |
| 새 API 패턴/버그 패턴/프로젝트 규칙 발견 | `.claude/skills/manage_skills.md` — 스킬 또는 문서에 기록 |
| KIS API 신규 기능 구현 또는 기존 기능 수정 | `.claude/skills/implement_kis_feature.md` — `docs/kis-api/` 명세 확인 후 구현 |