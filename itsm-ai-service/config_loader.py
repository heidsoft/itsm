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
class BackendConfig:
    """ITSM Go 后端的地址与凭据。

    与 llm.* 严格分开：llm.base_url 是「直连的模型厂商端点」（形如
    https://api.openai.com/v1，guidance sidecar 会在其后拼 /chat/completions），
    而这里的 base_url 是「本系统 Go 后端」，SDK 风格为 .../api/v1/ai/*。
    两者曾共用 llm.base_url 一个字段，导致 api/triage.py、api/rca.py 把
    模型端点当成后端地址去请求（https://api.openai.com/v1/api/v1/ai/...），
    永远 404、永远静默降级 —— 功能看起来「有」，实际一次 LLM 调用都没发出去。
    """

    base_url: str = "http://localhost:8080"
    api_key: str = ""  # 后端 JWT（服务间调用），不是模型厂商的 API key
    timeout: int = 120


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
    backend: BackendConfig
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
            backend=BackendConfig(),
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
        backend=BackendConfig(**data.get("backend", {})),
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
    # A5：环境变量为空串时不得覆盖文件/默认值。Compose 里 `LLM_API_KEY=${LLM_API_KEY:-}`
    # 会注入一个空串，旧写法 os.getenv(KEY, fallback) 会把 config.yaml 里配好的
    # 非空 key 直接抹掉，于是「配置明明有、运行时却没有」。
    config.service.host = os.getenv("SERVICE_HOST") or config.service.host
    config.service.port = int(os.getenv("SERVICE_PORT") or config.service.port)
    config.service.log_level = os.getenv("LOG_LEVEL") or config.service.log_level

    # A4：compose 注入了 CORS_ORIGINS，但这里从来没有读过它；
    # main.py 一直用 config.yaml 的 localhost 白名单。
    cors_env = os.getenv("CORS_ORIGINS")
    if cors_env:
        origins = [o.strip() for o in cors_env.split(",") if o.strip()]
        if origins:
            config.service.cors_origins = origins

    config.llm.provider = (os.getenv("LLM_PROVIDER") or config.llm.provider).strip().lower()
    config.llm.api_key = os.getenv("LLM_API_KEY") or config.llm.api_key
    config.llm.model = os.getenv("LLM_MODEL") or config.llm.model
    config.llm.base_url = (
        os.getenv("LLM_BASE_URL")
        or os.getenv("LLM_ENDPOINT")
        or config.llm.base_url
    )
    # MiniMax's supported chat endpoint is Anthropic-compatible, not OpenAI
    # chat/completions. Compose historically supplied the OpenAI default even
    # when LLM_PROVIDER=minimax, so normalize that inherited default here.
    if config.llm.provider == "minimax" and config.llm.base_url.rstrip("/") in {
        "",
        "https://api.openai.com/v1",
    }:
        config.llm.base_url = "https://api.minimaxi.com/anthropic/v1"
    config.llm.timeout = int(os.getenv("LLM_TIMEOUT") or config.llm.timeout)

    # 后端地址独立覆盖：ITSM_BACKEND_URL（首选）/ BACKEND_URL（兼容）。
    config.backend.base_url = (
        os.getenv("ITSM_BACKEND_URL")
        or os.getenv("BACKEND_URL")
        or config.backend.base_url
    ).rstrip("/")
    config.backend.api_key = os.getenv("ITSM_BACKEND_TOKEN") or config.backend.api_key
    config.backend.timeout = int(os.getenv("BACKEND_TIMEOUT") or config.backend.timeout)
    return config


# Global config instance
_config: Optional[Config] = None


# 本地/自建推理端点不需要 API key；其余 provider 缺 key 就是配置错误。
_LOCAL_PROVIDERS = {"ollama", "vllm", "local", "lmstudio", "localai"}


def llm_is_configured(config: Optional[Config] = None) -> bool:
    """供 lifespan 与 /health 共用的单一判定口径（A5）。

    注意不能用「进程启动时看一眼」就完事：env 在容器里可能为空串，
    而 config.yaml 里其实配了 key。这里按与 apply_environment_overrides
    相同的 or 语义取值。
    """
    cfg = config or get_config()
    if cfg.llm.provider in _LOCAL_PROVIDERS:
        return True
    return bool((os.getenv("LLM_API_KEY") or cfg.llm.api_key or "").strip())


def get_config() -> Config:
    """获取全局配置实例"""
    global _config
    if _config is None:
        config_path = os.getenv("ITSM_AI_CONFIG", "config.yaml")
        _config = load_config(config_path)
    return _config
