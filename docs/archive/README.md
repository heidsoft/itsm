# 文档归档

这里保存阶段性报告、历史问题清单、旧版本计划和一次性复盘材料。这些内容对追溯项目演进有价值，但不应作为新用户或部署人员的默认阅读入口。

## 目录说明

| 目录 | 内容 |
|:---|:---|
| `bug-reports/` | 历史缺陷报告、修复报告、Schema 问题分析 |
| `reviews/` | 阶段性产品、架构、商用就绪和交付复盘 |
| `plans/` | 历史增强计划、迁移计划、前端计划 |
| `testing-reports/` | 旧版多角色、多端测试报告、itst-* 早期计划 |
| `workflow-reports/` | 中文企业流程、BPMN 部署和验证相关历史材料 |
| `scripts/` | 已废弃脚本 / 历史脚本快照 |

## 归档准则（v1.5 起）

文档满足下列任一条件时，应在下一轮文档清理 PR 中移入 `docs/archive/`：

1. **默认凭据过期**：文档中的 `admin/admin123`、`admin123`、`postgres123` 等示例密码已经与当前 bootstrap 流程不一致。
2. **状态机变更**：文档基于的状态机、BPMN 网关、CMDB 字段命名已被新规则覆盖。
3. **结论无 revision 锚点**：报告结论（"全部通过""立即上线""零阻断"）未附 Git SHA、镜像 digest、数据库类型或未验证范围。
4. **路线图重复**：内容已被 [`ROADMAP.md`](../../ROADMAP.md) 完全取代，且报告本身不携带额外证据。
5. **被新规范覆盖**：评审中识别的规则已迁移到 [`docs/architecture/`](../architecture/) 或 [`docs/testing/`](../testing/)，原报告仅保留追溯价值。
6. **一次性的烟测记录**：非可重复执行的开发日烟测（如 `dev-environment-test-report.md`），除非与未来回归测试强相关。

每条迁移在 PR 描述中必须包含：

- 原文件路径与目标路径；
- 触发归档的准则编号（1-6）；
- 是否仍引用该文件（若是，必须同步更新引用方）。

## 已完成的迁移

| 迁移文件 | 目标 | 触发准则 | 时间 |
|---|---|---|---|
| `docs/test-plan/itst-test-plan-v1.md` | `archive/testing-reports/itst-test-plan-v1.md` | #1 #2 #3 | 2026-08 |
| `docs/test-plan/itst-similar-bugs-test-plan-v1.md` | `archive/testing-reports/itst-similar-bugs-test-plan-v1.md` | #2 | 2026-08 |
| `docs/test-plan/itst-test-report-p0/p1/p2-v1.md` | `archive/testing-reports/` | #1 #3 | 2026-08 |
| `docs/test-plan/itst-test-summary-v1.md` | `archive/testing-reports/` | #3 | 2026-08 |
| `docs/review/servicenow-benchmark-2026-06-18.md` | `archive/reviews/` | #4 #5 | 2026-08 |

## 使用原则

- 需要了解当前如何部署、开发或运维时，不从归档开始，先看 [文档中心](../README.md)。
- 归档文档可以引用旧路径、旧文件名或旧环境，不保证与当前代码完全一致。
- 如果归档内容重新成为长期有效指南，应移动回合适的正式目录，并更新 [文档中心](../README.md)。
- 评审 / 测试 / 发布报告的强制字段（Git SHA / 镜像 digest / 数据库 / 未验证范围）由 [`scripts/docs-gate/check-release-claims.sh`](../../scripts/docs-gate/check-release-claims.sh) 校验，缺失字段的归档文档允许继续保留（historical 状态不阻断），但新增文档必须满足。

## 归档清理频率

- **每个版本发布后**：release manager 评审 `docs/review/`、`docs/test-plan/`，按准则 1-6 决定是否迁移。
- **每年 1 月**：归档 owner 清理超出 24 个月的快照，必要时可物理删除（保留 index 文件）。
- **任何迁移**：必须在 [`docs/archive/README.md`](./README.md) 的"已完成的迁移"表格登记。
