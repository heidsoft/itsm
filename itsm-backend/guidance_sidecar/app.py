"""
Guidance Sidecar Service for ITSM AI-Native Features

HTTP API that Go backend calls for Guidance-constrained LLM generation.
"""

import os
import time
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field
import uvicorn

from guidance_triage import classify

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger("guidance-sidecar")


class TriageRequest(BaseModel):
    title: str = Field(..., min_length=1, max_length=500)
    description: str = Field(..., min_length=0, max_length=5000)
    tenant_id: int = Field(default=1)


class TriageResponse(BaseModel):
    category: str
    priority: str
    confidence: float
    explanation: str
    suggested_fix: str | None = None
    assignee_id: int
    method: str = "guidance"
    latency_ms: float


class HealthResponse(BaseModel):
    status: str
    model: str
    provider: str


# Global model instance
_model = None


def get_model():
    """Initialize the LLM model."""
    global _model
    if _model is not None:
        return _model

    provider = os.getenv("GUIDANCE_PROVIDER", "minimax")
    model_name = os.getenv("GUIDANCE_MODEL", "gpt-4o-mini")

    logger.info(f"Initializing Guidance model: provider={provider}, model={model_name}")

    if provider == "minimax":
        # MiniMax OpenAI-compatible API
        from guidance.models import OpenAI
        _model = OpenAI(
            model=model_name,
            api_key=os.getenv("MINIMAX_API_KEY", ""),
            base_url="https://api.minimax.io/v1",
        )
        logger.info("Using MiniMax via OpenAI-compatible API")
    else:
        # Standard OpenAI
        from guidance.models import OpenAI
        _model = OpenAI(model_name)
        logger.info("Using standard OpenAI")

    return _model


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Startup and shutdown."""
    logger.info("Starting Guidance Sidecar...")
    try:
        model = get_model()
        logger.info(f"Guidance Sidecar ready: {os.getenv('GUIDANCE_PROVIDER', 'minimax')}/{os.getenv('GUIDANCE_MODEL', 'gpt-4o-mini')}")
    except Exception as e:
        logger.error(f"Failed to initialize model: {e}")
        raise
    yield
    logger.info("Shutting down Guidance Sidecar...")


app = FastAPI(
    title="Guidance Sidecar for ITSM",
    description="AI-Native ticket triage with Guidance-constrained generation",
    version="1.0.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/health", response_model=HealthResponse)
async def health():
    return HealthResponse(
        status="healthy",
        model=os.getenv("GUIDANCE_MODEL", "gpt-4o-mini"),
        provider=os.getenv("GUIDANCE_PROVIDER", "minimax"),
    )


@app.post("/triage", response_model=TriageResponse)
async def triage(request: TriageRequest):
    """Classify a ticket using Guidance-constrained generation."""
    start_time = time.time()

    try:
        model = get_model()
        result = classify(
            title=request.title,
            description=request.description,
            model=model,
        )

        latency_ms = (time.time() - start_time) * 1000

        logger.info(
            f"Triage: category={result['category']}, priority={result['priority']}, "
            f"confidence={result['confidence']}, latency={latency_ms:.1f}ms"
        )

        return TriageResponse(
            category=result["category"],
            priority=result["priority"],
            confidence=result["confidence"],
            explanation=result["explanation"],
            suggested_fix=result.get("suggested_fix"),
            assignee_id=result["assignee_id"],
            method="guidance",
            latency_ms=latency_ms,
        )

    except Exception as e:
        logger.error(f"Triage failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


if __name__ == "__main__":
    port = int(os.getenv("GUIDANCE_PORT", "8091"))
    uvicorn.run(app, host="0.0.0.0", port=port, log_level="info")
