"""
配置管理
"""
import os
from dataclasses import dataclass, field
from typing import List, Optional
import yaml


@dataclass
class LLMConfig:
    provider: str = "openai"
    api_key: str = ""
    base_url: str = "https://api.openai.com/v1"
    model: str = "gpt-4o-mini"
    timeout: int = 120
    max_tokens: int = 4096


@dataclass
class DatabaseConfig:
    host: str = "localhost"
    port: int = 5432
    user: str = "itsm"
    password: str = "itsm_password_2026"
    database: str = "itsm"
    pool_size: int = 10
    max_overflow: int = 20
    sslmode: str = "disable"

    @property
    def url(self) -> str:
        return f"postgresql://{self.user}:{self.password}@{self.host}:{self.port}/{self.database}?sslmode={self.sslmode}"


@dataclass
class RedisConfig:
    host: str = "localhost"
    port: int = 6379
    password: str = ""
    db: int = 0
    key_prefix: str = "itsm-ai:"


@dataclass
class RCAConfig:
    max_findings: int = 10
    default_method: str = "5_whys"
    similarity_threshold: float = 0.7


@dataclass
class RiskConfig:
    auto_approve_threshold: str = "low"
    normal_approval_threshold: str = "medium"
    cab_review_threshold: str = "high"
    impact_weight: float = 0.6
    probability_weight: float = 0.4


@dataclass
class TriageConfig:
    confidence_threshold: float = 0.6
    max_related_articles: int = 5
    max_keywords: int = 10


@dataclass
class ServiceConfig:
    name: str = "itsm-ai-service"
    version: str = "1.0.0"
    host: str = "0.0.0.0"
    port: int = 8000
    log_level: str = "info"
    cors_origins: List[str] = field(default_factory=lambda: ["*"])


@dataclass
class Config:
    service: ServiceConfig
    database: DatabaseConfig
    redis: RedisConfig
    llm: LLMConfig
    rca: RCAConfig
    risk: RiskConfig
    triage: TriageConfig


def load_config(path: str = "config.yaml") -> Config:
    """加载配置文件"""
    if not os.path.exists(path):
        # Return default config if file doesn't exist
        return Config(
            service=ServiceConfig(),
            database=DatabaseConfig(),
            redis=RedisConfig(),
            llm=LLMConfig(),
            rca=RCAConfig(),
            risk=RiskConfig(),
            triage=TriageConfig(),
        )

    with open(path, "r") as f:
        data = yaml.safe_load(f)

    return Config(
        service=ServiceConfig(**data.get("service", {})),
        database=DatabaseConfig(**data.get("database", {})),
        redis=RedisConfig(**data.get("redis", {})),
        llm=LLMConfig(**data.get("llm", {})),
        rca=RCAConfig(**data.get("rca", {})),
        risk=RiskConfig(**data.get("risk", {})),
        triage=TriageConfig(**data.get("triage", {})),
    )


# Global config instance
_config: Optional[Config] = None


def get_config() -> Config:
    """获取全局配置实例"""
    global _config
    if _config is None:
        config_path = os.getenv("ITSM_AI_CONFIG", "config.yaml")
        _config = load_config(config_path)
    return _config