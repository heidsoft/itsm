# 文档归档

> Status: historical

这里保存具有长期参考价值的阶段性评审、设计文档和集成指南。已过时的一次性快照（旧缺陷报告、烟测记录、已执行完毕的计划）已在 2026-09 清理中物理删除，git 历史仍可追溯。

## 当前保留文件

| 文件 | 内容 | 保留理由 |
|:---|:---|:---|
| `feishu.md` | 飞书连接器配置与使用指南 | 集成参考，持续有效 |
| `frontend-backend-alignment-audit-2026-08-05.md` | 前后端对齐审计 | 系统性架构发现，有参考价值 |
| `product-debt-review-2026-09-18.md` | 产品债务评审（v1.6.x） | 当前基线 |
| `tech-sharing-guide.md` | 技术分享框架 | 流程参考，持续有效 |
| `reviews/enterprise-v1-readiness-2026-06-07.md` | v1 就绪度分析 | 战略里程碑 |
| `reviews/product-architecture-review-2026-09-03.md` | 产品架构评审 | 当前架构状态与差距 |
| `reviews/product-commercial-review-2026-05-02.md` | 商用价值评审 | 产品定位参考 |
| `reviews/product-retrospective-2026-05-21.md` | 产品复盘 | 里程碑参考 |
| `reviews/screenshot-issues-product-review-2026-09-12.md` | UI 问题根因分析 | 持续改善参考 |
| `superpowers-specs/2026-06-20-logout-bug-fix-design.md` | 登出缺陷修复设计 | Auth cookie 架构参考 |
| `workflow-reports/cn-enterprise-workflow-design.md` | 中国企业工作流设计 | 流程设计理念 |

## 归档准则（v1.5 起）

文档满足下列任一条件时，应在下一轮文档清理中移入 `docs/archive/` 或直接删除：

1. **默认凭据过期**：文档中的示例密码已与当前 bootstrap 流程不一致。
2. **状态机变更**：文档基于的状态机、BPMN 网关、CMDB 字段命名已被新规则覆盖。
3. **结论无 revision 锚点**：报告结论未附 Git SHA、镜像 digest 或未验证范围。
4. **路线图重复**：内容已被 [`ROADMAP.md`](../../ROADMAP.md) 完全取代，且报告本身不携带额外证据。
5. **被新规范覆盖**：评审中识别的规则已迁移到正式文档目录，原报告仅保留追溯价值。
6. **一次性的烟测记录**：非可重复执行的开发日烟测，除非与未来回归测试强相关。

## 使用原则

- 需要了解当前如何部署、开发或运维时，不从归档开始，先看 [文档中心](../README.md)。
- 归档文档可以引用旧路径、旧文件名或旧环境，不保证与当前代码完全一致。
- 如果归档内容重新成为长期有效指南，应移动回合适的正式目录，并更新 [文档中心](../README.md)。

## 归档清理频率

- **每个版本发布后**：release manager 评审归档目录，按准则 1-6 决定是否迁移或删除。
- **每年 1 月**：归档 owner 清理超出 12 个月的一次性快照，物理删除（git 历史保留）。
- **2026-09 清理**：删除 82 个过期文件（10 个缺陷报告、12 个测试报告、11 个计划、10 个评审、5 个 superpowers 计划、5 个 superpowers 设计、3 个工作流报告、7 个根目录旧文件、16 个脚本、3 个未跟踪文件），保留 12 个有持续参考价值的文件。
