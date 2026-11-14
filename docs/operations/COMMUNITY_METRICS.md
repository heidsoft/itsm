# 开源经营指标（Community Metrics）· 占位骨架

> 对应手册条款：§6.6 开源经营指标 / §10.2 增长飞轮量化。
> **状态：PLACEHOLDER — 口径定义先行，轻量采集待建。不得虚构数字。**

## 指标口径（定义）

| 指标 | 定义 | 采集源 |
|---|---|---|
| README → 安装 | 访问 README 后实际 clone / docker 拉起次数 | 仓库 clone 统计 / 镜像拉取 |
| 首次部署成功率 | 按文档首次部署成功比例 | 部署脚本埋点 / issue 反馈 |
| PR 合并周期 | 从开 PR 到合并中位时长 | GitHub API |
| 外部贡献者数 | 非核心成员的合并 PR 数 | GitHub API |
| 社区信号 | Star / Fork / Issue / PR 趋势 | star-history（已有 SVG） |

## 轻量采集建议（待建）

- 复用 `scripts/generate-star-history.mjs` 已有 Star 趋势。
- 周/月脚本导出 GitHub API 指标到本文件（或独立 dashboard）。

## 待填字段

- 真实基线值（首月/首季目标）。
- 与 §5 商业化路径的关联（部署→PoC→签约转化率）。

## 红线

所有指标须来自真实采集；不刷量、不美化 Star/Contributor 数字。
