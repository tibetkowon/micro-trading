# Manage Skills Skill

## Purpose
- 세션 중 발생하는 반복적인 오류나 새로운 요구사항을 감지하여 `.claude/skills/`의 지침을 업데이트한다.
- `CLAUDE.md`의 내용이 너무 길어지면 적절히 스킬 파일로 분리하여 토큰 효율을 높인다.

## Maintenance Rules
- 스킬 파일은 항상 독립적이고 구체적이어야 한다.
- 새로운 스킬을 생성할 때는 `CLAUDE.md`의 `## Skills` 테이블에도 자동으로 추가한다.
- 중복되는 지침은 하나로 통합하여 클로드의 혼란을 방지한다.
