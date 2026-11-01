package ai

import (
	"context"
	"regexp"
	"strings"
)

// AI 中心的**记忆层**：让多轮对话里的上下文真正参与回答。
//
// ## 为什么需要它
//
// 此前 AI Chat 是完全无状态的：每轮只拿当前 query 去检索，会话消息虽然
// 落进了 messages 表，却从不读回。用户问「那它怎么修？」「上一个工单呢？」
// 这类指代时，检索层拿到的 query 里没有任何可匹配的实体，检索必然落空
// ——不是「答得不好」，是**结构上不可能答对**。
//
// ## 分层对照（参照 OpenViking 的记忆分层）
//
//	层级            OpenViking            ITSM 现状                      本次落地
//	短期 session    会话上下文 + TTL      表在、历史从不读回            ★ 会话记忆回读 + 指代消解
//	情景 episodic   捕获的交互轨迹        ai_analysis_results / RCA      已有（历史处置经验）
//	长期 semantic   向量索引 + 知识库     RAG + KnowledgeArticle        已有
//	程序性 procedural Skills              handlers/ai/skills.go          已有
//
// 缺口只在短期层：数据早就有了，缺的是「读回来并用上」。
//
// ## 三条不可让步的原则
//
// 1. **不新建表**。记忆的数据本来就在 messages 表里，另起一套表只会造出
//    两套互相漂移的事实源。
// 2. **不拼接 query**。关键词检索是整串包含（TitleContainsFold OR
//    ContentContainsFold），把历史拼进当前 query 会让原本能命中的 query
//    命不中。做法是**多路检索后合并**：当前 query 一路，历史里抽出的
//    实体/关键词各一路，各走完整的检索语义。
// 3. **记忆不得冒充知识**。记忆召回的条目会与知识库检索结果一起返回，
//    但必须标注来源（fromMemory），让调用方能区分「库里写着」与
//    「我们之前聊过」。

// memoryContext 是一次记忆召回的结果。
//
// MemoryTurns 是回读的最近若干轮会话消息（按时间正序），
// MemoryQueries 是从中抽出的、值得单独检索的补充 query。
type memoryContext struct {
	// Turns 是回读的历史消息（原样，不做加工）。
	Turns []*Message
	// Queries 是从历史里抽出的补充检索 query（去重、限量）。
	Queries []string
	// Used 为 true 表示本轮确实用到了记忆（历史存在且抽出了补充 query）。
	Used bool
}

// maxMemoryTurns 是回读的历史轮次上限。
//
// 记忆不是越多越好：整段历史塞进检索既慢又会挤掉当前 query 的命中。
// 取最近的若干轮足够覆盖「上一个话题」这类指代。
const maxMemoryTurns = 6

// maxMemoryQueries 是从历史里抽出的补充 query 上限。
const maxMemoryQueries = 3

// maxMemoryTurnLength 是单条历史消息参与抽词的长度上限（防止超长消息
// 把整篇文章塞进检索）。
const maxMemoryTurnLength = 200

// entityPattern 识别历史里出现的业务实体：工单号、事件号、CI 名、SLA 名等。
//
// 这些实体是「当前 query 里没有、但历史里有」的语义锚点——用户问「它怎么
// 修」时，历史里的工单号就是最该拿去检索的东西。
var entityPattern = regexp.MustCompile(`(?i)(INC|TKT|CHG|PRB|SR|CI|SLA)[-_]?\d{2,}`)

// stopwords 是抽词时跳过的通用词：它们几乎不出现在知识文章标题里，
// 带上只会稀释检索。
//
// 按长度降序排列：截断前缀时要先去掉长的（"为什么"）再去短的（"什么"），
// 否则先去掉"什么"会留下"为什"这种更糟的残片。map 迭代无序，所以这里
// 用有序切片。
var stopwords = []string{
	// 尾部疑问短语必须排在更短的片段之前：先去掉「怎么处理」再去掉「怎么」，
	// 否则「磁盘空间不足怎么处理」会先被削成「磁盘空间不足怎么办」，
	// 再削成「磁盘空间不足办」——留下「办」这种碎片，比不去更糟。
	"怎么办", "怎么处理", "如何处理", "怎么样", "是什么", "为什么",
	"请问", "谢谢", "你好", "帮忙", "他们", "我们", "你们",
	"如何", "怎么", "什么", "哪些", "哪里", "这个", "那个",
	"处理", "解决", "问题", "情况", "相关", "一下", "还有", "然后",
	"现在", "之前", "上面", "刚才", "可以", "需要", "它",
}

// recallConversationMemory 回读会话历史并抽出可用于补充检索的 query。
//
// 边界：
//   - 只读**当前用户自己**的会话（GetMessages 带租户谓词，且调用方已校验
//     属主），记忆不得跨租户、跨用户；
//   - convID == 0（新会话）时直接返回空记忆，不做无意义的读取；
//   - 历史读失败**不阻断回答**——记忆是增强，不是前置依赖，读不到就按
//     无记忆处理，把错误如实记日志。
func (s *Service) recallConversationMemory(
	ctx context.Context, tenantID, userID, convID int,
) *memoryContext {
	if convID == 0 || s.repo == nil {
		return &memoryContext{}
	}

	msgs, err := s.repo.GetMessages(ctx, convID, tenantID)
	if err != nil {
		// 记忆读不到不是致命错误：回答照常出，只是没有上下文增强。
		s.logger.Warnw("AI memory: 读会话历史失败，按无记忆处理",
			"conversation_id", convID, "tenant_id", tenantID, "error", err)
		return &memoryContext{}
	}

	turns := recentTurns(msgs, maxMemoryTurns)
	if len(turns) == 0 {
		return &memoryContext{}
	}

	queries := extractMemoryQueries(turns, maxMemoryQueries)
	return &memoryContext{Turns: turns, Queries: queries, Used: len(queries) > 0}
}

// recentTurns 取最近 N 条消息（按时间正序返回，便于按序阅读）。
//
// 消息表里 user/assistant 交替，这里不做配对——记忆只需要「最近聊过什么」，
// 不需要精确的轮次结构。
func recentTurns(msgs []*Message, limit int) []*Message {
	if limit <= 0 || len(msgs) == 0 {
		return nil
	}
	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	out := make([]*Message, 0, len(msgs))
	for _, m := range msgs {
		if m == nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

// extractMemoryQueries 从历史轮次里抽出值得单独检索的 query。
//
// 优先级：
//  1. 业务实体（工单号、CI 名等）——它们是最强的检索锚点；
//  2. 用户消息里的实义短语（去掉疑问词与泛用词后剩下的长片段）。
//
// 抽出来的是**独立的检索词**，不是拼接后的长句——每一条都单独走一遍
// 完整的检索语义（整串包含），这样一条不命中不会拖垮另一条。
func extractMemoryQueries(turns []*Message, limit int) []string {
	if limit <= 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, limit)

	add := func(q string) {
		q = strings.TrimSpace(q)
		if q == "" || len([]rune(q)) < 2 {
			return
		}
		key := strings.ToLower(q)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, q)
	}

	// 实体优先（从最新的历史往前找）。
	for i := len(turns) - 1; i >= 0 && len(out) < limit; i-- {
		if turns[i] == nil || turns[i].Role != "user" {
			continue
		}
		content := truncateRunes(turns[i].Content, maxMemoryTurnLength)
		for _, entity := range entityPattern.FindAllString(content, -1) {
			if len(out) >= limit {
				break
			}
			add(entity)
		}
	}

	// 再补实义短语。
	for i := len(turns) - 1; i >= 0 && len(out) < limit; i-- {
		if turns[i] == nil || turns[i].Role != "user" {
			continue
		}
		content := truncateRunes(turns[i].Content, maxMemoryTurnLength)
		for _, phrase := range salientPhrases(content) {
			if len(out) >= limit {
				break
			}
			add(phrase)
		}
	}

	return out
}

// salientPhrases 从一条消息里抽出实义短语。
//
// 做法是按标点切句，逐句去掉句首的疑问/客套前缀与末尾的疑问词，保留
// 剩下的主体部分。太短的片段丢弃——两个字的中文短语大多是泛用词。
func salientPhrases(content string) []string {
	if content == "" {
		return nil
	}
	parts := strings.FieldsFunc(content, func(r rune) bool {
		switch r {
		case '，', ',', '。', '.', '！', '!', '？', '?', ';', '；', ':', '：',
			'\n', '\r', '\t', '、':
			return true
		}
		return false
	})

	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		for _, w := range stopwords {
			p = strings.TrimPrefix(p, w)
		}
		// 前缀去掉后再去一次尾部疑问词（"它怎么修" → 去前缀后剩"怎么修"）。
		for _, w := range stopwords {
			p = strings.TrimSuffix(p, w)
		}
		p = strings.TrimSpace(p)
		runes := []rune(p)
		if len(runes) < 3 {
			continue
		}
		// 太长的片段多半是整句，检索里整串包含几乎必然落空；截断到
		// 一个可命中的长度上限。
		if len(runes) > 24 {
			runes = runes[:24]
		}
		out = append(out, string(runes))
	}
	return out
}

// truncateRunes 按 rune 截断（不切碎多字节字符）。
func truncateRunes(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit])
}

// mergeMemoryResults 把记忆召回的结果并入知识库检索结果。
//
// 合并规则：
//   - 知识库结果在前（它们是「库里写着」的权威来源）；
//   - 记忆结果按「object_type:id」去重，不覆盖已有条目；
//   - 记忆条目打上 fromMemory=true 与 memoryQuery 标注，让调用方能区分
//     两类来源——记忆是「我们之前聊过」，不是「库里写着」。
func mergeMemoryResults(primary, memory []map[string]any, memoryQuery string) []map[string]any {
	if len(memory) == 0 {
		return primary
	}
	seen := make(map[string]struct{}, len(primary))
	for _, item := range primary {
		seen[resultKey(item)] = struct{}{}
	}
	out := primary
	for _, item := range memory {
		key := resultKey(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		marked := make(map[string]any, len(item)+2)
		for k, v := range item {
			marked[k] = v
		}
		marked["fromMemory"] = true
		if memoryQuery != "" {
			marked["memoryQuery"] = memoryQuery
		}
		out = append(out, marked)
	}
	return out
}

// resultKey 是检索结果的身份键（跨来源去重用）。
func resultKey(item map[string]any) string {
	typ, _ := item["object_type"].(string)
	id, _ := item["id"].(int)
	if id == 0 {
		// 没有 id 的条目用标题兜底，避免两条不同来源的同题条目互相覆盖。
		title, _ := item["title"].(string)
		return typ + ":" + title
	}
	return typ + ":" + itoa(id)
}

// itoa 是小整数转字符串（避免为此引入 strconv 的完整依赖）。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
