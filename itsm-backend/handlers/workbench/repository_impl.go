package workbench

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"itsm-backend/internal/datascope"
	"itsm-backend/service/lifecycle"
)

type repositoryImpl struct {
	db *sql.DB
}

// workbenchDomain 描述一个参与统一工作台的 ITIL 领域及其物理表。
// softDelete 必须与 ent/schema 实际列一致：changes 没有 deleted_at，
// 给它加 deleted_at IS NULL 会让整条 UNION 直接报错。
//
// requesterCol 是该域"发起人"列名。行级数据权限（owner 维度）需要按各域
// 自己的列过滤 —— 各域命名不统一：tickets.requester_id / incidents.reporter_id
// / changes.created_by / problems.created_by。写成常量是为了让映射关系显式可见，
// 避免在 SQL 里散落魔法字符串。
type workbenchDomain struct {
	name         string
	table        string
	softDelete   bool
	requesterCol string
}

var workbenchDomains = []workbenchDomain{
	{name: "incident", table: "incidents", softDelete: true, requesterCol: "reporter_id"},
	{name: "change", table: "changes", softDelete: false, requesterCol: "created_by"},
	{name: "problem", table: "problems", softDelete: true, requesterCol: "created_by"},
	{name: "ticket", table: "tickets", softDelete: true, requesterCol: "requester_id"},
}

// NewRepository creates a new workbench repository.
func NewRepository(db *sql.DB) Repository {
	return &repositoryImpl{db: db}
}

// Query 聚合四域条目并分页返回。
//
// SQL 由本函数内的固定字面量与占位符拼成：所有用户输入（assignee_id、priority、
// record_type、status 等）都经 nextArg() 变成 $N 占位符并进 args，LIMIT/OFFSET 用
// %d 格式化 int（类型层面不可注入）。gosec G201 只看得到 fmt.Sprintf 拼 SQL 的
// 形态，看不到占位符来源，故在此按函数抑制并把依据写明。
//
// #nosec G201
func (r *repositoryImpl) Query(ctx context.Context, query WorkbenchQuery) (*WorkbenchResponse, error) {
	argIndex := 0
	nextArg := func() string {
		argIndex++
		return fmt.Sprintf("$%d", argIndex)
	}

	var unions []string
	var args []interface{}

	for _, d := range workbenchDomains {
		statuses, ok := lifecycle.StatusesForPhases(lifecycle.Domain(d.name), query.Phase)
		if !ok {
			continue
		}

		conds := []string{fmt.Sprintf("tenant_id = %s", nextArg())}
		args = append(args, query.TenantID)
		if d.softDelete {
			conds = append(conds, "deleted_at IS NULL")
		}
		if len(statuses) > 0 {
			placeholders := make([]string, len(statuses))
			for i, s := range statuses {
				placeholders[i] = nextArg()
				args = append(args, s)
			}
			conds = append(conds, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ",")))
		}

		// 行级数据权限（owner/assignee 维度）：非管理角色只看本人发起或受理的单据。
		// 收窄条件放在各域子查询内，保证 UNION 的每一条分支都带上 ——
		// 放外层会因各域列名不同而无法统一引用。
		// viewerID <= 0 时返回恒假条件（安全收窄），不做"无过滤"处理。
		if query.DataScope != datascope.DataScopeAll {
			viewerArg := nextArg()
			args = append(args, query.ViewerID)
			if query.ViewerID <= 0 {
				conds = append(conds, "1 = 0")
			} else {
				conds = append(conds, fmt.Sprintf(
					"(%s = %s OR assignee_id = %s)", d.requesterCol, viewerArg, viewerArg))
			}
		}

		unions = append(unions, fmt.Sprintf(
			"SELECT '%s' as record_type, id, title, description, priority, status, assignee_id, tenant_id, created_at, updated_at FROM %s WHERE %s",
			d.name, d.table, strings.Join(conds, " AND "),
		))
	}

	if len(unions) == 0 {
		return &WorkbenchResponse{
			Items:    []WorkbenchItem{},
			Total:    0,
			Page:     query.Page,
			PageSize: query.PageSize,
		}, nil
	}

	unionSQL := strings.Join(unions, "\nUNION ALL\n")

	var whereClauses []string
	if query.AssigneeID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("assignee_id = %s", nextArg()))
		args = append(args, *query.AssigneeID)
	}

	if len(query.Priority) > 0 {
		placeholders := make([]string, len(query.Priority))
		for i, p := range query.Priority {
			placeholders[i] = nextArg()
			args = append(args, p)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("priority IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(query.RecordType) > 0 {
		placeholders := make([]string, len(query.RecordType))
		for i, rt := range query.RecordType {
			placeholders[i] = nextArg()
			args = append(args, rt)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("record_type IN (%s)", strings.Join(placeholders, ",")))
	}

	baseQuery := fmt.Sprintf("SELECT * FROM (%s) AS combined", unionSQL)
	if len(whereClauses) > 0 {
		baseQuery += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS counted", baseQuery)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count workbench items: %w", err)
	}

	outerQuery := baseQuery + " ORDER BY created_at DESC"
	if query.PageSize > 0 {
		outerQuery += fmt.Sprintf(" LIMIT %d", query.PageSize)
		if query.Page > 0 {
			offset := (query.Page - 1) * query.PageSize
			outerQuery += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, outerQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query workbench items: %w", err)
	}
	defer rows.Close()

	items := []WorkbenchItem{}
	for rows.Next() {
		var item WorkbenchItem
		var assigneeID sql.NullInt64
		var description sql.NullString
		if err := rows.Scan(
			&item.RecordType,
			&item.ID,
			&item.Title,
			&description,
			&item.Priority,
			&item.Status,
			&assigneeID,
			&item.TenantID,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan workbench item: %w", err)
		}
		if assigneeID.Valid {
			id := int(assigneeID.Int64)
			item.AssigneeID = &id
		}
		if description.Valid {
			item.Description = description.String
		}
		item.Phase = string(lifecycle.StatusToPhase(lifecycle.Domain(item.RecordType), item.Status))
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workbench rows: %w", err)
	}

	totalPages := 0
	if query.PageSize > 0 {
		totalPages = (total + query.PageSize - 1) / query.PageSize
	}

	return &WorkbenchResponse{
		Items:      items,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: totalPages,
	}, nil
}
