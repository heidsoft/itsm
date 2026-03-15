# 前端编辑/删除404问题审查报告

## 审查结论

经过全面审查，后端路由和控制器实现均完整（CMDB、知识库、工单的 PUT/DELETE 端点都已正确注册）。404 问题来自以下几个具体原因，已全部修复。

---

## 已修复问题

### 问题1：知识库路由 `/categories` 被 `/:id` 捕获（已修复）

**文件**：`itsm-backend/router/router.go`

**原因**：`/knowledge-articles/categories` 静态路由注册在 `/:id` 动态路由之后，Gin 会将 `categories` 当作 `:id` 参数处理，导致查询 ID 为 `categories` 的文章，返回 404。

**修复**：将 `/categories` 路由移到 `/:id` 之前注册。

```go
// 修复前（错误）
kbGrp.GET("/:id", ...)
kbGrp.GET("/categories", ...)  // 被 /:id 捕获

// 修复后（正确）
kbGrp.GET("/categories", ...)  // 静态路由先注册
kbGrp.GET("/:id", ...)
```

---

### 问题2：知识库编辑页 `categoryId` 字段映射错误（已修复）

**文件**：`itsm-frontend/src/app/(main)/knowledge/articles/[id]/edit/page.tsx`

**原因**：编辑页提交时使用 `categoryId` 字段，HTTP 客户端将其转为 snake_case 的 `category_id`，但后端 `UpdateKnowledgeArticleRequest` DTO 期望的字段是 `category`，导致分类字段更新失效。

```typescript
// 修复前（错误）
await KnowledgeBaseApi.updateArticle(id, {
  categoryId: values.category,  // 转为 category_id，后端不识别
});

// 修复后（正确）
await KnowledgeBaseApi.updateArticle(id, {
  category: values.category,  // 直接传 category
});
```

---

### 问题3：工单"编辑"按钮跳转到不存在的路由（已修复）

**文件**：
- `itsm-frontend/src/components/ticket/TicketList.tsx`
- `itsm-frontend/src/components/ticket/TicketKanban.tsx`

**原因**：两个组件的"编辑"菜单项都跳转到 `/tickets/${id}/edit`，但该路由页面根本不存在（`/tickets/[ticketId]/` 下只有 `page.tsx`，没有 `edit/` 子目录），导致 404。

工单详情页 `[ticketId]/page.tsx` 已内置编辑弹窗（Modal），无需独立编辑页。

```typescript
// 修复前（错误）
onClick: () => router.push(`/tickets/${record.id}/edit`)  // 404

// 修复后（正确）
onClick: () => router.push(`/tickets/${record.id}`)  // 跳转详情页，内有编辑弹窗
```

---

## 后端路由完整性确认

审查确认以下路由均已正确注册，无404问题：

| 模块 | PUT | DELETE | 备注 |
|------|-----|--------|------|
| 工单 `/api/v1/tickets/:id` | ✅ | ✅ | 走 TicketController |
| CMDB `/api/v1/cmdb/cis/:id` | ✅ | ✅ | 走 DDD CMDBHandler |
| 知识库 `/api/v1/knowledge-articles/:id` | ✅ | ✅ | 走 DDD KnowledgeHandler |
| 事件 `/api/v1/incidents/:id` | ✅ | — | 走 DDD IncidentHandler |
| 问题 `/api/v1/problems/:id` | ✅ | ✅ | 走 DDD ProblemHandler |
| 变更 `/api/v1/changes/:id` | ✅ | — | 走 DDD ChangeHandler |

**注意**：`CMDBController`（`/api/v1/configuration-items/`）路由缺少 PUT/DELETE，但前端 `CMDBApi` 实际调用的是 `/api/v1/cmdb/cis/`（DDD Handler），不受影响。

---

## 其他发现（不影响404，但需关注）

1. **`CMDBController` 路由不完整**：`/api/v1/configuration-items/:id` 只有 GET，没有 PUT/DELETE。虽然前端不调用这个路径，但建议补全或移除该旧控制器，避免混淆。

2. **知识库 `UpdateArticleRequest` 类型定义**：前端 `types/knowledge-base.ts` 中 `CreateArticleRequest` 使用 `categoryId?: string`，与后端 DTO 的 `category` 字段不一致。建议统一类型定义。

3. **工单详情页无独立编辑路由**：工单编辑通过详情页内的 Modal 实现，这是合理的设计，但需确保所有入口都跳转到详情页而非 `/edit` 路径。
