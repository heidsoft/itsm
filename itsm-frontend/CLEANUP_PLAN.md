# ITSM 前端代码清理计划

本文档列出了需要清理的遗留代码和文件，以及清理后的现代化架构。

## 🗑️ 需要清理的遗留文件

### 1. 重复的 API 文件
以下 API 文件存在功能重复，应该整合到统一的现代 API 架构中：

```
# 遗留的 ticket API 文件 (应删除)
├── lib/api/ticket-api.ts (OLD - 替换为新的 lib/api/ticket.ts)
├── lib/api/ticket-api-enhanced.ts (功能已合并)
├── lib/api/ticket-attachment-api.ts (合并到 ticket.ts)
├── lib/api/ticket-comment-api.ts (合并到 ticket.ts)
├── lib/api/ticket-assignment-api.ts (合并到 ticket.ts)
├── lib/api/ticket-approval-api.ts (合并到 ticket.ts)
├── lib/api/ticket-notification-api.ts (合并到 ticket.ts)
├── lib/api/ticket-rating-api.ts (合并到 ticket.ts)
├── lib/api/ticket-analytics-api.ts (合并到 ticket.ts)
├── lib/api/ticket-prediction-api.ts (合并到 ticket.ts)
├── lib/api/ticket-root-cause-api.ts (合并到 ticket.ts)
├── lib/api/ticket-relations-api.ts (合并到 ticket.ts)
├── lib/api/ticket-view-api.ts (合并到 ticket.ts)
├── lib/api/ticket-automation-rule-api.ts (合并到 ticket.ts)
└── modules/ticket/api/ticket-api.ts (功能重复)

# 遗留的统一 API 文件 (功能分散，设计混乱)
├── lib/api/api-unified.ts (DELETE)
├── lib/api/api-unified-v2.ts (DELETE)
├── lib/api/base-api-handler.ts (替换为 client.ts)
├── lib/api/base-api.ts (替换为 client.ts)
└── app/lib/api-config.ts (功能重复)
```

### 2. 重复的状态管理文件
存在多套状态管理实现，应统一为现代化的 Zustand 架构：

```
# 遗留的 store 文件 (应删除)
├── lib/stores/ticket-store.ts (OLD - 保留但标记弃用)
├── modules/ticket/store/ticket-store.ts (功能重复)
├── app/lib/store/ticket-store.ts (功能重复)
├── app/lib/store/ticket-data-store.ts (功能重复)
├── app/lib/store/ticket-filter-store.ts (功能重复)
├── app/lib/store/ticket-ui-store.ts (功能重复)
├── lib/stores/base-store.ts (功能重复)
└── app/lib/store.ts (混合实现，需重构)
```

### 3. 重复的设计系统文件
存在多套设计系统实现：

```
# 遗留设计系统 (应删除)
├── lib/design-system.ts (OLD)
├── lib/design-system/colors.ts (已合并到 theme/index.ts)
├── lib/design-system/spacing.ts (已合并到 theme/index.ts)
├── lib/design-system/theme.tsx (已合并到 theme/components.tsx)
├── lib/antd-theme.ts (不再使用 Ant Design)
└── components/ui/InteractionPatterns.tsx (功能重复)
```

### 4. 遗留组件文件
以下组件使用旧的设计模式或功能重复：

```
# 遗留 UI 组件 (应删除或重构)
├── components/ui/LoadingSkeleton.tsx (替换为 ResponsiveLayout 中的 LoadingState)
├── components/ui/LoadingEmptyError.tsx (功能重复)
├── components/ui/LoadingEmptyError.example.tsx (示例文件)
├── components/ui/VirtualList.tsx (功能重复，已有现代实现)
├── components/ui/NotificationContainer.tsx (功能重复)
├── components/ui/Input.tsx (OLD - 替换为 theme/components.tsx 中的 Input)
├── components/ui/Select.tsx (OLD)
├── components/ui/Badge.tsx (OLD - 替换为 theme/components.tsx 中的 Badge)
├── components/ui/Modal.tsx (OLD)
├── components/ui/Toast.tsx (OLD)
└── components/ui/__tests__/UnifiedTable.test.tsx (测试文件需要更新)

# 遗留表单组件 (应删除)
├── components/forms/FormInput.tsx (OLD)
├── components/forms/FormTextarea.tsx (OLD)
└── components/forms/form-*.tsx (一套完整的旧表单系统)

# 遗留布局组件 (应删除)
├── components/layout/ErrorBoundary.tsx (替换为 ResponsiveLayout 中的 ErrorBoundaryFallback)
├── components/layout/AuthGuard.tsx (功能重复)
└── components/layout/auth-guard.tsx (小写版本)

# 遗留业务组件 (标记为 TODO，需要重构)
└── components/business/*.tsx (大部分包含 TODO 标记，需要使用新的设计模式重构)
```

### 5. 工具文件清理
一些工具文件功能重复或过时：

```
# 遗留工具文件
├── lib/component-utils.ts (功能重复)
├── lib/formatters.ts (功能重复)
├── app/lib/user-preferences.ts (功能重复)
├── app/lib/cmdb-relations.ts (业务逻辑应在 domain 层)
├── app/lib/ai-service.ts (功能重复)
├── app/lib/mock-data.ts (开发数据，生产不需要)
├── lib/hooks/useCache.ts (功能重复)
├── lib/hooks/useResponsive.ts (替换为 theme/components.tsx 中的 useBreakpoint)
└── lib/hooks/usePerformance.ts (功能重复)
```

## ✅ 现代化架构结构

清理后的推荐目录结构：

```
src/
├── lib/
│   ├── api/
│   │   ├── client.ts                    # ✅ 统一 API 客户端
│   │   ├── ticket.ts                    # ✅ 现代化 Ticket API
│   │   ├── incident.ts                  # ✅ 事件 API
│   │   ├── change.ts                    # ✅ 变更 API
│   │   ├── service-request.ts          # ✅ 服务请求 API
│   │   └── auth.ts                     # ✅ 认证 API
│   ├── stores/
│   │   ├── modern-ticket-store.ts      # ✅ 现代化状态管理
│   │   ├── auth-store.ts               # ✅ 认证状态
│   │   └── ui-store.ts                 # ✅ UI 状态
│   ├── theme/
│   │   ├── index.ts                    # ✅ 设计系统主文件
│   │   └── components.tsx              # ✅ 主题组件
│   └── utils.ts                        # ✅ 通用工具函数
├── components/
│   ├── layout/
│   │   └── ResponsiveLayout.tsx        # ✅ 现代布局组件
│   ├── tickets/
│   │   └── ModernTicketList.tsx        # ✅ 现代票据列表
│   └── forms/
│       └── (新的表单组件系统)
└── types/
    ├── api.ts                          # ✅ API 类型定义
    ├── ticket.ts                       # ✅ 票据类型
    └── common.ts                       # ✅ 通用类型
```

## 🔧 清理步骤

### 第一阶段：删除明显的重复文件

```bash
# 删除重复的 ticket API 文件
rm src/lib/api/ticket-api-enhanced.ts
rm src/lib/api/ticket-attachment-api.ts
rm src/lib/api/ticket-comment-api.ts
rm src/lib/api/ticket-assignment-api.ts
rm src/lib/api/ticket-approval-api.ts
rm src/lib/api/ticket-notification-api.ts
rm src/lib/api/ticket-rating-api.ts
rm src/lib/api/ticket-analytics-api.ts
rm src/lib/api/ticket-prediction-api.ts
rm src/lib/api/ticket-root-cause-api.ts
rm src/lib/api/ticket-relations-api.ts
rm src/lib/api/ticket-view-api.ts
rm src/lib/api/ticket-automation-rule-api.ts

# 删除遗留的统一 API 文件
rm src/lib/api/api-unified.ts
rm src/lib/api/api-unified-v2.ts
rm src/lib/api/base-api-handler.ts
rm src/lib/api/base-api.ts

# 删除重复的状态管理文件
rm src/modules/ticket/store/ticket-store.ts
rm src/app/lib/store/ticket-data-store.ts
rm src/app/lib/store/ticket-filter-store.ts
rm src/app/lib/store/ticket-ui-store.ts
rm src/lib/stores/base-store.ts

# 删除遗留设计系统
rm src/lib/design-system.ts
rm -rf src/lib/design-system/
rm src/lib/antd-theme.ts
```

### 第二阶段：更新导入引用

1. 搜索所有文件中对已删除文件的引用
2. 更新导入路径指向新的现代化文件
3. 更新类型引用

### 第三阶段：重构标记为 TODO 的组件

1. 使用新的设计系统重构 `components/business/` 下的组件
2. 应用 DDD 模式和现代化状态管理
3. 移除内联样式，使用主题系统

## 📊 清理效果预估

清理前后的文件对比：

| 类别 | 清理前文件数 | 清理后文件数 | 减少比例 |
|------|-------------|-------------|----------|
| API 文件 | 47 | 12 | 74% |
| Store 文件 | 12 | 4 | 67% |
| 设计系统 | 8 | 2 | 75% |
| UI 组件 | 25 | 8 | 68% |
| 总计 | 92 | 26 | 72% |

## 🎯 清理后的优势

1. **代码维护性提升**：减少 72% 的冗余文件
2. **开发效率提升**：统一的 API 和状态管理模式
3. **类型安全**：完整的 TypeScript 支持
4. **性能优化**：现代化的状态管理和组件设计
5. **团队协作**：统一的设计系统和代码规范

## ⚠️ 注意事项

1. 在删除文件前，确保所有引用都已更新
2. 保留必要的业务逻辑，避免功能丢失
3. 逐步迁移，确保系统稳定性
4. 更新相关的测试文件
5. 更新文档和类型定义

## 🔍 验证清理效果

清理完成后，进行以下验证：

1. **编译检查**：`npm run build` 无错误
2. **类型检查**：`npm run type-check` 通过
3. **代码规范**：`npm run lint` 通过
4. **功能测试**：核心功能正常运行
5. **性能测试**：页面加载速度提升