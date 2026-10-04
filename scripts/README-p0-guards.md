# 前端质量守卫

防止编译失败、假功能入口、冗余引导文案、契约漂移的自动化检查。

## P0 守卫（已接入 CI）

| # | 守卫 | 脚本 | 防什么 |
|---|------|------|--------|
| 1 | Unicode 非标点检测 | `check-unicode-punctuation.sh` | 智能引号 `""''`、全角标点混入 TSX 导致编译失败 |
| 2 | 占位文案零新增 | `check-placeholder-text.sh` | "即将推出/敬请期待"等假功能入口泄漏到用户界面 |
| 3 | Alert 冗余描述 | `check-alert-verbosity.js` | Alert 同时写 message + 长 description 导致冗余引导 |
| 4 | unused import | 已有 `npm run lint` | 删除 UI 元素后遗留 `import Link`/`Avatar` 等 |

**统一入口**: `bash scripts/check-frontend-p0-guards.sh`

## P1 守卫（存量问题跟踪）

| # | 守卫 | 脚本 | 防什么 | 当前状态 |
|---|------|------|--------|----------|
| 5 | 列表信封契约 | `check-list-envelope-contract.js` | 列表接口返回 `records`/`list` 而非 `items` | 23 处违规待修复 |
| 6 | snake_case 零新增 | `check-snake-case-zero-new.sh` | 后端 DTO 新增 snake_case JSON tag | ✅ 通过 |
| 7 | 空状态文案守卫 | `check-empty-state-verbosity.sh` | "点击下方按钮创建第一个X"长文案 | 10 处违规待修复 |
| 8 | 假数据渲染检测 | `check-fake-data-rendering.js` | Card/Table 硬编码假数据 | ✅ 通过 |

**统一入口**: `bash scripts/check-frontend-p1-guards.sh`

## 本地运行

```bash
# 运行 P0 守卫（CI 必过）
bash scripts/check-frontend-p0-guards.sh

# 运行 P1 守卫（存量跟踪）
bash scripts/check-frontend-p1-guards.sh

# 或单独运行某个守卫
bash scripts/check-unicode-punctuation.sh
bash scripts/check-placeholder-text.sh
node scripts/check-alert-verbosity.js
node scripts/check-list-envelope-contract.js
bash scripts/check-snake-case-zero-new.sh
bash scripts/check-empty-state-verbosity.sh
node scripts/check-fake-data-rendering.js
```

## CI 集成

P0 守卫已接入 `.github/workflows/frontend-ci.yml`，在 lint 之前运行。失败会阻断 PR 合并。

P1 守卫当前作为 advisory，不阻断 CI，但会输出违规清单供跟踪修复。

## 规则说明

### Unicode 非标点

零容忍。匹配字符：
- U+201C/U+201D: `""` (智能双引号)
- U+2018/U+2019: `''` (智能单引号)
- U+3001/U+3002: `、` `。` (中文逗号句号)
- U+FF0C/U+FF1A/U+FF1B: `，` `：` `；` (全角标点)

修复：替换为 ASCII 等价字符。

### 占位文案

排除 i18n 翻译文件和内部错误，只检查 JSX 渲染路径。匹配模式：
- 即将推出
- 敬请期待
- coming soon
- 功能开发中
- 暂未开放

修复：
1. 功能已实现 → 删除占位文案，接入真实数据
2. 功能未实现 → 从菜单/路由中移除入口
3. 内部注释 → 确保不在 JSX 渲染路径中

### Alert 冗余描述

Alert 组件如果同时有 `message` 和 `description`，description 长度不应超过 60 字符。

修复：
- 合并 message + description 为单条 message
- 或删除冗余 description

示例：
```tsx
// ❌ 冗余
<Alert message="使用说明" description="MSP Manager 可以为 MSP 员工分配客户租户。分配后，MSP 员工即可通过 X-Customer-Tenant-ID 头访问对应客户的工单。" />

// ✅ 精简
<Alert message="MSP Manager 可为员工分配客户租户，分配后通过 X-Customer-Tenant-ID 头访问对应工单" />
```

## 历史背景

这些守卫源自 2026-10-03 的产品质量巡检（R1-R8 + P0-P2），当时发现：
- 智能引号导致 TypeScript 编译失败
- 多个页面显示"即将推出"假功能入口
- Alert 组件写长段引导文案，信息密度低
- 删除 UI 元素后遗留 unused import

这些问题大多是人工巡检发现的，不是自动化测试或 CI 门禁拦住的。P0 守卫的目标是让这些问题无法回潮。

## 后续计划

- P1: 列表信封契约测试、snake_case 零新增扫描
- P2: 租户隔离回归测试、状态机迁移测试
- P3: docs-gate 自动化、handler 接线检查、E2E 关键路径
