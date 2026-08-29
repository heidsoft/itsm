# Security Policy

## Supported Versions

| 版本 | 状态 |
|------|------|
| v1.5.x | ✅ Supported（推荐生产部署） |
| v1.1.x | ⚠️ 仅安全补丁 |
| v1.0.x | ❌ EOL（2026-06-30 起） |
| < v1.0 | ❌ EOL |

仅对标 ✅ Supported 的版本接收新功能 + 安全修复；⚠️ 仅安全补丁；❌ 无任何补丁。

## Reporting a Vulnerability

请**不要**通过公开 Issue、PR、Discussion 报告安全漏洞。

### 首选渠道：GitHub 私有漏洞报告

<https://github.com/heidsoft/itsm/security/advisories/new>

GitHub 原生的私有披露通道，仅维护者可见，支持修复发布前的协同讨论与 CVE 申请。

### 备选渠道：邮件

若无法使用上述通道（例如组织网络限制 GitHub 访问），请发送邮件至维护者安全邮箱。

> **维护者待办 —— 发布前必须完成，否则本节形同虚设**
>
> 1. 在仓库 Settings → Code security 中启用 *Private vulnerability reporting*；
> 2. 将本节占位邮箱替换为**真实可接收**的邮箱。原示例 `security@itsm.local`
>    使用了 `.local` 保留域名，外部邮件**无法投递**；
> 3. 如需提供 PGP 加密，请先实际发布公钥再更新路径 —— 原引用
>    `docs/security/pgp-key.asc` 所在的 `docs/security/` 目录并不存在。

请在报告中包含：

1. 漏洞类型（XSS / SSRF / SQL 注入 / 认证绕过 / 信息泄露 / 越权 / ...）
2. 受影响的版本号（commit SHA 优先）
3. 复现步骤或 PoC（命令、请求、截图）
4. 影响的范围（租户隔离是否被绕过？是否需要认证？）
5. 已知的缓解措施（若有）

## Response Timeline

| 阶段 | 承诺 |
|------|------|
| 首次响应 | 报告后 72 小时内 |
| 初步评估 | 7 天内给出严重级别（CRITICAL / HIGH / MEDIUM / LOW） |
| 修复发布（CRITICAL） | 14 天内 |
| 修复发布（HIGH） | 30 天内 |
| 修复发布（MEDIUM / LOW） | 下个 minor 版本 |

## Scope

范围内：仓库内所有 Go 后端（`itsm-backend/`、`itsm-ai-service/`、`itsm-rag/`）、前端（`itsm-frontend/`）、脚本与 CI（`.github/workflows/`）。

不在范围内：第三方依赖的已知 CVE → 请提交到对应上游；私有部署实例由运营方自行负责。

## Recognition

公开致谢名单将在首位外部报告者出现后建立于 `docs/security/hall-of-fame.md`（该目录目前尚未创建）。报告者可在修复发布后选择是否匿名披露。