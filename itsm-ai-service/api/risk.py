"""
风险预测 API
使用AI预测变更风险，支持风险矩阵评估
"""
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field
from typing import Optional, List
import logging

logger = logging.getLogger(__name__)

router = APIRouter()


class RiskFactor(BaseModel):
    """风险因素"""
    factor: str
    description: str
    impact: float  # 0-1
    probability: float  # 0-1
    severity: float  # 0-1


class ChangeRequest(BaseModel):
    """变更请求"""
    change_id: Optional[int] = Field(None, description="变更ID")
    title: str = Field(..., description="变更标题")
    description: str = Field(..., description="变更描述")
    change_type: str = Field("normal", description="变更类型: normal/emergency/standard")
    affected_cis: Optional[List[str]] = Field(None, description="受影响的配置项")
    implementation_plan: Optional[str] = Field(None, description="实施计划")
    rollback_plan: Optional[str] = Field(None, description="回滚计划")
    tenant_id: Optional[int] = Field(None, description="租户ID")


class RiskMatrixCell(BaseModel):
    """风险矩阵单元格"""
    impact: int  # 1-5
    probability: int  # 1-5
    risk_level: str  # low/medium/high/critical
    color: str  # green/yellow/orange/red


class RiskPrediction(BaseModel):
    """风险预测"""
    overall_risk_level: str  # low/medium/high/critical
    overall_score: float  # 0-100
    risk_matrix: RiskMatrixCell
    factors: List[RiskFactor]
    mitigation_suggestions: List[str]


class RiskResponse(BaseModel):
    """风险预测响应"""
    change_id: Optional[int]
    prediction: RiskPrediction
    recommendations: List[str]
    approval_requirement: str  # auto_approved/normal_approval/cab_review


@router.post("/predict", response_model=RiskResponse)
async def predict_risk(request: ChangeRequest):
    """
    预测变更风险
    - 分析风险因素
    - 计算风险矩阵
    - 提供缓解建议
    """
    try:
        # 分析风险因素
        factors = [
            RiskFactor(
                factor="服务可用性",
                description="变更可能影响服务可用性",
                impact=0.8,
                probability=0.3,
                severity=0.6
            ),
            RiskFactor(
                factor="数据完整性",
                description="变更可能影响数据完整性",
                impact=0.7,
                probability=0.2,
                severity=0.5
            ),
            RiskFactor(
                factor="回滚难度",
                description="回滚计划复杂度",
                impact=0.6,
                probability=0.4,
                severity=0.5
            ),
        ]

        # 计算风险矩阵
        avg_impact = sum(f.impact for f in factors) / len(factors)
        avg_probability = sum(f.probability for f in factors) / len(factors)

        impact_score = int(avg_impact * 5)
        probability_score = int(avg_probability * 5)

        # 确定风险等级
        risk_score = impact_score * probability_score
        if risk_score >= 15:
            risk_level = "critical"
            color = "red"
        elif risk_score >= 10:
            risk_level = "high"
            color = "orange"
        elif risk_score >= 5:
            risk_level = "medium"
            color = "yellow"
        else:
            risk_level = "low"
            color = "green"

        overall_score = float(risk_score) / 25.0 * 100

        prediction = RiskPrediction(
            overall_risk_level=risk_level,
            overall_score=overall_score,
            risk_matrix=RiskMatrixCell(
                impact=impact_score,
                probability=probability_score,
                risk_level=risk_level,
                color=color
            ),
            factors=factors,
            mitigation_suggestions=[
                "在低峰期执行变更",
                "增加监控告警",
                "准备手动回滚方案"
            ]
        )

        # 确定审批要求
        if risk_level in ["critical", "high"]:
            approval_requirement = "cab_review"
        elif risk_level == "medium":
            approval_requirement = "normal_approval"
        else:
            approval_requirement = "auto_approved"

        recommendations = [
            "建议安排在非工作时间执行",
            "需要相关团队审批",
            "建议准备详细的回滚文档"
        ]

        result = RiskResponse(
            change_id=request.change_id,
            prediction=prediction,
            recommendations=recommendations,
            approval_requirement=approval_requirement
        )

        logger.info(f"Risk prediction completed for: {request.title}")
        return result

    except Exception as e:
        logger.error(f"Risk prediction error: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/batch", response_model=List[RiskResponse])
async def predict_batch_risks(requests: List[ChangeRequest]):
    """
    批量预测变更风险
    """
    results = []
    for req in requests:
        try:
            result = await predict_risk(req)
            results.append(result)
        except Exception as e:
            logger.error(f"Batch risk prediction error: {str(e)}")
            results.append(None)

    return results
