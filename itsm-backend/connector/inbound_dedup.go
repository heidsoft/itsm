package connector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/connectorinbounddedup"
)

// ErrDuplicateInbound 表示 (tenant, connector, event_id) 三元组命中现有去重行；
// handler 应按已处理语义返回（通常是 cached response）。
var ErrDuplicateInbound = errors.New("connector: duplicate inbound event")

// InboundDeduper 把 IM/Webhook 入站事件持久化到 ent.ConnectorInboundDedup 表，
// 用 (tenant_id, connector_name, event_id) UNIQUE 约束替代 in-memory nonce map，
// 让重启 / 多实例部署不再丢去重窗口。HandleBeforeProcess 返回 true 时说明事件
// 已经被处理过（同 event_id），handler 应直接返回上次的响应；返回 false 时
// 表示本轮是首次，handler 在处理成功后必须调 MarkProcessed 落地结果。
type InboundDedup struct {
	client      *ent.Client
	defaultTTL time.Duration
	now         func() time.Time
}

// NewInboundDeduper 创建持久化入站去重器。ttl 默认 5 分钟，与常见 IM 平台 timestamp 窗口一致。
func NewInboundDeduper(client *ent.Client, ttl time.Duration) *InboundDedup {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &InboundDedup{client: client, defaultTTL: ttl, now: time.Now}
}

// HandleBeforeProcess 试图登记 (tenant, connector, event_id)：
//   - 返回 (true, nil)  → 已存在，handler 必须短路
//   - 返回 (false, nil)  → 首次出现，handler 处理后必须调 MarkProcessed
//   - 返回 (_, err)     → DB 错误，handler 应 fail-closed
//
// 为避免并发场景下"同时进入都看到不存在"，插入失败 UNIQUE 约束时返回 ErrDuplicateInbound。
func (d *InboundDedup) HandleBeforeProcess(ctx context.Context, tenantID int, connectorName, eventID string, payload []byte) (bool, error) {
	if tenantID <= 0 || connectorName == "" || eventID == "" {
		return false, fmt.Errorf("inbound dedup: missing required fields")
	}
	if d.client == nil {
		return false, fmt.Errorf("inbound dedup: ent client is nil")
	}
	expires := d.now().Add(d.defaultTTL)
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	_, err := d.client.ConnectorInboundDedup.Create().
		SetTenantID(tenantID).
		SetConnectorName(connectorName).
		SetEventID(eventID).
		SetExpiresAt(expires).
		SetPayloadHash(payloadHash).
		SetResponseStatus("processing").
		Save(ctx)
	if err == nil {
		return false, nil
	}
	if ent.IsConstraintError(err) {
		return true, nil
	}
	return false, fmt.Errorf("inbound dedup: insert: %w", err)
}

// MarkProcessed 把事件结果状态写回（"ok" / "ignored" / "rejected"），用于运维排障。
// 仅在该 (tenant, connector, event_id) 未过期时才有意义，过期行被清理后无影响。
func (d *InboundDedup) MarkProcessed(ctx context.Context, tenantID int, connectorName, eventID, status string) error {
	if d.client == nil {
		return nil
	}
	_, err := d.client.ConnectorInboundDedup.Update().
		Where(
			connectorinbounddedup.TenantIDEQ(tenantID),
			connectorinbounddedup.ConnectorNameEQ(connectorName),
			connectorinbounddedup.EventIDEQ(eventID),
		).
		SetResponseStatus(status).
		Save(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return fmt.Errorf("inbound dedup: mark processed: %w", err)
	}
	return nil
}

// CleanupExpired 删除已过期的去重行。运维可周期性调用（5min TTL → 5min 一次足够）。
func (d *InboundDedup) CleanupExpired(ctx context.Context) (int, error) {
	if d.client == nil {
		return 0, nil
	}
	n, err := d.client.ConnectorInboundDedup.Delete().
		Where(connectorinbounddedup.ExpiresAtLT(d.now())).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("inbound dedup: cleanup: %w", err)
	}
	return n, nil
}

// LookupStatus 给 handler / 运维一个查询入口，看到底是不是已经处理过。
func (d *InboundDedup) LookupStatus(ctx context.Context, tenantID int, connectorName, eventID string) (string, bool, error) {
	if d.client == nil {
		return "", false, nil
	}
	row, err := d.client.ConnectorInboundDedup.Query().
		Where(
			connectorinbounddedup.TenantIDEQ(tenantID),
			connectorinbounddedup.ConnectorNameEQ(connectorName),
			connectorinbounddedup.EventIDEQ(eventID),
		).
		Only(ctx)
	if ent.IsNotFound(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return row.ResponseStatus, true, nil
}