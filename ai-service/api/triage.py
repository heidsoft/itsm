"""
智能分类 API
根据工单/事件描述自动分类、确定优先级和分配建议
"""
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field
from typing import Optional, List
import logging

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
    try:
        # TODO: 集成实际的分类模型
        # 这里返回模拟结果
        result = TriageResponse(
            categories=[
                CategorySuggestion(
                    category="infrastructure",
                    subcategory="network",
                    confidence=0.85
                )
            ],
            priority=PrioritySuggestion(
                priority="high",
                confidence=0.78
            ),
            assignment=AssignmentSuggestion(
                assignee_group="network_team",
                reason="网络相关问题，分配至网络组"
            ),
            keywords=["网络", "连接", "故障"],
            related_articles=[
                {"id": 1, "title": "网络连接问题排查"}
            ]
        )

        logger.info(f"Triage completed for: {request.title}")
        return result

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
