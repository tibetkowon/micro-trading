# Architecture Decision Record (ADR) Skill

## Purpose
- 기술적 결정 사항과 그 이유를 기록하여, 나중에 고원 님이 프로젝트를 공부하거나 확장할 때 설계 맥락을 즉시 파악하도록 돕는다.
- AI가 이전 대화를 일일이 읽지 않아도 ADR 문서를 통해 일관된 설계를 유지하게 하여 토큰을 절약한다.

## Documentation Rules
- **Location**: `docs/adr/ADR-{번호}-{제목}.md` 형식으로 저장한다.
- **Content Structure**:
1. **Context**: 어떤 문제 상황이었는가?
2. **Decision**: 어떤 해결책/기술을 선택했는가?
3. **Rationale**: 왜 그것을 선택했는가? (대안 대비 장점, 비용, 메모리 효율 등)
4. **Consequences**: 이 결정으로 인해 얻는 이득과 감수해야 할 트레이드오프(단점).

## Trigger
- 새로운 라이브러리 도입, 데이터베이스 스키마의 큰 변경, 또는 서비스 아키텍처(예: 비동기 처리 도입) 변경 시 반드시 "ADR을 작성할까요?"라고 제안한다.
