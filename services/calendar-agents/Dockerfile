# ---------- builder ----------
FROM python:3.12-slim AS builder

WORKDIR /build

COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

# ---------- runtime ----------
FROM python:3.12-slim AS runtime

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

RUN addgroup --system agents && \
    adduser --system --ingroup agents agents

COPY --from=builder /install /usr/local
COPY . .

RUN chown -R agents:agents /app
USER agents

EXPOSE 3337

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["python", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:3337/health')"]

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "3337"]
