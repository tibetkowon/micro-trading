# Micro-Trading (Python)

## Core Context
- **Goal**: Lightweight trading bot for NCP Micro server.
- **Requirements**: Refer to `requirements.md` for functional specs.
- **Resources**: 1GB Physical RAM + 2GB Swap Memory.
- **Stack**: Python 3.11+, FastAPI, SQLite, CCXT.
- **Constraints**: Optimize memory usage. Refer to `.claude/skills/memory.md`.

## Essential Commands
- **Env**: `source venv/bin/activate`
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
| `verify-implementation` | Runs all verify skills sequentially to generate an integrated report. |
| `manage-skills` | Analyzes session changes, updates skills, and manages CLAUDE.md. |
| `study-mentor` | 구현 로직 설명 및 자바 비교 분석을 포함한 학습용 문서 생성 |
