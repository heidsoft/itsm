package middleware

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"itsm-backend/internal/authz"
)

// =============================================================================
// 权限缓存跨实例失效广播
//
// 问题：permissionCache 是进程内 map，InvalidateRolePermissionCache 只能
// 清除本进程缓存。多副本部署下，副本 A 改权限后，副本 B/C 需等 TTL（5 分钟）
// 自然过期才能感知，存在权限变更不一致窗口。
//
// 方案：本地失效后向 Redis 频道发布消息，各实例订阅并失效各自进程内缓存。
// 未配置 Redis（单副本）时自动降级为仅本进程失效，行为与既有完全一致。
// =============================================================================

const (
	permissionCacheInvalidateChannel = "itsm:permission-cache:invalidate"
	// 策略/ACL 缓存失效的独立频道：与角色权限缓存的载荷格式不同，
	// 老实例不认识该频道，不会误把租户号当角色名解析。
	authzCacheInvalidateChannel = "itsm:authz-cache:invalidate"
)

var (
	permissionBroadcastOnce sync.Once
	permissionBroadcaster   *redis.Client
)

// ConfigurePermissionCacheBroadcast 启用权限缓存失效的跨实例广播（需 Redis）。
// 重复调用安全：订阅仅建立一次。client 为 nil 时为 no-op（单副本降级）。
func ConfigurePermissionCacheBroadcast(client *redis.Client) {
	if client == nil {
		return
	}
	permissionBroadcastOnce.Do(func() {
		permissionBroadcaster = client
		go subscribePermissionInvalidation(client)
		go subscribeAuthzCacheInvalidation(client)
	})
}

// broadcastPermissionInvalidation 向其它实例广播角色权限缓存失效。
// 发布失败不阻断本地失效（本地始终是权威兜底，TTL 最终一致）。
func broadcastPermissionInvalidation(roleName string, tenantID int) {
	if permissionBroadcaster == nil {
		return
	}
	msg := fmt.Sprintf("%s|%d", roleName, tenantID)
	if err := permissionBroadcaster.Publish(context.Background(),
		permissionCacheInvalidateChannel, msg).Err(); err != nil {
		zap.S().Warnw("权限缓存失效广播发布失败（其它实例将等TTL过期）",
			"role", roleName, "tenant_id", tenantID, "error", err)
	}
}

// subscribePermissionInvalidation 订阅失效广播并清除本进程对应缓存。
func subscribePermissionInvalidation(client *redis.Client) {
	sub := client.Subscribe(context.Background(), permissionCacheInvalidateChannel)
	defer sub.Close()

	for msg := range sub.Channel() {
		roleName, tenantID, ok := parseInvalidateMessage(msg.Payload)
		if !ok {
			continue
		}
		if PermissionConfig.EnableCache {
			invalidatePermissionCache(roleName, tenantID)
		}
		zap.S().Debugw("收到权限缓存失效广播，已清除本进程缓存",
			"role", roleName, "tenant_id", tenantID)
	}
}

// parseInvalidateMessage 解析 "role|tenant" 载荷。
// 租户段取最后一个 "|" 之后的内容（角色名本身允许含 "|"），且必须非空、全为数字。
func parseInvalidateMessage(payload string) (string, int, bool) {
	idx := strings.LastIndex(payload, "|")
	if idx <= 0 {
		return "", 0, false
	}
	roleName := payload[:idx]
	tenantPart := payload[idx+1:]
	if tenantPart == "" {
		return "", 0, false
	}
	tenantID := 0
	for _, c := range tenantPart {
		if c < '0' || c > '9' {
			return "", 0, false
		}
		tenantID = tenantID*10 + int(c-'0')
	}
	return roleName, tenantID, true
}

// InvalidateAuthzCaches 使某租户的策略目录（authz.Registry）与动态 ACL 缓存失效，
// 并向其它副本广播。权限策略写入后必须调用它，而不是只调 authz.Shared().Invalidate：
// 后者只清本进程，多副本下另一实例仍会按旧策略判定最长 5 分钟。
func InvalidateAuthzCaches(tenantID int) {
	authz.Shared().Invalidate(tenantID)
	InvalidateTenantACLCache(tenantID)
	broadcastAuthzCacheInvalidation(tenantID)
}

func broadcastAuthzCacheInvalidation(tenantID int) {
	if permissionBroadcaster == nil {
		return
	}
	msg := fmt.Sprintf("%d", tenantID)
	if err := permissionBroadcaster.Publish(context.Background(),
		authzCacheInvalidateChannel, msg).Err(); err != nil {
		zap.S().Warnw("策略缓存失效广播发布失败（其它实例将等TTL过期）",
			"tenant_id", tenantID, "error", err)
	}
}

func subscribeAuthzCacheInvalidation(client *redis.Client) {
	sub := client.Subscribe(context.Background(), authzCacheInvalidateChannel)
	defer sub.Close()

	for msg := range sub.Channel() {
		tenantID, ok := parseAuthzInvalidateMessage(msg.Payload)
		if !ok {
			continue
		}
		authz.Shared().Invalidate(tenantID)
		InvalidateTenantACLCache(tenantID)
		zap.S().Debugw("收到策略缓存失效广播，已清除本进程缓存", "tenant_id", tenantID)
	}
}

// parseAuthzInvalidateMessage 只接受纯数字租户号；拒绝空串和非数字载荷，
// 避免一条脏消息清掉整个进程的缓存。
func parseAuthzInvalidateMessage(payload string) (int, bool) {
	if payload == "" {
		return 0, false
	}
	tenantID := 0
	for _, c := range payload {
		if c < '0' || c > '9' {
			return 0, false
		}
		tenantID = tenantID*10 + int(c-'0')
	}
	return tenantID, true
}
