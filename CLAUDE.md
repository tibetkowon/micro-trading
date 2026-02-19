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

## Skills
Custom validation and maintenance skills are defined in `.claude/skills/`.

| Skill | Purpose |
| :--- | :--- |
| [`verify-implementation`](./.claude/skills/verify-implementation.md) | 통합 검증 보고서 생성 및 실행 |
| [`manage-skills`](./.claude/skills/manage-skills.md) | 세션 변경사항 분석 및 스킬/CLAUDE.md 관리 |
| [`study-mentor`](./.claude/skills/study-mentor.md) | 구현 로직 설명 및 자바 비교 분석 학습 문서 생성 |
| [`adr`](./.claude/skills/adr.md) | 설계 결정 사항 및 배경 기록 (Architecture Decision Record) |
| [`context-slimming`](./.claude/skills/context-slimming.md) | 토큰 절약 및 코드 최적화 관리 |
