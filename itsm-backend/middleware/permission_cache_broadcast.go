package middleware

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// =============================================================================
// 权限缓存跨实例失效广播（Phase 1：P1-4）
//
// 问题：permissionCache 是进程内 map，InvalidateRolePermissionCache 只能
// 清除本进程缓存。多副本部署下，副本 A 改权限后，副本 B/C 需等 TTL（5 分钟）
// 自然过期才能感知，存在权限变更不一致窗口。
//
// 方案：本地失效后向 Redis 频道发布消息，各实例订阅并失效各自进程内缓存。
// 未配置 Redis（单副本）时自动降级为仅本进程失效，行为与既有完全一致。
// =============================================================================

const permissionCacheInvalidateChannel = "itsm:permission-cache:invalidate"

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
