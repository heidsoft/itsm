package middleware

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// AUDIT-022：此前只有角色权限缓存有跨实例广播，authz.Registry（策略目录）
// 与动态 ACL 缓存都只清本进程。多副本下一次策略变更最长 5 分钟不一致。
// 本测试锁住消息解析的 fail-closed 与本地失效端口；真实 Redis 端到端需要
// 独立环境，见下方 env-gated 用例。

func TestParseAuthzInvalidateMessage(t *testing.T) {
	cases := []struct {
		payload string
		want    int
		ok      bool
	}{
		{"12", 12, true},
		{"1", 1, true},
		{"", 0, false},
		{"12|x", 0, false},
		{"-1", 0, false},
		{" 12", 0, false},
	}
	for _, tc := range cases {
		got, ok := parseAuthzInvalidateMessage(tc.payload)
		require.Equal(t, tc.ok, ok, "payload=%q", tc.payload)
		if tc.ok {
			require.Equal(t, tc.want, got, "payload=%q", tc.payload)
		}
	}
}

func TestInvalidateAuthzCaches_ClearsLocalACLCache(t *testing.T) {
	const tenantID = 987654
	aclCacheLock.Lock()
	aclCache[tenantID] = &cachedACL{expiresAt: time.Now().Add(time.Hour)}
	aclCacheLock.Unlock()
	t.Cleanup(func() {
		aclCacheLock.Lock()
		delete(aclCache, tenantID)
		aclCacheLock.Unlock()
	})

	aclCacheLock.RLock()
	_, exists := aclCache[tenantID]
	aclCacheLock.RUnlock()
	require.True(t, exists, "夹具必须先写入缓存")

	// 未配置 Redis 时广播是 no-op，本进程失效仍必须发生。
	require.NotPanics(t, func() { InvalidateAuthzCaches(tenantID) })

	aclCacheLock.RLock()
	_, exists = aclCache[tenantID]
	aclCacheLock.RUnlock()
	require.False(t, exists, "策略缓存失效必须同时清掉本进程的 ACL 缓存")
}

// 权限停用属于人工 RBAC 判定输入（role→permission 缓存）。此前写入路径只清
// Registry 与 ACL，人类用户在 TTL 窗口内仍能通过 RequirePermission；这里锁住
// 「一次失效清三类缓存」的契约。反向验证：把 invalidateTenantAuthzCachesLocal
// 改回只清后两类，本测试必须红在断言上（roleCache 仍在）。
func TestInvalidateAuthzCaches_ClearsRolePermissionCache(t *testing.T) {
	const tenantID = 987655
	const roleName = "cache_invalidate_probe_role"
	cacheKey := roleName + "_" + strconv.Itoa(tenantID)

	cacheRolePermissions(cacheKey, []Permission{{Resource: "ticket", Action: "read"}}, nil)
	t.Cleanup(func() {
		permissionCacheLock.Lock()
		delete(permissionCache, cacheKey)
		permissionCacheLock.Unlock()
	})

	permissionCacheLock.RLock()
	_, exists := permissionCache[cacheKey]
	permissionCacheLock.RUnlock()
	require.True(t, exists, "夹具必须先写入角色权限缓存")

	InvalidateAuthzCaches(tenantID)

	permissionCacheLock.RLock()
	_, exists = permissionCache[cacheKey]
	permissionCacheLock.RUnlock()
	require.False(t, exists, "策略失效必须同时清掉角色权限缓存（本副本立即撤权）")
}

// 真实 Redis 端到端：证明策略写入会向独立频道发布租户号，另一副本可据此失效。
// 未设置 ITSM_TEST_REDIS_ADDR 时跳过（单副本无需广播）。
func TestAuthzCacheBroadcast_RedisPublish(t *testing.T) {
	addr := os.Getenv("ITSM_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set ITSM_TEST_REDIS_ADDR to a throwaway Redis to run the broadcast e2e")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Ping(ctx).Err())

	sub := client.Subscribe(ctx, authzCacheInvalidateChannel)
	t.Cleanup(func() { _ = sub.Close() })
	_, err := sub.Receive(ctx)
	require.NoError(t, err, "订阅必须先就绪")

	ConfigurePermissionCacheBroadcast(client)
	InvalidateAuthzCaches(4242)

	select {
	case msg := <-sub.Channel():
		require.Equal(t, "4242", msg.Payload)
	case <-ctx.Done():
		t.Fatal("未收到策略缓存失效广播")
	}
}
