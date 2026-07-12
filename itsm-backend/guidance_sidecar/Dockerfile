# syntax=docker/dockerfile:1.7
#
# ITSM Guidance Sidecar — multi-stage Python build
#
# Previous single-stage image copied the build context into a slim runtime
# and installed deps as root. This version:
#   - Builds a venv in a builder stage
#   - Ships only the venv + app source into a slim runtime
#   - Runs as a non-root user
#   - Adds a .dockerignore so local venvs/caches never enter the build context

# ============================================================
# Stage 1: Build venv
# ============================================================
FROM python:3.11-slim-bookworm AS builder

WORKDIR /app

ENV PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

COPY requirements.txt ./

RUN python -m venv /opt/venv \
    && . /opt/venv/bin/activate \
    && pip install --upgrade pip \
    && pip install -r requirements.txt

# ============================================================
# Stage 2: Runtime
# ============================================================
FROM python:3.11-slim-bookworm AS runtime

WORKDIR /app

ENV PYTHONUNBUFFERED=1 \
    PATH="/opt/venv/bin:$PATH"

COPY --from=builder /opt/venv /opt/venv

RUN useradd -m -u 1001 app
COPY --chown=app:app . .
RUN chown -R app:app /app

USER app

EXPOSE 8091

CMD ["python", "app.py"]
