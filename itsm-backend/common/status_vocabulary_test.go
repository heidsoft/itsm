package common

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// statusCheckMigration 是状态词表 CHECK 约束的唯一迁移文件。
const statusCheckMigration = "../migrations/20261031_status_check_constraints.sql"

// constraintPattern 从迁移 SQL 中提取「表 / 约束名 / 列 / 字面量集合」。
//
// 刻意解析真实 DDL 而不是维护第二份 Go 镜像：如果测试里再抄一份集合，
// 它只会证明「两份抄写一致」，证明不了「数据库约束与代码一致」。
var constraintPattern = regexp.MustCompile(
	`(?s)ALTER TABLE ([a-z_]+)\s+ADD CONSTRAINT ([a-z_]+)\s+CHECK \(([a-z_]+) = ANY \(ARRAY\[([^\]]*)\]::text\[\]\)\)`,
)

var sqlLiteralPattern = regexp.MustCompile(`'([^']*)'`)

// parseStatusCheckMigration 返回 "table.column" -> 允许值集合，以及约束名。
func parseStatusCheckMigration(t *testing.T) (map[string][]string, map[string]string) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(".", statusCheckMigration))
	require.NoError(t, err, "状态词表迁移文件必须存在：%s", statusCheckMigration)
	sql := string(raw)

	sets := make(map[string][]string)
	names := make(map[string]string)

	for _, m := range constraintPattern.FindAllStringSubmatch(sql, -1) {
		table, constraint, column, literals := m[1], m[2], m[3], m[4]
		key := table + "." + column

		matches := sqlLiteralPattern.FindAllStringSubmatch(literals, -1)
		require.NotEmpty(t, matches, "约束 %s 的 ARRAY 字面量为空", constraint)

		values := make([]string, 0, len(matches))
		for _, lit := range matches {
			values = append(values, lit[1])
		}
		sort.Strings(values)

		_, dup := sets[key]
		require.False(t, dup, "迁移文件重复声明了 %s 的 CHECK 约束", key)
		sets[key] = values
		names[key] = constraint
	}

	require.NotEmpty(t, sets, "未从 %s 解析到任何 CHECK 约束——正则或文件格式已漂移", statusCheckMigration)
	return sets, names
}

// TestStatusVocabularyMatchesCheckMigration 锁定「Go 词表 ↔ DB CHECK 字面量」双向一致。
//
// 任何一侧单独新增/删除状态值都会让本测试变红：
//   - 只改 Go 清单：数据库仍拒绝新值（运行时 23514），测试先一步拦住；
//   - 只改 SQL：代码仍会写入词表外的值，测试同样拦住。
func TestStatusVocabularyMatchesCheckMigration(t *testing.T) {
	sqlSets, sqlNames := parseStatusCheckMigration(t)

	goSets := make(map[string][]string, len(statusVocabulary))
	for _, c := range statusVocabulary {
		key := c.Table + "." + c.Column
		values := append([]string(nil), c.Allowed...)
		sort.Strings(values)
		goSets[key] = values

		assert.Equal(t, "ck_"+c.Table+"_"+c.Column+"_vocabulary", sqlNames[key],
			"%s 的约束命名必须与词表登记一致（便于运维按名定位）", key)
	}

	var goKeys, sqlKeys []string
	for k := range goSets {
		goKeys = append(goKeys, k)
	}
	for k := range sqlSets {
		sqlKeys = append(sqlKeys, k)
	}
	sort.Strings(goKeys)
	sort.Strings(sqlKeys)

	require.Equal(t, goKeys, sqlKeys,
		"受 CHECK 约束保护的列集合必须与 Go 词表完全一致（多一列少一列都是漂移）")

	for _, key := range goKeys {
		assert.Equal(t, goSets[key], sqlSets[key],
			"%s 的 CHECK 字面量必须与 status_vocabulary.go 逐值相等", key)
	}
}

// TestStatusVocabularyLegacyValuesAreAllowed 保证 Legacy 只是「兼容存量」，
// 不能出现「标记为 legacy 却不在允许集合内」的自相矛盾。
func TestStatusVocabularyLegacyValuesAreAllowed(t *testing.T) {
	for _, c := range statusVocabulary {
		for _, legacy := range c.Legacy {
			assert.True(t, StatusValueAllowed(c.Table, c.Column, legacy),
				"%s.%s 的 legacy 值 %q 不在允许集合内：约束会拒绝存量数据", c.Table, c.Column, legacy)
		}
	}
}

// TestStatusMachinesOnlyUseVocabulary 保证状态机不会产生 CHECK 约束拒绝的值。
//
// 这是最容易出事的漂移方向：状态机新增一个状态、词表没跟上，单元测试
// （mock 仓储）全绿，生产写入却撞上 23514。
func TestStatusMachinesOnlyUseVocabulary(t *testing.T) {
	cases := []struct {
		table  string
		column string
		m      map[string][]string
	}{
		{"tickets", "status", ticketStatusTransitions},
		{"incidents", "status", incidentStatusTransitions},
	}

	for _, tc := range cases {
		require.NotEmpty(t, tc.m, "%s 状态机迁移表为空", tc.table)
		for from, targets := range tc.m {
			assert.True(t, StatusValueAllowed(tc.table, tc.column, from),
				"%s 状态机以 %q 为源状态，但该值不在 CHECK 词表内", tc.table, from)
			for _, to := range targets {
				assert.True(t, StatusValueAllowed(tc.table, tc.column, to),
					"%s 状态机允许迁移到 %q，但该值不在 CHECK 词表内（生产会 23514）", tc.table, to)
			}
		}
	}
}

// TestStatusConstantsBelongToVocabulary 锁定「导出的状态常量必须可写入」。
//
// 2026-10-31 的实测缺陷就是这条失守：ProblemStatusNew = "new" 从未被任何
// 代码写入（创建时用 "open"），一旦有人按常量写入，数据库 CHECK 会直接拒绝。
// 幽灵常量比缺常量更危险，因为它看起来是权威的。
func TestStatusConstantsBelongToVocabulary(t *testing.T) {
	cases := []struct {
		table  string
		column string
		values map[string]string
	}{
		{"tickets", "status", map[string]string{
			"TicketStatusNew": TicketStatusNew, "TicketStatusOpen": TicketStatusOpen,
			"TicketStatusInProgress": TicketStatusInProgress, "TicketStatusPending": TicketStatusPending,
			"TicketStatusResolved": TicketStatusResolved, "TicketStatusClosed": TicketStatusClosed,
			"TicketStatusCancelled": TicketStatusCancelled, "TicketStatusAssigned": TicketStatusAssigned,
			"TicketStatusApproved": TicketStatusApproved, "TicketStatusRejected": TicketStatusRejected,
		}},
		{"incidents", "status", map[string]string{
			"IncidentStatusNew": IncidentStatusNew, "IncidentStatusAcknowledged": IncidentStatusAcknowledged,
			"IncidentStatusAssigned": IncidentStatusAssigned, "IncidentStatusInProgress": IncidentStatusInProgress,
			"IncidentStatusTriaged": IncidentStatusTriaged, "IncidentStatusEscalated": IncidentStatusEscalated,
			"IncidentStatusOnHold": IncidentStatusOnHold, "IncidentStatusResolved": IncidentStatusResolved,
			"IncidentStatusClosed": IncidentStatusClosed, "IncidentStatusCancelled": IncidentStatusCancelled,
		}},
		{"problems", "status", map[string]string{
			"ProblemStatusOpen": ProblemStatusOpen, "ProblemStatusInvestigating": ProblemStatusInvestigating,
			"ProblemStatusIdentified": ProblemStatusIdentified, "ProblemStatusResolved": ProblemStatusResolved,
			"ProblemStatusClosed": ProblemStatusClosed, "ProblemStatusInProgress": ProblemStatusInProgress,
		}},
	}

	for _, tc := range cases {
		for name, value := range tc.values {
			assert.True(t, StatusValueAllowed(tc.table, tc.column, value),
				"%s = %q 不在 %s.%s 的 CHECK 词表内——常量与数据库约束不一致",
				name, value, tc.table, tc.column)
		}
	}
}

// TestStatusVocabularyHasNoDuplicateValues 防止同一列里重复列值掩盖真实集合。
func TestStatusVocabularyHasNoDuplicateValues(t *testing.T) {
	for _, c := range statusVocabulary {
		seen := make(map[string]struct{}, len(c.Allowed))
		for _, v := range c.Allowed {
			require.NotEmpty(t, v, "%s.%s 含空状态值", c.Table, c.Column)
			_, dup := seen[v]
			assert.False(t, dup, "%s.%s 重复登记状态值 %q", c.Table, c.Column, v)
			seen[v] = struct{}{}
		}
	}
}

// TestStatusVocabularyKeysAreSorted 让失败信息与迁移文件顺序可对照。
func TestStatusVocabularyKeysAreSorted(t *testing.T) {
	keys := sortedStatusVocabularyKeys()
	sorted := append([]string(nil), keys...)
	sort.Strings(sorted)
	assert.Equal(t, sorted, keys)
	assert.True(t, strings.Contains(keys[0], "."))
}
