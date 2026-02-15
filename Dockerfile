FROM python:3.11-slim

WORKDIR /app

# 의존성 정의만 먼저 복사 → pip install 레이어 캐싱
COPY pyproject.toml .
RUN pip install --no-cache-dir . && rm -rf /tmp/*

# 소스 코드는 나중에 복사 (변경되어도 pip 캐시 유지)
COPY app/ app/
COPY run.py .

# 데이터 디렉토리
RUN mkdir -p /data

ENV DATABASE_URL=sqlite+aiosqlite:////data/trading.db
ENV HOST=0.0.0.0
ENV PORT=8000

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/api/health')" || exit 1

CMD ["python", "run.py"]
