# 프로젝트 개요
저사양 서버에서 효율적으로 동작하는 **규칙 기반 자동매매 시스템**.

**매매 사이클 (반복):**
1. 종목 순위 조회 (KIS API)
2. 하드 필터링 (조건 미달 종목 제거)
3. 스코어링 (지표 기반 점수 산출)
4. 목표 점수 이상 종목 선정
   - 선정 → 해당 종목 매수
   - 미선정 → 다음 스케줄까지 대기
5. 매수 후 종목 모니터링 (목표가 / 손절가 / 지표)
6. 매도 조건 도달 시 매도
7. 반복

**사용자 역할:**
- 설정 UI를 통한 계좌 정보 및 API 키 관리
- 주문 내역, 잔고, 수익률 모니터링
- KIS API 연동 에러 로그 모니터링

# 기술 스택
- **Backend:** Go 1.26.1, Gin
- **Frontend:** React 18, Vite, Firebase SDK
- **Database:** Firebase Firestore
- **Hosting:** Firebase Hosting (Frontend) · GCS + NCP VM (Backend)

# 아키텍처 원칙
- Backend / Frontend 명확한 분리
- GitHub Actions 기반 CI/CD
- 민감 정보는 `.env`로만 관리 (절대 하드코딩 금지)
- **KIS API 핵심 설계:**
  - Access Token 자동 갱신: 만료 24시간 중 **20시간 간격**으로 갱신 (안전 마진 확보)
  - Rate Limiting (TPS 제어): KIS API 제한 준수, IP 차단 방지

# 코딩 규칙 & 보안
- **커밋 메시지:** 기능 변경이 없는 커밋(문서 업데이트 등)은 메시지에 `[skip actions]` 추가 → 불필요한 CI 실행 방지
- **보안:** API 키, 계좌번호 등 민감 데이터 하드코딩 절대 금지. 모든 설정은 `.env`로만 관리.
- **로깅:** 백엔드 에러 및 거래는 구조화 로그(JSON) 사용. KIS API 에러 로그에는 반드시 에러코드, 타임스탬프, KIS 원본 응답 메시지 포함.

# 스킬 전략 & 컨텍스트 최적화
토큰 낭비를 줄이기 위해 워크스페이스 전체를 무분별하게 스캔하지 않는다.
- `docs/kis-api/` — KIS API 명세서 (API별 개별 파일, TR_ID 기준)
- `.claude/skills/` — AI 행동 지침 (스킬 파일)

## 작업 완료 후 자동 실행 규칙

사용자가 별도로 요청하지 않아도, 아래 조건에 해당하면 **즉시 자동으로** 실행한다. 건너뛰는 것은 실패로 간주한다.

1. **코드를 작성하거나 수정했다면** → 커밋 전에 반드시 `go fmt ./...` + `go build ./...` + `npm run build` 실행. 빌드 실패 시 먼저 수정하고 커밋한다.

2. **작업 완료 후 컨텍스트 파악이 필요하면** → `git log --oneline -20` 으로 최근 이력 확인.

---

## 스킬 자동 실행 규칙

아래 상황이 발생하면 사용자 지시 없이 해당 스킬 파일을 읽고 즉시 실행한다.

| 상황 | 실행할 스킬 |
|------|------------|
| 코드 작성/수정 완료 → **커밋 전** | `.claude/skills/verify_implementation.md` — Go 빌드/테스트, React 린트/빌드 실행 |
| 세션 시작 후 최근 작업 파악 필요 시 | `.claude/skills/read_git_context.md` — `git log` 로 최근 이력 확인 |
| KIS API 에러 조사 또는 자동매매 실패 | `.claude/skills/analyze_trade_logs.md` — 로그 스마트 추출 및 분석 |
| 새 API 패턴/버그 패턴/프로젝트 규칙 발견 | `.claude/skills/manage_skills.md` — 스킬 또는 문서에 기록 |
| KIS API 신규 기능 구현 또는 기존 기능 수정 | `.claude/skills/implement_kis_feature.md` — `docs/kis-api/` 명세 확인 후 구현 |