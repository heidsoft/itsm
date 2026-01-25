# 开发规范 (Development Rules)

## 前端 (Frontend)

### API 响应命名 (API Response Naming)

- **命名约定**: API 响应类型和接口中的所有字段必须使用 **camelCase**（小驼峰命名法）。这符合 JavaScript/TypeScript 标准和前端编码风格。
- **自动转换**: 系统在 `http-client.ts` 中内置了拦截器，会自动将所有 API 响应的键从 `snake_case`（下划线命名）转换为 `camelCase`，并将所有请求体的键从 `camelCase` 转换为 `snake_case`。
- **组件开发**: 所有新组件必须严格使用 **camelCase** 来命名 Props、State 变量和数据访问。**严禁**在前端代码中使用 `snake_case`（例如，必须使用 `ticketNumber` 而不是 `ticket_number`）。
- **一致性**: 确保 `api-config.ts` 或特定功能 API 文件中定义的所有新 API 接口严格遵循此规则。

### UI 框架 (UI Framework)

- **组件库**: 使用 **Ant Design (antd)** 作为主要的 UI 组件库，用于布局、表单、表格和其他交互元素。
- **样式**: 尽可能利用 Ant Design 的主题功能和工具类。对于自定义样式，使用 Tailwind CSS (v4) 或 CSS Modules，确保不与 Ant Design 的基础样式冲突。
- **布局**: 遵循 Ant Design 的布局原则 (Layout, Header, Sider, Content) 以保持页面结构一致。

### 图标使用 (Icon Usage)

- **标准库**: 在应用程序中统一使用 `lucide-react` 作为图标库，以确保一致性和现代外观。
- **限制**: 除非绝对必要（例如，当特定 Ant Design 组件需要 `lucide-react` 无法轻易满足的特定图标类型时，尽管通常可以封装），否则避免使用 `@ant-design/icons`。
- **迁移**: 在处理现有文件时，主动将 `@ant-design/icons` 导入替换为对应的 `lucide-react` 图标。
- **样式**: 根据设计系统保持一致的图标尺寸（例如 16x16 或 20x20）。
