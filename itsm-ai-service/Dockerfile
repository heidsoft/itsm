# syntax=docker/dockerfile:1.7
#
# ITSM AI Service — multi-stage Python build
#
# The previous single-stage image pulled in torch + transformers as CUDA
# wheels, bloating the image past ~3GB. This version:
#   - Installs into a venv in a builder stage
#   - Uses CPU-only torch wheels (--extra-index-url .../whl/cpu) to avoid
#     the multi-GB CUDA packages (override TORCH_INDEX for GPU builds)
#   - Copies only the venv + app source into a slim runtime
#   - Runs as a non-root user
#   - Healthcheck uses stdlib urllib (no extra dependency at probe time)

# ============================================================
# Stage 1: Build venv
# ============================================================
FROM python:3.12-slim-bookworm AS builder

WORKDIR /app

ENV PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

# CPU-only torch index. For GPU builds pass:
#   --build-arg TORCH_INDEX=https://download.pytorch.org/whl/cu121
ARG TORCH_INDEX=https://download.pytorch.org/whl/cpu

COPY requirements.txt ./

RUN python -m venv /opt/venv \
    && . /opt/venv/bin/activate \
    && pip install --upgrade pip \
    && pip install -r requirements.txt --extra-index-url "${TORCH_INDEX}"

# ============================================================
# Stage 2: Runtime
# ============================================================
FROM python:3.12-slim-bookworm AS runtime

WORKDIR /app

ENV PYTHONUNBUFFERED=1 \
    PATH="/opt/venv/bin:$PATH" \
    SERVICE_NAME=itsm-ai-service

# libgomp1 is required by torch at runtime; keep the image lean.
RUN apt-get update \
    && apt-get install -y --no-install-recommends libgomp1 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /opt/venv /opt/venv

# Create a non-root user and copy app source owned by it.
RUN useradd -m -u 1000 app
COPY --chown=app:app . .
RUN chown -R app:app /app

USER app

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD python -c "import urllib.request,sys; sys.exit(0 if urllib.request.urlopen('http://localhost:8000/health').status==200 else 1)"

CMD ["python", "main.py"]
