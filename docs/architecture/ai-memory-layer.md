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
| 情景 episodic | 捕获的交互轨迹 | 历史工单处置经验（`tickets.resolution` + `root_cause_analyses`） | ★ 本次落地 |
| 长期 semantic | 向量索引 + 知识库 | RAGService + KnowledgeArticle + 本体图谱 | 已有 |
| 程序性 procedural | Skills / 规则 | `handlers/ai/skills.go` + `builtin_skills.go` + 工具注册表 | 已有 |

短期层与情景层是本次落地范围：都不新建表、不引入新依赖，只把已有数据接进
回答链路。

- **短期层**：会话历史（`messages` 表）→ 抽实体/短语 → 多路检索补充；
- **情景层**：历史工单的处置结论（`tickets.resolution`）→ 按标题关键词匹配 →
  作为独立来源的上下文块进 prompt。

两层的分工：短期层回答「我们刚才说了什么」，情景层回答「这个租户以前怎么
处理过同类问题」。

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

## 4. 情景层：历史工单处置经验

「上次这个故障是怎么解决的？」是 ITSM 里最高频的问题之一，但系统此前答不了
——RAG 的关键词检索**只覆盖知识文章**（`keywordSearch` 查的是
`knowledge_articles`），历史工单的处置结论（`tickets.resolution`）与根因分析
（`root_cause_analyses.analysis_summary`）都躺在库里，从未被召回。

### 4.1 过滤链（顺序即安全边界）

| 顺序 | 过滤 | 为什么 |
|---|---|---|
| 1 | 租户 | 跨租户绝对不可见 |
| 2 | 未软删 | 软删工单不应作为先例 |
| 3 | 已解决/已关闭 **且** `resolution` 非空 | 未解决的工单没有「经验」可言；把进行中的描述当经验会误导模型 |
| 4 | `datascope.ApplyTicketFilter` 行级范围 | **工单是业务数据**：把用户无权查看的工单处置喂给 LLM 等于绕过权限外泄内容 |
| 5 | 标题关键词匹配 | 相关性 |

第 4 条是情景层与短期层最大的区别：短期层读的是自己的会话，情景层读的是
**业务数据**，必须经行级范围过滤。权限不可得时**收窄**是唯一安全的方向——
`datascope.ForTicketReader` 对空角色返回最保守档（本人创建或分配给本人），
不会放宽到全租户。

### 4.2 关键词为什么是候选列表

工单标题常是**多词短语**（「CDN 回源超时」），用户问的却是「CDN 回源超时
怎么办」：

- 只取最长片段 → 削掉限定词（「CDN」被切掉），标题里没有「怎么办」，命不中；
- 只取单词 → 「CDN」能命中但过于宽泛，把不相关的工单拉进来。

所以按空格切词后生成相邻词组合（3 词、2 词优先，单词兜底），**逐个独立
检索、命中即止**——把候选 OR 在一起会让泛用词稀释结果。命中够数后不再放宽
关键词：更宽的匹配只会引入噪声。

### 4.3 来源标注

情景召回的条目形态与知识库条目一致（便于前端统一渲染引用），但：

- `object_type=ticket_experience`（知识库正文是 `kb`）；
- `fromMemory=true`、`memorySource=ticket_experience`、带 `ticket_number`；
- 进 LLM prompt 时标注「（来源：历史工单处置经验，非知识库正文；可信度低于
  知识库文档）」，并要求模型引用时注明工单号。

不标的话模型会把历史工单结论说成「知识库文档里写了」——那是伪造溯源。历史
结论是同类问题的**先例**，不是标准流程。

### 4.4 边界

| 场景 | 行为 |
|:---|:---|
| 没有已解决工单 / 无相关标题 | 返回空，不报错（这是正常结果） |
| 用户无权查看该工单 | 行级过滤挡住，表现为「没有历史经验」 |
| 召回查询失败 | 记 Warn 日志，返回已召回的部分；不阻断回答 |
| `entClient` 未装配 | 返回空（无数据源即无经验，不猜） |

## 5. 渐进式披露：摘要层与全文下钻

参照 OpenViking 的「Read the summary before the source」（L0 摘要 / L1 概览 /
L2 全文），知识检索分两层：

| 层 | 内容 | 获取方式 |
|---|---|---|
| 摘要层 | `snippet`（正文截断到 160 字符） | 检索命中即返回 |
| 全文层 | `content`（完整正文） | AI 工具 `get_kb_article` 按需下钻 |

### 5.1 为什么此前下钻不了

检索命中后模型拿到的是截断片段，遇到需要细节的问题（具体步骤、参数、例外
条件）就答不了——而它**没有任何办法下钻**：工具清单里只有 `list_kb`（检索），
没有「读全文」。这正是 OpenViking 说的 evidence gap 无法闭合。

### 5.2 信号位与下钻路径

- 检索结果带 `truncated`（是否截断）与 `contentLength`（正文长度）——没有
  这两个字段，模型无从判断片段是不是完整的，会拿着半截内容当全文硬答；
- 截断条目在 prompt 里标注「（以下内容为节选，末尾可能有省略）」；
- **有工具路径**（provider 支持工具调用）prompt 引导「调用 get_kb_article 按
  文档 id 读取完整正文」；
- **无工具路径不能承诺「可以调工具」**——那时根本没有工具可用。空承诺比不
  提示更糟：模型会说自己要调用工具，然后**编造工具的返回内容**。该路径如实
  说明「只有节选」，让模型在片段范围内作答并声明信息不完整。

### 5.3 读与检索走同一套准入

`RAGService.GetArticleFull` 与 `keywordSearch` 共用同一套过滤（租户/未软删/
已发布/时效/分类可见性含作者豁免）。**读与检索口径必须一致**，否则会出现：
「检索命中了但工具读不到」（模型困惑、反复重试）或「工具能读到但检索永远
命中不了」（那是一条绕过权限的读通道）。工具层不重复实现准入——两处各自
实现必然漂移。

读不到时返回错误而非空内容：空内容会让模型把「读不到」当成「这篇文章没有
正文」，然后基于空输入编造答案。

## 6. 短期层：记忆抽取规则

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

## 7. 边界与失败处理

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

## 8. 相关但独立的三件事

避免把它们混为一谈：

- **可追溯性**（auditId / promptVersion / model）：回答的溯源底账，
  与记忆无关，但两者都要「如实」——记忆管上下文来源，溯源管决策来源。
- **能力矩阵的 LLM 健康维度**：`ai` 能力的 health 反映「LLM 配没配」，
  与记忆无关。
- **评测语料验收**：`cmd/ai-eval` 的召回率验收针对知识库检索质量，不测
  记忆层（记忆层的验证在 `handlers/ai/memory_test.go` 与
  `service/rag_multi_test.go`）。

## 9. 验证方式

| 层次 | 测试 | 锁住什么 |
|---|---|---|
| 抽取 | `TestMemoryExtractsEntityFromHistory` 等 9 条 | 实体抽取、跳过助手消息、无历史不编造、限量、跨租户隔离、来源标注 |
| 隔离 | `TestMemoryIsolatedFromRealRepository` | 真实 SQL 下跨租户读不到（mock 测不出谓词写错） |
| 价值 | `TestAskMulti_ReferentialQueryHitsViaSupplemental` | 指代型 query 从零命中 → 有记忆时命中 |
| 去重 | `TestAskMulti_DeduplicatesAcrossRoutes` | 同一文档不跨路重复 |
| 降级 | `TestAskMulti_SupplementalFailureDoesNotBreakPrimary` | 补充路失败不影响主路 |
| 越权 | `handlers/ai/conversation_tenant_guard_test.go` | 会话读面按租户 + 属主双校验 |
| 情景权限 | `TestEpisodeRecallRespectsDataScope` | agent 不得召回他人工单经验；管理角色可以 |
| 情景门槛 | `TestEpisodeRecallOnlyResolvedWithConclusion` | 只有已解决且有处置结论的工单才是经验 |
| 情景隔离 | `TestEpisodeRecallRejectsCrossTenant` / `TestEpisodeRecallSkipsSoftDeleted` | 跨租户与软删工单不进召回 |
| 情景标注 | `TestEpisodeResultsCarrySource` / `TestEpisodePromptBlockMarksSource` | 来源标注与工单号不得丢 |
| 披露一致性 | `TestGetArticleFull_MatchesRetrievalVisibility` | 检索能看到的，全文也能读到（同一篇文章两条路） |
| 披露准入 | `TestGetArticleFull_RejectsUnpublishedAndDeleted` / `TestGetArticleFull_RejectsCrossTenant` | 草稿/软删/跨租户不可读 |
| 披露可见性 | `TestGetArticleFullMatchesKnowledgeGuard` | 受限分类在无权时：检索与全文**都**读不到 |
| 信号位 | `TestSearchResultsCarryTruncationSignal` | 长文必须带 truncated=true 与 contentLength |
| 工具端到端 | `TestGetKbArticleToolExecutesThroughService` | 完整链路（RBAC→审计→执行）可拿到全文 |
