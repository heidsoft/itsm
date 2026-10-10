# CLAUDE.md

本仓库 AI 编码助手规范的**唯一事实源是 [AGENTS.md](./AGENTS.md)**，请直接阅读它。

本文件只保留跳转说明，不再承载规范内容，以避免两份长期演进的规则分叉：

- 项目定位、产品方向、当前阶段、开发命令 → [AGENTS.md](./AGENTS.md)
- 后端分层（`handlers/<domain>/`）、身份与租户、PATCH/状态机、Outbox 与幂等、能力失败语义、Context/日志、Schema 迁移等强制规则 → [AGENTS.md](./AGENTS.md)
- 前端契约、命名、Ant Design v6、Hook 正确性与 **Frontend Hook 设计基线（Pattern A–G / 审查 H1–H7）** → [AGENTS.md](./AGENTS.md)

历史说明：本文件曾包含一份 AGENTS.md 的早期副本，并独占「Frontend Hook Patterns」一节。该节已于 2026-10-10 完整并入 AGENTS.md，重叠内容随之删除。新增长期有效规范一律写进 AGENTS.md。

发布与运维事实源仍按 [文档状态与事实源](./docs/documentation-governance.md) 的权威层级判断：**源码、迁移、运行时 API 与当前 CI 优先于任何文档**。

<!-- SPECKIT START -->
<!-- SPECKIT END -->
