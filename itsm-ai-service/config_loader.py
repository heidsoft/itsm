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
    password: str = ""
    database: str = "itsm"
    pool_size: int = 10
    max_overflow: int = 20
    sslmode: str = "disable"

    def __post_init__(self) -> None:
        # Never ship a default credential in the repository. When config.yaml
        # leaves the password empty, fall back to the DB_PASSWORD environment
        # variable (or point ITSM_AI_CONFIG at a private config file).
        if not self.password:
            self.password = os.getenv("DB_PASSWORD", "")

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
        config = Config(
            service=ServiceConfig(),
            database=DatabaseConfig(),
            redis=RedisConfig(),
            llm=LLMConfig(),
            rca=RCAConfig(),
            risk=RiskConfig(),
            triage=TriageConfig(),
        )
        return apply_environment_overrides(config)

    with open(path, "r") as f:
        data = yaml.safe_load(f)

    config = Config(
        service=ServiceConfig(**data.get("service", {})),
        database=DatabaseConfig(**data.get("database", {})),
        redis=RedisConfig(**data.get("redis", {})),
        llm=LLMConfig(**data.get("llm", {})),
        rca=RCAConfig(**data.get("rca", {})),
        risk=RiskConfig(**data.get("risk", {})),
        triage=TriageConfig(**data.get("triage", {})),
    )
    return apply_environment_overrides(config)


def apply_environment_overrides(config: Config) -> Config:
    """Apply deployment-safe environment overrides to file/default config.

    Docker Compose injects provider credentials at runtime.  Keeping this
    translation here makes the Python guidance sidecar use the same provider,
    model and timeout as the Go LLM gateway without storing secrets in YAML.
    """
    config.service.host = os.getenv("SERVICE_HOST", config.service.host)
    config.service.port = int(os.getenv("SERVICE_PORT", str(config.service.port)))
    config.service.log_level = os.getenv("LOG_LEVEL", config.service.log_level)

    config.llm.provider = os.getenv("LLM_PROVIDER", config.llm.provider).strip().lower()
    config.llm.api_key = os.getenv("LLM_API_KEY", config.llm.api_key)
    config.llm.model = os.getenv("LLM_MODEL", config.llm.model)
    config.llm.base_url = os.getenv(
        "LLM_BASE_URL",
        os.getenv("LLM_ENDPOINT", config.llm.base_url),
    )
    # MiniMax's supported chat endpoint is Anthropic-compatible, not OpenAI
    # chat/completions. Compose historically supplied the OpenAI default even
    # when LLM_PROVIDER=minimax, so normalize that inherited default here.
    if config.llm.provider == "minimax" and config.llm.base_url.rstrip("/") in {
        "",
        "https://api.openai.com/v1",
    }:
        config.llm.base_url = "https://api.minimaxi.com/anthropic/v1"
    config.llm.timeout = int(os.getenv("LLM_TIMEOUT", str(config.llm.timeout)))
    return config


# Global config instance
_config: Optional[Config] = None


def get_config() -> Config:
    """获取全局配置实例"""
    global _config
    if _config is None:
        config_path = os.getenv("ITSM_AI_CONFIG", "config.yaml")
        _config = load_config(config_path)
    return _config
