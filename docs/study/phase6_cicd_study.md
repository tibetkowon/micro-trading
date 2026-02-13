# Phase 6: GitHub Actions CI/CD - 학습 문서

## 1. Overview: 왜 CI/CD가 필요했는가?

Phase 5까지 기능 구현이 완성됐지만, 배포 과정이 수동이었다:

- **수동 배포**: 코드를 push할 때마다 직접 서버에 SSH 접속 → `git pull` → `docker compose up --build` 실행
- **테스트 없는 배포**: 깨진 코드가 메인 브랜치에 들어가면 즉시 운영 서버에 반영됨
- **배포 실패 감지 불가**: 컨테이너가 뜨더라도 실제로 동작하는지 확인하지 않음
- **반복 작업**: 같은 배포 절차를 매번 수동으로 실행

CI/CD는 "코드를 push하면 자동으로 테스트 → 배포 → 헬스체크"가 이루어지게 한다.

---

## 2. Logic Flow: CI/CD 파이프라인 전체 흐름

### 2-1. 전체 파이프라인

```
개발자: git push origin main
    │
    ▼
GitHub Actions 트리거 (on: push: branches: [main])
    │
    ▼
[JOB 1] test (ubuntu-latest)
    ├── actions/checkout@v4     → 코드 체크아웃
    ├── actions/setup-python@v5 → Python 3.11 설치
    ├── pip install .[dev]      → 의존성 설치
    └── pytest -v               → 테스트 실행
         ├── 실패 → 파이프라인 중단 (deploy 실행 안 됨)
         └── 성공 → [JOB 2] 시작
    │
    ▼
[JOB 2] deploy (needs: test)
    │  if: github.ref == 'refs/heads/main'
    │
    └── appleboy/ssh-action@v1
            │  host: ${{ secrets.NCP_HOST }}
            │  key:  ${{ secrets.NCP_SSH_KEY }}
            │
            ▼
            [NCP 서버에서 실행]
            cd /service/micro-trading
            git pull origin main
            docker compose down
            docker compose up -d --build
            │
            ▼
            Health Check (최대 30초)
            for i in 1..6:
                sleep 5
                curl -sf http://localhost:8000/api/health
                ├── 성공 → exit 0 (배포 완료)
                └── 실패 → 다음 시도
            모두 실패 → exit 1 (배포 실패 알림)
```

### 2-2. Docker 멀티스테이지 빌드

```
[Stage 1: builder]
FROM python:3.11-slim AS builder
    ├── COPY pyproject.toml, app/, run.py
    └── pip install --prefix=/install . → /install에 패키지 설치

[Stage 2: runtime]
FROM python:3.11-slim
    ├── COPY --from=builder /install → /usr/local (패키지만 복사)
    ├── COPY --from=builder /build/app → ./app (소스만 복사)
    └── COPY --from=builder /build/run.py → ./
    (빌드 도구, pip, build 캐시 등은 포함되지 않음)
```

### 2-3. Docker HEALTHCHECK 흐름

```
컨테이너 시작
    │
    ▼
start-period: 10초 대기 (앱 초기화 시간)
    │
    ▼
30초마다 실행:
python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/api/health')"
    ├── 성공 (2xx) → healthy
    └── 실패 or 5초 초과 → unhealthy 카운트
        3회 연속 실패 → 컨테이너 상태: unhealthy
```

---

## 3. Pythonic Point: CI/CD와 Docker 핵심 개념

### 3-1. GitHub Actions YAML 구조

```yaml
# .github/workflows/deploy.yml
name: CI & Deploy to NCP

on:
  push:
    branches:
      - main          # main 브랜치 push 시에만 실행

jobs:
  test:               # Job 이름 (임의)
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4   # 공개 Action 사용

      - name: Set up Python
        uses: actions/setup-python@v5
        with:
          python-version: "3.11"
          cache: "pip"              # pip 캐시 → 재실행 시 빠름

      - name: Install dependencies
        run: pip install .[dev]     # pyproject.toml의 [dev] extra 설치

      - name: Run tests
        run: pytest -v
```

### 3-2. Job 의존성: needs

```yaml
deploy:
  needs: test          # ← test job이 성공해야만 실행
  runs-on: ubuntu-latest
  if: github.ref == 'refs/heads/main'   # main 브랜치일 때만
```

- `needs: test`: `test` job이 완료(성공)되어야 `deploy` job 시작
- 실패하면 GitHub에서 빨간 X 표시 + 이메일 알림 (설정 시)
- 테스트 없이 배포되는 사고를 원천 차단

### 3-3. Secrets 관리

```yaml
uses: appleboy/ssh-action@v1
with:
  host: ${{ secrets.NCP_HOST }}         # IP 주소
  username: ${{ secrets.NCP_USERNAME }}  # SSH 사용자명
  key: ${{ secrets.NCP_SSH_KEY }}        # 비밀키 내용 (PEM 전체)
  passphrase: ${{ secrets.NCP_SSH_PASSPHRASE }}
```

- GitHub Settings → Secrets and variables → Actions에서 설정
- 코드에 직접 IP/비밀키가 노출되지 않음
- `${{ secrets.NAME }}` 형식으로 참조 (워크플로우 실행 시 주입)

### 3-4. 헬스체크 스크립트 (Bash)

```bash
# Health check (최대 30초 대기)
echo "Waiting for service to start..."
for i in $(seq 1 6); do      # 1 2 3 4 5 6 → 6번 반복
  sleep 5                    # 5초 대기
  if curl -sf http://localhost:8000/api/health > /dev/null 2>&1; then
    echo "Service is healthy!"
    exit 0                   # 성공 → 즉시 종료
  fi
  echo "Attempt $i/6: waiting..."
done
echo "WARNING: Health check failed after 30s"
exit 1                       # 실패 → CI 실패 처리
```

- `curl -sf`: `-s` silent(출력 없음), `-f` fail(HTTP 오류 시 비정상 종료)
- `> /dev/null 2>&1`: stdout과 stderr 모두 버림 (로그 안 남김)
- 6번 × 5초 = 최대 30초 대기

### 3-5. Docker 멀티스테이지 Dockerfile

```dockerfile
# Stage 1: 빌드 스테이지 (의존성 설치)
FROM python:3.11-slim AS builder

WORKDIR /build
COPY pyproject.toml .
COPY app/ app/
COPY run.py .

# --prefix=/install: 패키지를 /install 디렉토리에 설치
RUN pip install --no-cache-dir --prefix=/install .


# Stage 2: 런타임 스테이지 (실행만)
FROM python:3.11-slim

WORKDIR /app

# 빌드 스테이지의 /install만 복사 (pip, build 도구 제외)
COPY --from=builder /install /usr/local
COPY --from=builder /build/app ./app
COPY --from=builder /build/run.py .

RUN mkdir -p /data

ENV DATABASE_URL=sqlite+aiosqlite:////data/trading.db
ENV HOST=0.0.0.0
ENV PORT=8000

EXPOSE 8000

# Docker 레벨 헬스체크
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/api/health')" || exit 1

CMD ["python", "run.py"]
```

### 3-6. pip install .[dev] 패턴

```toml
# pyproject.toml
[project]
name = "micro-trading"
dependencies = [
    "fastapi",
    "sqlalchemy",
    ...
]

[project.optional-dependencies]
dev = [
    "pytest",
    "pytest-asyncio",
    "aiosqlite",
]
```

- `pip install .`: 프로덕션 의존성만 설치
- `pip install .[dev]`: 프로덕션 + dev 의존성 설치 (CI 테스트용)
- Docker 이미지에는 dev 의존성이 포함되지 않음 → 이미지 크기 절감

---

## 4. Java vs Python: CI/CD 비교

### 4-1. 빌드 도구 비교

**Java (Maven)**
```yaml
# GitHub Actions
- name: Build with Maven
  run: mvn -B package --file pom.xml

- name: Run tests
  run: mvn test
```

**Python (pip + pytest)**
```yaml
- name: Install dependencies
  run: pip install .[dev]

- name: Run tests
  run: pytest -v
```

| 항목 | Java (Maven/Gradle) | Python (pip+pytest) |
|:---|:---|:---|
| 빌드 결과물 | JAR/WAR 파일 | 없음 (소스 그대로 실행) |
| 의존성 정의 | `pom.xml`, `build.gradle` | `pyproject.toml`, `requirements.txt` |
| 테스트 실행 | `mvn test`, `gradle test` | `pytest` |
| 빌드 시간 | 컴파일 포함 (느림) | 의존성 설치만 (빠름) |
| 캐시 | Maven 로컬 저장소 | pip 캐시 |

### 4-2. Docker 이미지 비교

**Java (Spring Boot)**
```dockerfile
FROM eclipse-temurin:17-jre AS runtime
COPY target/app.jar /app.jar
ENTRYPOINT ["java", "-jar", "/app.jar"]
```

**Python (FastAPI)**
```dockerfile
FROM python:3.11-slim AS builder
RUN pip install --prefix=/install .

FROM python:3.11-slim
COPY --from=builder /install /usr/local
COPY app/ ./app/
CMD ["python", "run.py"]
```

| 항목 | Java (Spring Boot) | Python (FastAPI) |
|:---|:---|:---|
| 베이스 이미지 | JRE (~200MB) | python:3.11-slim (~50MB) |
| 빌드 결과물 | JAR (수십 MB) | 소스 파일 (수 MB) |
| 실행 방식 | `java -jar app.jar` | `python run.py` |
| 멀티스테이지 필요성 | Maven 도구 제거용 | pip 빌드 도구 제거용 |
| 콜드 스타트 | JVM 기동 (느림) | 빠름 |

### 4-3. 배포 방식 비교

**Java (Spring Boot - 일반적)**
```yaml
# GitHub Actions
- name: Deploy to server
  run: |
    scp target/app.jar user@server:/app/
    ssh user@server "systemctl restart myapp"
```

**Python (Docker Compose)**
```yaml
# GitHub Actions (appleboy/ssh-action)
script: |
  cd /service/micro-trading
  git pull origin main
  docker compose down
  docker compose up -d --build
```

| 항목 | Java (JAR 배포) | Python (Docker) |
|:---|:---|:---|
| 배포 단위 | JAR 파일 전송 | 소스 pull + Docker build |
| 환경 재현성 | JVM 버전 의존 | Docker로 완전 격리 |
| 롤백 | 이전 JAR 복구 | 이전 이미지 태그 사용 |
| 배포 시간 | SCP 전송 시간 | Docker build 시간 |

---

## 5. Key Concept: 초보자가 놓치기 쉬운 포인트

### 5-1. CI와 CD의 차이

| 용어 | 의미 | 이 프로젝트에서 |
|:---|:---|:---|
| CI (Continuous Integration) | 코드 통합 시 자동 테스트 | `test` job: pytest 실행 |
| CD (Continuous Delivery) | 테스트 통과 후 자동 배포 | `deploy` job: NCP 서버 배포 |
| CD (Continuous Deployment) | 자동 배포 + 운영 반영 | docker compose up --build |

### 5-2. Docker HEALTHCHECK vs CI 헬스체크

두 개의 헬스체크가 존재한다:

**Docker HEALTHCHECK** (컨테이너 내부):
```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/api/health')" || exit 1
```
- Docker 데몬이 컨테이너 상태를 지속적으로 모니터링
- `docker ps`에서 `(healthy)` / `(unhealthy)` 상태 표시
- Docker Swarm/Compose에서 자동 재시작 트리거 가능

**CI 헬스체크** (배포 직후 일회성):
```bash
for i in $(seq 1 6); do
  sleep 5
  curl -sf http://localhost:8000/api/health && exit 0
done
exit 1
```
- 배포가 성공했는지 확인하는 일회성 검증
- 실패 시 GitHub Actions 파이프라인 실패 처리

### 5-3. `--no-cache-dir` 옵션

```dockerfile
RUN pip install --no-cache-dir --prefix=/install .
```

- pip는 기본적으로 패키지를 `~/.cache/pip`에 캐시
- Docker 빌드 시 캐시는 이미지 레이어에 포함됨 → 이미지 크기 증가
- `--no-cache-dir`: 캐시 저장 안 함 → 이미지 크기 절감
- CI 환경에서는 `actions/setup-python@v5`의 `cache: "pip"`가 대신 처리

### 5-4. 멀티스테이지 빌드의 핵심 이점

```
빌드 스테이지 포함 시:
  python:3.11-slim + pip + build tools + .whl 캐시 + 소스 = 크고 무거움

런타임 스테이지만:
  python:3.11-slim + 실행에 필요한 패키지 + 소스 = 작고 가벼움
```

1GB RAM NCP 서버에서:
- 이미지 다운로드 시간 단축
- 컨테이너 실행 시 메모리 사용량 감소
- 빌드 도구(`gcc`, `make` 등)가 없어 공격 표면 축소

### 5-5. `docker compose down` + `up --build` vs `restart`

```bash
docker compose down        # 컨테이너 완전 중지 및 제거
docker compose up -d --build  # 이미지 재빌드 + 새 컨테이너 시작
```

**왜 `restart`가 아닌가?**
- `docker compose restart`: 이미지를 재빌드하지 않고 기존 컨테이너만 재시작
- 코드를 변경했어도 반영되지 않음
- `--build`가 있어야 새 소스코드로 이미지를 다시 빌드함

**왜 `down` 후 `up`인가?**
- `down` 없이 `up --build`를 실행하면 이름 충돌이 발생할 수 있음
- 포트 바인딩이 깨끗하게 정리됨

### 5-6. pyproject.toml의 optional-dependencies 패턴

```toml
[project.optional-dependencies]
dev = [
    "pytest>=8",
    "pytest-asyncio",
    "aiosqlite",   # 인메모리 SQLite 비동기 드라이버
]
```

- **프로덕션**: `pip install .` → dev 패키지 미설치
- **개발/CI**: `pip install .[dev]` → dev 패키지 포함
- Docker 이미지에는 `pip install .`만 실행 → `pytest` 등이 포함되지 않아 이미지 경량화

### 5-7. GitHub Secrets의 중요성

```yaml
host: ${{ secrets.NCP_HOST }}
key: ${{ secrets.NCP_SSH_KEY }}
```

**절대 코드에 직접 넣지 말아야 하는 것들:**
- 서버 IP 주소
- SSH 비밀키 (PEM 파일 내용)
- 패스프레이즈
- API 키, DB 비밀번호

GitHub Secrets는 암호화되어 저장되고, 워크플로우 실행 시에만 복호화된다. 로그에도 `***`로 마스킹된다.
