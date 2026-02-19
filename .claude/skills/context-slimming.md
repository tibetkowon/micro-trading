# Context Slimming Skill

## Purpose
- AI가 읽어야 할 파일 크기와 범위를 최소화하여 토큰 소모를 줄이고 응답 속도를 높인다.
- 프로젝트 구조를 가볍게 유지하여 1GB RAM 환경에서의 운용 효율을 극대화한다.

## Optimization Rules
- **Snippetization**: 하나의 파일이 300라인을 넘어가면 기능별 모듈 분리를 제안한다.
- **Dead Code Elimination**: 사용되지 않는 함수, 클래스, 임포트를 주기적으로 체크하여 삭제를 제안한다.
- **Ignore Management**: 로그(`*.log`), 임시 데이터(`*.tmp`), 대형 JSON 파일 등이 `.claudeignore`에 적절히 포함되어 있는지 확인한다.
- **Prompt Compression**: AI에게 요청할 때 불필요한 서술을 줄이고 핵심 요구사항 위주로 전달하도록 가이드한다.

## Trigger
- 세션 시작 시 또는 대규모 코드 수정 전, 현재 컨텍스트에서 불필요하게 크게 잡힌 파일이 있는지 확인하고 보고한다.
