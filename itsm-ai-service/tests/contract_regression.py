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

# 断言用异常类：装了真 fastapi 时必须用真类——生产代码 raise 的是
# fastapi.HTTPException，测试 except 用 stub 类会因类身份不同而捕获不到
# （本机装依赖后 test_a2 未捕获 501 的根因）。未装依赖时才落到 stub 类。
import fastapi as _fastapi_mod  # noqa: E402  # 此时必在 sys.modules（真库或 stub）

HTTPExceptionForAssert = _fastapi_mod.HTTPException
JSONResponseForAssert = _fastapi_mod.responses.JSONResponse


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
from api import guidance, risk, rca, triage  # noqa: E402
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
            except HTTPExceptionForAssert as e:
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
        check(isinstance(denied, JSONResponseForAssert) and denied.status_code == 401,
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
    # stub 环境：added_middleware = [(cls, kw)]；真库环境：user_middleware = [Middleware(cls, *args, **kwargs)]。
    # 两种形态都读 kwargs，兼容断言（同一生产代码，两种运行环境）。
    middlewares = getattr(main.app, "added_middleware", None)
    if middlewares is None:
        middlewares = [(m.cls, getattr(m, "kwargs", {})) for m in getattr(main.app, "user_middleware", [])]
    cors = [kw for _cls, kw in middlewares if "allow_origins" in kw]
    # wildcard 前提只能在 import main 之前用环境注入；脚本 import 在顶部，
    # 运行期 CORS_ORIGINS 是缺省白名单（非 wildcard）→ allow_credentials=True
    # 是合法形态。wildcard 禁带凭据的契约由下面的源码断言锁实现。
    if any("*" in (kw.get("allow_origins") or []) for kw in cors):
        check(cors[0]["allow_credentials"] is False,
              "CORS_ORIGINS 含 '*' 时不得 allow_credentials=True")
    else:
        check(cors and cors[0]["allow_credentials"] is True,
              "显式白名单（非 wildcard）下 allow_credentials=True 是合法形态")
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




def test_b1_rca_camel_case_contract():
    """批次 E：rca/risk DTO 契约 camelCase 化回归锁。

    Python 字段名保持 snake_case（语言惯例），JSON 契约必须 camelCase：
    - 对外接受 camelCase 别名（model_validate）；
    - 序列化（FastAPI 默认 by_alias）只输出 camelCase；
    - populate_by_name 允许内部 snake_case 构造，两者并存。
    """
    # 请求侧：camelCase 别名解析
    req = rca.RCARequest.model_validate({
        "problemId": 7, "problemTitle": "订单库主从延迟",
        "problemDescription": "延迟持续增长", "analysisType": "5_whys",
        "tenantId": 1,
    })
    check(req.problem_id == 7 and req.analysis_type == "5_whys", "camelCase 别名必须可解析")
    # 内部构造（populate_by_name）
    req2 = rca.RCARequest(problem_title="t", problem_description="d")
    check(req2.problem_title == "t", "内部 snake_case 构造必须继续可用")
    # 序列化：by_alias 只输出 camelCase，不出现 snake_case
    dumped = req2.model_dump(by_alias=True)
    for k in ("problemTitle", "problemDescription", "analysisType", "tenantId"):
        check(k in dumped, f"序列化必须输出 camelCase 键 {k}")
    for k in ("problem_title", "problem_description", "analysis_type", "tenant_id"):
        check(k not in dumped, f"序列化不得输出 snake_case 键 {k}")
    # 响应侧：RCAResponse 含既有 camelCase 字段（degradedReason）与新别名字段并存
    resp = rca.RCAResponse(
        problem_id=1, analysis_type="5_whys", findings=[],
        root_cause=rca.RCARootCause(cause="c", confidence=0.0, evidence=[]),
        recommendations=[], degraded=True, degradedReason="llm unavailable")
    rd = resp.model_dump(by_alias=True)
    check("problemId" in rd and "rootCause" in rd and "relatedKnownErrors" in rd,
          "RCAResponse 序列化键必须 camelCase")
    check(rd["degraded"] is True and rd["degradedReason"] == "llm unavailable",
          "既有 camelCase 字段不受别名改造影响")
    # risk.py 同批
    rr = risk.RiskResponse.model_validate({
        "changeId": 5,
        "prediction": {"overallRiskLevel": "high", "overallScore": 80.0,
                        "riskMatrix": {"impact": 4, "probability": 4,
                                        "riskLevel": "high", "color": "orange"},
                        "factors": [], "mitigationSuggestions": []},
        "recommendations": [], "approvalRequirement": "cab_review",
    })
    rd2 = rr.model_dump(by_alias=True)
    check("changeId" in rd2 and "approvalRequirement" in rd2, "RiskResponse 序列化键必须 camelCase")
    check(rd2["prediction"]["overallRiskLevel"] == "high", "嵌套 model 同样按 camelCase 序列化")
    print("B1 rca/risk camelCase contract OK")


def test_b1_rca_error_not_echoed():
    """批次 E：rca 端点不得把异常原文回显给调用方。

    旧实现 raise HTTPException(detail=str(e))——异常原文可能携带后端地址、
    凭据片段等内部细节。对齐 triage.py 已修好的口径：日志只记类型名，
    响应给通用文案（502）。
    """
    import inspect as _inspect

    for fn_name in ("analyze_root_cause", "find_similar_problems"):
        src = _inspect.getsource(getattr(rca, fn_name))
        check("detail=str(e)" not in src and "detail=str(exc)" not in src,
              f"{fn_name} 不得把异常原文写进 HTTPException detail")
        check("type(e).__name__" in src, f"{fn_name} 日志只记异常类型名")
        check("暂时不可用" in src, f"{fn_name} 响应必须给通用文案而非内部细节")
    print("B1 rca error sanitization OK")


if __name__ == "__main__":
    test_a1_triage_mapping()
    test_a1_llm_client_contract()
    test_a2_risk_not_implemented()
    test_a3_guidance_async_and_config_error()
    test_a5_config_and_health()
    test_a4_service_token_and_cors()
    test_a6_llm_response_robustness()
    test_b1_rca_camel_case_contract()
    test_b1_rca_error_not_echoed()
    print("ALL CONTRACT CHECKS PASSED")
