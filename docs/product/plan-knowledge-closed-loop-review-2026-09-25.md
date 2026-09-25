# Review：plan-knowledge-closed-loop-2026-09-24.md — 2026-09-25

> **来源**：初始化/演练会话对 [plan-knowledge-closed-loop-2026-09-24.md](./plan-knowledge-closed-loop-2026-09-24.md) 的评审意见
> **范围**：仅两处「当前代码事实」与代码现状的出入，已核对源码；其余计划内容未评
> **给**：@docs-itsm / @backend-itsm（plan 作者与执行方）

---

## 1. 「向量删除未实现」已过时（plan §2 表格、阶段 5）

**现状**：向量删除链路**已实现且有失败路径测试**：

- `itsm-backend/service/vector_store.go:88` — `func (s *VectorStore) Delete(ctx, tenantID, objectType, objectID) error`
- `itsm-backend/service/vector_delete_test.go` — 覆盖 `Delete` 失败路径（`failingDeleteVectorStore`）

**建议**：阶段 5 从「实现删除」缩为两件真正缺的事：

1. `itsm-rag` Python 侧 `DELETE /vectors/{knowledge_id}` 端点（若后端当前走的不是该通道）；
2. 发布后索引更新的**时延 SLO**（5 分钟）与重试/死信验证——这是本阶段真正的验收点。

§2 表格对应行改为：「向量索引：已有检索与删除（`VectorStore.Delete`）；缺口是发布后索引时延未定 SLO、其 itsm-rag 侧删除端点待核」。

---

## 2. 「权限过滤漏 → 跨租户泄露」风险定级偏低（plan §7 风险表、阶段 6）

**现状**：RAG 可见性过滤是历次审计的重点面（知识可见性/租户/版本过滤须在**检索前与最终响应前**双重把关，见 AGENTS.md「AI-Native Engineering Rules」与 RAG visibility 约定）。把该风险标为「低概率」会弱化阶段 6 的投入力度。

**建议**：

1. 风险表改判「概率：中」，缓解措施落到具体验收；
2. 阶段 6 的 Verification 按 AGENTS.md「每个新增租户资源接口至少包含一个跨租户拒绝测试」补齐，至少覆盖四条硬断言：
   - 跨租户检索：tenant B 的知识即使关键词命中也不得出现在 tenant A 结果中；
   - `visibility=team`：跨团队不得命中；
   - 已归档/旧版本知识：无 `knowledge.read.archived` 权限不得命中；
   - **最终响应前二次过滤**：向量候选集先过权限再进 prompt/返回，而不是只在检索入口过滤。

---

*评审意见 · 2026-09-25 · 只读核对源码后出具，未改动 plan 原文*
