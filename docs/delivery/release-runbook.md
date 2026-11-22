# Release Runbook

> Status: current
> **适用范围**：v1.6.x+ 版本发布（tag-triggered 或手动重跑）
> **前置条件**：main 分支绿色 CI、CHANGELOG 已更新、Docker daemon 可用

---

## 发布流程总览

```
1. 准备 → 2. 打 tag → 3. CI 自动发布 → 4. 验证产物 → 5. 通知
```

---

## 1. 准备阶段

### 1.1 确认发布资格

```bash
# 确认 main 分支 CI 全绿
gh run list --branch main --limit 5

# 确认无未合并的阻断 PR
gh pr list --state open --search "label:release-blocker"
```

### 1.2 更新 CHANGELOG

确保 `CHANGELOG.md` 中 `[Unreleased]` 下的条目已整理到新版本号下：

```markdown
## [1.6.12] - 2026-10-15

### 发布说明

- 工单列表支持按处理人筛选。值班组可以快速找到待跟进的工单。
- 升级前请备份数据库；如需执行迁移，在此写明命令和验证方法。

### Fixed

- 在此记录内部修复、实现和测试细节。
```

每个新版本在对应版本标题下写一节 `### 发布说明`。每条先写用户看得见的变化，涉及升级时再补充前置条件、迁移和验证步骤。维护者记录继续放在 Added、Changed、Fixed 等小节。预览命令为 `bash scripts/release-notes.sh v1.6.12`。

### 1.3 使用准备脚本（推荐）

```bash
# 预览将要做什么
./scripts/prepare-release.sh v1.6.12 --dry-run

# 确认无误后执行
./scripts/prepare-release.sh v1.6.12

# 只打 tag 不创建 Release（CI 会自动创建）
./scripts/prepare-release.sh v1.6.12 --tag-only
```

脚本会依次执行：
1. 工作区清洁检查（无未提交变更）
2. CHANGELOG 条目存在性验证
3. 提取 release notes
4. 检查远程同步状态
5. 创建 annotated tag 并推送
6. 创建 GitHub Release（`--tag-only` 模式跳过此步）

---

## 2. 打 tag

若手动打 tag：

```bash
# annotated tag（必须，轻量 tag 不含元信息）
git tag -a v1.6.12 -m "Release v1.6.12"

# 推送 tag
git push origin v1.6.12
```

推送后 `.github/workflows/release.yml` 自动触发。

---

## 3. CI 自动发布

### 3.1 监控 workflow 进度

```bash
# 查看最新 workflow run
gh run list --workflow release.yml --limit 1

# 实时跟踪日志
gh run watch
```

### 3.2 Workflow 包含 4 个 job

| Job | 产物 | 预计耗时 |
|-----|------|---------|
| build-backend | 5 个平台二进制（linux/darwin/windows × amd64/arm64） | ~3 min |
| build-frontend | Next.js standalone 构建 | ~4 min |
| release | GitHub Release + zip 归档 | ~1 min |
| publish-containers | backend + frontend GHCR 镜像 | ~5 min |

### 3.3 手动重跑

若 CI 失败但代码无误（如网络抖动），使用 `workflow_dispatch`：

```bash
gh workflow run release.yml -f tag=v1.6.12
```

或在 GitHub UI：Actions → Build & Release → Run workflow → 输入 tag。

---

## 4. 验证产物

### 4.1 GitHub Release

```bash
# 检查 Release 状态
gh release view v1.6.12

# 验证附件
gh release view v1.6.12 --json assets --jq '.assets[].name'
```

预期附件：
- `itsm-v1.6.12-backends.zip`
- `itsm-v1.6.12-frontend.zip`

新版本的 Release 正文来自对应版本的「发布说明」小节。旧版本没有这个小节时，发布流程读取该版本的完整条目。

### 4.2 Docker 镜像

```bash
# 验证镜像已推送 GHCR
docker pull ghcr.io/heidsoft/itsm-backend:v1.6.12
docker pull ghcr.io/heidsoft/itsm-frontend:v1.6.12

# 验证 tag 完整性
docker pull ghcr.io/heidsoft/itsm-backend:1.6.12
docker pull ghcr.io/heidsoft/itsm-backend:1.6
docker pull ghcr.io/heidsoft/itsm-backend:latest
```

预期 tag 矩阵（每个镜像）：

| Tag | 来源 |
|-----|------|
| `v1.6.12` | git ref tag |
| `1.6.12` | semver pattern |
| `1.6` | semver major.minor |
| `latest` | 始终启用 |

### 4.3 快速冒烟

```bash
# 启动容器验证健康检查
docker run -d --name itsm-smoke \
  -e DB_HOST=host.docker.internal \
  -e DB_PASSWORD=your_password \
  -p 8090:8090 \
  ghcr.io/heidsoft/itsm-backend:v1.6.12

curl -s http://localhost:8090/api/v1/health | jq .
```

---

## 5. 常见问题

### Release 创建失败但 tag 已推送

tag 已存在时 `softprops/action-gh-release` 不会重复创建。解决：

```bash
# 删除 Release（不删 tag）
gh release delete v1.6.12 --yes

# 手动重跑
gh workflow run release.yml -f tag=v1.6.12
```

### CHANGELOG 提取为空

发布流程会报错。检查目标 tag 的版本标题和「发布说明」是否存在且有正文，再按版本流程重新准备发布。

### Docker 镜像推送失败

通常是 GHCR 认证问题。检查：
1. `GITHUB_TOKEN` 是否包含 `packages:write` 权限
2. workflow 的 `permissions` 块是否完整

### workflow_dispatch 重跑后镜像 tag 不对

确认输入的 tag 格式为 `v*`（如 `v1.6.12`），不带前缀空格或多余字符。

---

## 6. 发布后

1. 更新 `CHANGELOG.md` 的 `[Unreleased]` 小节为空（为下个版本准备）
2. 若需要发布报告，按 `docs/delivery/release-guide.md` 创建认证报告
3. 通知相关方 Release 链接与镜像地址

---

## 引用

- [`.github/workflows/release.yml`](../../.github/workflows/release.yml) — 发布 workflow 定义
- [`scripts/prepare-release.sh`](../../scripts/prepare-release.sh) — 发布准备脚本
- [`docs/delivery/release-guide.md`](./release-guide.md) — 发布报告规范
- [`CHANGELOG.md`](../../CHANGELOG.md) — 变更日志
