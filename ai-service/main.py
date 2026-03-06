"""
ITSM AI Service - FastAPI Entry Point
智能服务模块，提供分类、根因分析、风险预测等功能
"""
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from contextlib import asynccontextmanager
import logging

from api import triage, rca, risk

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """应用生命周期管理"""
    logger.info("ITSM AI Service starting up...")
    # 初始化服务
    yield
    logger.info("ITSM AI Service shutting down...")


# 创建FastAPI应用
app = FastAPI(
    title="ITSM AI Service",
    description="IT服务管理智能分析服务",
    version="1.0.0",
    lifespan=lifespan
)

# 配置CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
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
    return {"status": "healthy", "service": "itsm-ai-service"}


@app.get("/")
async def root():
    """根路径"""
    return {
        "service": "ITSM AI Service",
        "version": "1.0.0",
        "endpoints": {
            "triage": "/api/v1/triage",
            "rca": "/api/v1/rca",
            "risk": "/api/v1/risk"
        }
    }


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
