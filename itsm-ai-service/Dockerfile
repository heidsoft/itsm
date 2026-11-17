# syntax=docker/dockerfile:1.7
#
# ITSM AI Service — multi-stage Python build
#
# The previous single-stage image pulled in torch + transformers as CUDA
# wheels, bloating the image past ~3GB. This version:
#   - Installs into a venv in a builder stage
#   - Uses CPU-only torch wheels via a dedicated --index-url (NOT
#     --extra-index-url, which silently falls back to PyPI's CUDA wheel)
#   - BuildKit pip cache mounts so repeated builds stay fast
#   - Copies only the venv + app source into a slim runtime
#   - Runs as a non-root user
#   - Healthcheck uses stdlib urllib (no extra dependency at probe time)

# ============================================================
# Stage 1: Build venv
# ============================================================
FROM python:3.12-slim-bookworm AS builder

WORKDIR /app

# Keep pip's cache enabled: the BuildKit cache mount below persists it across
# builds. PIP_NO_CACHE_DIR=1 would make those mounts dead weight.
ENV PIP_DISABLE_PIP_VERSION_CHECK=1

# CPU-only torch index. For GPU builds pass:
#   --build-arg TORCH_INDEX=https://download.pytorch.org/whl/cu121
ARG TORCH_INDEX=https://download.pytorch.org/whl/cpu

COPY requirements.txt ./

# Install torch FIRST from the dedicated index only.
#   --index-url (NOT --extra-index-url) is deliberate. With an extra index,
#   pip still resolves `torch>=2.1.0` (requirements.txt) against PyPI, whose
#   Linux wheel pulls multi-GB nvidia-* CUDA packages; and if the PyTorch
#   index is unreachable, pip silently falls back to that CUDA wheel while
#   the build stays green. Splitting the install keeps step 2 on PyPI for
#   every other dependency, while the already-installed torch satisfies the
#   `torch>=2.1.0` constraint so it is never reinstalled from PyPI.
RUN --mount=type=cache,target=/root/.cache/pip \
    python -m venv /opt/venv \
    && . /opt/venv/bin/activate \
    && pip install --upgrade pip \
    && pip install --index-url "${TORCH_INDEX}" torch

RUN --mount=type=cache,target=/root/.cache/pip \
    . /opt/venv/bin/activate \
    && pip install -r requirements.txt

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
