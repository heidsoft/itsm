"""
后端 AI 服务客户端。

命名注意：本模块调的是**本系统 Go 后端**的 /api/v1/ai/* 接口，不是模型厂商
的 chat/completions。厂商端点属于 guidance sidecar（api/guidance.py），
两者的地址配置已拆分（llm.base_url vs backend.base_url）——共用同一个字段时
这里会把 https://api.openai.com/v1 当成后端地址，请求恒 404、静默降级。
"""
import httpx
import logging
from typing import List, Dict, Any, Optional

from config_loader import get_config

logger = logging.getLogger(__name__)


class LLMClient:
    """LLM 客户端，调用后端 AI 服务"""

    def __init__(self, backend_url: Optional[str] = None):
        cfg = get_config()
        # 后端地址与模型端点严格分开：用 backend.*，不要回落到 llm.base_url。
        self.backend_url = (backend_url or cfg.backend.base_url).rstrip("/")
        # 服务间调用凭据是后端 JWT，不是模型厂商的 API key；未配置时宁可不发
        # Authorization，也不要把厂商 key 泄漏给后端。
        self.api_key = cfg.backend.api_key
        self.model = cfg.llm.model
        self.timeout = cfg.backend.timeout
        self._client = None

    @property
    def client(self) -> httpx.AsyncClient:
        if self._client is None:
            self._client = httpx.AsyncClient(
                base_url=self.backend_url,
                timeout=httpx.Timeout(self.timeout),
                headers={"Authorization": f"Bearer {self.api_key}"} if self.api_key else {}
            )
        return self._client

    async def close(self):
        if self._client:
            await self._client.aclose()
            self._client = None

    async def chat(self, messages: List[Dict[str, str]], model: Optional[str] = None) -> str:
        """发送聊天请求（Go: POST /api/v1/ai/chat）。"""
        try:
            response = await self.client.post(
                "/api/v1/ai/chat",
                json={
                    # Go ChatRequest 是驼峰：query/pageSize/conversationId。
                    # 此前发 conversation_id/limit，Go 一律读成零值，
                    # 多轮会话与检索条数静默失效。
                    "query": messages[-1]["content"] if messages else "",
                    "pageSize": 5,
                    "conversationId": 0,
                }
            )
            response.raise_for_status()
            data = response.json()
            if data.get("code") == 0:
                answers = data.get("data", {}).get("answers", [])
                if answers:
                    return answers[0].get("answer", "")
            return ""
        except Exception as e:
            logger.error(f"LLM chat error: {e}")
            return ""

    async def triage_ticket(self, title: str, description: str, category: str = "", priority: str = "") -> Dict[str, Any]:
        """工单分诊"""
        try:
            response = await self.client.post(
                "/api/v1/ai/triage",
                json={
                    "title": title,
                    "description": description,
                    "category": category,
                    "priority": priority
                }
            )
            response.raise_for_status()
            data = response.json()
            if data.get("code") == 0:
                return data.get("data", {})
            return {}
        except Exception as e:
            logger.error(f"Triage error: {e}")
            return {}

    async def analyze_ticket(self, ticket_id: int) -> Dict[str, Any]:
        """分析工单"""
        try:
            response = await self.client.post(
                f"/api/v1/ai/tickets/{ticket_id}/analyze",
                json={}
            )
            response.raise_for_status()
            data = response.json()
            if data.get("code") == 0:
                return data.get("data", {})
            return {}
        except Exception as e:
            logger.error(f"Analyze ticket error: {e}")
            return {}

    async def search_knowledge(self, query: str, limit: int = 5) -> List[Dict[str, Any]]:
        """搜索知识库（Go: POST /api/v1/ai/rag/search）。"""
        try:
            response = await self.client.post(
                "/api/v1/ai/rag/search",
                # Go KnowledgeSearch 读 {query, pageSize, type}；
                # 旧实现打的是不存在的 /ai/knowledge/search 且发 limit，
                # 结果恒 404 → 永远返回空列表。
                json={"query": query, "pageSize": limit, "type": "kb"}
            )
            response.raise_for_status()
            data = response.json()
            if data.get("code") == 0:
                payload = data.get("data", {})
                # 响应字段是 items；旧实现读 results，即使路径修好也仍是空。
                return payload.get("items", []) or []
            return []
        except Exception as e:
            logger.error(f"Knowledge search error: {e}")
            return []


# 全局客户端实例
_llm_client: Optional[LLMClient] = None


def get_llm_client() -> LLMClient:
    """获取全局 LLM 客户端"""
    global _llm_client
    if _llm_client is None:
        _llm_client = LLMClient()
    return _llm_client