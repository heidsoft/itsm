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
    """智能分类响应"""
    categories: List[CategorySuggestion]
    priority: PrioritySuggestion
    assignment: AssignmentSuggestion
    keywords: List[str]
    related_articles: Optional[List[dict]] = None


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
        # 调用后端 AI 分诊
        triage_result = await llm.triage_ticket(
            title=request.title,
            description=request.description,
            category=request.category or "",
            priority=""
        )

        if triage_result:
            logger.info(f"Triage completed for: {request.title}")
            # 转换后端响应
            return _build_triage_response(request, triage_result)

        # 回退到规则-based 分类
        return _rule_based_triage(request)

    except Exception as e:
        logger.error(f"Triage error: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/batch", response_model=List[TriageResponse])
async def classify_batch_tickets(requests: List[TriageRequest]):
    """
    批量智能分类
    """
    results = []
    for req in requests:
        try:
            result = await classify_ticket(req)
            results.append(result)
        except Exception as e:
            logger.error(f"Batch triage error: {str(e)}")
            results.append(None)

    return results


def _rule_based_triage(request: TriageRequest) -> TriageResponse:
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
        related_articles=[{"id": 1, "title": f"{subcategory} 问题排查指南"}]
    )


def _build_triage_response(request: TriageRequest, triage: dict) -> TriageResponse:
    """从后端响应构建分类结果"""
    categories = []
    for cat in triage.get("categories", triage.get("category_suggestions", [])):
        if isinstance(cat, dict):
            categories.append(CategorySuggestion(
                category=cat.get("category", "general"),
                subcategory=cat.get("subcategory", "other"),
                confidence=cat.get("confidence", 0.8)
            ))

    if not categories:
        categories = [CategorySuggestion(category="general", subcategory="other", confidence=0.5)]

    priority_data = triage.get("priority", triage.get("priority_suggestion", {}))
    if isinstance(priority_data, dict):
        priority = priority_data.get("priority", "low")
        priority_confidence = priority_data.get("confidence", 0.8)
    else:
        priority = str(priority_data) if priority_data else "low"
        priority_confidence = 0.8

    assignment_data = triage.get("assignment", triage.get("assignment_suggestion", {}))
    if isinstance(assignment_data, dict):
        assignment = AssignmentSuggestion(
            assignee_id=assignment_data.get("assignee_id"),
            assignee_group=assignment_data.get("assignee_group", "support_team"),
            reason=assignment_data.get("reason", "基于分类分配")
        )
    else:
        assignment = AssignmentSuggestion(assignee_group="support_team", reason="默认分配")

    return TriageResponse(
        categories=categories,
        priority=PrioritySuggestion(priority=priority, confidence=priority_confidence),
        assignment=assignment,
        keywords=triage.get("keywords", []),
        related_articles=triage.get("related_articles", [])
    )