# Issue / PR 分诊（Triage）流程 · 占位骨架

> 对应手册条款：§6.5 标签体系与 triage / §6.6 经营指标。
> **状态：PLACEHOLDER — 引用 `docs/label-schema.md`，待填真实 SLA 执行记录。**

## 首次响应 SLA

- 新 Issue 创建后 **1 个工作日内** 必须打 `status/needs-triage` 或 `status/triaged`（label-schema §9）。
- 优先级按 `priority/p0`(4h) → `p3`(2w+) 处理（label-schema §4）。

## 分诊步骤

1. 判类别 → 打 `area/*`（backend/frontend/ai/...，见 label-schema §2）。
2. 判性质 → 打 `kind/*`（feature/bugfix/...，见 label-schema §3）。
3. 判优先级 → 打 `priority/*`；安全类打 `security`（私披露，勿公开）。
4. 判规模 → labeler 自动打 `size/*`（见 label-schema §6）。
5. 阻塞/重复/不修 → `status/blocked` / `duplicate` / `wontfix`（关闭前须说明原因）。

## 周 triage 机制

- 维护者每周集中清 `needs-triage`；`pinned` 不被 stale bot 关闭。
- 标签由 `label-sync.yml` 自动与 `label-schema.md` 同步。

## 待填字段

- 实际首次响应时长、周 triage  throughput（纳入 COMMUNITY_METRICS）。

## 红线

安全漏洞走 SECURITY.md 私披露通道，不在公开 Issue 讨论细节。
