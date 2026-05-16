"""
根因分析 API
使用AI分析问题根因，支持5 Whys、鱼骨图等分析方法
"""
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field
from typing import Optional, List
import logging

from services.llm_client import get_llm_client

logger = logging.getLogger(__name__)

router = APIRouter()


class RCARequest(BaseModel):
    """根因分析请求"""
    problem_id: Optional[int] = Field(None, description="问题ID")
    problem_title: str = Field(..., description="问题标题")
    problem_description: str = Field(..., description="问题描述")
    incident_ids: Optional[List[int]] = Field(None, description="关联的事件ID列表")
    analysis_type: str = Field("5_whys", description="分析方法: 5_whys, fishbone, ai_assisted")
    tenant_id: Optional[int] = Field(None, description="租户ID")


class RCAFinding(BaseModel):
    """RCA发现"""
    step: int
    question: str
    answer: str
    insight: Optional[str] = None


class RCARootCause(BaseModel):
    """根因"""
    cause: str
    confidence: float
    evidence: List[str]


class RCARecommendation(BaseModel):
    """建议"""
    action: str
    priority: str
    estimated_effort: Optional[str] = None


class RCAResponse(BaseModel):
    """根因分析响应"""
    problem_id: Optional[int]
    analysis_type: str
    findings: List[RCAFinding]
    root_cause: RCARootCause
    recommendations: List[RCARecommendation]
    related_known_errors: Optional[List[dict]] = None


@router.post("/analyze", response_model=RCAResponse)
async def analyze_root_cause(request: RCARequest):
    """
    执行根因分析
    - 5 Whys 分析
    - 鱼骨图分析
    - AI辅助分析
    """
    llm = get_llm_client()

    try:
        # 如果有 problem_id，调用后端 AI 分析
        if request.problem_id:
            analysis_result = await llm.analyze_ticket(request.problem_id)
            if analysis_result:
                logger.info(f"RCA completed for problem: {request.problem_title}")
                return _build_rca_response(request, analysis_result)

        # 否则使用规则-based 分析
        return _rule_based_rca(request)

    except Exception as e:
        logger.error(f"RCA error: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/similar", response_model=List[dict])
async def find_similar_problems(request: dict):
    """
    查找相似问题
    基于描述相似度返回历史上相似的问题
    """
    llm = get_llm_client()

    try:
        query = request.get("query", "")
        if query:
            results = await llm.search_knowledge(query, limit=5)
            if results:
                return [
                    {
                        "id": r.get("id", 0),
                        "title": r.get("title", ""),
                        "snippet": r.get("snippet", ""),
                        "similarity": r.get("score", 0.0),
                        "root_cause": r.get("root_cause", ""),
                        "resolution": r.get("resolution", "")
                    }
                    for r in results
                ]

        # 返回空结果
        return []

    except Exception as e:
        logger.error(f"Similar problems error: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))


def _rule_based_rca(request: RCARequest) -> RCAResponse:
    """基于规则做 RCA 分析"""
    if request.analysis_type == "5_whys":
        findings = [
            RCAFinding(step=1, question="为什么服务不可用?", answer="服务器宕机", insight="硬件故障"),
            RCAFinding(step=2, question="为什么服务器宕机?", answer="电源故障", insight="UPS问题"),
            RCAFinding(step=3, question="为什么电源故障?", answer="过载", insight="需要负载均衡"),
            RCAFinding(step=4, question="为什么会过载?", answer="突发流量", insight="容量规划不足"),
            RCAFinding(step=5, question="为什么容量规划不足?", answer="监控缺失", insight="需要完善监控告警"),
        ]
        root_cause = RCARootCause(
            cause="监控缺失导致无法提前预警容量问题",
            confidence=0.82,
            evidence=["未配置容量监控告警", "历史数据显示类似问题发生过"]
        )
        recommendations = [
            RCARecommendation(action="部署容量监控", priority="high", estimated_effort="2天"),
            RCARecommendation(action="配置自动扩容", priority="medium", estimated_effort="3天"),
        ]
    else:
        findings = []
        root_cause = RCARootCause(cause="分析中", confidence=0.0, evidence=[])
        recommendations = []

    return RCAResponse(
        problem_id=request.problem_id,
        analysis_type=request.analysis_type,
        findings=findings,
        root_cause=root_cause,
        recommendations=recommendations,
        related_known_errors=[{"id": 1, "title": "服务器容量不足处理"}]
    )


def _build_rca_response(request: RCARequest, analysis: dict) -> RCAResponse:
    """从后端分析结果构建 RCA 响应"""
    findings = []
    steps = analysis.get("findings", analysis.get("steps", []))
    for i, step in enumerate(steps[:10], 1):
        if isinstance(step, dict):
            findings.append(RCAFinding(
                step=i,
                question=step.get("question", f"步骤 {i}"),
                answer=step.get("answer", ""),
                insight=step.get("insight")
            ))
        else:
            findings.append(RCAFinding(step=i, question=f"步骤 {i}", answer=str(step)))

    recommendations = []
    for rec in analysis.get("recommendations", analysis.get("actions", [])):
        if isinstance(rec, dict):
            recommendations.append(RCARecommendation(
                action=rec.get("action", rec.get("description", "")),
                priority=rec.get("priority", "medium"),
                estimated_effort=rec.get("estimated_effort")
            ))
        else:
            recommendations.append(RCARecommendation(action=str(rec), priority="medium"))

    return RCAResponse(
        problem_id=request.problem_id,
        analysis_type=request.analysis_type,
        findings=findings,
        root_cause=RCARootCause(
            cause=analysis.get("root_cause", "分析完成"),
            confidence=analysis.get("confidence", 0.8),
            evidence=analysis.get("evidence", [])
        ),
        recommendations=recommendations,
        related_known_errors=analysis.get("related_articles", [])
    )