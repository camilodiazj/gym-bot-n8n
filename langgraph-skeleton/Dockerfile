FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY pyproject.toml ./
RUN pip install --no-cache-dir .

# Copy application code
COPY server.py ./
COPY src/ ./src/
COPY cases/ ./cases/

# Cloud Run sets PORT env var (default 8080)
ENV PORT=8080

CMD ["sh", "-c", "uvicorn server:app --host 0.0.0.0 --port $PORT"]
