# Verify Implementation Skill

## Verification Steps
1. **Linting Check**: `ruff` 또는 `flake8`을 사용하여 파이썬 코드 스타일을 확인한다.
2. **Type Safety**: `mypy`를 사용하여 타입 힌트가 누락되었거나 잘못되었는지 검사한다.
3. **Memory Audit**: 새로 추가된 라이브러리가 메모리를 과도하게 점유하는지 확인한다.
4. **Test Coverage**: 작성된 로직에 대해 유닛 테스트가 존재하는지 확인하고 실행한다.

## Reporting
- 검증 결과는 반드시 [PASS/FAIL] 형식으로 요약 보고서를 출력한다.
- 실패 시 구체적인 수정 제안을 포함해야 한다.
