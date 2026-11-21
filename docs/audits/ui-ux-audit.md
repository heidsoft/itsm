# UI/UX 与组件一致性静态审计

> 基准 commit：`b3bfd748604041d5082a320ed6d5c447221ac76a`
> **静态检查完成，运行验证待执行**：未安装依赖、未启动 dev server、未运行 Playwright，所有视觉影响标为 `suspected`。

## 1. 门禁实跑结果（confirmed）

在 worktree 执行（只读）：

| 命令 | 结果 | 证据 |
|:---|:---|:---|
| `bash tools/check-antd-legacy.sh` | **exit 1** | `Alert.message 6/0`、`Space.direction 2/0`、`valueStyle 1/0` 超基线 |
| `bash tools/check-antd-direction.sh` | **exit 1** | 两处 `direction="vertical"`，反向守卫要求的文件也不再含 `orientation` |
| `bash scripts/check-frontend-p0-guards.sh` | **exit 1** | `reports/cmdb-quality/page.tsx:147` Alert description 85 字 > 60 |

具体行：`admin/permissions/page.tsx:821,1102,1113`、`reports/cmdb-quality/page.tsx:147,263`、`components/change/ChangeDetail.tsx:1031`、`ai/approval/page.tsx:403`、`components/sla/EscalationChainEditor.tsx:230`、`cmdb/certification/page.tsx:260`。
这些是历史 AntD v6 迁移后的**回归**，不是未完成迁移；正确做法是修调用点，不上调基线。

另外 `scripts/check-unicode-punctuation.sh:17-19` 在 macOS BSD grep 下 `grep -P` 报 invalid option，但 `|| true` 使脚本输出"未发现智能引号"并 exit 0——**假绿**。

## 2. 结构性发现

```yaml
id: AUDIT-040
priority: P1
category: ui/sidebar
status: confirmed (代码) / suspected (视觉)
title: 两个 Sidebar.module.css 导致实际菜单的暗色覆盖为死代码
file: components/layout/sidebar/Sidebar.tsx:14 vs MenuItems.tsx:10
evidence: Sidebar 引入 './Sidebar.module.css'，MenuItems 引入 '../Sidebar.module.css'；CSS Modules 对同名本地类做不同哈希；MenuItems.tsx:156 应用的是父模块的 .customMenu，而暗色覆盖写在子模块
impact: 暗色下实际内联菜单项保留 #4b5563，与 #1f2937 背景约 1.7:1
fix: 统一 import 到同一模块并合并规则；扩展 dark-menu-contract 断言模块一致性
acceptance: 浏览器计算样式对比 + 亮色像素不变
```

```yaml
id: AUDIT-041
priority: P1
category: ui/select
status: confirmed (层叠优先级) / suspected (视觉)
title: 禁用态 Select 选项在两套主题中与可用态同色
file: src/app/globals.css:101-104,1945-1955,309-319
evidence: html:not(.dark) .ant-select-item (0,2,1) 覆盖 :root .ant-select-item-option-disabled (0,2,0)；暗色把 disabled 强制为 #fff !important
fix: 在 catch-all 之后补 disabled 规则，暗色用 --color-text-disabled；新增 select-option-states.test.ts
```

```yaml
id: AUDIT-042
priority: P1
category: ui/layout
status: confirmed
title: 平板宽度覆盖因选择器优先级永不生效
file: globals.css:2217-2220 vs styles/antd-layout-overrides.css:32-37
evidence: .ant-layout-sider (0,1,0) 输给 :not(.ant-layout-sider-collapsed) (0,2,0)，两者均 !important
fix: 媒体查询选择器补齐 :not(...)
```

```yaml
id: AUDIT-043
priority: P1
category: ui/layout
status: confirmed
title: 移动端遮罩 z-index 9 低于 header 10 与 sider 1000
file: app/(main)/layout.tsx:317-325；config/layout.config.ts:103-104；sidebar/Sidebar.module.css:11
impact: 遮罩无法盖住顶栏
fix: 从真实层叠值派生（如 999），去掉不一致的 !important
```

```yaml
id: AUDIT-044
priority: P1
category: ui/tokens
status: confirmed
title: 三套尺寸真源互相矛盾
evidence: layout.config.ts 头 64/侧栏 256/80；antd-layout-overrides.css 头 56；theme-variables.css 头 56/侧栏 240/64；Content minHeight 用 64 而实际头 56
fix: 以 LAYOUT_CONFIG 为唯一真源向 CSS 变量注入
```

```yaml
id: AUDIT-045
priority: P1
category: ui/loading
status: confirmed (代码) / suspected (视觉)
title: LoadingSkeleton 在渲染期使用 Math.random() 生成高度
file: components/ui/LoadingSkeleton.tsx:140（文件已标 @deprecated）
impact: SSR hydration 不一致、布局抖动、截图不可复现
fix: 用 index/常量数组确定高度
```

```yaml
id: AUDIT-046
priority: P2
category: ui/tokens
status: confirmed
title: 两套 :root 变量值冲突、z-index 三套并存
evidence: theme-variables.css --color-primary #3b82f6 vs globals.css:649 #2563eb；--spacing-xl 24 vs 20；globals.css:11-51 弹层统一 9999 !important vs theme-variables.css:133-139 的 1000/1050/1070
fix: 删除 globals.css 中重复 :root 声明，统一使用 token
```

```yaml
id: AUDIT-047
priority: P2
category: ui/dropdown
status: suspected
title: UserSelect 用 ReactNode 作为 optionLabelProp
file: components/common/UserSelect.tsx:48-65,82-95
impact: 多选 chip 可能变成头像+姓名+邮箱的多行块；mode 未限制为 multiple
fix: 分离纯文本 label 或提供 tagRender，并限制 mode
```

```yaml
id: AUDIT-048
priority: P2
category: ui/scroll
status: suspected
title: .ant-table overflow:visible 与表格内滚动冲突；.ant-modal overflow:hidden 可能裁剪弹层
file: globals.css:890-894,879-887,2147-2158
fix: 把 overflow:visible 限定到专用类；overflow:hidden 移到 .ant-modal-content
```

```yaml
id: AUDIT-049
priority: P2
category: ui/i18n
status: confirmed
title: 部分空态/日期快捷项硬编码中文且未透传 t
file: components/ui/AppDatePicker.tsx:37-44；LoadingEmptyError.tsx:234-260
fix: 透传 t 并本地化
```

## 3. 工具与验证条件

Playwright 已配置（`playwright.config.ts:17-121`，`webServer: npm run dev`，`reuseExistingServer`），已有 `tests/e2e/roles/` 7 个角色规格和 `responsive-layout.spec.ts`（1366×768→3840×2160 + 820×1180 + 1024×768）。缺：显式暗色视觉断言、screenshot-diff 基线、390×844 与 360×640 的系统覆盖。

**在声称任何视觉修复完成之前，必须提供**：桌面 1440×900/1920×1080、平板 820×1180/1024×768、手机 390×844/360×640；明暗两套主题；end-user/engineer/approver/tenant-admin/super-admin 角色；对比度与 `scrollWidth ≤ innerWidth+2` 断言；失败时 trace/截图。当前工作树无 `node_modules` 且未启动后端，故本轮不提供这些证据。
