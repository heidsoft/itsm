# ITSM 前端 UX 审查报告 — 开源用户首次使用视角

> **审查日期**: 2026-06-19  
> **审查人**: 许清楚（产品经理）  
> **审查范围**: `itsm-frontend/` 前端工程，聚焦开源用户下载 → 安装 → 首次登录 → 核心功能探索的完整旅程  
> **技术栈**: Next.js 15 (App Router) + React 19 + Ant Design v6 + Tailwind CSS v4 + Zustand + TanStack Query

---

## 一、开源用户旅程分析

### 旅程总览

```
GitHub Clone → Docker 启动 → 访问 localhost:3000 → 中间件重定向 → 登录页 → 仪表盘 → 功能探索 → 错误场景
     ✅            ✅           ⚠️ P0-1            ✅          ⚠️ P1-5      ✅        ⚠️ 多项       ⚠️ P0-2/3
```

### 1.1 首次访问（根页面 → 登录）

**当前状态**: 中间件 `src/middleware.ts` 正确将 `/` 重定向到 `/login`（未认证）或 `/dashboard`（已认证）。但 `src/app/page.tsx` 仍保留了一个完整的 "CloudMesh" 商业营销落地页（含定价表 ¥9,800/年、虚假统计数字 "100+ 服务企业"、"100万+ 处理工单"、销售联系方式 sales@cloudmesh.top）。

**问题**:
- 中间件正常工作时用户看不到此页面，但它仍存在于代码库中，会被静态生成（SSG），在 GitHub 源码浏览时暴露
- 页面中的品牌名 "CloudMesh" 与项目实际名称 "AI-Native ITSM" 不一致
- 定价表、虚假统计数字对开源项目信誉有严重损害
- 页面引用了 FontAwesome (`fas fa-cloud`) 但项目实际使用 Lucide Icons，存在依赖不一致
- 页面中的 `/deploy`、`/docs`、`/blog`、`/about`、`/privacy` 等链接指向不存在的路由

**建议**: 将 `page.tsx` 替换为极简的重定向页面或直接删除（依赖中间件处理），或在页面中放置项目介绍 + 快速开始指引。

### 1.2 登录流程（登录页 → 仪表盘）

**当前状态**: 登录页 `src/app/(auth)/login/page.tsx` 使用 Ant Design 构建，视觉设计专业。登录成功后跳转 `/dashboard`。

**问题**:

| 元素 | 问题 | 严重程度 |
|------|------|----------|
| "忘记密码" 按钮 | 无 `onClick`、无 `href`，点击无反应。`/forgot-password` 页面实际已存在 | P1 |
| "立即注册" 按钮 | 无 `onClick`、无 `href`，点击无反应。`/register` 页面实际已存在 | P1 |
| "SSO 登录" 按钮 | 无 `onClick`，点击无反应，无任何提示 | P1 |
| MSPService 引用 | 登录成功后调用 `MSPService.refreshCache()`，这是 MSP 多租户商业模式的功能，开源用户不需要 | P2 |
| 默认凭据提示 | 登录页未显示默认管理员凭据 `admin/admin123`，用户需查 README 才知道 | P1 |
| Remember Me | `rememberMe` 状态被管理但未实际使用（不传给 AuthService） | P2 |

**建议**: 
- 三个死按钮用 `<Link>` 包裹或添加 `onClick` 导航
- 移除或条件化 MSPService 引用
- 在登录表单下方添加默认凭据提示（开发模式）

### 1.3 首次使用引导（Onboarding）

**当前状态**: **完全没有 onboarding 机制**。无欢迎页、无功能引导、无首次登录检测、无新手向导。

**问题**:
- 用户登录后直接进入仪表盘，面对 90+ 页面的系统不知从何开始
- 没有"创建第一个工单"、"浏览 CMDB"、"查阅知识库"等引导步骤
- 仪表盘的 QuickActions 区域依赖后端 API 返回数据，空数据时无引导

**建议**: 
- P1: 仪表盘空数据状态增加"快速上手"引导卡片
- P2: 实现轻量级 onboarding tour（使用 shepherd.js 或类似库）
- P2: 首次登录后跳转到引导页而非直接进仪表盘

### 1.4 核心功能探索

#### 工单创建
**当前状态**: `/tickets/create` 页面存在，有表单组件。
**问题**: 需验证表单提交是否完整工作，模板选择是否可用。

#### CMDB 浏览
**当前状态**: `/cmdb` 页面存在，有拓扑图组件。
**问题**: 首次使用时 CMDB 为空，需确认空状态是否有引导。

#### 知识库查阅
**当前状态**: `/knowledge` 页面存在。
**问题**: 需确认空状态处理。

#### 通用组件
**良好发现**: `LoadingEmptyError` 组件已封装 loading/empty/error 三种状态，且有针对 tickets/incidents/problems/changes/cmdb/users/workflows 的预定义配置，这是做得好的地方。

### 1.5 错误场景

| 场景 | 当前处理 | 问题 | 严重程度 |
|------|----------|------|----------|
| 404 - 页面不存在 | Next.js 默认 404 | 无自定义 `not-found.tsx`，体验差 | P0 |
| 路由级渲染错误 | 无 `error.tsx` | 任何页面级错误冒泡到全局 ErrorBoundary，整个应用白屏 | P0 |
| 全局未捕获错误 | ErrorBoundary 组件 | 存在但"返回首页"按钮跳转到 `/`（营销页/重定向），应跳 `/dashboard` | P1 |
| API 不可用 | 各页面自行处理 | 依赖各页面单独实现，不一致；部分页面仅 `console.error` | P1 |
| 网络断开 | 无全局处理 | 无离线提示，无网络恢复自动重连 | P1 |
| Token 过期 | 中间件 JWT 检查 | 中间件检查 token 过期会重定向到登录页，但无"会话已过期"提示 | P2 |
| 构建时类型错误 | 忽略 | `next.config.ts` 配置 `ignoreBuildErrors: true`，类型错误进入生产 | P0 |

---

## 二、需求池（按优先级排列）

### P0 — 必须修复（阻塞开源发布）

| # | 需求 | 问题描述 | 建议方案 | 影响文件 |
|---|------|---------|---------|---------|
| P0-1 | 替换根页面营销内容 | `src/app/page.tsx` 是 CloudMesh 商业营销页（定价表/虚假数据/销售联系方式），严重损害开源项目形象 | 方案A: 删除 `page.tsx`，依赖 middleware 重定向；方案B: 替换为项目介绍页（GitHub 链接 + 快速开始 + 功能截图） | `src/app/page.tsx` |
| P0-2 | 添加 404 页面 | 无 `not-found.tsx`，用户访问不存在路径看到 Next.js 默认 404 | 创建 `src/app/not-found.tsx`，包含返回仪表盘/搜索建议，使用 Ant Design Result 组件，保持视觉一致性 | `src/app/not-found.tsx` (新建) |
| P0-3 | 添加路由级错误页面 | 无 `error.tsx`，页面级 React 错误冒泡到全局 ErrorBoundary 导致整页白屏 | 创建 `src/app/error.tsx`（路由级）和 `src/app/global-error.tsx`（根级），提供重试/返回操作，不白屏整个应用 | `src/app/error.tsx` (新建), `src/app/global-error.tsx` (新建) |
| P0-4 | 修复构建配置 | `next.config.ts` 中 `eslint.ignoreDuringBuilds: true` 和 `typescript.ignoreBuildErrors: true` 导致类型错误和 lint 问题进入生产构建 | 移除两个 ignore 配置；修复现有类型错误和 lint 问题；在 CI 中强制 `tsc --noEmit` 和 `next lint` | `next.config.ts` |

### P1 — 应该修复（影响首次使用体验）

| # | 需求 | 问题描述 | 建议方案 | 影响文件 |
|---|------|---------|---------|---------|
| P1-1 | 修复登录页死按钮 | "忘记密码"、"立即注册"、"SSO 登录" 三个按钮均无 onClick/href，点击无反应 | 用 `<Link href="/forgot-password">` 和 `<Link href="/register">` 包裹按钮；SSO 按钮添加 onClick 提示"SSO 暂未配置"或条件渲染 | `src/app/(auth)/login/page.tsx` |
| P1-2 | 添加路由级加载状态 | 无 `loading.tsx`，页面切换时无骨架屏，用户看到白屏或旧内容闪烁 | 创建 `src/app/(main)/loading.tsx`，使用 Ant Design Skeleton 组件，匹配各页面布局结构 | `src/app/(main)/loading.tsx` (新建) |
| P1-3 | 修复 ErrorBoundary 跳转 | `handleGoHome` 跳转到 `/`（营销页/重定向到登录），用户遇到错误后被踢出登录 | 改为 `window.location.href = '/dashboard'` 或更智能地根据认证状态跳转 | `src/components/common/ErrorBoundary.tsx` L94 |
| P1-4 | 登录页添加默认凭据提示 | 开源用户首次使用不知道 `admin/admin123`，需翻阅 README | 在登录表单下方添加开发模式提示："默认管理员账户: admin / admin123（请及时修改密码）"，生产模式隐藏 | `src/app/(auth)/login/page.tsx` |
| P1-5 | 添加仪表盘空数据引导 | 仪表盘 QuickActions 依赖后端 API，空数据时无引导，用户不知下一步做什么 | 当 `data?.quickActions` 为空时，显示静态引导卡片："创建第一个工单"、"浏览服务目录"、"配置 CMDB"、"查阅知识库" | `src/app/(main)/dashboard/page.tsx` |
| P1-6 | 添加网络状态全局处理 | 无离线检测、无 API 不可用全局提示 | 添加全局网络状态监听组件，断网时顶部显示 Ant Design Alert 横幅；API 超时统一提示 | 新建 `src/components/common/NetworkStatus.tsx` |
| P1-7 | 移除 Google Fonts 残留 | 根布局保留 `preconnect` 到 `fonts.googleapis.com` 和 `fonts.gstatic.com`，但字体导入已注释。导致不必要的 DNS 查询，离线环境产生控制台错误 | 删除 L79-80 和 L93-94 的 preconnect/dns-prefetch 链接 | `src/app/layout.tsx` L79-80, L93-94 |
| P1-8 | Token 过期友好提示 | 中间件检测到 token 过期直接重定向到 `/login`，用户不知为何被登出 | 在重定向 URL 添加 `?reason=session_expired` 参数，登录页读取后显示"您的会话已过期，请重新登录"提示 | `src/middleware.ts`, `src/app/(auth)/login/page.tsx` |

### P2 — 打磨优化（提升整体品质）

| # | 需求 | 问题描述 | 建议方案 | 影响文件 |
|---|------|---------|---------|---------|
| P2-1 | 清理 console 语句 | 全代码库 **386 处** `console.error/warn/log` 调用，部分在生产环境暴露内部错误细节 | 统一使用 `src/lib/env.ts` 中的 `logger` 工具；生产构建通过 babel/swc 插件自动剔除 `console.log/warn`；保留 `logger.error` 用于真正需要生产可见的错误 | 全 `src/` 目录 (386 处) |
| P2-2 | 修复 package.json 名称 | `"name": "itsm-prototype"` 不符合开源项目命名规范 | 改为 `"name": "itsm-frontend"` 或 `"@itsm/frontend"` | `itsm-frontend/package.json` L2 |
| P2-3 | 修复侧边栏菜单 TODO | `FORCE_STATIC_MENU = false` 带 TODO 注释，后端菜单 API 返回重复 key，导致强制使用静态菜单 | 后端修复菜单数据去重；前端添加 key 去重逻辑；移除 TODO 和 FORCE_STATIC_MENU 标志 | `src/components/layout/sidebar/Sidebar.tsx` L105-106 |
| P2-4 | 移除/条件化 MSPService | 登录页引用 `MSPService.refreshCache()`，这是 MSP 商业模式功能，开源用户不需要 | 根据 `DEPLOYMENT_MODE` 环境变量条件引入 MSPService；或移到登录成功后的路由守卫中 | `src/app/(auth)/login/page.tsx` L28, L61 |
| P2-5 | 实现 Remember Me 功能 | `rememberMe` 状态被管理但未传给 AuthService，功能不完整 | 将 rememberMe 传给 AuthService，控制 token 存储策略（sessionStorage vs localStorage） | `src/app/(auth)/login/page.tsx`, `src/lib/services/auth-service.ts` |
| P2-6 | 添加前端健康检查端点 | 无 `/.well-known/health` 或 `/api/health` 前端健康检查 | 创建 `src/app/api/health/route.ts`，返回 `{ status: 'ok', timestamp, version }` | `src/app/api/health/route.ts` (新建) |
| P2-7 | monitoring.js 清理 | `public/scripts/monitoring.js` 在生产环境输出 `console.log('性能指标:', metrics)` | 生产环境条件化输出或使用 Performance API 上报 | `public/scripts/monitoring.js` |
| P2-8 | 添加首次登录 Onboarding | 无新手引导，90+ 页面系统对新用户不友好 | 实现轻量级 onboarding tour：首次登录检测 → 3-5 步引导（仪表盘概览 → 创建工单 → CMDB → 知识库 → 完成） | 新建 `src/components/onboarding/` 目录 |
| P2-9 | 统一 ErrorBoundary 与 GlobalErrorBoundary | 存在两个错误边界组件（`common/ErrorBoundary.tsx` 和 `common/GlobalErrorBoundary.tsx`），功能重叠 | 合并为一个，或明确分工：路由级 vs 组件级 | `src/components/common/ErrorBoundary.tsx`, `src/components/common/GlobalErrorBoundary.tsx` |
| P2-10 | AccessDenied 组件跳转修复 | `AuthGuard.tsx` 中 AccessDenied 的"返回上一页"在无历史记录时跳转 `/`（营销页） | 改为跳转 `/dashboard` | `src/components/auth/AuthGuard.tsx` L402 |

---

## 三、开源就绪检查清单

### 文档与配置

| 检查项 | 状态 | 说明 |
|--------|------|------|
| README 和快速启动文档 | ✅ 通过 | `README.md` 内容详尽，含一键启动、本地开发、初始化说明、截图 |
| 环境变量示例文件 | ✅ 通过 | 前端 `.env.example` 和后端 `.env.example` 均存在且配置项完整 |
| 默认管理员凭据提示 | ⚠️ 部分 | README 中有 `admin/admin123` 说明，但登录页未显示，用户需翻阅文档 |
| CONTRIBUTING 指南 | ✅ 通过 | README 引用了 CONTRIBUTING.md |
| LICENSE 文件 | ✅ 通过 | Apache 2.0 |
| CHANGELOG | ✅ 通过 | 存在 |

### 首次运行体验

| 检查项 | 状态 | 说明 |
|--------|------|------|
| Docker 一键启动 | ✅ 通过 | `docker compose up -d --build` + `itsm-init` 初始化任务 |
| 首次运行引导 | ❌ 缺失 | 无 onboarding flow，无首次登录检测 |
| 健康检查页面 | ❌ 缺失 | 前端无 `/api/health` 端点（后端有 `/api/v1/health`） |
| 中间件路由保护 | ✅ 通过 | middleware 正确处理认证重定向 |
| 错误处理全覆盖 | ❌ 缺失 | 无 `not-found.tsx`、`error.tsx`、`loading.tsx`、`global-error.tsx` |

### 代码质量

| 检查项 | 状态 | 说明 |
|--------|------|------|
| 构建时类型检查 | ❌ 失败 | `typescript.ignoreBuildErrors: true` |
| 构建时 Lint 检查 | ❌ 失败 | `eslint.ignoreDuringBuilds: true` |
| console 语句清理 | ❌ 失败 | 386 处 console 调用 |
| 项目命名规范 | ❌ 失败 | `itsm-prototype` 应改为正式名称 |
| 错误边界一致性 | ⚠️ 部分 | 存在两个功能重叠的 ErrorBoundary 组件 |

### 响应式适配

| 检查项 | 状态 | 说明 |
|--------|------|------|
| 主布局响应式 | ✅ 通过 | `(main)/layout.tsx` 有移动端检测、侧边栏自动折叠、遮罩层 |
| 登录页响应式 | ✅ 通过 | 使用 Ant Design Row/Col 响应式栅格，`xs={0} lg={10}` 隐藏左侧品牌区 |
| 仪表盘响应式 | ✅ 通过 | KPI 卡片使用 `xs={24} sm={12} md={12} lg={6}` 栅格 |
| 营销页响应式 | ⚠️ N/A | 该页面应被移除 |
| 无障碍 | ⚠️ 部分 | 有 skip-to-content 链接，但部分按钮缺少 aria-label |

### 开源品牌一致性

| 检查项 | 状态 | 说明 |
|--------|------|------|
| 根页面品牌 | ❌ 失败 | 显示 "CloudMesh" 而非 "AI-Native ITSM" |
| 登录页品牌 | ⚠️ 部分 | 显示 "ITSM Pro"，与 README 中 "AI-Native ITSM" 不完全一致 |
| 侧边栏品牌 | ✅ 通过 | 显示 "ITSM 系统" |
| 元数据品牌 | ✅ 通过 | layout.tsx metadata 使用 "ITSM Platform" |
| 商业内容残留 | ❌ 失败 | 定价表、销售联系方式、虚假统计数字 |

---

## 四、待确认问题

| # | 问题 | 需确认方 | 背景 |
|---|------|---------|------|
| Q1 | 项目正式名称是什么？ | 用户/主理人 | 代码中出现 "CloudMesh"、"ITSM Pro"、"ITSM Platform"、"AI-Native ITSM" 四种名称，需统一 |
| Q2 | 根页面 `page.tsx` 的预期行为？ | 用户/主理人 | 中间件已将 `/` 重定向到 `/login` 或 `/dashboard`，`page.tsx` 实际不会展示。是删除还是替换为项目介绍页？ |
| Q3 | SSO 登录是否为开源版本的功能？ | 用户/架构师 | 登录页有 SSO 按钮但无实现。如不支持，应移除按钮而非留死链接 |
| Q4 | 注册功能是否开放给开源用户？ | 用户/架构师 | 注册页面存在，但 ITSM 通常由管理员创建用户。需确认是否保留自助注册 |
| Q5 | MSP 模块是否包含在开源版本中？ | 用户/架构师 | 登录页引用 MSPService，README 提到 `saas_msp` 模式。需确认开源版本是否包含 MSP 功能 |
| Q6 | `next.config.ts` 忽略构建错误的背景？ | 架构师/工程师 | 是临时措施还是有意为之？需评估修复所有类型错误的工作量 |
| Q7 | 后端菜单 API 重复 key 问题的修复计划？ | 后端/架构师 | Sidebar 中 TODO 标注后端菜单数据有重复 key，需确认修复时间线 |
| Q8 | 是否需要 i18n 国际化？ | 用户 | 登录页使用 `useI18n` hook，但系统默认中文。开源项目面向全球用户，是否需要英文支持 |

---

## 五、总结

### 做得好的方面
1. **中间件认证** — `middleware.ts` 正确处理路由保护和根路径重定向，JWT 过期检查逻辑完善
2. **LoadingEmptyError 组件** — 统一封装了 loading/empty/error 三态，有 7 种业务场景预配置
3. **仪表盘体验** — 有骨架屏、错误重试、自动刷新、连接状态指示器
4. **响应式布局** — 主布局有完整的移动端适配（侧边栏折叠、遮罩层、栅格响应）
5. **无障碍** — 有 skip-to-content 链接
6. **README 文档** — 快速启动指南详尽，含 Docker 部署、本地开发、初始化验证
7. **AuthGuard 组件** — 认证守卫体系完整，支持路由级、组件级、操作级权限控制

### 必须改进的方面
1. **根页面营销内容必须移除** — 这是开源项目的第一印象，定价表和虚假数据不可接受
2. **Next.js 特殊文件缺失** — `not-found.tsx`、`error.tsx`、`loading.tsx` 是 App Router 的基本要求
3. **构建配置必须严格** — 忽略类型检查和 lint 是生产环境的定时炸弹
4. **登录页死按钮** — 三个按钮无响应是基本的 UX 缺陷
5. **386 处 console 语句** — 生产环境暴露内部逻辑，需统一使用 logger 工具

### 优先级建议
- **第一波（P0）**: P0-1 ~ P0-4，预计 1-2 人天
- **第二波（P1）**: P1-1 ~ P1-8，预计 3-5 人天
- **第三波（P2）**: P2-1 ~ P2-10，可分批迭代，console 清理可自动化
