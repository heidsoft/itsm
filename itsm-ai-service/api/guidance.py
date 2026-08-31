"""
Guidance Sidecar 兼容端点

后端 GuidanceClient 期望独立 sidecar 提供受限 JSON 生成：
  POST /triage  ->  {category, priority, confidence, explanation, suggestedFix, assigneeId, method, latencyMs}
  GET  /health  ->  已在 main.py 提供

本模块让 itsm-ai-service 直接扮演该 sidecar：调用配置的 LLM provider
（OpenAI 兼容接口）生成受约束的分类 JSON，避免回代理后端造成鉴权环。
"""
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from typing import Optional
import json
import logging
import os
import time

import httpx

from config_loader import get_config

logger = logging.getLogger(__name__)

router = APIRouter()

# 与后端 triage_service.go 的 validCategories / validPriorities 保持一致
_VALID_CATEGORIES = [
    "database", "network", "server", "application",
    "security", "storage", "user_access", "general",
]
_VALID_PRIORITIES = ["critical", "high", "medium", "low"]


class GuidanceTriageRequest(BaseModel):
    title: str
    description: str = ""
    tenantId: int = 1


class GuidanceTriageResponse(BaseModel):
    category: str
    priority: str
    confidence: float
    explanation: str
    suggestedFix: Optional[str] = None
    assigneeId: int = 0
    method: str = "guidance"
    latencyMs: float = 0.0


_SYSTEM_PROMPT = (
    "You are an ITSM ticket triage classifier. "
    "Respond ONLY with a single JSON object, no markdown, no commentary. "
    "Required fields: "
    "category (one of: database, network, server, application, security, storage, user_access, general), "
    "priority (one of: critical, high, medium, low), "
    "confidence (float 0.0-1.0), "
    "explanation (short string), "
    "suggestedFix (short string or null)."
)


def _build_user_prompt(title: str, description: str) -> str:
    desc = description or "(no description)"
    return f"Classify this ITSM ticket.\nTitle: {title}\nDescription: {desc}"


def _coerce_category(value: str) -> str:
    if value in _VALID_CATEGORIES:
        return value
    lowered = (value or "").strip().lower()
    for c in _VALID_CATEGORIES:
        if c in lowered:
            return c
    return "general"


def _coerce_priority(value: str) -> str:
    if value in _VALID_PRIORITIES:
        return value
    lowered = (value or "").strip().lower()
    if "crit" in lowered:
        return "critical"
    if "high" in lowered or "urgent" in lowered:
        return "high"
    if "low" in lowered:
        return "low"
    return "medium"


def _call_llm(title: str, description: str) -> dict:
    """直连 OpenAI 兼容 chat/completions，返回解析后的分类字典。"""
    cfg = get_config()
    api_key = os.getenv("LLM_API_KEY") or cfg.llm.api_key
    base_url = (cfg.llm.base_url or "https://api.openai.com/v1").rstrip("/")
    model = cfg.llm.model or "gpt-4o-mini"
    url = f"{base_url}/chat/completions"

    payload = {
        "model": model,
        "messages": [
            {"role": "system", "content": _SYSTEM_PROMPT},
            {"role": "user", "content": _build_user_prompt(title, description)},
        ],
        "temperature": 0.2,
        "response_format": {"type": "json_object"},
    }
    headers = {"Content-Type": "application/json"}
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"

    with httpx.Client(timeout=httpx.Timeout(cfg.llm.timeout or 120)) as client:
        resp = client.post(url, json=payload, headers=headers)
        resp.raise_for_status()
        data = resp.json()

    content = data["choices"][0]["message"]["content"]
    parsed = json.loads(content)
    if not isinstance(parsed, dict):
        raise ValueError("LLM did not return a JSON object")
    return parsed


@router.post("/triage", response_model=GuidanceTriageResponse)
async def guidance_triage(req: GuidanceTriageRequest):
    """Guidance sidecar 兼容的受限分诊入口。"""
    start = time.perf_counter()
    try:
        raw = _call_llm(req.title, req.description)
    except Exception as e:  # noqa: BLE001 - 任何失败都降级为 general/medium
        logger.error(f"Guidance triage LLM failed: {e}")
        latency = (time.perf_counter() - start) * 1000
        return GuidanceTriageResponse(
            category="general",
            priority="medium",
            confidence=0.5,
            explanation="guidance LLM unavailable, keyword fallback",
            method="guidance-fallback",
            latencyMs=round(latency, 2),
        )

    latency = (time.perf_counter() - start) * 1000
    category = _coerce_category(str(raw.get("category", "general")))
    priority = _coerce_priority(str(raw.get("priority", "medium")))
    try:
        confidence = float(raw.get("confidence", 0.5))
    except (TypeError, ValueError):
        confidence = 0.5
    confidence = max(0.0, min(1.0, confidence))

    suggested = raw.get("suggestedFix")
    if suggested is not None:
        suggested = str(suggested)

    return GuidanceTriageResponse(
        category=category,
        priority=priority,
        confidence=round(confidence, 3),
        explanation=str(raw.get("explanation", ""))[:500],
        suggestedFix=suggested,
        assigneeId=0,
        method="guidance",
        latencyMs=round(latency, 2),
    )
