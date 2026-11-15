"""跨端点共享的契约基类。

AI-Native 计划批次 E：sidecar 对外 JSON 契约统一 camelCase（与 Go 后端、
前端同一套字段命名）。Python 侧字段名保持 snake_case（语言惯例），序列化
经 to_camel 别名输出 camelCase；populate_by_name 允许内部代码继续用
snake_case 字段名构造，互不干扰。

历史背景：rca/risk 端点早于 guidance 兼容端点存在，DTO 用了 snake_case
（problem_id / root_cause / change_type / overall_risk_level …），与
itsm-backend 的 camelCase 契约规范冲突。这些端点当前无后端消费方
（guidance_client.go 只调 camelCase 的 /triage 兼容端点），camelCase 化
零破坏；后续若有新消费方，从第一天起读到的就是统一命名。
"""
from pydantic import BaseModel, ConfigDict
from pydantic.alias_generators import to_camel


class CamelModel(BaseModel):
    """snake_case 字段名 ↔ camelCase JSON 契约的基类。

    - 序列化/反序列化按 camelCase 别名（FastAPI response_model 默认 by_alias）；
    - populate_by_name：内部构造仍可用 snake_case 字段名。
    """

    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)
