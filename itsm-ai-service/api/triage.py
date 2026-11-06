"""
智能分类 API
根据工单/事件描述自动分类、确定优先级和分配建议
"""
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field
from typing import Optional, List
import logging

from services.llm_client import get_llm_client

logger = logging.getLogger(__name__)

router = APIRouter()


class TriageRequest(BaseModel):
    """智能分类请求"""
    title: str = Field(..., description="工单标题")
    description: str = Field(..., description="工单描述")
    category: Optional[str] = Field(None, description="已有分类")
    source: Optional[str] = Field("manual", description="来源")
    tenant_id: Optional[int] = Field(None, description="租户ID")


class CategorySuggestion(BaseModel):
    """分类建议"""
    category: str
    subcategory: str
    confidence: float


class PrioritySuggestion(BaseModel):
    """优先级建议"""
    priority: str
    confidence: float


class AssignmentSuggestion(BaseModel):
    """分配建议"""
    assignee_id: Optional[int] = None
    assignee_group: Optional[str] = None
    reason: str


class TriageResponse(BaseModel):
    """智能分类响应（字段来自 Go 后端 POST /api/v1/ai/triage 的 data）。"""
    categories: List[CategorySuggestion]
    priority: PrioritySuggestion
    assignment: AssignmentSuggestion
    keywords: List[str]
    related_articles: Optional[List[dict]] = None
    # 溯源与降级标记：调用方必须能区分「模型结论」与「关键词凑数」。
    # Go 侧在 Guidance/LLM 不可用时返回 degraded=true + degradedReason，
    # 本服务在后端不可达时走 _rule_based_triage，也必须如实标注。
    degraded: bool = False
    degradedReason: str = ""
    model: str = ""
    promptVersion: str = ""
    auditId: Optional[int] = None


# 批量端点上限：无边界的 List 会让单请求串行 await 几十万次后端调用，
# 把 sidecar 与 Go 后端一起拖死（A6 的体量约束）。
MAX_BATCH_SIZE = 50


@router.post("/", response_model=TriageResponse)
async def classify_ticket(request: TriageRequest):
    """
    智能分类工单/事件
    - 自动分类 (category/subcategory)
    - 确定优先级
    - 建议分配
    """
    llm = get_llm_client()

    try:
        # 调用后端 AI 分诊（返回的是 Go 的 data 对象，不是模型厂商响应）
        triage_result = await llm.triage_ticket(
            title=request.title,
            description=request.description,
            category=request.category or "",
            priority=""
        )

        if triage_result:
            logger.info(f"Triage completed for: {request.title}")
            return _build_triage_response(request, triage_result)

        # 后端无产出（不可达 / 业务降级）：走本地规则，并显式标降级。
        return _rule_based_triage(request, "backend triage returned no data")

    except Exception as e:
        # 不把内部异常原文回显给调用方（可能含后端地址、凭据片段）。
        logger.error(f"Triage error: {type(e).__name__}")
        raise HTTPException(status_code=502, detail="工单分诊服务暂时不可用")


@router.post("/batch", response_model=List[TriageResponse])
async def classify_batch_tickets(requests: List[TriageRequest]):
    """批量智能分类。单项失败时返回标记了 degraded 的降级项，不让整批 500。"""
    if len(requests) > MAX_BATCH_SIZE:
        raise HTTPException(
            status_code=413,
            detail=f"批量分诊最多 {MAX_BATCH_SIZE} 条，实际 {len(requests)} 条",
        )
    results = []
    for req in requests:
        try:
            results.append(await classify_ticket(req))
        except Exception as e:
            # 旧实现在这里 append(None)，与 response_model=List[TriageResponse]
            # 直接冲突：一条失败会让整个响应 500。
            logger.error(f"Batch triage error: {type(e).__name__}")
            results.append(_rule_based_triage(req, "backend triage call failed"))
    return results


def _rule_based_triage(request: TriageRequest, reason: str = "backend unavailable") -> TriageResponse:
    """基于规则的分类"""
    # 简单的关键词匹配
    keywords = []
    text = f"{request.title} {request.description}".lower()

    # 关键词映射
    keyword_map = {
        "网络": ["网络", "连接", "网卡", "eth0", "ping", "dns"],
        "服务器": ["服务器", "宕机", "重启", "宕机", "crash"],
        "存储": ["存储", "磁盘", "空间", "满了", "i/o"],
        "数据库": ["数据库", "mysql", "postgres", "查询", "慢"],
        "安全": ["攻击", "入侵", "异常", "漏洞", "病毒"],
    }

    for kw, patterns in keyword_map.items():
        for p in patterns:
            if p in text:
                keywords.append(kw)
                break

    # 根据关键词确定分类
    if "网络" in keywords:
        category = "infrastructure"
        subcategory = "network"
        group = "network_team"
    elif "服务器" in keywords:
        category = "infrastructure"
        subcategory = "server"
        group = "infra_team"
    elif "数据库" in keywords:
        category = "application"
        subcategory = "database"
        group = "dba_team"
    elif "安全" in keywords:
        category = "security"
        subcategory = "incident"
        group = "security_team"
    else:
        category = "general"
        subcategory = "other"
        group = "support_team"

    # 优先级判断
    if any(p in text for p in ["紧急", "critical", "宕机", "服务不可用"]):
        priority = "high"
    elif any(p in text for p in ["严重", "major", "故障"]):
        priority = "medium"
    else:
        priority = "low"

    return TriageResponse(
        categories=[
            CategorySuggestion(category=category, subcategory=subcategory, confidence=0.85)
        ],
        priority=PrioritySuggestion(priority=priority, confidence=0.78),
        assignment=AssignmentSuggestion(
            assignee_group=group,
            reason=f"{subcategory} 相关问题，分配至 {group}"
        ),
        keywords=keywords,
        # 不伪造「相关文档」：Go 不返回这个字段，凭空造一条 id=1 的排查指南
        # 会让调用方以为后端真的检索过知识库。
        related_articles=[],
        degraded=True,
        degradedReason=reason,
        model="keyword-fallback",
    )


def _clamp_confidence(value) -> float:
    try:
        return max(0.0, min(1.0, float(value)))
    except (TypeError, ValueError):
        return 0.0


def map_backend_triage(triage: dict) -> dict:
    """把 Go 后端的 data 对象映射成本端 TriageResponse 的字段。

    Go 的真实契约（handlers/ai/service.go TriageTicket）：
      {title, description, promptVersion, model,
       suggestions: {category, priority, confidence, reasoning, urgency},
       degraded?, degradedReason?, auditId?}

    旧实现读的是 categories[] / priority_suggestion / assignment_suggestion /
    keywords / related_articles —— 这些字段后端一个都不返回，于是分类恒
    general、优先级恒 low、置信度恒默认 0.8。这里按真实契约逐字段映射，
    后端没有的字段一律留空，不猜测、不伪造。
    """
    suggestions = triage.get("suggestions")
    if not isinstance(suggestions, dict):
        suggestions = {}
    confidence = _clamp_confidence(suggestions.get("confidence"))
    category = str(suggestions.get("category") or "unclassified")
    priority = str(suggestions.get("priority") or "unclassified")
    reasoning = str(suggestions.get("reasoning") or "")

    return {
        "categories": [{
            "category": category,
            "subcategory": "",  # Go 无二级分类概念，留空而不是编 "other"
            "confidence": confidence,
        }],
        "priority": {"priority": priority, "confidence": confidence},
        "assignment": {
            "assignee_id": None,       # Go TriageResult.AssigneeID 未进响应
            "assignee_group": None,
            "reason": reasoning or "后端未给出分配建议",
        },
        "keywords": [],               # Go 不返回关键词
        "related_articles": [],        # Go 不返回关联文档
        "degraded": bool(triage.get("degraded", False)),
        "degradedReason": str(triage.get("degradedReason") or ""),
        "model": str(triage.get("model") or ""),
        "promptVersion": str(triage.get("promptVersion") or ""),
        "auditId": triage.get("auditId"),
    }


def _build_triage_response(request: TriageRequest, triage: dict) -> TriageResponse:
    """从后端响应构建分类结果（薄包装，映射逻辑见 map_backend_triage）。"""
    return TriageResponse(**map_backend_triage(triage))