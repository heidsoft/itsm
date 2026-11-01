package ai

import (
	"context"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/internal/datascope"

	"go.uber.org/zap"
)

// AI 中心的**情景记忆层**：召回历史工单的处置经验。
//
// ## 为什么需要它
//
// 「上次这个故障是怎么解决的？」是 ITSM 里最高频的问题之一，但系统此前
// 答不了——RAG 的关键词检索**只覆盖知识文章**（keywordSearch 查的是
// knowledge_articles），历史工单的处置结论（tickets.resolution）与根因
// 分析（root_cause_analyses.analysis_summary）都躺在库里，从未被召回。
//
// 短期记忆层（memory.go）解决的是「本会话内上文说了什么」，情景层解决的是
// 「这个租户以前怎么处理过同类问题」——两者数据源与语义都不同。
//
// ## 三条硬边界
//
// 1. **必须经 DataScope 行级过滤**。工单是业务数据，有「本人/本部门/全租户」
//    的可见范围；把用户无权查看的工单处置喂给 LLM，等于绕过权限外泄内容。
//    场景记忆是**增强**，权限不可得时必须返回空，而不是放宽到全量。
// 2. **只取已解决/已关闭且有处置结论的工单**。未解决的工单没有「经验」可言，
//    把进行中的描述当经验会误导模型。
// 3. **不得冒充知识**。情景召回的条目 object_type=ticket_experience，与知识库
//    正文（object_type=kb）区分；进 prompt 时显式标注来源。

// episodeCandidate 是一条历史工单处置经验。
type episodeCandidate struct {
	TicketID           int
	TicketNumber       string
	Title              string
	Resolution         string
	ResolutionCategory string
	ClosedAt           time.Time
}

// maxEpisodeResults 是情景召回的最大条数。
//
// 经验条目进 prompt 是有成本的（token + 噪声），且低相关的历史工单会把模型
// 带偏。少量高相关胜过多条泛相关。
const maxEpisodeResults = 3

// maxEpisodeQueryLength 是参与匹配的 query 长度上限。
const maxEpisodeQueryLength = 120

// recallEpisodes 召回与当前 query 相关的历史工单处置经验。
//
// 过滤链（顺序即安全边界）：
//  1. 租户（跨租户绝对不可见）；
//  2. 未软删；
//  3. 已解决/已关闭且 resolution 非空（没有处置结论就不是经验）；
//  4. datascope.ApplyTicketFilter 行级范围（用户无权查看的工单不进召回）；
//  5. 标题关键词匹配（相关性）。
//
// 返回空切片表示「无可用经验」——这是正常结果，不报错。
func (s *Service) recallEpisodes(
	ctx context.Context, tenantID, userID int, role, query string,
) []episodeCandidate {
	if s.entClient == nil || tenantID <= 0 || userID <= 0 {
		return nil
	}
	keywords := episodeKeywords(query)
	if len(keywords) == 0 {
		return nil
	}

	// 行级范围：按角色词表解析（agent/end_user 只见本人或分配给本人的）。
	scope := datascope.ForTicketReader(role)

	q := s.entClient.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.DeletedAtIsNil(),
			ticket.StatusIn(common.TicketStatusResolved, common.TicketStatusClosed),
			ticket.ResolutionNEQ(""),
		)
	q = datascope.ApplyTicketFilter(q, userID, 0, scope)

	// 相关性：标题包含关键词。工单标题是问题的一句话概括，与用户问法的
	// 词汇重合度最高；description 太长且噪声大，不参与匹配（但可作为正文）。
	//
	// 候选按「具体→宽泛」依次尝试（多词组合优先）：工单标题常是
	// 「CDN 回源超时」这样的多词短语，用户问的却是「CDN 回源超时怎么办」，
	// 只取最长片段会丢掉限定词（把 "CDN" 切掉），单词又太泛。逐个独立
	// 检索、命中即止，比把候选 OR 在一起更可控（OR 会让泛用词把结果稀释）。
	seen := map[int]struct{}{}
	out := make([]episodeCandidate, 0, maxEpisodeResults)
	for _, keyword := range keywords {
		if len(out) >= maxEpisodeResults {
			break
		}
		rows, err := q.Clone().Where(ticket.TitleContainsFold(keyword)).
			Order(ent.Desc(ticket.FieldClosedAt)).
			Limit(maxEpisodeResults).
			All(ctx)
		if err != nil {
			// 情景召回失败不影响主链路：知识库检索照常。
			s.logger.Warnw("AI memory: 召回历史工单经验失败，按无经验处理",
				"tenant_id", tenantID, "user_id", userID, "keyword", keyword, "error", err)
			return out
		}
		for _, t := range rows {
			if len(out) >= maxEpisodeResults {
				break
			}
			if _, dup := seen[t.ID]; dup {
				continue
			}
			// SQL 的 != '' 挡不住全空白：没有实质处置结论的工单不是「经验」。
			if strings.TrimSpace(t.Resolution) == "" {
				continue
			}
			seen[t.ID] = struct{}{}
			c := episodeCandidate{
				TicketID:           t.ID,
				TicketNumber:       t.TicketNumber,
				Title:              t.Title,
				Resolution:         t.Resolution,
				ResolutionCategory: t.ResolutionCategory,
			}
			if t.ClosedAt != nil {
				c.ClosedAt = *t.ClosedAt
			}
			out = append(out, c)
		}
		// 已经召回够数就不再放宽关键词——更宽的匹配只会引入噪声。
		if len(out) > 0 {
			break
		}
	}
	return out
}

// episodeKeywords 从 query 里提取用于匹配工单标题的候选关键词，按「具体→宽泛」
// 排序。
//
// 为什么是候选列表而不是单个关键词：工单标题常是**多词短语**（「CDN 回源
// 超时」），用户问的却是「CDN 回源超时怎么办」——
//   - 只取最长片段：会把限定词切掉（「CDN 回源超时怎么办」→ 只剩「回源超时
//     怎么办」，标题里没有「怎么办」，命不中）；
//   - 只取单词：「CDN」能命中但过于宽泛，把不相关的工单也拉进来。
//
// 所以按空格切词后生成相邻词组合（2 词、3 词优先），逐个独立检索，命中即止——
// 把候选 OR 在一起会让泛用词稀释结果。
func episodeKeywords(query string) []string {
	q := truncateRunes(strings.TrimSpace(query), maxEpisodeQueryLength)
	if q == "" {
		return nil
	}
	// 按标点与空白切分，去掉疑问/客套词。
	parts := strings.FieldsFunc(q, func(r rune) bool {
		switch r {
		case '，', ',', '。', '.', '！', '!', '？', '?', ';', '；', ':', '：',
			'\n', '\r', '\t', '、', ' ':
			return true
		}
		return false
	})
	words := make([]string, 0, len(parts))
	for _, p := range parts {
		for _, w := range stopwords {
			p = strings.TrimPrefix(p, w)
			p = strings.TrimSuffix(p, w)
		}
		p = strings.TrimSpace(p)
		if len([]rune(p)) < 2 {
			continue
		}
		words = append(words, p)
	}
	if len(words) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(words)*2)
	add := func(s string) {
		s = strings.TrimSpace(s)
		if len([]rune(s)) < 3 {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	// 长组合优先（更具体），再退到单词（更宽泛）。
	for size := 3; size >= 2; size-- {
		for i := 0; i+size <= len(words); i++ {
			add(strings.Join(words[i:i+size], " "))
		}
	}
	for _, w := range words {
		add(w)
	}
	return out
}

// episodesToResults 把情景经验转成检索结果条目。
//
// 形态与知识库条目一致（object_type/id/title/snippet/score），以便前端统一
// 渲染引用；但 object_type 不同，且带 fromMemory/memorySource 标注——调用方
// 必须能区分「知识库正文」与「历史工单经验」。
func episodesToResults(episodes []episodeCandidate) []map[string]any {
	if len(episodes) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(episodes))
	for _, e := range episodes {
		snippet := e.Resolution
		if snippet == "" {
			snippet = e.Title
		}
		out = append(out, map[string]any{
			"object_type":      "ticket_experience",
			"id":               e.TicketID,
			"title":            e.Title,
			"snippet":          snippet,
			"category":         e.ResolutionCategory,
			"score":            0.0,
			"ticket_number":    e.TicketNumber,
			"fromMemory":       true,
			"memorySource":     "ticket_experience",
			"memoryResolution": e.Resolution,
		})
	}
	return out
}

// episodePromptBlock 生成情景经验的 prompt 片段。
//
// 显式标注来源是硬要求：模型必须知道这是「以前处理过类似问题的工单结论」，
// 而不是「知识库文档」。把两者混淆会让模型编造出「文档里写了」的说法。
func episodePromptBlock(episodes []episodeCandidate) string {
	if len(episodes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("以下是本租户历史上处理过类似问题的已结工单结论（来源：历史工单处置经验，非知识库文档正文）：\n\n")
	for i, e := range episodes {
		b.WriteString("【历史工单")
		b.WriteString(itoa(i + 1))
		b.WriteString("】")
		b.WriteString(e.TicketNumber)
		b.WriteString(" ")
		b.WriteString(e.Title)
		b.WriteString("\n处置结论：")
		b.WriteString(e.Resolution)
		b.WriteString("\n\n")
	}
	b.WriteString("如引用这些历史结论，请注明工单号；它们是同类问题的先例，可信度低于知识库正文，不得当作标准流程。\n\n")
	return b.String()
}

// episodeLogger 供调用方在装配时记录情景召回情况（可选依赖，便于测试替身）。
type episodeLogger interface {
	Debugw(msg string, keysAndValues ...interface{})
}

var _ episodeLogger = (*zap.SugaredLogger)(nil)
