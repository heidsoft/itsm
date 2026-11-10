"""AI sidecar 契约回归锁（A1/A2/A3/A5）。

运行：python3 tests/contract_regression.py

本机/CI 可能没有 fastapi / pydantic / httpx / PyYAML，因此这里用轻量 stub
只替换框架外壳，跑的是**仓库里的真实模块与真实函数**：字段映射、请求
payload、501 行为、async 化、配置覆盖与健康检查全部走生产代码路径。
安装依赖后同样可运行（stub 只在模块缺失时注入）。
"""
import asyncio
import inspect
import os
import pathlib
import sys
import types

ROOT = pathlib.Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
os.chdir(ROOT)


# ---------------------------------------------------------------- framework stubs
class _BaseModel:
    def __init__(self, **kw):
        # 先把类上声明的默认值铺开，再覆盖显式传入的字段。
        for name, val in type(self).__dict__.items():
            if not name.startswith("_") and not callable(val) and not isinstance(val, (classmethod, staticmethod, property)):
                setattr(self, name, val)
        for k, v in kw.items():
            setattr(self, k, v)


def _field(default=None, **_kw):
    return default


if "pydantic" not in sys.modules:
    try:
        import pydantic  # noqa: F401
    except ImportError:
        m = types.ModuleType("pydantic")
        m.BaseModel, m.Field = _BaseModel, _field
        sys.modules["pydantic"] = m


class _Router:
    def post(self, *_a, **_kw):
        return lambda fn: fn

    def get(self, *_a, **_kw):
        return lambda fn: fn


class _HTTPException(Exception):
    def __init__(self, status_code=500, detail=""):
        super().__init__(str(detail))
        self.status_code = status_code
        self.detail = detail


class _JSONResponse:
    def __init__(self, status_code=200, content=None, **_kw):
        self.status_code = status_code
        self.content = content


class _App:
    def __init__(self, **_kw):
        self.state = types.SimpleNamespace()
        self.added_middleware = []

    def add_middleware(self, cls, **kw):
        self.added_middleware.append((cls, kw))

    def middleware(self, *_a, **_kw):
        # @app.middleware("http") 必须把被装饰函数原样留下，测试才能直接调。
        return lambda fn: fn

    def include_router(self, *_a, **_kw):
        pass

    def get(self, *_a, **_kw):
        return lambda fn: fn


if "fastapi" not in sys.modules:
    try:
        import fastapi  # noqa: F401
    except ImportError:
        m = types.ModuleType("fastapi")
        m.APIRouter, m.HTTPException, m.FastAPI = _Router, _HTTPException, _App
        m.Request = object
        sys.modules["fastapi"] = m
        cors_pkg = types.ModuleType("fastapi.middleware")
        cors_mod = types.ModuleType("fastapi.middleware.cors")
        cors_mod.CORSMiddleware = object
        sys.modules["fastapi.middleware"] = cors_pkg
        sys.modules["fastapi.middleware.cors"] = cors_mod
        responses_mod = types.ModuleType("fastapi.responses")
        responses_mod.JSONResponse = _JSONResponse
        sys.modules["fastapi.responses"] = responses_mod


class _Response:
    def __init__(self, payload):
        self._payload = payload
        self.status_code = 200

    def raise_for_status(self):
        return None

    def json(self):
        return self._payload


class _AsyncClient:
    """记录每次请求；post 是协程，用来证明没有同步阻塞调用。"""
    calls = []

    def __init__(self, **kw):
        self.kw = kw
        self.closed = False

    async def post(self, url, **kw):
        _AsyncClient.calls.append((url, kw))
        if _AsyncClient.fail_once:
            _AsyncClient.fail_once = False
            raise _httpx_stub.TimeoutException("simulated transient timeout")
        return _Response(_AsyncClient.next_payload or {"code": 0, "data": {}})

    async def aclose(self):
        self.closed = True


_AsyncClient.next_payload = None
_AsyncClient.fail_once = False


class _SyncClient:
    def __init__(self, *a, **kw):
        raise AssertionError("guidance 端点不得再使用同步 httpx.Client（会阻塞事件循环）")


if "httpx" not in sys.modules:
    try:
        import httpx  # noqa: F401
    except ImportError:
        m = types.ModuleType("httpx")
        m.AsyncClient = _AsyncClient
        m.Client = _SyncClient
        m.Timeout = lambda *a, **kw: object()
        m.Limits = lambda **kw: object()
        m.TransportError = type("TransportError", (Exception,), {})
        m.TimeoutException = type("TimeoutException", (m.TransportError,), {})
        m.HTTPStatusError = type("HTTPStatusError", (Exception,), {})
        sys.modules["httpx"] = m
_httpx_stub = sys.modules["httpx"]

if "yaml" not in sys.modules:
    try:
        import yaml  # noqa: F401
    except ImportError:
        m = types.ModuleType("yaml")
        m.safe_load = lambda _f: {}
        sys.modules["yaml"] = m


import config_loader  # noqa: E402
from api import guidance, risk, triage  # noqa: E402
from services import llm_client as lc  # noqa: E402
import main  # noqa: E402


def check(cond, msg):
    if not cond:
        raise AssertionError(msg)


def reset_config():
    config_loader._config = None
    for key in ("LLM_API_KEY", "LLM_PROVIDER", "CORS_ORIGINS", "LLM_BASE_URL", "AI_SERVICE_TOKEN"):
        os.environ.pop(key, None)


# ---------------------------------------------------------------- A1 字段映射
GO_TRIAGE_RESPONSE = {
    "title": "生产 MySQL 慢查询",
    "description": "连接池耗尽",
    "promptVersion": "triage-v2",
    "model": "guidance",
    "suggestions": {
        "category": "database",
        "priority": "high",
        "confidence": 0.82,
        "reasoning": "标题含 MySQL 与慢查询",
        "urgency": "high",
    },
    "auditId": 4242,
}


def test_a1_triage_mapping():
    mapped = triage.map_backend_triage(GO_TRIAGE_RESPONSE)
    check(mapped["categories"][0]["category"] == "database", "分类必须来自 suggestions.category，不能恒 general")
    check(mapped["priority"]["priority"] == "high", "优先级必须来自 suggestions.priority，不能恒 low")
    check(mapped["categories"][0]["confidence"] == 0.82, "置信度必须来自 suggestions.confidence")
    check(mapped["priority"]["confidence"] == 0.82, "优先级置信度同样来自 suggestions.confidence")
    check(mapped["assignment"]["reason"] == "标题含 MySQL 与慢查询", "reasoning 应作为分配说明")
    check(mapped["model"] == "guidance" and mapped["promptVersion"] == "triage-v2", "溯源字段必须透传")
    check(mapped["auditId"] == 4242, "auditId 必须透传")
    check(mapped["keywords"] == [] and mapped["related_articles"] == [],
          "后端不返回的字段不得伪造（旧实现凭空造 id=1 的排查指南）")
    check(mapped["degraded"] is False, "正常响应不得标降级")

    degraded = dict(GO_TRIAGE_RESPONSE, degraded=True, degradedReason="guidance sidecar returned guidance-fallback")
    check(triage.map_backend_triage(degraded)["degraded"] is True, "后端自报降级必须透传")
    check(triage.map_backend_triage(degraded)["degradedReason"].startswith("guidance sidecar"), "降级原因必须透传")
    check(triage.map_backend_triage({})["categories"][0]["category"] == "unclassified",
          "空响应不得伪装成 general")
    check(triage.map_backend_triage({"suggestions": {"confidence": 9}})["priority"]["confidence"] == 1.0,
          "置信度必须 clamp 到 [0,1]")

    fallback = triage._rule_based_triage(
        triage.TriageRequest(title="网络不通", description="ping 丢包"), "backend down")
    check(fallback.degraded is True and fallback.model == "keyword-fallback", "本地规则降级必须自报家门")
    check(fallback.related_articles == [], "降级路径也不得伪造关联文档")
    print("A1 triage mapping OK")


def test_a1_llm_client_contract():
    reset_config()
    os.environ["ITSM_BACKEND_URL"] = "http://go-backend:8090"
    client = lc.LLMClient()
    _AsyncClient.calls.clear()
    _AsyncClient.next_payload = {"code": 0, "data": {"answers": [{"answer": "ok"}]}}

    async def run():
        await client.chat([{"role": "user", "content": "hi"}])
        _AsyncClient.next_payload = {"code": 0, "data": {"items": [{"id": 7}]}}
        found = await client.search_knowledge("nginx", limit=3)
        check(found == [{"id": 7}], "RAG 响应字段是 items，不是 results")
        _AsyncClient.next_payload = {"code": 0, "data": GO_TRIAGE_RESPONSE}
        got = await client.triage_ticket("t", "d")
        check(got["model"] == "guidance", "triage 应返回完整 data 对象")
        await client.close()

    asyncio.run(run())
    urls = [u for u, _ in _AsyncClient.calls]
    payloads = {u: kw.get("json", {}) for u, kw in _AsyncClient.calls}
    check("/api/v1/ai/chat" in urls, "chat 路径错误")
    chat_body = payloads["/api/v1/ai/chat"]
    check("conversationId" in chat_body and "pageSize" in chat_body,
          "Go ChatRequest 是驼峰 conversationId/pageSize")
    check("conversation_id" not in chat_body and "limit" not in chat_body,
          "snake_case 字段会让 Go 读成零值")
    check("/api/v1/ai/rag/search" in urls, "知识检索必须打 /ai/rag/search")
    check("/api/v1/ai/knowledge/search" not in urls, "旧路径在 Go 侧不存在（恒 404）")
    rag_body = payloads["/api/v1/ai/rag/search"]
    check(rag_body.get("pageSize") == 3 and "query" in rag_body, "RAG body 需 query/pageSize")
    check(client.backend_url == "http://go-backend:8090", "backend.base_url 不能被厂商端点覆盖")
    print("A1 llm_client contract OK")


# ---------------------------------------------------------------- A2 501
def test_a2_risk_not_implemented():
    req = risk.ChangeRequest(change_id=1, title="上线", description="改配置")

    async def run():
        for fn, arg in ((risk.predict_risk, req), (risk.predict_batch_risks, [req])):
            try:
                await fn(arg)
            except _HTTPException as e:
                check(e.status_code == 501, f"{fn.__name__} 必须 501，实际 {e.status_code}")
                check(isinstance(e.detail, dict) and e.detail.get("code") == "not_implemented",
                      "501 必须带 not_implemented 标识")
                check("approval_requirement" not in e.detail and "prediction" not in e.detail,
                      "501 不得再输出任何风险等级/审批要求结论")
            else:
                raise AssertionError(f"{fn.__name__} 居然返回了假的风险结论")

    asyncio.run(run())
    print("A2 risk 501 OK")


# ---------------------------------------------------------------- A3 async
def test_a3_guidance_async_and_config_error():
    reset_config()  # LLM_API_KEY 未配置
    check(inspect.iscoroutinefunction(guidance._call_llm), "_call_llm 必须是 async")
    check(inspect.iscoroutinefunction(guidance._call_minimax), "_call_minimax 必须是 async")
    src = (ROOT / "api" / "guidance.py").read_text()
    # 注释里会提到旧实现，所以按「真实调用形态」断言，不做裸字符串扫描。
    check("await get_http_client().post" in src, "出网点必须 await 异步 post")
    check("with httpx.Client(timeout=" not in src, "不得残留同步 Client 的 with 块")
    check("AsyncClient" in src and "get_http_client()" in src, "必须复用模块级 AsyncClient")

    guidance._http_client = None
    first = guidance.get_http_client()
    second = guidance.get_http_client()
    check(first is second, "AsyncClient 必须模块级复用（连接池）")

    async def run():
        req = guidance.GuidanceTriageRequest(title="VPN 连不上", description="dns 失败")
        # 凭据缺失是配置错误，必须体现在 degradedReason，且不能阻塞事件循环。
        result = await asyncio.wait_for(guidance.guidance_triage(req), timeout=1.0)
        check(result.degraded is True, "无凭据只能给降级结果")
        check("configuration error" in result.degradedReason,
              "配置错误不能被当成普通的 LLM 调用失败")
        await guidance.close_http_client()
        check(first.closed is True, "lifespan 关闭时必须能真正 aclose")

    asyncio.run(run())
    print("A3 async + config error OK")


# ---------------------------------------------------------------- A5 配置与健康检查
def test_a5_config_and_health():
    reset_config()
    cfg = config_loader.Config(
        service=config_loader.ServiceConfig(),
        database=config_loader.DatabaseConfig(),
        redis=config_loader.RedisConfig(),
        llm=config_loader.LLMConfig(api_key="from-config-yaml"),
        backend=config_loader.BackendConfig(),
        rca=config_loader.RCAConfig(),
        risk=config_loader.RiskConfig(),
        triage=config_loader.TriageConfig(),
    )
    os.environ["LLM_API_KEY"] = ""  # compose 的 `:-` 会注入空串
    os.environ["CORS_ORIGINS"] = "http://itsm-frontend:3000,http://nginx"
    out = config_loader.apply_environment_overrides(cfg)
    check(out.llm.api_key == "from-config-yaml", "env 空串不得抹掉文件里的 key")
    check(out.service.cors_origins == ["http://itsm-frontend:3000", "http://nginx"],
          "CORS_ORIGINS 必须真正生效")
    check(config_loader.llm_is_configured(out) is True, "有 key 应判已配置")

    reset_config()
    config_loader._config = None
    health = asyncio.run(main.health_check())
    check(health["status"] == "degraded", "无凭据时 /health 必须报 degraded")
    check(health["llm"]["configured"] is False, "/health 必须给 llm.configured")
    check("LLM_API_KEY" in health["llm"]["reason"], "降级原因要能指引修复")

    os.environ["LLM_API_KEY"] = "sk-test"
    healthy = asyncio.run(main.health_check())
    check(healthy["status"] == "healthy" and healthy["llm"]["configured"] is True,
          "配好 key 后必须恢复 healthy")
    reset_config()
    print("A5 config + health OK")


# ---------------------------------------------------------------- A4 服务间鉴权 / CORS
def test_a4_service_token_and_cors():
    reset_config()
    os.environ["AI_SERVICE_TOKEN"] = "unit-test-token"
    check(config_loader.service_token_enabled() is True, "配置了令牌必须启用鉴权")
    check(config_loader.verify_service_token("unit-test-token") is True, "正确令牌必须通过")
    check(config_loader.verify_service_token("wrong") is False, "错误令牌必须拒绝")
    check(config_loader.verify_service_token("") is False, "缺令牌必须拒绝")

    class _Req:
        def __init__(self, path, token="", method="POST"):
            self.url = types.SimpleNamespace(path=path)
            self.headers = {"x-service-token": token}
            self.method = method

    async def _next(_request):
        return "PASSED"

    async def run():
        denied = await main.service_token_guard(_Req("/triage"), _next)
        check(isinstance(denied, _JSONResponse) and denied.status_code == 401,
              "无令牌请求必须 401，不能落到端点上消耗 LLM 凭据")
        check(await main.service_token_guard(_Req("/triage", "unit-test-token"), _next) == "PASSED",
              "带正确令牌必须放行")
        check(await main.service_token_guard(_Req("/health"), _next) == "PASSED",
              "/health 必须始终开放，否则 healthcheck 触发重启循环")
        check(await main.service_token_guard(_Req("/api/v1/triage", "", "OPTIONS"), _next) == "PASSED",
              "CORS 预检不得被鉴权拦住")

    asyncio.run(run())

    health = asyncio.run(main.health_check())
    check(health["auth"]["serviceToken"] == "enabled", "/health 必须暴露鉴权状态")
    reset_config()
    health = asyncio.run(main.health_check())
    check(health["auth"]["serviceToken"] == "disabled", "未配令牌时降级态必须可观测")

    # 通配符 + allow_credentials 是非法组合：必须自动关掉凭据。
    cors = [kw for _cls, kw in main.app.added_middleware if "allow_origins" in kw]
    check(cors and cors[0]["allow_credentials"] is False,
          "CORS_ORIGINS 含 '*' 时不得 allow_credentials=True")
    src = (ROOT / "main.py").read_text()
    check("allow_credentials=not wildcard" in src, "显式白名单时才允许带凭据跨域")

    reset_config()
    print("A4 service token + CORS OK")


# ---------------------------------------------------------------- A6 LLM 响应解析
_VALID_ENVELOPE = {
    "choices": [{
        "message": {"content": '```json\n{"category":"database","priority":"high"}\n```'},
        "finish_reason": "stop",
    }],
}


def test_a6_llm_response_robustness():
    check(guidance._parse_triage_json('```json\n{"category":"database"}\n```') == {"category": "database"},
          "必须剥掉 markdown 围栏")
    check(guidance._parse_triage_json('说明如下：{"category":"security"} 以上') == {"category": "security"},
          "模型前后加缀说明文字时仍应取到 JSON 对象")
    for bad in (
        {"choices": []},
        {"choices": [{"message": {"content": ""}}]},
        {"choices": [{"message": {"content": "{}"}, "finish_reason": "length"}]},
        {"error": "upstream"},
        None,
    ):
        try:
            guidance._chat_content(bad)
        except guidance.LLMResponseError:
            pass
        else:
            raise AssertionError(f"非法响应信封必须抛 LLMResponseError: {bad!r}")
    try:
        guidance._parse_triage_json("")
    except guidance.LLMResponseError:
        pass
    else:
        raise AssertionError("空 content 不得被当成成功结果")

    reset_config()
    os.environ["LLM_PROVIDER"] = "openai"
    os.environ["LLM_API_KEY"] = "sk-test"
    guidance._http_client = None
    _AsyncClient.calls.clear()
    _AsyncClient.fail_once = False
    _AsyncClient.next_payload = _VALID_ENVELOPE

    async def run():
        parsed = await guidance._call_llm("MySQL 慢查询", "连接池耗尽")
        check(parsed["category"] == "database", "带围栏的 JSON 必须解析成功，不再静默降级")
        await guidance.close_http_client()

    asyncio.run(run())
    payload = _AsyncClient.calls[-1][1]["json"]
    check(payload.get("max_tokens") == 1024,
          "OpenAI 路径必须下发 max_tokens（config.yaml 定义了却从未使用）")
    check(payload.get("response_format", {}).get("type") == "json_object", "结构化输出约束不能丢")

    # 一次可恢复故障必须重试；重试耗尽仍抛错，而不是伪装成关键词结果。
    guidance._http_client = None
    _AsyncClient.calls.clear()
    _AsyncClient.fail_once = True
    _AsyncClient.next_payload = _VALID_ENVELOPE
    asyncio.run(run())
    check(len(_AsyncClient.calls) == 2, "单次可恢复故障应恰好重试一次")

    # 格式错误同样重试一次；都不成功时错误必须是显式类型。
    guidance._http_client = None
    _AsyncClient.calls.clear()
    _AsyncClient.fail_once = False
    _AsyncClient.next_payload = {"choices": []}

    async def run_bad():
        try:
            await guidance._call_llm("t", "d")
        except guidance.LLMResponseError:
            return
        else:
            raise AssertionError("格式错误必须显式抛出，不能静默返回")
        finally:
            await guidance.close_http_client()

    asyncio.run(run_bad())
    check(len(_AsyncClient.calls) == 2, "格式偏差也只重试一次，不能无限打厂商接口")

    reset_config()
    print("A6 LLM response parsing OK")


if __name__ == "__main__":
    test_a1_triage_mapping()
    test_a1_llm_client_contract()
    test_a2_risk_not_implemented()
    test_a3_guidance_async_and_config_error()
    test_a5_config_and_health()
    test_a4_service_token_and_cors()
    test_a6_llm_response_robustness()
    print("ALL CONTRACT CHECKS PASSED")
