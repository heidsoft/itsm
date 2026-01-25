# 知识库 404 错误修复计划

## 问题确认
1.  **文件系统**: 存在 `src/app/(main)/knowledge/page.tsx`，说明正确的路由是 `/knowledge`。
2.  **Sidebar 配置**: 在 `src/components/layout/Sidebar.tsx` 中，知识库菜单项的配置如下：
    ```typescript
    {
      key: '/knowledge-base',
      icon: <HelpCircle style={iconStyle} />,
      label: '知识库',
      path: '/knowledge',
      permission: 'knowledge:view',
      description: t('knowledgeBase.description'),
    },
    ```
    虽然 `path` 是正确的 `/knowledge`，但 `key` 是 `/knowledge-base`。
3.  **跳转逻辑**: `handleMenuClick` 函数使用了 `key` 进行跳转：
    ```typescript
    const handleMenuClick = ({ key }: { key: string }) => {
      router.push(key);
    };
    ```
    这就是导致 404 的根本原因。`router.push('/knowledge-base')` 被调用，而该路由不存在。

## 修复方案
修改 `src/components/layout/Sidebar.tsx` 文件，将知识库菜单项的 `key` 修改为 `/knowledge`，使其与 `path` 和实际路由一致。

## 具体修改
-   **文件**: `src/components/layout/Sidebar.tsx`
-   **改动**:
    ```typescript
    // Before
    key: '/knowledge-base',
    
    // After
    key: '/knowledge',
    ```

这不仅仅修复了跳转错误，还能确保当前路由高亮逻辑（`selectedKeys={[pathname]}`）正确工作，因为 `pathname` 将是 `/knowledge`。