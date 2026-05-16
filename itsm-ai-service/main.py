"""
ITSM AI Service - FastAPI Entry Point
智能服务模块，提供分类、根因分析、风险预测等功能
"""
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from contextlib import asynccontextmanager
import logging

from config_loader import get_config
from api import triage, rca, risk

# 配置日志
cfg = get_config()
logging.basicConfig(
    level=getattr(logging, cfg.service.log_level.upper(), logging.INFO),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """应用生命周期管理"""
    cfg = get_config()
    logger.info(f"{cfg.service.name} v{cfg.service.version} starting up...")
    logger.info(f"LLM provider: {cfg.llm.provider}, model: {cfg.llm.model}")
    yield
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


@app.get("/health")
async def health_check():
    """健康检查"""
    cfg = get_config()
    return {
        "status": "healthy",
        "service": cfg.service.name,
        "version": cfg.service.version,
        "llm": {
            "provider": cfg.llm.provider,
            "model": cfg.llm.model,
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