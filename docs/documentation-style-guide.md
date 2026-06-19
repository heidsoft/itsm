# 文档命名与维护规范

本文档定义 `docs/` 目录的长期维护规则。目标是让新用户能快速找到当前有效文档，同时保留项目演进过程中的历史材料。

## 目录分层

| 目录 | 用途 |
|:---|:---|
| `docs/` | 长期有效的核心指南和总入口 |
| `docs/product/` | 产品能力、产品设计、发布能力说明 |
| `docs/architecture/` | 架构设计和长期技术方案 |
| `docs/prd/` | PRD 和需求规格 |
| `docs/articles/` | 对外传播文章、专题说明 |
| `docs/review/` | 当前仍有参考价值的评审、验收、测试结论 |
| `docs/testing/` | 测试方案、测试用例入口、当前测试报告 |
| `docs/delivery/` | 发布工程、生产就绪、CI/CD 方案 |
| `docs/tools/` | 工具配置和辅助指南 |
| `docs/scripts/` | 文档配套脚本 |
| `docs/images/` | README 和文档引用的图片资产 |
| `docs/archive/` | 历史报告、过期计划、阶段性复盘 |

## 文件命名

使用 lowercase kebab-case：

```text
deployment.md
commercial-ready-architecture.md
browser-e2e-test-report-2026-06-18.md
```

规则：

- 文件名使用英文小写、数字和连字符。
- 长期指南不带日期，例如 `deployment.md`。
- 阶段性报告带日期，例如 `frontend-ux-review-2026-06-19.md`。
- 文章可以带序号保持阅读顺序，例如 `07-ai-native-architecture-guidance-harness-skill.md`。
- 测试用例文件允许保留用例编号前缀，例如 `TC-TICKET.md`，因为编号本身是测试资产的一部分。
- 不提交 `.DS_Store`、临时截图、未引用的图片和本地导出文件。

## 内容质量

长期文档至少包含：

- 目标读者
- 适用场景
- 前置条件
- 可执行命令或明确步骤
- 验证方式
- 常见问题或下一步入口

报告类文档至少包含：

- 测试/评审日期
- 范围
- 环境
- 结论
- 阻塞问题
- 后续建议

## 归档规则

满足任一条件时移动到 `docs/archive/`：

- 内容只描述某次历史修复或临时问题。
- 文件名包含旧日期，且不再作为当前流程入口。
- 已被新的正式指南替代。
- 包含旧环境、旧路径或旧结论，继续留在根目录会误导读者。

归档后不要求逐条修复内部旧链接，但需要在 [文档中心](./README.md) 提供当前有效入口。
