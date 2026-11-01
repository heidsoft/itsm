# AI 中心记忆层设计

> 适用版本：v1.7 起。代码事实源：`itsm-backend/handlers/ai/memory.go`、
> `itsm-backend/service/rag_service.go`（AskMulti / AskWithLLMStreamMulti）。

本文说明 ITSM AI 中心的**记忆层**是什么、按什么分层、每个决策为什么这样定，
以及各层的边界在哪。

---

## 1. 为什么需要记忆层

AI Chat 此前是**完全无状态**的：每轮只拿当前 query 去检索知识库，会话消息虽然
落进了 `messages` 表，却从不读回。

后果不是「回答质量差一点」，而是**结构上不可能答对**：用户在多轮对话里问
「那它怎么修？」「上一个工单呢？」时，当前 query 里没有任何可匹配的实体
（没有工单号、没有 CI 名、没有主题词），关键词检索是整串包含，这种 query
在知识库里必然零命中。

数据其实早就在那里——缺的是「读回来并用上」。

## 2. 分层模型

参照 OpenViking 的记忆分层（session / episodic / semantic / procedural），
ITSM 的对应关系：

| 层级 | OpenViking | ITSM 的实现 | 状态 |
|---|---|---|---|
| 短期 session | 会话上下文 + TTL | 会话历史回读 → 抽实体/短语 → 多路检索补充 | ★ 本次落地 |
| 情景 episodic | 捕获的交互轨迹 | `ai_analysis_results`（分诊/RCA/摘要历史）、`root_cause_analyses` | 已有 |
| 长期 semantic | 向量索引 + 知识库 | RAGService + KnowledgeArticle + 本体图谱 | 已有 |
| 程序性 procedural | Skills / 规则 | `handlers/ai/skills.go` + `builtin_skills.go` + 工具注册表 | 已有 |

**缺口只在短期层**。这也是本次改动的全部范围：短期层不新建表、不引入新
依赖，只把已有数据接进回答链路。

## 3. 三条不可让步的原则

### 3.1 不新建表

记忆的数据本来就在 `messages` 表里。另起一套表只会造出两套互相漂移的
事实源——「会话里记着 A、记忆表里记着 B」这类不一致在 ITSM 的排障场景里
代价很高。

### 3.2 不拼接 query，只做多路检索

关键词检索的语义是整串包含（`TitleContainsFold(query) OR
ContentContainsFold(query)`）。如果把历史拼进当前 query：

```
当前 query：「那它怎么修」
拼接后：    「那它怎么修 TKT-20260831 磁盘告警」
```

拼接后的字符串在知识库里**整串不存在**，本来能命中的当前 query 也一起
命不中——记忆越强，检索越差。

正确做法是**多路检索后合并**：当前 query 一路，历史里抽出的实体/短语各
一路，每路各自走完整的检索语义，一路不命中不影响另一路。合并时按
`object_type:id` 去重。

### 3.3 记忆不得冒充知识

记忆召回的条目会与知识库结果一起进入上下文，因此必须可区分：

- 检索结果条目带 `fromMemory=true` 与 `memoryQuery`（哪一路召回了它）；
- 进入 LLM prompt 时，记忆条目标注「（来源：会话记忆上下文线索，非知识库
  正文）」。

不标的后果是模型把「我们之前聊过」说成「文档里写着」——那是**伪造溯源**，
比没有记忆更危险。

## 4. 记忆抽取规则

从最近若干轮会话消息（默认 6 轮，回读上限 200 字符/条）里抽取补充检索词：

| 优先级 | 抽什么 | 例 |
|---|---|---|
| 1 | 业务实体（工单号/事件号/CI/SLA/变更/问题号） | `TKT-20260831`、`CI-88` |
| 2 | 实义短语（去疑问词与泛用词后的片段，3–24 字） | 「CDN 缓存预热」 |

规则细节：

- **只从 user 消息抽取**。助手的回答里有系统生成措辞（「已为您查询…」），
  那不是用户的检索意图。
- **跳过泛用词**（怎么/如何/处理/解决/请问…）。这些词几乎不出现在知识文章
  标题里，带上只会稀释检索。
- **实体优先**。实体是最强的检索锚点——它是「当前 query 里没有、但历史里
  有」的语义信息。
- **补充词限量**（默认 3 条）。记忆不是越多越好：整段历史塞进检索既慢，又
  会挤掉当前 query 的有效命中。

## 5. 边界与失败处理

| 场景 | 行为 |
|---|---|
| 新会话（`conversationId == 0`） | 不做记忆读取，直接返回空记忆 |
| 历史为空 | 无补充路，行为与引入记忆前完全一致 |
| 记忆读取失败 | **不阻断回答**：记 Warn 日志，按无记忆处理 |
| 补充路检索失败 | 只记录不抛出，主路结果照常返回 |
| 记忆跨租户/跨用户 | 仓储层带租户谓词 + 会话属主校验，读不到即无记忆 |
| 记忆条目在 prompt 中 | 显式标注来源，不与知识库正文混为一谈 |

记忆是**增强**，不是**依赖**。任何一步失败都必须降级到「无记忆」，而不是
让整个问答失败。

## 6. 相关但独立的三件事

避免把它们混为一谈：

- **可追溯性**（auditId / promptVersion / model）：回答的溯源底账，
  与记忆无关，但两者都要「如实」——记忆管上下文来源，溯源管决策来源。
- **能力矩阵的 LLM 健康维度**：`ai` 能力的 health 反映「LLM 配没配」，
  与记忆无关。
- **评测语料验收**：`cmd/ai-eval` 的召回率验收针对知识库检索质量，不测
  记忆层（记忆层的验证在 `handlers/ai/memory_test.go` 与
  `service/rag_multi_test.go`）。

## 7. 验证方式

| 层次 | 测试 | 锁住什么 |
|---|---|---|
| 抽取 | `TestMemoryExtractsEntityFromHistory` 等 9 条 | 实体抽取、跳过助手消息、无历史不编造、限量、跨租户隔离、来源标注 |
| 隔离 | `TestMemoryIsolatedFromRealRepository` | 真实 SQL 下跨租户读不到（mock 测不出谓词写错） |
| 价值 | `TestAskMulti_ReferentialQueryHitsViaSupplemental` | 指代型 query 从零命中 → 有记忆时命中 |
| 去重 | `TestAskMulti_DeduplicatesAcrossRoutes` | 同一文档不跨路重复 |
| 降级 | `TestAskMulti_SupplementalFailureDoesNotBreakPrimary` | 补充路失败不影响主路 |
| 越权 | `handlers/ai/conversation_tenant_guard_test.go` | 会话读面按租户 + 属主双校验 |
