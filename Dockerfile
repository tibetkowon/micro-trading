FROM python:3.11-slim AS builder

WORKDIR /build
COPY pyproject.toml .
COPY app/ app/
COPY run.py .

RUN pip install --no-cache-dir --prefix=/install .


FROM python:3.11-slim

WORKDIR /app

# 빌드 스테이지에서 설치된 패키지 복사
COPY --from=builder /install /usr/local
COPY --from=builder /build/app ./app
COPY --from=builder /build/run.py .

# 데이터 디렉토리
RUN mkdir -p /data

ENV DATABASE_URL=sqlite+aiosqlite:////data/trading.db
ENV HOST=0.0.0.0
ENV PORT=8000

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/api/health')" || exit 1

CMD ["python", "run.py"]
