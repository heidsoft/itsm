package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"itsm-backend/ent"
	changePred "itsm-backend/ent/change"
	"itsm-backend/ent/cirelationship"
	ciPred "itsm-backend/ent/configurationitem"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/incidentalert"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/release"
	"itsm-backend/ent/ticket"
)

// OntologyService 基于 ITSM 领域本体（实体-关系-语义）的图检索服务。
// 它从用户 query 中识别业务实体（工单/事件/问题/变更/发布/CI），
// 沿本体的类型化关系边做 1 跳扩展，产出供 LLM 推理的结构化上下文。
//
// 设计约束（见 output/ontology-design-2026-08-29.md）：
//   - 只读：本服务不写任何实体
//   - 租户隔离：所有查询强制 TenantIDEQ，fail-closed
//   - 限流：每实体邻居 ≤ maxNeighbors（默认 20）
type OntologyService struct {
	client       *ent.Client
	logger       *zap.SugaredLogger
	maxNeighbors int
}

func NewOntologyService(client *ent.Client, logger *zap.SugaredLogger) *OntologyService {
	return &OntologyService{
		client:       client,
		logger:       logger,
		maxNeighbors: 20,
	}
}

// ---- 实体编号识别（不依赖 LLM 的轻量正则） ----

var (
	ticketNumberRe   = regexp.MustCompile(`TKT-\d{4,6}-\d+`)
	incidentNumberRe = regexp.MustCompile(`INC-\d{4,6}-\d+`)
	releaseNumberRe  = regexp.MustCompile(`REL-\d{8}-\w+`)
	problemNumberRe  = regexp.MustCompile(`PRB-\d{8}-\d+`)
	changeNumberRe   = regexp.MustCompile(`CHG-\d{8}-\d+`)
)

// OntologyEntity 识别并扩展出的一个业务实体卡
type OntologyEntity struct {
	ObjectType string // ticket / incident / problem / change / release / ci
	ID         int
	Number     string // 工单号/事件号/发布号；CI 与无编号实体为空
	Title      string
	Status     string
	Snippet    string            // 实体摘要（邻居关系的人话描述）
	Neighbors  []string          // 关系邻居清单（"TKT-xxx: 标题 [关系: related_ticket]"）
	Extra      map[string]string // 附加事实（如 CI 的 criticality/environment）
}

// OntologyContext 一次 ExtractAndExpand 的全部产物
type OntologyContext struct {
	Entities []*OntologyEntity
}

// Sources 把实体卡转换为 RAG sources 格式（前端 SourceList 天然兼容多 objectType）
func (oc *OntologyContext) Sources() []map[string]any {
	if oc == nil || len(oc.Entities) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(oc.Entities))
	for _, e := range oc.Entities {
		s := map[string]any{
			"objectType": e.ObjectType,
			"id":         e.ID,
			"title":      e.displayTitle(),
			"snippet":    e.Snippet,
			"score":      1.0, // 图谱实体是确定性命中，置满置信
			"searchType": "ontology",
		}
		if e.Number != "" {
			s["number"] = e.Number
		}
		out = append(out, s)
	}
	return out
}

// PromptBlock 把实体卡序列化为注入 LLM prompt 的结构化文本块
func (oc *OntologyContext) PromptBlock() string {
	if oc == nil || len(oc.Entities) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n【业务实体图谱上下文】以下是从 CMDB/工单系统中识别到的实体及其关系（本体扩展 1 跳），回答时优先基于这些图谱事实：\n")
	for _, e := range oc.Entities {
		b.WriteString(fmt.Sprintf("● %s（%s，状态: %s）\n", e.displayTitle(), e.ObjectType, e.Status))
		if e.Snippet != "" {
			b.WriteString("  摘要：" + e.Snippet + "\n")
		}
		for _, n := range e.Neighbors {
			b.WriteString("  关联 - " + n + "\n")
		}
	}
	return b.String()
}

func (e *OntologyEntity) displayTitle() string {
	if e.Number != "" {
		return fmt.Sprintf("%s %s", e.Number, e.Title)
	}
	return fmt.Sprintf("#%d %s", e.ID, e.Title)
}

// Empty 是否为空上下文
func (oc *OntologyContext) Empty() bool {
	return oc == nil || len(oc.Entities) == 0
}

// ExtractAndExpand 识别 query 中的实体并做 1 跳图扩展。
// 任一单实体查询失败只记日志不中断整体（降级为不含该实体）。
func (s *OntologyService) ExtractAndExpand(ctx context.Context, tenantID int, query string) *OntologyContext {
	oc := &OntologyContext{Entities: make([]*OntologyEntity, 0, 4)}

	// 1. 编号实体：TKT- / INC- / REL-
	for _, num := range dedupeStringsLocal(matchAll(ticketNumberRe, query)) {
		t, err := s.client.Ticket.Query().
			Where(
				ticket.TenantIDEQ(tenantID),
				ticket.TicketNumberEQ(num),
				ticket.DeletedAtIsNil(),
			).
			Only(ctx)
		if err != nil {
			if !ent.IsNotFound(err) {
				s.logger.Warnw("ontology: ticket lookup failed", "number", num, "error", err)
			}
			continue
		}
		oc.Entities = append(oc.Entities, s.expandTicket(ctx, tenantID, t))
	}
	for _, num := range dedupeStringsLocal(matchAll(incidentNumberRe, query)) {
		in, err := s.client.Incident.Query().
			Where(
				incident.TenantIDEQ(tenantID),
				incident.IncidentNumberEQ(num),
				incident.DeletedAtIsNil(),
			).
			Only(ctx)
		if err != nil {
			if !ent.IsNotFound(err) {
				s.logger.Warnw("ontology: incident lookup failed", "number", num, "error", err)
			}
			continue
		}
		oc.Entities = append(oc.Entities, s.expandIncident(ctx, tenantID, in))
	}
	for _, num := range dedupeStringsLocal(matchAll(releaseNumberRe, query)) {
		rel, err := s.client.Release.Query().
			Where(
				release.TenantIDEQ(tenantID),
				release.ReleaseNumberEQ(num),
			).
			Only(ctx)
		if err != nil {
			if !ent.IsNotFound(err) {
				s.logger.Warnw("ontology: release lookup failed", "number", num, "error", err)
			}
			continue
		}
		oc.Entities = append(oc.Entities, s.expandRelease(ctx, tenantID, rel))
	}
	for _, num := range dedupeStringsLocal(matchAll(problemNumberRe, query)) {
		p, err := s.client.Problem.Query().
			Where(
				problem.TenantIDEQ(tenantID),
				problem.ProblemNumberEQ(num),
				problem.DeletedAtIsNil(),
			).
			Only(ctx)
		if err != nil {
			if !ent.IsNotFound(err) {
				s.logger.Warnw("ontology: problem lookup failed", "number", num, "error", err)
			}
			continue
		}
		oc.Entities = append(oc.Entities, s.expandProblem(ctx, tenantID, p))
	}
	for _, num := range dedupeStringsLocal(matchAll(changeNumberRe, query)) {
		c, err := s.client.Change.Query().
			Where(
				changePred.TenantIDEQ(tenantID),
				changePred.ChangeNumberEQ(num),
			).
			Only(ctx)
		if err != nil {
			if !ent.IsNotFound(err) {
				s.logger.Warnw("ontology: change lookup failed", "number", num, "error", err)
			}
			continue
		}
		oc.Entities = append(oc.Entities, s.expandChange(ctx, tenantID, c))
	}

	// 2. CI 实体：抽取 query 中的引号词与中文/字母词组，做 name 精确→前缀匹配
	ciNames := extractCandidateCINames(query)
	for _, name := range ciNames {
		ci, err := s.lookupCI(ctx, tenantID, name)
		if err != nil {
			continue
		}
		oc.Entities = append(oc.Entities, s.expandCI(ctx, tenantID, ci))
	}

	// 3. Problem 标题模糊匹配（兜底识别：用户未带编号时按标题关键词命中）
	seenProblem := map[int]bool{}
	for _, e := range oc.Entities {
		if e.ObjectType == "problem" {
			seenProblem[e.ID] = true
		}
	}
	for _, kw := range ciNames {
		probs, err := s.client.Problem.Query().
			Where(
				problem.TenantIDEQ(tenantID),
				problem.TitleContainsFold(kw),
				problem.DeletedAtIsNil(),
			).
			Order(ent.Desc(problem.FieldCreatedAt)).
			Limit(2).
			All(ctx)
		if err != nil {
			s.logger.Warnw("ontology: problem title fuzzy lookup failed", "keyword", kw, "error", err)
			continue
		}
		for _, p := range probs {
			if seenProblem[p.ID] {
				continue
			}
			seenProblem[p.ID] = true
			oc.Entities = append(oc.Entities, s.expandProblem(ctx, tenantID, p))
		}
	}

	// 3b. 反向兜底：中文 query 无分词边界时（整句 CJK 连续段），
	// 拉取近期问题用标题片段反向匹配 query，只取最新一个命中避免噪声。
	if len(seenProblem) == 0 {
		probs, err := s.client.Problem.Query().
			Where(
				problem.TenantIDEQ(tenantID),
				problem.DeletedAtIsNil(),
			).
			Order(ent.Desc(problem.FieldUpdatedAt)).
			Limit(50).
			All(ctx)
		if err != nil {
			s.logger.Warnw("ontology: problem reverse title lookup failed", "error", err)
		} else {
			lowerQuery := strings.ToLower(query)
			for _, p := range probs {
				if problemTitleMatchesQuery(p.Title, lowerQuery) {
					oc.Entities = append(oc.Entities, s.expandProblem(ctx, tenantID, p))
					break
				}
			}
		}
	}

	return oc
}

// problemTitleMatchesQuery 判断问题标题的任一关键词片段（CJK 段/字母数字段，≥3 字符）
// 是否出现在 query 中（大小写不敏感）。用于无分词边界的中文 query 反向兜底匹配。
func problemTitleMatchesQuery(title, lowerQuery string) bool {
	if title == "" || lowerQuery == "" {
		return false
	}
	for _, seg := range titleSegments(title) {
		if strings.Contains(lowerQuery, strings.ToLower(seg)) {
			return true
		}
	}
	return false
}

// titleSegments 把标题切分为 CJK 连续段与字母数字段（各 ≥3 字符），供模糊匹配使用。
func titleSegments(title string) []string {
	re := regexp.MustCompile(`[\p{Han}]+|[A-Za-z][A-Za-z0-9_-]{2,}`)
	out := make([]string, 0, 4)
	for _, w := range re.FindAllString(title, -1) {
		if len([]rune(w)) >= 3 {
			out = append(out, w)
		}
	}
	return out
}

// ---- 实体扩展（1 跳邻居） ----

func (s *OntologyService) expandTicket(ctx context.Context, tenantID int, t *ent.Ticket) *OntologyEntity {
	e := &OntologyEntity{
		ObjectType: "ticket",
		ID:         t.ID,
		Number:     t.TicketNumber,
		Title:      t.Title,
		Status:     t.Status,
		Snippet:    truncate(t.Description, 160),
		Neighbors:  make([]string, 0, 8),
	}

	// 相关工单（M2M 双向）
	related, err := t.QueryRelatedTickets().
		Where(ticket.TenantIDEQ(tenantID), ticket.DeletedAtIsNil()).
		Limit(s.maxNeighbors).
		All(ctx)
	if err != nil {
		s.logger.Warnw("ontology: expand ticket related failed", "ticket_id", t.ID, "error", err)
	}
	for _, r := range related {
		e.Neighbors = append(e.Neighbors, fmt.Sprintf("%s「%s」[%s]（关联工单）", r.TicketNumber, r.Title, r.Status))
	}

	// 关联 CI（通过中间表反查：Ticket 侧无正向查询方法）
	cis, err := s.client.ConfigurationItem.Query().
		Where(
			ciPred.TenantIDEQ(tenantID),
			ciPred.HasTicketsWith(ticket.ID(t.ID)),
		).
		Limit(s.maxNeighbors).
		All(ctx)
	if err != nil {
		s.logger.Warnw("ontology: expand ticket CIs failed", "ticket_id", t.ID, "error", err)
	}
	for _, ci := range cis {
		e.Neighbors = append(e.Neighbors, fmt.Sprintf("CI「%s」[%s]（受影响配置项）", ci.Name, ci.Status))
	}

	// 经办人路由
	if a := t.Edges.Assignee; a != nil {
		e.Neighbors = append(e.Neighbors, fmt.Sprintf("经办人：%s", userDisplayName(a)))
	}
	if r := t.Edges.Requester; r != nil {
		e.Neighbors = append(e.Neighbors, fmt.Sprintf("请求人：%s", userDisplayName(r)))
	}
	return e
}

func (s *OntologyService) expandIncident(ctx context.Context, tenantID int, in *ent.Incident) *OntologyEntity {
	e := &OntologyEntity{
		ObjectType: "incident",
		ID:         in.ID,
		Number:     in.IncidentNumber,
		Title:      in.Title,
		Status:     in.Status,
		Snippet:    truncate(in.Description, 160),
		Neighbors:  make([]string, 0, 8),
	}

	// 关联问题（M2M：问题汇聚点，RCA 主线）
	problems, err := in.QueryProblems().
		Where(problem.TenantIDEQ(tenantID), problem.DeletedAtIsNil()).
		Limit(s.maxNeighbors).
		All(ctx)
	if err != nil {
		s.logger.Warnw("ontology: expand incident problems failed", "incident_id", in.ID, "error", err)
	}
	for _, p := range problems {
		e.Neighbors = append(e.Neighbors, fmt.Sprintf("问题#%d「%s」[%s]（根因汇聚）", p.ID, p.Title, p.Status))
	}

	// 受影响 CI（M2M 正向）
	cis, err := in.QueryConfigurationItems().
		Where(ciPred.TenantIDEQ(tenantID)).
		Limit(s.maxNeighbors).
		All(ctx)
	if err != nil {
		s.logger.Warnw("ontology: expand incident CIs failed", "incident_id", in.ID, "error", err)
	}
	for _, ci := range cis {
		e.Neighbors = append(e.Neighbors, fmt.Sprintf("CI「%s」[%s]（受影响配置项，criticality=%s）", ci.Name, ci.Status, ci.Criticality))
	}

	// 触发告警（RCA 上游）
	alerts, err := in.QueryIncidentAlerts().
		Where(incidentalert.TenantIDEQ(tenantID)).
		Limit(s.maxNeighbors).
		All(ctx)
	if err != nil {
		s.logger.Warnw("ontology: expand incident alerts failed", "incident_id", in.ID, "error", err)
	} else {
		for _, a := range alerts {
			e.Neighbors = append(e.Neighbors, fmt.Sprintf("告警「%s」[%s/%s]（触发信号）", a.AlertName, a.AlertType, a.Severity))
		}
	}
	return e
}

func (s *OntologyService) expandRelease(ctx context.Context, tenantID int, rel *ent.Release) *OntologyEntity {
	e := &OntologyEntity{
		ObjectType: "release",
		ID:         rel.ID,
		Number:     rel.ReleaseNumber,
		Title:      rel.Title,
		Status:     rel.Status,
		Snippet:    truncate(rel.Description, 160),
		Neighbors:  make([]string, 0, 4),
	}
	return e
}

func (s *OntologyService) expandProblem(ctx context.Context, tenantID int, p *ent.Problem) *OntologyEntity {
	e := &OntologyEntity{
		ObjectType: "problem",
		ID:         p.ID,
		Number:     p.ProblemNumber,
		Title:      p.Title,
		Status:     p.Status,
		Snippet:    truncate(p.Description, 160),
		Neighbors:  make([]string, 0, 8),
	}

	// 汇聚的事件（M2M：RCA 主线，问题为汇聚点）
	incidents, err := p.QueryIncidents().
		Where(incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		Limit(s.maxNeighbors).
		All(ctx)
	if err != nil {
		s.logger.Warnw("ontology: expand problem incidents failed", "problem_id", p.ID, "error", err)
	}
	for _, in := range incidents {
		e.Neighbors = append(e.Neighbors, fmt.Sprintf("%s「%s」[%s]（关联事件）", in.IncidentNumber, in.Title, in.Status))
	}

	// 衍生变更（问题修复的实施载体）
	changes, err := p.QueryChanges().
		Where(changePred.TenantIDEQ(tenantID)).
		Limit(s.maxNeighbors).
		All(ctx)
	if err != nil {
		s.logger.Warnw("ontology: expand problem changes failed", "problem_id", p.ID, "error", err)
	}
	for _, c := range changes {
		e.Neighbors = append(e.Neighbors, fmt.Sprintf("%s「%s」[%s]（修复变更）", c.ChangeNumber, c.Title, c.Status))
	}

	// 关联工单
	tickets, err := p.QueryTickets().
		Where(ticket.TenantIDEQ(tenantID), ticket.DeletedAtIsNil()).
		Limit(s.maxNeighbors).
		All(ctx)
	if err != nil {
		s.logger.Warnw("ontology: expand problem tickets failed", "problem_id", p.ID, "error", err)
	}
	for _, t := range tickets {
		e.Neighbors = append(e.Neighbors, fmt.Sprintf("%s「%s」[%s]（关联工单）", t.TicketNumber, t.Title, t.Status))
	}
	return e
}

func (s *OntologyService) expandChange(ctx context.Context, tenantID int, c *ent.Change) *OntologyEntity {
	e := &OntologyEntity{
		ObjectType: "change",
		ID:         c.ID,
		Number:     c.ChangeNumber,
		Title:      c.Title,
		Status:     c.Status,
		Snippet:    truncate(c.Description, 160),
		Neighbors:  make([]string, 0, 8),
	}

	// 溯源问题（变更的实施动因）
	problems, err := c.QueryProblems().
		Where(problem.TenantIDEQ(tenantID), problem.DeletedAtIsNil()).
		Limit(s.maxNeighbors).
		All(ctx)
	if err != nil {
		s.logger.Warnw("ontology: expand change problems failed", "change_id", c.ID, "error", err)
	}
	for _, p := range problems {
		e.Neighbors = append(e.Neighbors, fmt.Sprintf("%s「%s」[%s]（溯源问题）", p.ProblemNumber, p.Title, p.Status))
	}
	return e
}

func (s *OntologyService) expandCI(ctx context.Context, tenantID int, ci *ent.ConfigurationItem) *OntologyEntity {
	e := &OntologyEntity{
		ObjectType: "ci",
		ID:         ci.ID,
		Title:      ci.Name,
		Status:     ci.Status,
		Snippet:    fmt.Sprintf("类型 %s / 环境 %s / 关键度 %s", ci.CiType, ci.Environment, ci.Criticality),
		Neighbors:  make([]string, 0, 12),
	}

	// 出边关系（本体类型化边：depends_on/hosts/...）
	out, err := s.client.CIRelationship.Query().
		Where(
			cirelationship.TenantIDEQ(tenantID),
			cirelationship.SourceCiID(ci.ID),
			cirelationship.IsActiveEQ(true),
		).
		WithTargetCi().
		Limit(s.maxNeighbors).
		All(ctx)
	if err != nil {
		s.logger.Warnw("ontology: expand CI outgoing failed", "ci_id", ci.ID, "error", err)
	}
	for _, r := range out {
		target := "未知CI"
		if r.Edges.TargetCi != nil {
			target = r.Edges.TargetCi.Name
		}
		e.Neighbors = append(e.Neighbors, fmt.Sprintf("CI「%s」 <-%s(强度:%s)- 本CI（%s）", target, r.RelationshipType, r.Strength, directionLabel(r.RelationshipType)))
	}

	// 入边关系
	in, err := s.client.CIRelationship.Query().
		Where(
			cirelationship.TenantIDEQ(tenantID),
			cirelationship.TargetCiID(ci.ID),
			cirelationship.IsActiveEQ(true),
		).
		WithSourceCi().
		Limit(s.maxNeighbors).
		All(ctx)
	if err != nil {
		s.logger.Warnw("ontology: expand CI incoming failed", "ci_id", ci.ID, "error", err)
	}
	for _, r := range in {
		source := "未知CI"
		if r.Edges.SourceCi != nil {
			source = r.Edges.SourceCi.Name
		}
		e.Neighbors = append(e.Neighbors, fmt.Sprintf("本CI <-%s(强度:%s)- CI「%s」（%s）", r.RelationshipType, r.Strength, source, directionLabel(reverseRelation(r.RelationshipType))))
	}

	// 受影响事件（CI 正向边）
	incidents, err := ci.QueryIncidents().
		Where(incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		Limit(s.maxNeighbors).
		All(ctx)
	if err != nil {
		s.logger.Warnw("ontology: expand CI incidents failed", "ci_id", ci.ID, "error", err)
	}
	for _, in2 := range incidents {
		e.Neighbors = append(e.Neighbors, fmt.Sprintf("%s「%s」[%s]（关联事件）", in2.IncidentNumber, in2.Title, in2.Status))
	}
	return e
}

// lookupCI 名称匹配：先精确，再前缀（ContainsFold 语义太宽，避免误命中）
func (s *OntologyService) lookupCI(ctx context.Context, tenantID int, name string) (*ent.ConfigurationItem, error) {
	ci, err := s.client.ConfigurationItem.Query().
		Where(ciPred.TenantIDEQ(tenantID), ciPred.NameEQ(name)).
		Only(ctx)
	if err == nil {
		return ci, nil
	}
	if !ent.IsNotFound(err) {
		s.logger.Warnw("ontology: CI exact lookup failed", "name", name, "error", err)
		return nil, err
	}
	// 前缀匹配兜底（CI 命名常带环境后缀）
	prefix, err := s.client.ConfigurationItem.Query().
		Where(ciPred.TenantIDEQ(tenantID), ciPred.NameHasPrefix(name)).
		Limit(2).
		All(ctx)
	if err != nil {
		s.logger.Warnw("ontology: CI prefix lookup failed", "name", name, "error", err)
		return nil, err
	}
	if len(prefix) == 1 {
		return prefix[0], nil
	}
	return nil, &ent.NotFoundError{}
}

// ---- 工具函数 ----

func matchAll(re *regexp.Regexp, s string) []string {
	return re.FindAllString(strings.ToUpper(s), -1)
}

func dedupeStringsLocal(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// extractCandidateCINames 从 query 中抽取候选 CI 名称：
// 优先「」『』"" ” 引号内内容；否则抽取 ≥3 字符的连续 CJK/字母数字词组。
func extractCandidateCINames(query string) []string {
	var candidates []string
	quotePairs := [][2]string{{"「", "」"}, {"『", "』"}, {"\"", "\""}, {"“", "”"}, {"'", "'"}}
	rest := query
	for _, pair := range quotePairs {
		for {
			i := strings.Index(rest, pair[0])
			if i < 0 {
				break
			}
			after := rest[i+len(pair[0]):]
			j := strings.Index(after, pair[1])
			if j < 0 {
				break
			}
			inner := strings.TrimSpace(after[:j])
			if len([]rune(inner)) >= 2 {
				candidates = append(candidates, inner)
			}
			rest = after[j+len(pair[1]):]
		}
	}
	if len(candidates) > 0 {
		return dedupeStringsLocal(candidates)
	}
	// 无引号：抽词组（CJK 连续段或字母数字段，≥3 字符）
	re := regexp.MustCompile(`[\p{Han}]+|[A-Za-z][A-Za-z0-9_-]{2,}`)
	for _, w := range re.FindAllString(query, -1) {
		// 过滤疑问词等噪声（黑名单）
		if len([]rune(w)) >= 3 && !isStopword(w) {
			candidates = append(candidates, w)
		}
	}
	return dedupeStringsLocal(candidates)[:minInt(3, len(dedupeStringsLocal(candidates)))]
}

var ciStopwords = map[string]bool{
	"有哪些": true, "怎么样": true, "是什么": true, "怎么办": true,
	"相关工单": true, "关联工单": true, "相关事件": true, "影响分析": true,
	"配置项": true, "关系图": true, "影响面": true, "根因分析": true,
	"the": true, "and": true, "for": true, "what": true, "how": true,
}

func isStopword(w string) bool { return ciStopwords[w] }

func reverseRelation(rel string) string {
	m := map[string]string{
		"depends_on": "impacted_by", "impacted_by": "depends_on",
		"hosts": "hosted_on", "hosted_on": "hosts",
		"contains": "part_of", "part_of": "contains",
		"impacts": "impacted_by", "owns": "owned_by", "owned_by": "owns",
		"uses": "used_by", "used_by": "uses",
	}
	if v, ok := m[rel]; ok {
		return v
	}
	return rel
}

func directionLabel(rel string) string {
	switch rel {
	case "depends_on", "impacted_by", "hosted_on", "part_of", "owned_by", "used_by":
		return "依赖/受影响方向"
	case "impacts", "hosts", "contains", "owns", "uses":
		return "影响/承载方向"
	}
	return "双向"
}

func userDisplayName(u *ent.User) string {
	if u == nil {
		return "未知"
	}
	if u.Name != "" {
		return u.Name
	}
	return u.Username
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
