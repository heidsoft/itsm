"""
Guidance Sidecar 兼容端点

后端 GuidanceClient 期望独立 sidecar 提供受限 JSON 生成：
  POST /triage  ->  {category, priority, confidence, explanation, suggestedFix, assigneeId, method, latencyMs}
  GET  /health  ->  已在 main.py 提供

本模块让 itsm-ai-service 直接扮演该 sidecar：调用配置的 LLM provider
（OpenAI 兼容接口）生成受约束的分类 JSON，避免回代理后端造成鉴权环。
"""
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field
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
    # A4：tenantId 旧实现默认 1 且全文零消费，任何调用方都能自称是租户 1。
    # 服务间令牌启用后调用方已是受信任后端，这里至少拒绝 0/负数这类明显
    # 伪造值，并把 tenantId 写进结构化日志供审计对齐。
    tenantId: int = Field(1, ge=1)


class GuidanceTriageResponse(BaseModel):
    category: str
    priority: str
    confidence: float
    explanation: str
    suggestedFix: Optional[str] = None
    assigneeId: int = 0
    method: str = "guidance"
    latencyMs: float = 0.0
    # 显式降级标记：LLM 不可用时返回的关键词结果不是模型结论。
    # 历史实现只靠 method 字符串区分，调用方（Go 后端）根本没读 method，
    # 于是降级结果被当成模型决策写进审计底账。这里加显式布尔位，
    # 让「这是降级」这件事不依赖字符串约定。
    degraded: bool = False
    degradedReason: str = ""


# A6：LLM 响应是外部输入，格式偏差不能靠 `data["choices"][0]` 硬下标
# 直接抛 KeyError 再被宽泛 except 吞成关键词降级（E-P0-3 的放大器）。
# 它必须是显式、可重试、可记录的类型。
class LLMResponseError(ValueError):
    """LLM 响应体不符合契约（缺 choices / 空 content / 截断 / 非 JSON）。"""


class LLMNotConfiguredError(RuntimeError):
    """凭据/端点缺失：这是配置错误，不是一次可重试的调用失败。

    A5：旧实现把它当普通 RuntimeError 抛进宽泛 except，伪装成"LLM 暂时
    不可用"的降级；容器全绿、/health 恒 healthy，配置错误永远发现不了。
    """


_http_client: Optional[httpx.AsyncClient] = None


def get_http_client() -> httpx.AsyncClient:
    """模块级复用 AsyncClient：连接池 + 不阻塞事件循环。

    A3：旧实现在 `async def` 端点里用同步 `with httpx.Client(...)`，
    timeout=120 会把整个 uvicorn 事件循环卡住最多两分钟，/health 同样
    挂起 → compose healthcheck 10s 超时 → restart: always 反复重启；
    且每请求新建 Client，没有连接池。
    """
    global _http_client
    if _http_client is None:
        cfg = get_config()
        _http_client = httpx.AsyncClient(
            timeout=httpx.Timeout(cfg.llm.timeout or 120, connect=10.0),
            limits=httpx.Limits(max_connections=20, max_keepalive_connections=10),
        )
    return _http_client


async def close_http_client() -> None:
    global _http_client
    if _http_client is not None:
        await _http_client.aclose()
        _http_client = None


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


# 一次调用最多尝试 2 次（1 次重试）。只对可恢复错误重试；凭据缺失
# （LLMNotConfiguredError）是配置错误，重试毫无意义且会把故障拖成两倍慢。
_MAX_LLM_ATTEMPTS = 2
_RETRYABLE = (
    httpx.TimeoutException,
    httpx.TransportError,
    httpx.HTTPStatusError,
    LLMResponseError,
)


def _strip_json_fence(text: str) -> str:
    """剥掉 ```json ... ``` 围栏；模型经常无视「不要 markdown」的指令。"""
    if not isinstance(text, str):
        raise LLMResponseError("LLM content is not a string")
    stripped = text.strip()
    if stripped.startswith("```"):
        first_newline = stripped.find("\n")
        if first_newline < 0:
            raise LLMResponseError("LLM returned an unterminated code fence")
        stripped = stripped[first_newline + 1:].strip()
        if stripped.endswith("```"):
            stripped = stripped[:-3].strip()
    return stripped


def _parse_triage_json(text: str) -> dict:
    """把 LLM 自由文本解析成 JSON 对象；容忍围栏与前后点缀文字。"""
    stripped = _strip_json_fence(text)
    if not stripped:
        raise LLMResponseError("LLM returned empty content")
    try:
        parsed = json.loads(stripped)
    except json.JSONDecodeError:
        # 只在整体不是 JSON 时尝试截取首个 {...} 块，避免把合法 JSON
        # 里的字符串内容误当新对象。
        start, end = stripped.find("{"), stripped.rfind("}")
        if start < 0 or end <= start:
            raise LLMResponseError("LLM response is not a JSON object") from None
        try:
            parsed = json.loads(stripped[start:end + 1])
        except json.JSONDecodeError:
            raise LLMResponseError("LLM response is not a JSON object") from None
    if not isinstance(parsed, dict):
        raise LLMResponseError("LLM did not return a JSON object")
    return parsed


def _chat_content(data: dict) -> str:
    """校验 OpenAI 兼容响应信封，不依赖硬下标。"""
    if not isinstance(data, dict):
        raise LLMResponseError("LLM response envelope is not an object")
    choices = data.get("choices")
    if not isinstance(choices, list) or not choices:
        raise LLMResponseError("LLM response has no choices")
    first = choices[0]
    if not isinstance(first, dict):
        raise LLMResponseError("LLM choice is not an object")
    # finish_reason=length 说明 JSON 被 max_tokens 截断，解析出来的半截对象
    # 绝不能当成模型结论（旧实现会直接 json.loads 抛错 → 静默降级）。
    if first.get("finish_reason") == "length":
        raise LLMResponseError("LLM output truncated (finish_reason=length)")
    message = first.get("message")
    content = message.get("content") if isinstance(message, dict) else None
    if not isinstance(content, str) or not content.strip():
        raise LLMResponseError("LLM choice has no text content")
    return content


async def _with_one_retry(fetch):
    """单次重试：网络抖动 / 5xx / 格式偏差各给一次机会。"""
    last_error = None
    for attempt in range(_MAX_LLM_ATTEMPTS):
        try:
            return await fetch()
        except _RETRYABLE as exc:
            last_error = exc
            if attempt + 1 < _MAX_LLM_ATTEMPTS:
                logger.warning(
                    "LLM call failed (attempt %d/%d), retrying once: %s",
                    attempt + 1, _MAX_LLM_ATTEMPTS, type(exc).__name__,
                )
    raise last_error


def _keyword_classify(title: str, description: str) -> GuidanceTriageResponse:
    """基于关键词的降级分类，与后端 triage_service.go validCategories 对齐。"""
    text = f"{title} {description}".lower()

    category = "general"
    if any(k in text for k in ["mysql", "postgres", "mongodb", "redis", "oracle", "数据库", "查询", "慢查询"]):
        category = "database"
    elif any(k in text for k in ["wifi", "router", "switch", "firewall", "vpn", "网络", "连接", "网卡", "dns", "ping"]):
        category = "network"
    elif any(k in text for k in ["cpu", "memory", "disk", "linux", "windows server", "服务器", "宕机", "重启", "硬件"]):
        category = "server"
    elif any(k in text for k in ["software", "app", "api", "deployment", "软件", "应用", "程序", "部署"]):
        category = "application"
    elif any(k in text for k in ["vulnerability", "attack", "permission", "auth", "安全", "漏洞", "攻击", "权限"]):
        category = "security"
    elif any(k in text for k in ["backup", "snapshot", "storage", "磁盘", "存储", "空间"]):
        category = "storage"
    elif any(k in text for k in ["account", "login", "password", "access", "账号", "密码", "登录"]):
        category = "user_access"

    priority = "medium"
    if any(k in text for k in ["critical", "urgent", "emergency", "紧急", "严重", "宕机", "无法工作", "服务不可用"]):
        priority = "critical"
    elif any(k in text for k in ["high", "important", "重要", "影响工作", "故障"]):
        priority = "high"
    elif any(k in text for k in ["low", "minor", "不紧急", "轻微"]):
        priority = "low"

    return GuidanceTriageResponse(
        category=category,
        priority=priority,
        # 关键词命中给不出校准过的置信度，0.7 只是手感值：写低会被当成
        # 「模型很没把握」，写高会骗过自动采纳阈值。降到 0.5（中性），
        # 同时 degraded=True 让调用方知道这不是模型打分。
        confidence=0.5,
        explanation="keyword-based classification (LLM unavailable)",
        method="guidance-fallback",
        degraded=True,
        degradedReason="LLM call unavailable; keyword fallback used",
        latencyMs=0.0,
    )


async def _call_llm(title: str, description: str) -> dict:
    """Call the configured provider and return a constrained triage object."""
    cfg = get_config()
    api_key = os.getenv("LLM_API_KEY") or cfg.llm.api_key
    if not api_key:
        raise LLMNotConfiguredError("LLM_API_KEY not configured")

    if cfg.llm.provider == "minimax":
        return await _call_minimax(title, description, api_key)

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
        # A6：config.yaml 早就定义了 max_tokens，但 OpenAI 路径从未下发。
        # 上限夹紧到 1024：分诊输出只要几个字段，不设上限会把成本与
        # finish_reason=length 的截断风险一起放大。
        "max_tokens": min(int(cfg.llm.max_tokens or 512), 1024),
        "response_format": {"type": "json_object"},
    }
    headers = {"Content-Type": "application/json"}
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"

    async def fetch() -> dict:
        resp = await get_http_client().post(url, json=payload, headers=headers)
        resp.raise_for_status()
        return _parse_triage_json(_chat_content(resp.json()))

    return await _with_one_retry(fetch)


async def _call_minimax(title: str, description: str, api_key: str) -> dict:
    """Call MiniMax's Anthropic-compatible Messages API.

    The Go gateway uses this same protocol. Keeping it explicit prevents the
    guidance sidecar from silently falling back to OpenAI when MiniMax is the
    deployment provider.
    """
    cfg = get_config()
    base_url = (cfg.llm.base_url or "https://api.minimaxi.com/anthropic/v1").rstrip("/")
    payload = {
        "model": cfg.llm.model,
        "max_tokens": min(int(cfg.llm.max_tokens or 512), 1024),
        "system": _SYSTEM_PROMPT,
        "temperature": 0.2,
        "messages": [{"role": "user", "content": _build_user_prompt(title, description)}],
    }
    headers = {
        "Content-Type": "application/json",
        "x-api-key": api_key,
        "anthropic-version": "2023-06-01",
    }

    async def fetch() -> dict:
        resp = await get_http_client().post(f"{base_url}/messages", json=payload, headers=headers)
        resp.raise_for_status()
        data = resp.json()
        blocks = data.get("content", []) if isinstance(data, dict) else []
        text = next(
            (block.get("text", "") for block in blocks
             if isinstance(block, dict) and block.get("type") == "text"),
            "",
        )
        if not text:
            raise LLMResponseError("MiniMax returned no text block")
        return _parse_triage_json(text)

    return await _with_one_retry(fetch)


@router.post("/triage", response_model=GuidanceTriageResponse)
async def guidance_triage(req: GuidanceTriageRequest):
    """Guidance sidecar 兼容的受限分诊入口。"""
    start = time.perf_counter()
    logger.info("guidance triage request", extra={"tenant_id": req.tenantId})
    try:
        raw = await _call_llm(req.title, req.description)
    except LLMNotConfiguredError as e:
        # 配置错误不能被包装成「运行时降级」就悄悄过去：打 ERROR 并让
        # /health 的 llm.configured=false 暴露它。业务请求仍返回可用的降级
        # 结果（不能因为没配 LLM 让工单分诊整条链路 500）。
        logger.error(f"Guidance LLM is not configured: {e}")
        fallback = _keyword_classify(req.title, req.description)
        fallback.degradedReason = f"configuration error: {e}"
        fallback.latencyMs = round((time.perf_counter() - start) * 1000, 2)
        return fallback
    except Exception as e:  # noqa: BLE001 - 调用失败才降级为关键词分类
        logger.warning(f"Guidance triage LLM unavailable, using keyword fallback: {e}")
        latency = (time.perf_counter() - start) * 1000
        fallback = _keyword_classify(req.title, req.description)
        fallback.latencyMs = round(latency, 2)
        return fallback

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
