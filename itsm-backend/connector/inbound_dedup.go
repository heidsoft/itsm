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
	client     *ent.Client
	defaultTTL time.Duration
	now        func() time.Time
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
//
// 过期行可被覆盖：schema 注释承诺「expires_at 后允许覆盖」，但唯一约束不会因为时间流逝
// 自己消失——只有清理任务真的跑过才腾出位置。若清理任务停摆（它曾经根本没有生产调用点），
// 同一个 event_id 会被永久判为重复。这里在命中约束时顺手删掉已过期的同一行再重试一次，
// 让承诺不依赖后台任务的健康度。
func (d *InboundDedup) HandleBeforeProcess(ctx context.Context, tenantID int, connectorName, eventID string, payload []byte) (bool, error) {
	return d.register(ctx, tenantID, connectorName, eventID, payload, true)
}

// register 是 HandleBeforeProcess 的实现。allowExpiredOverride 只允许为 true 一次，
// 重试路径传 false，避免「删除-插入」在并发下反复循环。
func (d *InboundDedup) register(ctx context.Context, tenantID int, connectorName, eventID string, payload []byte, allowExpiredOverride bool) (bool, error) {
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
	if !ent.IsConstraintError(err) {
		return false, fmt.Errorf("inbound dedup: insert: %w", err)
	}
	if !allowExpiredOverride {
		return true, nil
	}
	// 命中 (tenant, connector, event_id) 唯一约束：先判断是不是「过期但没被清理」。
	deleted, delErr := d.client.ConnectorInboundDedup.Delete().
		Where(
			connectorinbounddedup.TenantIDEQ(tenantID),
			connectorinbounddedup.ConnectorNameEQ(connectorName),
			connectorinbounddedup.EventIDEQ(eventID),
			connectorinbounddedup.ExpiresAtLT(d.now()),
		).
		Exec(ctx)
	if delErr != nil {
		return false, fmt.Errorf("inbound dedup: purge expired: %w", delErr)
	}
	if deleted == 0 {
		// 未过期 → 真重复。
		return true, nil
	}
	return d.register(ctx, tenantID, connectorName, eventID, payload, false)
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

// Forget 删除一条去重行，用于「本次处理失败、允许平台重投重新接管」的场景。
//
// HandleBeforeProcess 先登记后处理，若不回滚，平台在 TTL 内的重投会被判重复——
// 一次瞬时失败（DB 抖动、下游超时）就让这条事件永久丢失，而平台那边看到的是
// 200/重复，不会再有下一次重试。
func (d *InboundDedup) Forget(ctx context.Context, tenantID int, connectorName, eventID string) error {
	if d.client == nil {
		return nil
	}
	_, err := d.client.ConnectorInboundDedup.Delete().
		Where(
			connectorinbounddedup.TenantIDEQ(tenantID),
			connectorinbounddedup.ConnectorNameEQ(connectorName),
			connectorinbounddedup.EventIDEQ(eventID),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("inbound dedup: forget: %w", err)
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
