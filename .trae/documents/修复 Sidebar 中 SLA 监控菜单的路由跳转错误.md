# SLA 监控页面 404 问题修复计划

## 问题分析
用户访问 `http://localhost:3000/sla` 出现 404 错误。
经检查 Sidebar 配置，发现“SLA监控”菜单项的配置如下：
```typescript
{
  key: '/sla',
  icon: <Calendar style={iconStyle} />,
  label: 'SLA监控',
  path: '/sla-dashboard',
  permission: 'sla:view',
  description: t('sla.description'),
},
```
- `key` 是 `/sla`，导致点击时 `router.push('/sla')`。
- 但 `path` 是 `/sla-dashboard`，且文件系统中存在 `src/app/(main)/sla-dashboard/page.tsx`。
- 这意味着点击菜单会跳转到不存在的 `/sla` 路由，而不是预期的 `/sla-dashboard`。

## 修复计划
修改 `src/components/layout/Sidebar.tsx` 文件，将 SLA 监控菜单项的 `key` 从 `/sla` 修改为 `/sla-dashboard`，使其与实际页面路由一致。

## 具体修改
- **文件**: `src/components/layout/Sidebar.tsx`
- **改动**:
    ```typescript
    // Before
    key: '/sla',
    
    // After
    key: '/sla-dashboard',
    ```

这将修复点击菜单时的 404 错误，并确保路由高亮正确。