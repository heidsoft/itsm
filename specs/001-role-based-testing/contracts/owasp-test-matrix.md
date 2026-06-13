# Contract: OWASP Top 10 → 测试 ID → 自动化层 映射

**Feature**: 001-role-based-testing
**Authority**: spec FR-901..908 / SC-401。
**Layers**: `smoke` = `docs/scripts/smoke-api.sh`，`e2e/<role>` = Playwright 角色 spec，`config` = 启动期 / 运维校验，`out-of-scope` = post-GA 跟进。

## 映射表

| OWASP | 描述 | spec FR | 测试 ID | 自动化层 | 期望 |
|-------|------|---------|---------|----------|------|
| A01 失效访问控制 | 越权调用管理 API | FR-901 | DENY-01..04 | e2e/end-user / e2e/security | 403 |
| A01 横向越权 | 用户 A 改用户 B 工单 | FR-901 | DENY-04 | e2e/end-user | 403 |
| A02 加密失效 | JWT 篡改 | FR-902 | SEC-A02-01 | e2e/security | 篡改任意位 → 401 |
| A02 加密失效 | 过期 token | FR-902 | SEC-A02-02 | smoke | exp -1 → 401 |
| A03 注入 | title 含 `' OR 1=1 --` | FR-903 | SEC-A03-01 | smoke | 持久化但渲染转义；DB 不破坏 |
| A03 注入 | CI 名 `<script>...</script>` | FR-903 | SEC-A03-02 | e2e/engineer | 持久化但 HTML 转义 |
| A05 安全错配 | SERVER_MODE / CORS / Redis | FR-904 | SEC-A05-01 | config | 生产 release + 显式 CORS + 强密码 |
| A05 安全错配 | Redis NOAUTH fallback | FR-904 | SEC-A05-02 | smoke | 应用不崩溃 + warning 日志 |
| A06 已过期组件 | 依赖 SBOM 扫描 | — | — | out-of-scope | post-GA |
| A07 鉴权失效 | 暴力登录 5 次 | FR-905 | SEC-A07-01 | smoke | 第 6 次 429 + audit `login_failed` |
| A07 鉴权失效 | 弱密码注册 | FR-905 | — | out-of-scope | 当前无注册接口 |
| A08 数据完整性 | 篡改请求 `tenant_id` | FR-906 | DENY-11 | e2e/tenant-admin | 服务端忽略，仍记 token tenant |
| A08 数据完整性 | 篡改请求 `requester_id` | FR-906 | SEC-A08-02 | e2e/end-user | 服务端忽略，仍记 token user |
| A09 日志缺失 | 关键操作审计 | FR-907 / FR-803 | SEC-A09-01 | smoke | 创建工单后 `/audit-logs` 至少 1 条 |
| A09 日志缺失 | 失败登录审计 | FR-907 | SEC-A09-02 | smoke | 5 次失败登录后 `/audit-logs` ≥5 条 `login_failed` |
| A10 SSRF | 连接器 webhook 内网 URL | FR-908 | SEC-A10-01 | smoke | `http://127.0.0.1:22` 拒绝 |
| A10 SSRF | 连接器 webhook 公有 URL | FR-908 | SEC-A10-02 | smoke | 公有合法 URL 通过 |

## 验收

- SC-401：以上 `smoke` + `e2e` 行 100% 通过。
- A06 在 GA 范围之外，由 post-GA SBOM/扫描专项跟踪。

## 编号约定

- `DENY-XX` / `ALLOW-XX` → 引用自 `role-permission-matrix.md`，由角色 E2E 实现。
- `SEC-AXX-NN` → OWASP 专项，按所属层（smoke/e2e）实现并打 tag 便于过滤。

## 失败处理

- `smoke` 层失败 → 冒烟脚本累加 fails，最后 exit 1。
- `e2e` 层失败 → Playwright 报告 + screenshots + traces 上传 CI artifact。
- `config` 层失败 → 进程启动失败 fail-fast，CI step 失败。
