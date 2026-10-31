package common

import "sort"

// StatusColumn 描述一张表的某个状态列的权威词表。
//
// 为什么需要这份清单：状态值散落在四个地方——common/dto 的常量、各域 service
// 的状态机、ent schema 的默认值、数据库里的存量数据。它们之间没有任何机制
// 保证一致，于是一边写 "open"、一边定义 "new" 的情况可以长期共存而无人发现
// （2026-10-31 实测：common.ProblemStatusNew = "new"，而问题域全线使用 "open"）。
//
// 本清单是「数据库 CHECK 约束」的 Go 侧事实源：
//   - migrations/20261031_status_check_constraints.sql 的 CHECK 字面量必须与
//     本清单逐值相等（common/status_vocabulary_test.go 解析 SQL 比对，双向）；
//   - 各域状态机的迁移表必须只包含本清单里的值（各域各自的守卫测试断言）；
//   - Legacy 列出的值只为兼容存量数据而保留，新代码不得写入。
//
// 新增状态值时，必须同时改三处：本清单、迁移 SQL、域状态机——只改一处会让
// 守卫测试变红，这正是它存在的意义。
type StatusColumn struct {
	Table  string
	Column string
	// Allowed 是当前代码可以写入的全部值（含 Legacy）。
	Allowed []string
	// Legacy 列出仅为兼容存量数据保留、新代码不应写入的值。
	Legacy []string
}

// statusVocabulary 是全部受 CHECK 约束保护的状态列。
//
// 顺序固定（表名字典序），让测试失败信息与迁移文件顺序可对照。
var statusVocabulary = []StatusColumn{
	{
		Table: "changes", Column: "status",
		// dto.ChangeStatus 的 11 个规范值 + 存量别名 submitted。
		// pending 与 submitted 是同一语义（待审批）的两个历史写法：
		// service.normalizeChangeTransitionStatus 在状态机入口把 pending 归一为
		// submitted，但数据库里两种写法都存在过，因此约束必须同时接受。
		Allowed: []string{
			"draft", "pending", "submitted", "approved", "rejected",
			"scheduled", "in_progress", "completed", "failed", "rolled_back",
			"cancelled", "closed",
		},
		Legacy: []string{"submitted"},
	},
	{
		Table: "incidents", Column: "status",
		Allowed: []string{
			"new", "acknowledged", "assigned", "in_progress", "triaged",
			"escalated", "on_hold", "resolved", "closed", "cancelled",
		},
	},
	{
		Table: "knowledge_articles", Column: "review_status",
		// 与 ent/schema/knowledgearticle.go 的 field.Enum 完全一致。
		Allowed: []string{"draft", "under_review", "approved", "rejected"},
	},
	{
		Table: "problems", Column: "status",
		// 权威词表：open → investigating/identified → resolved → closed。
		// in_progress 是存量别名：状态机只允许它迁出到规范状态，不产生新值。
		// 刻意不包含 "new"：common 曾经定义过 ProblemStatusNew = "new"，
		// 但没有任何代码写入它（创建时用 "open"），保留它等于给幽灵值发通行证。
		Allowed: []string{
			"open", "investigating", "identified", "in_progress", "resolved", "closed",
		},
		Legacy: []string{"in_progress"},
	},
	{
		Table: "service_requests", Column: "status",
		// domain/servicerequest/status.go 的 9 个值（审批链 + 交付链）。
		// 注意：ent/schema/servicerequest.go 的字段注释写的是
		// "submitted|approved|rejected|in_progress|completed|cancelled"，
		// 与实际词表不符——那段注释是错的，以本清单与 domain 常量为准。
		Allowed: []string{
			"submitted", "manager_approved", "it_approved", "security_approved",
			"rejected", "provisioning", "delivered", "failed", "cancelled",
		},
	},
	{
		Table: "tickets", Column: "status",
		Allowed: []string{
			"new", "open", "assigned", "in_progress", "pending", "resolved",
			"closed", "cancelled", "approved", "rejected",
		},
	},
}

// StatusVocabulary 返回受 CHECK 约束保护的状态列清单（副本，调用方可安全排序）。
func StatusVocabulary() []StatusColumn {
	out := make([]StatusColumn, len(statusVocabulary))
	for i, c := range statusVocabulary {
		out[i] = StatusColumn{
			Table:   c.Table,
			Column:  c.Column,
			Allowed: append([]string(nil), c.Allowed...),
			Legacy:  append([]string(nil), c.Legacy...),
		}
	}
	return out
}

// StatusVocabularyFor 返回某表某列的允许值集合；第二返回值表示该列是否受管。
func StatusVocabularyFor(table, column string) ([]string, bool) {
	for _, c := range statusVocabulary {
		if c.Table == table && c.Column == column {
			return append([]string(nil), c.Allowed...), true
		}
	}
	return nil, false
}

// StatusValueAllowed 判断某个值是否在受管词表内。未受管的列一律返回 true
// （本清单只声明自己负责的列，不对其他列做隐式约束）。
func StatusValueAllowed(table, column, value string) bool {
	allowed, managed := StatusVocabularyFor(table, column)
	if !managed {
		return true
	}
	for _, v := range allowed {
		if v == value {
			return true
		}
	}
	return false
}

// sortedStatusVocabularyKeys 供测试输出稳定顺序（"table.column"）。
func sortedStatusVocabularyKeys() []string {
	keys := make([]string, 0, len(statusVocabulary))
	for _, c := range statusVocabulary {
		keys = append(keys, c.Table+"."+c.Column)
	}
	sort.Strings(keys)
	return keys
}
