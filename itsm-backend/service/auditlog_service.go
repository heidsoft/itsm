package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/auditlog"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

// 审计日志读面的无界防护（L7）。audit_logs 此前只有单列 tenant_id 索引，
// Count + Order(created_at DESC) 在无时间窗口时是全租户扫描。
const (
	maxAuditLogWindowDays = 366
	maxAuditLogPage       = 10000
)

type AuditLogService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewAuditLogService(client *ent.Client, logger *zap.SugaredLogger) *AuditLogService {
	return &AuditLogService{client: client, logger: logger}
}

// ListAuditLogs 根据条件查询审计日志（分页）
func (s *AuditLogService) ListAuditLogs(ctx context.Context, req *dto.ListAuditLogsRequest, tenantID int) (*dto.ListAuditLogsResponse, error) {
	// 分页默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}
	// 深分页的 Offset 会让数据库跳过海量行；超过上限直接 400，
	// 让调用方改用时间窗口而不是无限翻页。
	if req.Page > maxAuditLogPage {
		return nil, common.NewBadRequestError(fmt.Sprintf("page 不得超过 %d", maxAuditLogPage), nil)
	}

	q := s.client.AuditLog.Query().Where(
		auditlog.TenantID(tenantID),
	)

	if req.UserID != nil {
		q = q.Where(auditlog.UserIDEQ(*req.UserID))
	}
	if req.Resource != "" {
		q = q.Where(auditlog.ResourceEQ(req.Resource))
	}
	if req.Action != "" {
		q = q.Where(auditlog.ActionEQ(req.Action))
	}
	if req.Method != "" {
		q = q.Where(auditlog.MethodEQ(req.Method))
	}
	if req.StatusCode != nil {
		q = q.Where(auditlog.StatusCodeEQ(*req.StatusCode))
	}
	if req.Path != "" {
		q = q.Where(auditlog.PathHasPrefix(req.Path))
	}
	if req.RequestID != "" {
		q = q.Where(auditlog.RequestIDEQ(req.RequestID))
	}

	// 时间范围过滤：非法格式不再静默忽略（否则调用方以为筛了时间，
	// 实际谓词根本没下推，仍是全租户扫描）。handler 已用 common.RespondError，
	// AppError 的 400 会原样透传，不会被兜底成 500。
	var fromTime, toTime time.Time
	if req.From != "" {
		t, err := time.Parse(time.RFC3339, req.From)
		if err != nil {
			return nil, common.NewBadRequestError("from 时间格式非法，应为 RFC3339", err)
		}
		fromTime = t
		q = q.Where(auditlog.CreatedAtGTE(t))
	}
	if req.To != "" {
		t, err := time.Parse(time.RFC3339, req.To)
		if err != nil {
			return nil, common.NewBadRequestError("to 时间格式非法，应为 RFC3339", err)
		}
		toTime = t
		q = q.Where(auditlog.CreatedAtLTE(t))
	}
	if !fromTime.IsZero() && !toTime.IsZero() {
		if toTime.Before(fromTime) {
			return nil, common.NewBadRequestError("to 不得早于 from", nil)
		}
		if toTime.Sub(fromTime) > time.Duration(maxAuditLogWindowDays)*24*time.Hour {
			return nil, common.NewBadRequestError(fmt.Sprintf("查询窗口不得超过 %d 天", maxAuditLogWindowDays), nil)
		}
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count audit logs", "error", err, "tenant_id", tenantID)
		return nil, err
	}

	items, err := q.
		Order(ent.Desc(auditlog.FieldCreatedAt)).
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list audit logs", "error", err, "tenant_id", tenantID)
		return nil, err
	}

	// 批量 join user 表查询 UserID 对应的姓名，避免前端只看到裸 ID。
	userNameMap := map[int]string{}
	if userIDs := collectAuditLogUserIDs(items); len(userIDs) > 0 {
		users, err := s.client.User.Query().
			Where(user.TenantIDEQ(tenantID), user.IDIn(userIDs...)).
			Select(user.FieldID, user.FieldName, user.FieldUsername).
			All(ctx)
		if err != nil {
			s.logger.Warnw("auditlog: load user names failed, fall back to id", "error", err, "tenant_id", tenantID)
		} else {
			for _, u := range users {
				name := u.Name
				if name == "" {
					name = u.Username
				}
				if name != "" {
					userNameMap[u.ID] = name
				}
			}
		}
	}

	logs := make([]*dto.AuditLog, 0, len(items))
	for _, it := range items {
		var body string
		if it.RequestBody != nil {
			body = *it.RequestBody
		}
		log := &dto.AuditLog{
			ID:          it.ID,
			CreatedAt:   it.CreatedAt,
			TenantID:    it.TenantID,
			UserID:      it.UserID,
			RequestID:   it.RequestID,
			IP:          it.IP,
			Resource:    it.Resource,
			Action:      it.Action,
			Path:        it.Path,
			Method:      it.Method,
			StatusCode:  it.StatusCode,
			RequestBody: body,
		}
		if it.UserID > 0 {
			if name, ok := userNameMap[it.UserID]; ok {
				n := name
				log.UserName = &n
			}
		}
		logs = append(logs, log)
	}

	totalPages := 0
	if req.PageSize > 0 {
		totalPages = (total + req.PageSize - 1) / req.PageSize
	}
	return &dto.ListAuditLogsResponse{
		Items:      logs,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// collectAuditLogUserIDs 收集一批 audit log 中去重的非零 user id。
func collectAuditLogUserIDs(logs []*ent.AuditLog) []int {
	seen := map[int]struct{}{}
	out := make([]int, 0, len(logs))
	for _, l := range logs {
		if l == nil || l.UserID <= 0 {
			continue
		}
		if _, ok := seen[l.UserID]; ok {
			continue
		}
		seen[l.UserID] = struct{}{}
		out = append(out, l.UserID)
	}
	return out
}

// GetCIAuditLogs 获取CI相关的审计日志
func (s *AuditLogService) GetCIAuditLogs(ctx context.Context, tenantID, ciID, page, pageSize int) (*dto.ListAuditLogsResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	// 构造搜索模式 - 匹配包含 ci_<id> 的资源记录
	searchPattern := fmt.Sprintf("ci_%d", ciID)

	q := s.client.AuditLog.Query().Where(
		auditlog.TenantID(tenantID),
	)

	// 模糊匹配包含 CI ID 的资源记录
	// 由于resource字段可能存储不同的格式，我们使用Like来匹配
	q = q.Where(auditlog.ResourceContains(searchPattern))

	total, err := q.Clone().Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count CI audit logs", "error", err, "tenant_id", tenantID, "ci_id", ciID)
		return nil, err
	}

	items, err := q.
		Order(ent.Desc(auditlog.FieldCreatedAt)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list CI audit logs", "error", err, "tenant_id", tenantID, "ci_id", ciID)
		return nil, err
	}

	logs := make([]*dto.AuditLog, 0, len(items))
	for _, it := range items {
		var body string
		if it.RequestBody != nil {
			body = *it.RequestBody
		}
		logs = append(logs, &dto.AuditLog{
			ID:          it.ID,
			CreatedAt:   it.CreatedAt,
			TenantID:    it.TenantID,
			UserID:      it.UserID,
			RequestID:   it.RequestID,
			IP:          it.IP,
			Resource:    it.Resource,
			Action:      it.Action,
			Path:        it.Path,
			Method:      it.Method,
			StatusCode:  it.StatusCode,
			RequestBody: body,
		})
	}

	return &dto.ListAuditLogsResponse{
		Items:      logs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
	}, nil
}
