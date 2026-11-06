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


_NOT_IMPLEMENTED_DETAIL = {
    "code": "not_implemented",
    "capability": "change_risk_prediction",
    "message": "变更风险预测尚未实现，不能作为审批依据",
    "reason": (
        "旧实现不读取任何请求字段、也不调用模型：三个 RiskFactor 全是字面常量，"
        "risk_score 恒为 3，approval_requirement 恒为 auto_approved。"
        "让它继续返回 200 会让调用方误以为变更已通过风险评估。"
    ),
}


def _not_implemented():
    """统一的 501：明确告诉调用方这个能力不存在，而不是给一个假结论。"""
    logger.warning("change risk prediction endpoint called but capability is not implemented")
    raise HTTPException(status_code=501, detail=_NOT_IMPLEMENTED_DETAIL)


@router.post("/predict", response_model=RiskResponse)
async def predict_risk(request: ChangeRequest):
    """预测变更风险。

    A2：此端点此前是纯常量占位实现（与输入无关、与模型无关），
    config.yaml 的 risk.cab_review_threshold 是死配置。在接入真实规则/模型
    之前，宁可 501 也不能输出 "low / auto_approved" 这种会被当成放行依据的
    结论。已核查 Go 侧没有 approval_requirement 消费点、也没有对该端点的
    调用，因此不会破坏现有审批链。
    """
    _not_implemented()


@router.post("/batch", response_model=List[RiskResponse])
async def predict_batch_risks(requests: List[ChangeRequest]):
    """批量预测同样未实现；不逐项吞异常再 append(None) 造成 500。"""
    _not_implemented()
