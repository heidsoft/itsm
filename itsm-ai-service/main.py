"""
ITSM AI Service - FastAPI Entry Point
智能服务模块，提供分类、根因分析、风险预测等功能
"""
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from contextlib import asynccontextmanager
import logging

from config_loader import get_config, llm_is_configured
from api import triage, rca, risk, guidance

# 配置日志
cfg = get_config()
logging.basicConfig(
    level=getattr(logging, cfg.service.log_level.upper(), logging.INFO),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """应用生命周期管理。

    A5：凭据缺失必须在这里就能被发现。不能抛异常让 uvicorn 退出——
    compose 是 restart: always，直接崩会变成无限重启（A3 的 healthcheck
    超时已经是同类问题）。做法是把状态写进 app.state + ERROR 日志，
    再由 /health 以 status=degraded / llm.configured=false 对外暴露。
    """
    cfg = get_config()
    configured = llm_is_configured(cfg)
    app.state.llm_configured = configured
    logger.info(f"{cfg.service.name} v{cfg.service.version} starting up...")
    logger.info(f"LLM provider: {cfg.llm.provider}, model: {cfg.llm.model}, configured: {configured}")
    if not configured:
        logger.error(
            "LLM credentials are not configured: guidance sidecar will only serve "
            "keyword fallback. Set LLM_API_KEY (or use a local provider) — "
            "/health will report status=degraded."
        )
    yield
    # 优雅关闭模块级 AsyncClient，避免事件循环退出后遗留未关闭连接。
    await guidance.close_http_client()
    logger.info(f"{cfg.service.name} shutting down...")


# 创建FastAPI应用
cfg = get_config()
app = FastAPI(
    title=cfg.service.name,
    description="IT服务管理智能分析服务",
    version=cfg.service.version,
    lifespan=lifespan
)

# 配置CORS
origins = cfg.service.cors_origins
app.add_middleware(
    CORSMiddleware,
    allow_origins=origins if "*" not in origins else ["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# 注册路由
app.include_router(
    triage.router,
    prefix="/api/v1/triage",
    tags=["triage"]
)

app.include_router(
    rca.router,
    prefix="/api/v1/rca",
    tags=["rca"]
)

app.include_router(
    risk.router,
    prefix="/api/v1/risk",
    tags=["risk"]
)

# Guidance sidecar 兼容端点（后端 GuidanceClient 调用）: POST /triage
app.include_router(
    guidance.router,
    tags=["guidance"]
)


@app.get("/health")
async def health_check():
    """健康检查。

    HTTP 状态码保持 200：compose healthcheck 只看 status==200，一旦返回 503
    会触发 restart: always 的重启循环，反而把「配置错误」这个可诊断状态
    变成「容器一直起不来」的假故障。降级信息放在响应体里，让 Go 侧
    GuidanceClient.HealthCheck 与运维都能读到。
    """
    cfg = get_config()
    configured = llm_is_configured(cfg)
    return {
        "status": "healthy" if configured else "degraded",
        "service": cfg.service.name,
        "version": cfg.service.version,
        "llm": {
            "provider": cfg.llm.provider,
            "model": cfg.llm.model,
            "configured": configured,
            "reason": "" if configured else "LLM_API_KEY not configured; keyword fallback only",
        }
    }


@app.get("/")
async def root():
    """根路径"""
    cfg = get_config()
    return {
        "service": cfg.service.name,
        "version": cfg.service.version,
        "endpoints": {
            "triage": "/api/v1/triage",
            "rca": "/api/v1/rca",
            "risk": "/api/v1/risk"
        }
    }


if __name__ == "__main__":
    import uvicorn
    cfg = get_config()
    uvicorn.run(app, host=cfg.service.host, port=cfg.service.port)