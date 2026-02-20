package service

import (
    "context"
    "fmt"
    "time"

    "itsm-backend/dto"
    "itsm-backend/ent"
    "itsm-backend/ent/auditlog"

    "go.uber.org/zap"
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

    // 时间范围过滤
    if req.From != "" {
        if t, err := time.Parse(time.RFC3339, req.From); err == nil {
            q = q.Where(auditlog.CreatedAtGTE(t))
        } else {
            s.logger.Warnw("Invalid From time format", "from", req.From, "error", err)
        }
    }
    if req.To != "" {
        if t, err := time.Parse(time.RFC3339, req.To); err == nil {
            q = q.Where(auditlog.CreatedAtLTE(t))
        } else {
            s.logger.Warnw("Invalid To time format", "to", req.To, "error", err)
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
        Logs:     logs,
        Total:    total,
        Page:     req.Page,
        PageSize: req.PageSize,
    }, nil
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
        Logs:     logs,
        Total:    total,
        Page:     page,
        PageSize: pageSize,
    }, nil
}