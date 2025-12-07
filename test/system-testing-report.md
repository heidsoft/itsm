# ITSM 系统测试报告（前后端 + UI）

- 日期: 2025-12-07
- 范围: 代码质量、UI 交互、UI 布局、UI 设计
- 覆盖仓库: itsm-prototype（Next.js 前端）与 itsm-backend（Go 后端）

## 总览

- 前端依赖安装成功，自动化测试执行，存在较多类型检查与用例失败。
- 后端单元测试与静态检查存在编译错误，部分接口实现与测试不匹配。
- UI 层在交互与布局方面整体结构清晰，但存在规则冲突与一致性问题。

## 进展更新（P0 修复）

- 前端
  - 恢复 `next lint` 运行（简化 ESLint 配置，移除未安装插件规则）。
  - 修复 TypeScript 语法错误：`TicketDependencyManager.tsx` 与 `validation.ts` 已通过 `tsc --noEmit`。
  - 修正 Jest alias 与测试导入：`jest.config.js` 统一 `@/*` 映射；`api-integration.test.ts` 使用 `@/lib/api/*`。
- 集成测试通过：`src/app/lib/__tests__/api-integration.test.ts` 19/19 通过。
- 浏览器端到端测试（初次）
  - 登录页操作：输入“用户名/密码”后点击“登录”，出现提示“登录失败，请检查您的用户名和密码”（后端未登录或未联通）
  - 由于登录失败，暂无法进入仪表盘与业务页面进行交互验证；已在 `test/browser-e2e-run-results.md` 记录 PASS/FAIL 明细
  - 建议：
    - 启动并联通后端服务，提供有效测试账号与租户；或在前端添加本地 Mock 登录用于烟雾测试
    - 完成后重新执行 `test/browser-e2e-test-plan.md` 并更新结果
- 新增场景测试
  - 新增工单与事件交互用例：
    - `src/app/__tests__/tickets-list-interactions.test.tsx` 覆盖搜索与状态筛选、分页参数校验
    - `src/app/__tests__/incident-list-interactions.test.tsx` 覆盖 API 层分页参数校验
  - 登录表单测试：`login/page.test.tsx` 已通过
- 当前失败用例与原因
  - 部分页面级测试（Dashboard）出现 `Invalid hook call`，原因：测试环境未正确挂载必要 Provider 或 App Router 环境，建议在测试包装中统一注入 `QueryClientProvider`、Router 与 App 上下文，或将页面测试拆分为组件级测试
- 后端
  - 重新运行 `go test ./...`：BPMN 适配器编译问题不再出现；当前主要失败源于测试期望的 Ent 字段与现有 schema 不匹配（如 `SetActive`、`SetDisplayName`）。

## 前端测试结果

- 命令: `npm ci`, `npm run type-check`, `npm run test:ci`
- 依赖安装: 成功
- ESLint: 失败（缺少配置依赖）
  - 错误: Failed to load config "@typescript-eslint/recommended" from `.eslintrc.json`
- TypeScript 类型检查: 失败（46 个错误）
  - 典型错误示例:
    - `src/lib/hooks/useAccessibility.ts:73` 存在 JSX 但为 `.ts` 文件，触发 JSX 解析错误
    - `src/lib/utils/props-validator.ts:237` JSX 片段解析错误
    - `src/components/business/TicketDependencyManager.tsx:170` 语法不完整（尾部数组缺少元素或分隔）
    - `src/lib/error-handler.tsx:333`, `src/lib/validation.ts:253` 使用占位符 `...` 导致语法错误
- Jest 测试: 执行成功但结果失败
  - 测试套件: 10 套中 7 失败
  - 用例: 33 用例中 24 失败，9 通过
  - 典型失败示例:
    - 登录表单测试已修复并统一使用标准登录页面

## 后端测试结果

- 命令: `go test ./...`, `go vet ./...`
- 结果: 编译失败，`vet` 检查报告匹配的编译期问题
- 关键错误:
  - `itsm-backend/pkg/bpmn/engine_adapter.go:16` `bpmn_engine.New` 参数数量不匹配（需传入 `string`）
  - `itsm-backend/pkg/bpmn/engine_adapter.go:43,51,57,62` 访问未导出或不存在的方法（`Export`, `Load`, `NewTaskHandler`, `GetProcessCache`）
  - `itsm-backend/controller/user_controller_test.go:40,627,685` `NewUserController` 构造函数签名不匹配（缺少 `*zap.SugaredLogger` 参数）
  - `itsm-backend/controller/user_controller_test.go:435,444,445,454` `dto.SearchUsersRequest` 字段不匹配（不存在 `Page`, `Department`）
  - `itsm-backend/service/auth_service_test.go:53,156,249,324,396` `ent.UserCreate.SetStatus` 方法不存在
  - `itsm-backend/service/auth_service_test.go:206` `dto.RefreshTokenResponse.RefreshToken` 字段不存在
  - `itsm-backend/service/auth_service_test.go:362,363` `dto.UserInfo.DisplayName`, `Phone` 字段不存在

## UI 交互与布局评估

- 交互一致性:
  - 登录表单具备明确的标签与占位提示，含辅助图标与 ARIA 属性；MFA 流程测试覆盖较深但查询选择器不够精确，易命中非输入元素。
  - Toast 组件交互与动画合理，自动关闭逻辑明确：`itsm-prototype/src/components/ui/Toast.tsx:36` 定时触发 `handleClose()`；关闭按钮提供 `aria-label="关闭通知"`。
- 布局结构:
  - App Router 结构明确，模块化页面分区清晰（如 `src/app/(main)/tickets`, `src/app/(main)/dashboard` 等）。
  - 使用 Ant Design 的网格与容器组件大量存在，配合 Tailwind 类名；但双UI体系并存会加大一致性与样式维护成本。
- 性能与状态管理:
  - 引入 `@tanstack/react-query`/`swr` 等缓存策略，具备分页与虚拟列表组件，性能优化意识良好。

## UI 设计一致性与规范符合性

- 规则冲突/偏差:
  - 禁止使用 CSS Modules，但存在以下文件：
    - `itsm-prototype/src/components/layout/Header.module.css`
    - `itsm-prototype/src/components/layout/Sidebar.module.css`
    - `itsm-prototype/src/lib/error-handler.module.css`
  - 禁止在组件中直接调用 API：实际存在直接 `fetch` 与 `localStorage` 调用
    - 示例：`itsm-prototype/src/components/business/IncidentManagement.tsx:162-168` 直接使用 `fetch` 与 `localStorage`
  - 禁止直接使用 `console.log` 调试：项目内广泛存在
    - 统计：51+ 个文件，170+ 次出现（示例：`itsm-prototype/src/app/(main)/dashboard/page.tsx:1` 等）
  - React 版本不一致：规则建议 React 19.x，当前 `package.json` 使用 React 18.2.0
- 设计系统与一致性:
  - Tailwind 4.x 已配置，但与 Ant Design 大量混用；建议明确主设计系统，减少样式与交互差异。

## 重点问题清单（可直接修复）

- ESLint 配置缺失：未安装 `@typescript-eslint/eslint-plugin` 与 `@typescript-eslint/parser`，导致 `next lint` 无法运行。
- 类型检查错误：将含 JSX 的 `.ts` 文件改为 `.tsx` 并修复占位符 `...`；补齐不完整的数组/对象字面量。
- 测试选择器问题：使用更精确的 `getByRole`/`getByLabelText` 选择器，避免命中按钮图标等非输入元素。
- 直接 API/本地存储调用：将组件内 `fetch`/`localStorage` 替换为统一 API 层与封装服务。
- 后端测试/实现对齐：统一 DTO 与服务接口签名，修正 BPMN 引擎适配器与控制器构造函数。

## 建议与改进路线

- 代码质量
  - 安装并启用 ESLint TypeScript 规则；新增 CI 步骤：`npm run lint:check` + `npm run type-check`。
  - 严格类型化：清理 `...` 占位符，统一 `.tsx` 扩展名用于含 JSX 的模块。
- UI 交互与测试
  - 为关键交互（登录、筛选、提交、分页）编写更稳健的 RTL 测试；改用角色与显式 label 选择器。
  - 引入可访问性快照与 Axe 检查，提升 ARIA 与键盘可达性。
- UI 布局与设计
  - 明确设计系统：若以 Tailwind 为主，逐步减少 Ant Design 组件或统一其主题风格。
  - 移除 CSS Modules，改用 Tailwind 实现局部样式；或在规则允许下保留少量关键样式但需一致说明。
- 前后端一致性
  - 对齐 DTO 字段与服务签名，增加 handler 层测试；修复 BPMN 适配器对外 API 使用方法。

## 关键代码参考

- `itsm-prototype/src/components/ui/Toast.tsx:36` 自动关闭调用 `handleClose()`
- `itsm-prototype/src/components/business/IncidentManagement.tsx:162-168` 直接使用 `fetch` 与 `localStorage`
- `itsm-prototype/src/lib/hooks/useAccessibility.ts:73` JSX 解析错误（文件扩展与语法问题）
- `itsm-backend/pkg/bpmn/engine_adapter.go:16,43,51,57,62` BPMN 适配器 API 使用错误
- `itsm-backend/controller/user_controller_test.go:40,435,444,445,454,627,685` 控制器测试签名与字段不匹配
- `itsm-backend/service/auth_service_test.go:53,206,324,362,363,396` DTO/Ent 字段方法不一致

## 执行命令与结果摘要

- 前端
  - `npm ci` → 成功
  - `npm run lint:check` → 失败（缺少 `@typescript-eslint/recommended` 配置依赖）
  - `npm run type-check` → 46 个 TS 错误
  - `npm run test:ci` → 7/10 套件失败，24/33 用例失败
- 后端
  - `go test ./...` → 编译失败（多处签名/API 不一致）
  - `go vet ./...` → 报告与编译期错误一致

## 结论

- 当前系统具备完整的模块结构与良好性能意识，但在规范一致性与类型/测试稳定性方面需要集中整改。优先修复 ESLint/TS 错误与测试选择器问题，随后推进 API 层统一化与后端 DTO/服务对齐，可快速提升整体质量与可维护性。
- 端到端测试场景持续完善：新增 Tickets/Incidents 交互用例已落地；登录表单选择器问题已修复；Dashboard 等页面级用例需统一测试包装（Provider/Router/App），建议在后续迭代中完成。
