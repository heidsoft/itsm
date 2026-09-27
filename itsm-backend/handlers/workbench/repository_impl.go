package workbench

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"itsm-backend/service/lifecycle"
)

type repositoryImpl struct {
	db *sql.DB
}

// NewRepository creates a new workbench repository.
func NewRepository(db *sql.DB) Repository {
	return &repositoryImpl{db: db}
}

func (r *repositoryImpl) Query(ctx context.Context, query WorkbenchQuery) (*WorkbenchResponse, error) {
	argIndex := 0
	nextArg := func() string {
		argIndex++
		return fmt.Sprintf("$%d", argIndex)
	}

	var unions []string
	var args []interface{}

	domains := []struct {
		name  string
		table string
	}{
		{"incident", "incidents"},
		{"change", "changes"},
		{"problem", "problems"},
		{"ticket", "tickets"},
	}

	for _, d := range domains {
		tenantPlaceholder := nextArg()
		unions = append(unions, fmt.Sprintf(
			"SELECT '%s' as record_type, id, title, description, priority, status, assignee_id, tenant_id, created_at, updated_at FROM %s WHERE tenant_id = %s AND deleted_at IS NULL",
			d.name, d.table, tenantPlaceholder,
		))
		args = append(args, query.TenantID)
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

	outerQuery := fmt.Sprintf("SELECT * FROM (%s) AS combined", unionSQL)
	if len(whereClauses) > 0 {
		outerQuery += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s)", outerQuery)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count workbench items: %w", err)
	}

	outerQuery += " ORDER BY created_at DESC"
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

	var items []WorkbenchItem
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

	if len(query.Phase) > 0 {
		phaseSet := make(map[string]bool)
		for _, p := range query.Phase {
			phaseSet[p] = true
		}
		var filtered []WorkbenchItem
		for _, item := range items {
			if phaseSet[item.Phase] {
				filtered = append(filtered, item)
			}
		}
		items = filtered
		total = len(items)
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
