# Ant Design App 组件迁移指南

## 问题说明

浏览器控制台警告：
```
Warning: [antd: message] Static function can not consume context like dynamic theme. 
Please use 'App' component instead.
```

这是因为直接使用 `message.success()` 等静态方法无法访问动态主题上下文。

## 解决方案

### 1. 已完成的配置 ✅

在 `src/app/lib/providers/AntdProvider.tsx` 中已经正确配置了 `App` 组件：

```typescript
<ConfigProvider theme={antdTheme} locale={zhCN}>
  <App>
    {children}
  </App>
</ConfigProvider>
```

### 2. 创建的工具 Hooks ✅

已创建三个工具 hooks：

- `src/hooks/useMessage.ts` - 消息提示
- `src/hooks/useModal.ts` - 模态框
- `src/hooks/useNotification.ts` - 通知提醒

### 3. 迁移步骤

#### 方法 A: 使用 App.useApp() (推荐)

```typescript
// 修改前
import { message } from 'antd';

export const MyComponent = () => {
  const handleClick = () => {
    message.success('操作成功');
  };
  // ...
};

// 修改后
import { App } from 'antd';

export const MyComponent = () => {
  const { message } = App.useApp();
  
  const handleClick = () => {
    message.success('操作成功');
  };
  // ...
};
```

#### 方法 B: 使用自定义 Hook

```typescript
// 修改前
import { message } from 'antd';

export const MyComponent = () => {
  const handleClick = () => {
    message.success('操作成功');
  };
  // ...
};

// 修改后
import { useMessage } from '@/hooks/useMessage';

export const MyComponent = () => {
  const message = useMessage();
  
  const handleClick = () => {
    message.success('操作成功');
  };
  // ...
};
```

### 4. 需要迁移的组件列表

#### 高优先级（用户可见）

- [x] `TicketList.tsx` - 已使用 App.useApp()
- [ ] `TicketKanban.tsx` - 需要迁移
- [ ] `TicketDetail.tsx` - 已使用 App.useApp()
- [ ] `TicketSubtasks.tsx` - 已使用 App.useApp()
- [ ] `TicketDependencyManager.tsx` - 已使用 App.useApp()
- [ ] `SLAViolationMonitor.tsx` - 需要迁移
- [ ] `UserList.tsx` - 需要迁移
- [ ] `notifications/page.tsx` - 需要迁移

#### 中优先级（管理功能）

- [ ] `FieldDesigner.tsx` - 需要迁移
- [ ] `ChangeRollbackPlan.tsx` - 需要迁移
- [ ] 其他管理页面组件

#### 低优先级（测试文件）

测试文件中的 message mock 不需要修改，因为它们已经被 mock 了。

### 5. 批量迁移脚本

对于需要迁移的组件，按以下步骤操作：

1. **添加导入**：
   ```typescript
   import { App } from 'antd';
   ```

2. **移除旧导入**：
   ```typescript
   // 删除或修改
   import { message } from 'antd';
   // 改为
   import { /* 其他导入 */ } from 'antd';
   ```

3. **在组件内部获取 message**：
   ```typescript
   export const MyComponent = () => {
     const { message } = App.useApp();
     // ...
   };
   ```

### 6. 特殊情况处理

#### 非组件函数中使用 message

如果在非 React 组件的函数中使用 message（如工具函数、API 调用），有两种方案：

**方案 A: 将 message 作为参数传递**
```typescript
// utils.ts
export const handleError = (error: Error, message: MessageInstance) => {
  message.error(error.message);
};

// Component.tsx
const { message } = App.useApp();
handleError(error, message);
```

**方案 B: 保留静态方法（临时方案）**
```typescript
// 对于无法轻易重构的代码，可以暂时保留静态方法
// 但会有警告
import { message } from 'antd';
message.success('操作成功');
```

### 7. 验证迁移

迁移完成后，检查浏览器控制台：
- ✅ 不应该再有 `[antd: message]` 警告
- ✅ message/modal/notification 功能正常
- ✅ 主题切换时样式正确

### 8. 最佳实践

1. **优先使用 App.useApp()**
   - 更直接，减少额外的 hook 层
   - 可以同时获取 message、modal、notification

2. **组件顶层获取**
   ```typescript
   export const MyComponent = () => {
     const { message, modal, notification } = App.useApp();
     // 在整个组件中使用
   };
   ```

3. **避免在循环或条件中调用**
   ```typescript
   // ❌ 错误
   if (condition) {
     const { message } = App.useApp();
   }
   
   // ✅ 正确
   const { message } = App.useApp();
   if (condition) {
     message.success('...');
   }
   ```

### 9. 迁移检查清单

- [ ] 所有组件都使用 App.useApp() 或自定义 hook
- [ ] 移除了所有静态 message/modal/notification 导入
- [ ] 浏览器控制台无警告
- [ ] 功能测试通过
- [ ] 主题切换测试通过

---

## 快速修复命令

对于简单的组件，可以使用以下模式快速修复：

```bash
# 1. 添加 App 导入
# 2. 移除 message 从 antd 导入
# 3. 在组件内添加 const { message } = App.useApp();
```

---

**更新时间**: 2026-02-10  
**状态**: 进行中  
**预计完成**: 今天
