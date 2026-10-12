# Release Reports

> Status: current

> **Status**: current. 自 v1.5 起强制。
> **维护人**：Release Manager / 发布负责人 / 数据库负责人。
> **来源依据**：[`docs/documentation-governance.md`](../documentation-governance.md) §"文档状态标识"与 §"维护规则"。

本目录集中维护所有版本发布 / 初始化 / GA 验收报告。**所有报告必须满足 [`report-template.md`](./report-template.md) 中规定的强制字段**；缺失字段的 PR 由 `scripts/docs-gate/check-release-claims.sh` 标为 advisory（v1.5），未来升级为 hard gate。

---

## 目录结构

```
docs/delivery/
├── release-guide.md       # 本文件
├── report-template.md     # 发布报告模板（含 frontmatter 强制字段）
└── <version>/             # 按版本子目录组织
    └── certification-<date>.md
```

> 历史上散落在仓库根目录的认证报告（`docs/initialization-release-certification.md`、`docs/release-v1.5.0-certification-evidence.md`）已在 2026-10-12 文档清理中删除；发布报告一律按本目录规范新建，不再散放根目录。

---

## 当前报告索引

暂无按本目录规范建立的正式报告。

> v1.5.0 与初始化两轮 2026-07-30 的认证快照（原 `docs/release-v1.5.0-certification-evidence.md`、`docs/initialization-release-certification.md`）已在 2026-10-12 文档清理中删除；如需回溯原文，用 `git log --diff-filter=D -- <path>` 取回。历史快照不迁入本目录，避免误导读者认为其代表当前发布状态。

---

## 必须满足的强制字段

每个报告必须包含以下字段（详见 [`report-template.md`](./report-template.md)）：

| 字段 | 含义 | 缺失后果 |
|---|---|---|
| `git_sha` | 当次提交哈希（短 7 位） | 报告无效，CI advisory 标记 |
| `image_digest` | 生产镜像 SHA256 digest | 报告无效，CI advisory 标记 |
| `database` | PostgreSQL 版本 + 库名 | 报告无效，CI advisory 标记 |
| `unverified_scope` | 未验证层级与已知跳过 | 报告无效，CI advisory 标记 |
| `deployment_mode` | private / saas / saas_msp | 报告无效，CI advisory 标记 |
| `commands` | 关键执行命令 | 不可复现，记 P1 |
| `date` | 当次执行日期 | 缺时间锚点，记 P2 |
| `signoff` | 签字字段（release/security/dba） | 缺签字则禁止作为发布凭证 |

---

## 禁止出现的表述

任何报告若包含下列无 revision 锚点的断言，将被 `scripts/docs-gate/check-release-claims.sh` 标红（v1.5 advisory）：

- "全部通过" / "全部 OK" / "全部完成"
- "立即上线" / "可以发布"
- "零阻断" / "零问题" / "无 P0"
- "完美" / "彻底解决"
- "已 100% 覆盖" / "覆盖率达成"（不附 commit / 报告链接）

合规写法必须包含：

```md
| 项目 | 命令 | 结果 |
|---|---|---|
| 后端全量测试 | `cd itsm-backend && go test ./... -count=1` | ✅ 34 packages passed (commit abc1234, 2026-08-12) |
```

每条结论绑定 `commit SHA + 日期 + 命令`，缺一则不视为可签字证据。

---

## 与 GitHub Release 同步

每次正式发布的 GitHub Release **必须**包含：

1. `git_sha`（commit SHA）
2. `image_digest`（生产镜像 digest）
3. `database`（生产库信息）
4. `unverified_scope`（已知未验证项）
5. 指向本目录下报告的链接

模板段落示例：

```markdown
## 部署证据

- **Git SHA**: `abc1234`
- **Backend image digest**: `sha256:xxxxxxxxxxxxxxxxxxxx`
- **Frontend image digest**: `sha256:yyyyyyyyyyyyyyyyyyyy`
- **Database**: PostgreSQL 17.10 / itsm_prod
- **Deployment mode**: private
- **完整报告**: docs/delivery/v1.6.0/certification-2026-08-15.md（待创建）
- **已知未验证项**: 跨 PG 大版本 `pg_upgrade` 实机演练（跟踪中）
```

CI 校验 `gh release create` 的 description 包含这些字段（`scripts/docs-gate/check-release-claims.sh` 在发布 workflow 中调用）。

---

## 引用

- [`docs/documentation-governance.md`](../documentation-governance.md) — 文档权威层级
- [`report-template.md`](./report-template.md) — 报告模板
- [`docs/testing/test-invariants.md`](../testing/test-invariants.md) §6 — 报告字段约束
- `scripts/docs-gate/check-release-claims.sh` — 自动校验
