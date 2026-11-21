# 整改 Backlog

> 基准 commit：`b3bfd748604041d5082a320ed6d5c447221ac76a`
> 状态：confirmed / suspected / resolved / blocked（见各专项报告）
> 原则：一个 PR 一个明确问题域；数据库/权限/部署变更需人工审批。

## P0

| ID | 问题 | 状态 | 处置 |
|:---|:---|:---|:---|
| AUDIT-001 | 空卷首装被 `mcp:read` 迁移打断 | `resolved`（PR #114 已合并） | 继续跟踪 AUDIT-002 的首装门禁 |
| AUDIT-002 | `configuration_items` 重复索引阻断 fresh-install 门禁 | `confirmed` | 待审批；在 `duplicateIndexPairs` 补两对，保留 Ent 索引 |
| AUDIT-003 | Release 静默降级为 git-log 说明、无测试证据/制品校验 | `confirmed` | PR-04（先做 G1/G4/G6 硬门禁，后做结构守卫） |

## P1

| ID | 问题 | 状态 | 处置 | 审批要求 |
|:---|:---|:---|:---|:---|
| AUDIT-010 | `/tickets/types` 菜单/路由权限码不一致 | `confirmed` | PR-02：改路由为 `ticket_type:read`，重生成 precheck，补 HTTP 负例 | 权限变更，需审批 |
| AUDIT-011 | 服务请求两菜单用 `ticket:read` | `confirmed` | PR-02：基线码改 `service_request:read` | 低风险（仅显示） |
| AUDIT-012 | `/workflow/audit` 用 `audit:read` | `confirmed` | PR-02：基线码改 `bpmn:read` | 低风险（仅显示） |
| AUDIT-013 | 存量租户 AI 归位/重命名依赖手动端点 | `confirmed` | PR-02：迁移补 `name/parent_id` 修正 + 双租户测试 | 数据迁移，需审批 |
| AUDIT-020~022 | 权限停用语义、DBOnly L3 回退、缓存广播 | `confirmed`/`suspected` | RBAC 专项分支，逐条补真实 DB 负例后修复 | 权限语义变更，需审批 |
| AUDIT-023 | 权限/角色缺 DB 唯一约束 | `confirmed` | 版本化迁移 + 存量重复检测 | 数据库结构，需审批 |
| AUDIT-024 | seeder 回退清单资源名漂移 | `confirmed` | 删除回退清单或从 catalog 派生 | 低风险 |
| AUDIT-025 | 重复初始化覆盖托管菜单字段 | `confirmed` | 明确所有权清单或纳入保护 | 低风险 |
| AUDIT-051 | 备份调度未被部署验收强制 | `confirmed` | 文档 + 部署后断言 | 低风险 |
| AUDIT-053 | 重复索引（同 AUDIT-002） | `confirmed` | 见 P0 | 需审批 |
| AUDIT-060 | `lint:antd` / 方向守卫 / P0 文案守卫红 | `confirmed` | PR-05 机械修复，不上调基线 | 低风险 |
| AUDIT-061 | 两套 Sidebar CSS Module 导致暗色规则失效 | `confirmed`+`suspected` | PR-05：统一 import + 模块一致性守卫 + 浏览器验证 | 需浏览器证据 |
| AUDIT-062 | 禁用态 Select 选项同色 | `confirmed`+`suspected` | PR-05：层叠修复 + `select-option-states.test.ts` | 需浏览器证据 |
| AUDIT-063~065 | 平板宽度、移动遮罩 z-index、尺寸真源三方矛盾 | `confirmed` | PR-05：统一 token/选择器 | 需浏览器证据 |
| AUDIT-066 | LoadingSkeleton 渲染期随机高度 | `confirmed` | PR-05：确定性高度 | 低风险 |
| AUDIT-030~032 | 发布 provenance / 版本漂移 | `confirmed` | PR-04 | 低风险（CI 配置与脚本） |
| AUDIT-050~054 | 官网信任页、无来源数字、站点源码、法务入口、品牌 | `confirmed` | 官网专项（仓库外发布流程） | 需要产品/市场确认 |

## P2

| ID | 问题 | 状态 | 处置 |
|:---|:---|:---|:---|
| AUDIT-050(deploy) | `.env.prod.example` 关闭 metrics 鉴权 | `confirmed` → `resolved` | PR-01 已修（改示例 + 静态守卫） |
| AUDIT-052(deploy) | 基础设施镜像浮动标签 | `confirmed` | 固定次要版本/digest |
| AUDIT-014(menu) | `/ai/chat/list` 别名可复活退役菜单 | `confirmed` | 扩展退役迁移 WHERE |
| AUDIT-015 | 菜单文档 102→101 漂移 | `confirmed` | 重新生成表格 + docs-gate 禁止退役字符串 |
| AUDIT-016/017 | 菜单 fixture 假绿、缺跨层守卫 | `confirmed` | 改为读基线；新增路径/码守卫 |
| AUDIT-046~049 | token、z-index、表格/弹窗 overflow、i18n | `confirmed`/`suspected` | PR-05 分批 |
| AUDIT-055 | 博客数字标题待核实 | `suspected` | 核实来源后改写 |
| F-01(rbac) | `PermissionDefinition` 死模型 | `confirmed` | 标记退役或版本化删表，不接入 RBAC |

## 已解决/不要重复

PR #114（首装 P0001）、跨租户负例矩阵、RBAC 缓存中毒与角色移除即时失效、RLS readiness 门禁、权限策略信封与只增不减授权、菜单退役与 `bpmn:*` 对齐、AntD v6 主体迁移。

## 本批已实施

- AUDIT-050(deploy)：`.env.prod.example` 默认开启 `/metrics` 鉴权，新增静态守卫测试，形成 `audit/production-tenant-readiness` 分支的 PR-01 低风险子集。
