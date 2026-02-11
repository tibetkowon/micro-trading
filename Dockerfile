FROM python:3.11-slim

WORKDIR /app

# Copy application
COPY . .

# Install dependencies
RUN pip install --no-cache-dir .

# Create data directory
RUN mkdir -p /data

ENV DATABASE_URL=sqlite+aiosqlite:////data/trading.db
ENV HOST=0.0.0.0
ENV PORT=8000

EXPOSE 8000

CMD ["python", "run.py"]
