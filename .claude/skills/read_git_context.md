# Skill: Read Git Context

## Description
세션 시작 시 또는 프로젝트 현황 파악이 필요할 때, git log를 통해 최근 작업 이력을 빠르게 파악한다.
별도의 changelog 파일 없이 git이 기록한 정확한 이력을 직접 활용한다.

## Trigger
- 새 세션 시작 후 "최근에 뭐 했지?", "지금 어디까지 됐어?" 류의 질문 시
- 기능 구현 전 현재 상태 파악이 필요할 때
- 사용자가 명시적으로 "git 로그 확인해줘" 요청 시

## Instructions

1. **최근 커밋 목록 확인:**
   ```bash
   git log --oneline -20
   ```

2. **변경 파일 포함 요약 (더 상세히 필요할 때):**
   ```bash
   git log --oneline --stat -5
   ```

3. **특정 영역 이력 (특정 기능/파일 관련 작업 파악 시):**
   ```bash
   # backend 변경 이력
   git log --oneline -10 -- backend/

   # frontend 변경 이력
   git log --oneline -10 -- frontend/
   ```

4. **마지막 커밋 상세 내용:**
   ```bash
   git show --stat HEAD
   ```

5. **Report:** 파악한 내용을 한국어로 간략히 요약하여 사용자에게 전달.
   현재 진행 중인 작업, 최근 완료된 기능, 다음 단계를 정리해서 보고.

## 주의
- 전체 git log를 읽지 말 것 — `-20` 제한 유지
- 커밋 메시지만으로 파악이 어려우면 `git show <hash>` 로 특정 커밋만 상세 확인
