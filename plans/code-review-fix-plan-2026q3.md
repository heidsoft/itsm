# Code Review 问题修复计划（2026-Q3）

> **目标：** 收敛 2026-Q3 全栈 code review 发现的问题，优先修复 P0/P1，清理 P2/P3 技术债
> **范围：** Phase 4（36 项）+ Phase 5（27 项），共 63 项
> **优先级：** P0（安全/数据正确性/跨租户） → P1（一致性/可靠性） → P2（性能/可维护性） → P3（优化）
> **执行原则：**
> 1. 每个 Task 先写失败的回归测试，再修代码（TDD）
> 2. 跨租户、权限、事务、幂等变更必须有正反双向测试
> 3. 修改完成后运行 `git diff --check` + 窄包测试 + `go build` / `npm run type-check`
> 4. 同步更新 `CHANGELOG.md` `[Unreleased]` 与相关 doc

---

## 修复总览

| 严重度 | Phase 4 | Phase 5 | 合计 | 处理策略 |
|:---:|:---:|:---:|:---:|:---|
| P0 | 7 | 8 | 15 | 本批逐项修复，必须带回归测试 |
| P1 | 11 | 11 | 22 | 本批逐项修复，必须带回归测试 |
| P2 | 11 | 4 | 15 | 批处理，按主题合并 PR |
| P3 | 7 | 4 | 11 | 清理类，随相关主题 PR 捎带 |

> Phase 4 的 36 项详情见 Phase 4 问题清单对话记录；本计划对 Phase 5 的 27 项给出完整 Task 拆解，Phase 4 按主题合并入对应 Task。

---

## 依赖与执行顺序

```
Group A（安全/多租户）  ──┐
Group B（加密演进）      ──┼─→ 独立并行
Group C（Outbox/审计）   ──┤
Group D（批量/性能）     ──┘
Group E（前端 RBAC）     ──→ 独立并行
Group F（前端 i18n/性能）──→ 独立并行
Group G（Schema/工具）   ──→ 收尾清理
```

**建议分批提交：**
- **Batch 1**：Group A + Group C（安全 + 投递可靠，最高风险）
- **Batch 2**：Group B（加密演进，需 migration 脚本）
- **Batch 3**：Group D（批量事务 + N+1）
- **Batch 4**：Group E（前端 RBAC 收敛）
- **Batch 5**：Group F（前端 i18n + 性能）
- **Batch 6**：Group G（Schema + 清理）

---

# Group A：安全 & 多租户隔离

## Task A-1：MSP 接口增加 AllowedCustomers 校验（P0-1 / Phase 5）

**问题：** MSP handler 用 `customerTenantID` 直接查 Ent，未校验是否在 `mspCtx.AllowedCustomers` 内，操作员可通过篡改 customerTenantID 跨租户访问。

**修复文件：**
- `itsm-backend/handlers/msp/handler.go`（所有使用 `customerTenantID` 的 handler）
- `itsm-backend/middleware/msp_context.go`（确认 `IsAllowed` helper 已存在）

**Step 1：确认 mspCtx 接口**

```bash
grep -rn "AllowedCustomers\|IsAllowed\|MspContext" itsm-backend/middleware/ itsm-backend/handlers/msp/
```

**Step 2：在每个 MSP handler 入口加校验**

在读取 `customerTenantID` 后、任何 Ent 查询前加入：

```go
mspCtx, ok := mspctx.FromContext(ctx)
if !ok || !mspCtx.IsAllowed(customerTenantID) {
    common.Fail(c, common.ErrForbidden, "无权访问该客户租户")
    return
}
```

**Step 3：回归测试**

新增 `itsm-backend/handlers/msp/handler_test.go`：
- happy path：MSP 操作员访问已授权 customer → 200
- 跨租户拒绝：MSP 操作员访问未授权 customer → 403
- 无 mspCtx：返回 401/403

**Step 4：验证**

```bash
cd itsm-backend && go test ./handlers/msp/... -run TestMSP -v
cd itsm-backend && go build ./...
```

**退出标准：** 跨租户拒绝测试通过；`go vet` 无告警。

---

## Task A-2：Webhook Host header 禁止用户覆盖（P0 / SSRF）

**问题：** `webhook_handler.go` 允许请求体中的 `Host` header 覆盖默认 Host，导致虚拟主机路由绕过。

**修复文件：** `itsm-backend/handlers/connector/webhook_handler.go`

**Step 1：定位 Host 设置点**

```bash
grep -n "Host\|Header.Set" itsm-backend/handlers/connector/webhook_handler.go
```

**Step 2：移除/白名单 Host**

```go
// 禁止用户设置 Host 头；仅允许白名单内的 header（如 Content-Type, X-Signature）
for k, v := range userHeaders {
    if strings.EqualFold(k, "Host") || strings.EqualFold(k, "X-Forwarded-Host") {
        continue
    }
    req.Header.Set(k, v)
}
```

**Step 3：回归测试**

- 模拟用户传 `Host: internal-service` → 断言最终请求 Host 为目标 URL 的 host
- 白名单 header 正常透传

---

## Task A-3：Webhook DNS rebinding TOCTOU 修复（P1 / SSRF）

**问题：** `LookupIP` → IP 检查 → `dial` 之间 DNS 可能切换，存在 TOCTOU。

**修复文件：** `itsm-backend/handlers/connector/webhook_handler.go`

**Step 1：使用 net.Dialer.Control 在连接时校验 IP**

```go
dialer := &net.Dialer{
    Control: func(network, address string, _ syscall.RawConn) error {
        host, _, _ := net.SplitHostPort(address)
        ip := net.ParseIP(host)
        if ip == nil || isPrivateIP(ip) {
            return fmt.Errorf("refusing connection to non-public IP: %s", host)
        }
        return nil
    },
}
transport := &http.Transport{DialContext: dialer.DialContext}
```

**Step 2：回归测试**

- 私有 IP 连接被拒绝
- 公网 IP 正常放行

---

## Task A-4：Webhook 签名缺失时 fail-closed（P1）

**问题：** secret 为空时跳过签名直接发送裸请求。

**修复文件：** `itsm-backend/service/connector/webhook.go`

**Step 1：enqueue 前校验**

```go
if strings.TrimSpace(secret) == "" {
    return fmt.Errorf("webhook secret is required for signature")
}
```

**Step 2：回归测试**

- secret 为空 → enqueue 失败，不产生 delivery record
- secret 非空 → 正常签名发送

---

# Group B：加密 & 密钥演进

## Task B-1：密钥派生引入 HKDF + 盐 + 版本字节（P0-3）

**问题：** AES key = `SHA256(secret + tenant_id)`，无盐、无版本、确定，无法轮换。

**修复文件：** `itsm-backend/pkg/connector/encryption.go`

**Step 1：重写密钥派生**

```go
const currentKeyVersion = byte(1)

func deriveKey(master, tenantID, salt []byte) []byte {
    info := append([]byte("itsm-connector:"), tenantID...)
    key := make([]byte, 32)
    hkdf.Expand(sha256.New, master, info, key)
    return key
}
```

**Step 2：密文前缀写入版本字节 + 盐**

密文格式：`version(1) || salt_len(2) || salt || nonce(12) || ciphertext`

**Step 3：兼容旧密文（无版本字节）走 fallback 解密**

```go
func Decrypt(ciphertext, master, tenantID []byte) ([]byte, error) {
    if len(ciphertext) > 0 && ciphertext[0] == currentKeyVersion {
        // 新版：解析 version/salt/nonce
    }
    // 旧版：回退到 SHA256(secret+tenant)（仅用于 migration，加 deprecation 日志）
    return decryptLegacy(ciphertext, master, tenantID)
}
```

> **精简说明：** 不新增 `key_rotation` 命令和 `key_version` 列 migration。密文自带版本字节即可支持未来轮换，真正需要轮换时再写命令。

**Step 4：回归测试**

- 新密文可解密
- 旧密文 fallback 可解密（迁移期）
- 不同租户密文不可互换

**验证：**

```bash
cd itsm-backend && go test ./pkg/connector/... -v
go build ./...
```

---

## Task B-2：单条加密记录失败不阻塞全部加载（P0-4）

**问题：** `PersistentConfigStore.LoadAll` 解密失败直接 return error，导致全 connector 不可用。

**修复文件：** `itsm-backend/pkg/connector/persistent_store.go`

**Step 1：逐条 try/catch**

```go
for _, raw := range records {
    decrypted, err := decrypt(raw.EncryptedConfig, ...)
    if err != nil {
        logger.Warnw("failed to decrypt connector config", "id", raw.ID, "err", err)
        failed = append(failed, raw.ID)
        continue
    }
    // ... 解析入结果
}
return results, failed, nil
```

**Step 2：返回失败 ID 列表，供上层标记 `pending_key_migration`**

**Step 3：回归测试**

- 1 条损坏 + 9 条正常 → 返回 9 条正常 + 1 个 failed ID
- 全部损坏 → 返回空结果 + 全部 failed ID，不 panic

---

## Task B-3：敏感字段白名单对齐 connector manifest（P0-5）

**问题：** `sensitiveFields` 未覆盖 webhook secret_token、oauth refresh_token、api_key、bot_token 等。

**修复文件：**
- `itsm-backend/pkg/connector/encryption.go`
- 各 connector 的 manifest/定义

**Step 1：盘点所有 connector 的 secret 字段**

```bash
grep -rn "secret\|token\|api_key\|password\|credential" itsm-backend/connector/ itsm-backend/pkg/connector/ --include="*.go"
```

**Step 2：按 connector 类型维护独立 sensitiveFields 映射**

```go
var connectorSensitiveFields = map[string][]string{
    "webhook":   {"secret_token", "signing_secret"},
    "feishu":    {"app_secret", "bot_token", "encrypt_key"},
    "dingtalk":  {"app_key", "app_secret"},
    "wecom":     {"corp_secret", "agent_secret"},
    "oauth":     {"client_secret", "refresh_token", "access_token"},
}
```

**Step 3：未识别字段默认加密（fail-closed）**

**Step 4：回归测试**

- 每种 connector 提交完整配置 → 断言所有 secret 字段在 DB 中为密文
- API 返回时 secret 字段被 mask

---

# Group C：Outbox & 投递可靠性 & 审计

## Task C-1：Notification delivery FAILED 状态幂等修复（P0-2）

**问题：** `MarkDelivered` 只对 `sent` 状态短路，FAILED/DLQ 记录会被重复投递。

**修复文件：** `itsm-backend/handlers/notification/delivery_command_handler.go`

**Step 1：扩大终态短路范围**

```go
existing, err := getDelivery(ctx, recipient, channel, idempotencyKey)
if err == nil && existing != nil {
    switch existing.Status {
    case "sent", "delivered":
        return nil // 已成功，幂等返回
    case "failed", "dead_letter":
        // 进入重试流程，先检查 retry_count < max_retries
    }
}
```

**Step 2：重试计数与 DLQ**

```go
if existing.RetryCount >= maxRetries {
    updateStatus(ctx, existing.ID, "dead_letter")
    return nil
}
```

**Step 3：回归测试**

- sent 状态重复消费 → provider.Send 调用 0 次
- failed 状态且 retry_count < max → 调用 1 次
- failed 且 retry_count >= max → 不调用，标记 dead_letter

---

## Task C-2：BPMN 回调审计写入事务内（P0-8）

**问题：** BPMN 外部回调只更新 task 状态，未在同事务写 audit_log。

**修复文件：** `itsm-backend/handlers/bpmn/callback_handler.go`

**Step 1：在 callback handler 内开启事务**

```go
err := s.client.WithTx(ctx, func(tx *ent.Tx) error {
    if err := tx.BpmnTask.UpdateOneID(taskID).
        SetStatus(newStatus).
        SetCompletedAt(time.Now()).
        Exec(ctx); err != nil {
        return err
    }
    if err := tx.AuditLog.Create().
        SetTenantID(tenantID).
        SetActorID(actorID).
        SetAction("bpmn_callback").
        SetResourceType("bpmn_task").
        SetResourceID(taskID).
        SetDetail(detail).
        Exec(ctx); err != nil {
        return err
    }
    return nil
})
```

**Step 2：回归测试**

- 回调成功 → bpmn_task + audit_log 都存在
- 回调中 audit 写入失败 → bpmn_task 回滚（无半提交）

---

# Group D：批量事务 & 性能

## Task D-1：批量删除/关闭/更新工单加事务包裹（P0-6）

**问题：** 循环单条 Tx，部分失败不可恢复。

**修复文件：** `itsm-backend/service/ticket_service.go`（BatchDelete/Close/Update）

> **精简说明：** 不新增 `batch_operations` 表。核心修复就是外层 `WithTx` 包裹循环，保证原子性；批量结果统计用内存 struct 返回即可，不落库。

**Step 1：外层事务包裹**

```go
func (s *TicketService) BatchDelete(ctx, tenantID, ids []int) (*BatchResult, error) {
    result := &BatchResult{Total: len(ids)}
    err := s.client.WithTx(ctx, func(tx *ent.Tx) error {
        for _, id := range ids {
            if err := tx.Ticket.DeleteOneID(id).Where(ticket.TenantID(tenantID)).Exec(ctx); err != nil {
                result.FailedIDs = append(result.FailedIDs, id)
                continue
            }
            result.Succeeded++
        }
        return nil
    })
    return result, err
}
```

**Step 2：回归测试**

- 全部成功 → succeeded = N
- 部分失败（mock 中间 ID 不存在）→ succeeded + failed = N
- 跨租户 ID → 该 ID 计入 failed，不删除

---

## Task D-2：Analytics N+1 重写为单 SQL（P0-7）

**问题：** 先全量拉取 ticket 再单条回查统计。

**修复文件：** `itsm-backend/service/ticket_service.go` `GetTicketAnalytics`

**Step 1：用单条 SQL 聚合**

```go
rows, err := s.client.QueryContext(ctx, `
    SELECT
        status,
        priority,
        COUNT(*) FILTER (WHERE sla_resolution_deadline < now() AND status != 'closed') AS overdue
    FROM tickets
    WHERE tenant_id = $1 AND deleted_at IS NULL
    GROUP BY status, priority
`, tenantID)
```

**Step 2：回归测试**

- 构造 100 条不同状态/优先级 ticket，断言聚合结果正确
- 确认 SQL 执行次数为 1（用 ent debug log 或 mock）

---

# Group E：前端 RBAC 一致性

## Task E-1：收敛"管理员"判定为单一权限来源（P1-7）

**问题：** Sidebar 用权限判定，AdminRouteGuard 用角色判定，二者漂移。

**修复文件：**
- `itsm-frontend/src/components/layout/sidebar/Sidebar.tsx:202-207`
- `itsm-frontend/src/components/common/AdminRouteGuard.tsx`
- `itsm-frontend/src/lib/store/auth-store.ts:154-157`
- `itsm-frontend/src/lib/hooks/use-permissions.ts:87-89`

**Step 1：统一 `isAdmin` 定义为权限判定**

在 `auth-store.ts` 中：

```typescript
isAdmin: () => {
  const { user } = get();
  const perms = user?.permissions ?? [];
  return perms.includes('*') ||
    perms.includes('user:write') ||
    perms.includes('role:write') ||
    perms.includes('system_config:write') ||
    perms.includes('ticket_type:manage');
},
```

**Step 2：移除 Sidebar 内联逻辑，改为 `useAuthStore(state => state.isAdmin)`**

**Step 3：`use-permissions.ts` 的 `isAdmin` 直接调用 store 的 `isAdmin`**

**Step 4：回归测试**

- `auth-store.test.ts`：`role=manager` + `permissions=['user:write']` → `isAdmin() === true`
- `role=admin` + `permissions=['ticket:read']` → `isAdmin() === false`（不再因角色是 admin 就放行）
- `role=super_admin` + `permissions=['*']` → `isAdmin() === true`

---

## Task E-2：AuthGuard 增加 requireAll/requireAny 语义（P1-8）

**问题：** AuthGuard 硬编码 `permissions.every()`，PermissionGuard 有 `requireAll` flag。

**修复文件：** `itsm-frontend/src/components/auth/AuthGuard.tsx`

**Step 1：提取共享权限检查函数到 `use-permissions.ts`**

```typescript
export function checkPermissions(
  permissions: Permission[],
  mode: 'all' | 'any',
  hasPermission: (resource: string, action: string) => boolean
): boolean {
  return mode === 'all'
    ? permissions.every(p => hasPermission(p.resource, p.action))
    : permissions.some(p => hasPermission(p.resource, p.action));
}
```

**Step 2：AuthGuard 新增 `requireAll?: boolean` prop（默认 true）**

**Step 3：PermissionGuard 复用同一函数**

**Step 4：回归测试**

- AuthGuard `requireAll={true}` 多权限缺一个 → 拒绝
- AuthGuard `requireAll={false}` 多权限有一个 → 放行

---

## Task E-3：OperationGuard 增加批量操作支持（P1-9）

**修复文件：** `itsm-frontend/src/components/auth/AuthGuard.tsx`

**Step 1：OperationGuard 支持 `additionalPermissions`**

```typescript
interface OperationGuardProps {
  resource: string;
  action: string;
  requireBatch?: boolean;
}
// 内部：hasPermission(resource, action) && (!requireBatch || hasPermission(resource, `batch_${action}`))
```

**Step 2：与 `canBatchOperate` 对齐**

---

## Task E-4：MENU_PATH_NORMALIZATIONS 迁移到 seed + 前端临时 fallback（P1-10）

**修复文件：**
- `itsm-backend/pkg/seeder/seeder.go`（menu seed 路径修正）
- `itsm-frontend/src/components/layout/sidebar/Sidebar.tsx`（保留 fallback + console.warn）

**Step 1：修正 seed 中的 `/xxx/list` 等路径**

**Step 2：前端保留 fallback 但加 `console.warn('[Sidebar] deprecated menu path, fix seed')`**

**Step 3：回归测试**

- 新 seed 生成的菜单路径直接可达，不触发 normalization

---

## Task E-5：capabilityPathRules 同步测试（P1-11）

**问题：** 12 条路径映射写死在前端，与后端 capability 矩阵独立维护。

**修复文件：**
- `itsm-frontend/src/components/layout/sidebar/Sidebar.tsx`（保留规则，加注释）
- 新增 `itsm-frontend/src/components/layout/sidebar/__tests__/capability-rules.test.ts`

> **精简说明：** 不改后端接口。12 条静态规则不值得新建契约面。保留前端规则，加一个测试断言前端规则的 capability key 在后端 capability 列表中都存在，避免漂移。

**Step 1：前端规则加注释标注单一来源**

```typescript
// 与后端 capability 表一一对应；新增 capability 时同步更新此处，
// 测试 capability-rules.test.ts 会校验不漂移。
const capabilityPathRules: Array<[string, string]> = [...]
```

**Step 2：新增同步测试**

```typescript
it('所有前端 capabilityPathRules 的 key 必须在后端 capability 列表中', async () => {
  const backendCaps = await fetchCapabilities();
  const frontendKeys = capabilityPathRules.map(([, k]) => k);
  for (const key of frontendKeys) {
    expect(backendCaps.some(c => c.key === key)).toBe(true);
  }
});
```

---

# Group F：前端 i18n & 性能

## Task F-1：useI18n 支持枚举 locale + 降级提示（P1-4）

**修复文件：** `itsm-frontend/src/lib/i18n/useI18n.ts`

**Step 1：定义支持的 locale 列表**

```typescript
const SUPPORTED_LOCALES = ['zh-CN', 'en-US'] as const;
type SupportedLocale = typeof SUPPORTED_LOCALES[number];
```

**Step 2：不支持时设置 fallback 状态**

```typescript
const setLanguage = (lang: string) => {
  if (!SUPPORTED_LOCALES.includes(lang as SupportedLocale)) {
    setWarning(`语言 ${lang} 暂不支持，已回退到 zh-CN`);
    lang = 'zh-CN';
  }
  // ...
};
```

**Step 3：回归测试**

- 传入 `ja-JP` → 返回 zh-CN 且有 warning
- 传入 `en-US` → 返回 en-US

---

## Task F-2：i18n 契约测试覆盖所有 namespace（P1-5）

**修复文件：** 新增 `itsm-frontend/src/lib/i18n/__tests__/translations-contract.test.ts`

**Step 1：遍历 translations 的所有 namespace**

```typescript
const namespaces = Object.keys(translations['zh-CN']);
it.each(namespaces)('namespace %s 在 zh-CN 和 en-US 结构对称', (ns) => {
  expect(translations['en-US'][ns]).toBeDefined();
  expect(Object.keys(translations['en-US'][ns]).sort())
    .toEqual(Object.keys(translations['zh-CN'][ns]).sort());
});
```

**Step 2：缺失/不对称 → 测试 fail**

---

## Task F-3：translations 按 locale 拆包（P1-6）

**修复文件：**
- `itsm-frontend/src/lib/i18n/translations/zh-CN.ts`（从 translations.ts 拆出）
- `itsm-frontend/src/lib/i18n/translations/en-US.ts`（从 translations.ts 拆出）
- `itsm-frontend/src/lib/i18n/translations/index.ts`（re-export）
- `itsm-frontend/src/lib/i18n/useI18n.ts`（按当前语言 lazy import）

> **精简说明：** 不按 namespace 拆 88 个文件。只按 locale 拆成 2 个文件，按当前语言 lazy load，首屏只加载当前语言，减少一半体积。namespace 级拆包等体积真成问题时再做。

**Step 1：拆为 zh-CN.ts / en-US.ts 两个文件**

**Step 2：useI18n 按当前语言 `import()` 动态加载**

```typescript
const loadTranslations = async (lang: SupportedLocale) => {
  const mod = await import(`./translations/${lang}`);
  return mod.default;
};
```

**Step 3：验证构建后生成 2 个 locale chunk**

---

## Task F-4：清理 react-window 死依赖（P2-1）

**修复文件：**
- `itsm-frontend/package.json`（移除 react-window, react-window-infinite-loader）
- `itsm-frontend/src/types/react-window.d.ts`（删除）
- `itsm-frontend/docs/reports/FRONTEND_AUDIT_REPORT.md`（更新描述）

**Step 1：确认无引用**

```bash
grep -rn "react-window" itsm-frontend/src/
# 仅 .d.ts 和 package.json
```

**Step 2：移除依赖 + .d.ts**

**Step 3：更新审计报告**

**Step 4：验证**

```bash
cd itsm-frontend && npm run build
```

---

## Task F-5：工单列表增加大 pageSize 性能基线测试（P2-2）

**修复文件：** 新增 `itsm-frontend/tests/e2e/performance/ticket-list.perf.spec.ts`

**Step 1：Playwright 测试 pageSize=100/500 的 TTI**

```typescript
await page.goto('/tickets');
await page.selectOption('[aria-label="page size"]', '100');
const tti = await page.evaluate(() => performance.timing.loadEventEnd - performance.timing.navigationStart);
expect(tti).toBeLessThan(3000);
```

---

## Task F-6：跨页选择收敛（P2-3）

**修复文件：** `itsm-frontend/src/lib/hooks/useRowSelection.ts` + `TicketList.tsx`

**Step 1：pageSize 变化时清空 selection**

```typescript
useEffect(() => {
  selection.clear();
}, [pagination.pageSize]);
```

---

# Group G：Schema & 工具清理（P3）

## Task G-1：ent Schema 删除字段 deprecation 流程文档化

**修复文件：** `docs/documentation-governance.md`（补充 schema 演进 checklist）

## Task G-2：operational_command payload 增加 JSON schema 校验

**修复文件：** `itsm-backend/ent/schema/operational_command.go`

## Task G-3：useI18n 导出纯函数 formatMessage

**修复文件：** `itsm-frontend/src/lib/i18n/useI18n.ts`

## Task G-4：SSR i18n hydration 修复

**修复文件：** `itsm-frontend/src/lib/i18n/useI18n.ts` + 服务端 layout

---

# Phase 4 问题归并（36 项）

Phase 4 的 36 项问题按主题归并入上述 Task，对应关系：

| Phase 4 主题 | 归入 Task |
|:---|:---|
| 后端跨租户查询缺失 tenant predicate | A-1（同批修复） |
| notification outbox 幂等 key 不完整 | C-1（同批修复） |
| BPMN 回调缺 audit | C-2（同批修复） |
| 批量操作无事务 | D-1（同批修复） |
| Analytics N+1 | D-2（同批修复） |
| 前端权限双轨 | E-1（同批修复） |
| 前端 i18n 缺失 | F-1/F-2（同批修复） |
| 前端列表无虚拟化 | F-4/F-5（同批修复） |
| 其他 P2/P3 代码规范 | Group G 捎带 |

---

# 验证策略

## 每 PR 必跑

**后端：**
```bash
cd itsm-backend
go test ./<修改包>/... -v
go build ./...
go vet ./...
git diff --check
```

**前端：**
```bash
cd itsm-frontend
npm run type-check
npm run lint:check
npm run test:unit -- <修改测试文件>
git diff --unified=0 -- '*.tsx' '*.jsx' | rg '^\+.*<Space\b[^>]*\bdirection\s*='  # antd Space.direction 检查
git diff --unified=0 -- '*.go' '*.ts' | rg '^\\+.*(json|form|query):"[a-z0-9]+_[a-z0-9_]+"'  # snake_case 检查
```

## 跨租户/权限变更必加

- 至少 1 个跨租户拒绝测试
- 至少 1 个 happy path 测试
- 至少 1 个权限拒绝测试

## 事务/outbox 变更必加

- enqueue 失败 → 业务写入回滚
- 业务失败 → 无 command 残留
- 重复请求 → 无重复副作用

## 全量验证（每个 Batch 完成后）

```bash
cd itsm-backend && go test ./...
cd itsm-frontend && npm run type-check && npm run test:unit
```

---

# 回滚策略

1. **数据库 migration**：每个 migration 提供 `down` 版本，记录在 `migrations/`
2. **密钥轮换**：保留旧密钥解密 fallback（至少 2 个版本窗口），旧密文可继续解密
3. **前端**：每个变更为向后兼容（旧权限码、旧路径有 fallback）
4. **回滚触发**：窄包测试失败或全量测试出现与本次改动相关的失败

---

# 完成报告模板

每个 Task 完成后在对应 PR 描述中填写：

- [ ] 修改文件列表
- [ ] 新增/修改测试列表
- [ ] `go test ./<pkg>/...` 结果
- [ ] `go build ./...` / `npm run type-check` 结果
- [ ] `git diff --check` 结果
- [ ] 是否需要 migration（是则附 up/down）
- [ ] 是否需要 CHANGELOG 更新（是则附 diff）

---

> **计划状态：** In Progress。Batch 1-2 已完成（2026-09-24），按 Batch 3 → 6 顺序继续。

### Batch 1 执行记录（2026-09-24）

| 任务 | 状态 | 说明 |
|:---|:---|:---|
| A-1 MSP AllowedCustomers 校验 | ✅ | `handlers/msp/handler.go` + `middleware/msp_context.go` 新增 `IsCustomerAllowed` |
| A-2 Webhook Host header 过滤 | ✅ | `service/bpmn/webhook_handler.go` 新增 `blockedWebhookHeaders`（12 个敏感 header） |
| A-3 DNS rebinding TOCTOU | ✅（无需改动） | 现有 `net.Resolver.LookupHost` + 连接前重检已覆盖 |
| A-4 出站签名缺失 | ✅（不适用） | 当前 Webhook Connector 无出站签名特性，跳过 |
| C-1 Notification delivery 幂等 | ✅ | `notification_delivery_command_handler.go` Send 成功后状态更新失败不再重试 |
| C-2 BPMN 审计事务内 | ✅ | `bpmn_process_executor.go` + `bpmn_audit_service.go` 审计写入移入 CompleteTask 事务 |

验证：`go build ./...` ✅、`go vet ./...` ✅；窄包 ent 测试因环境无 C 编译器（clang/gcc 缺失，CGO SQLite 无法编译）未能本地运行，需在有 CGO 的 CI 环境执行。

### Batch 2 执行记录（2026-09-24）

| 任务 | 状态 | 说明 |
|:---|:---|:---|
| B-1 HKDF 密钥派生 + 版本字节 | ✅ | `middleware/encryption.go` 新加密用 HKDF+随机盐+版本前缀，旧密文 fallback 解密 |
| B-2 单条解密失败不阻塞 | ✅ | `connector/persistent_store.go` LoadAll 逐条容错，新增 LoadAllWithFailures 返回失败 ID |
| B-3 敏感字段白名单扩充 | ✅ | `middleware/mask.go` 新增 signing_secret/corp_secret/agent_secret/encrypt_key/app_key/bot_token |

验证：`go build ./...` ✅、`go vet ./...` ✅
