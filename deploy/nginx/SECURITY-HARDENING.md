# 公网安全加固记录（2026-11-15）

适用资产：官网 `cloudmesh.top` + ITSM SaaS `itsm.cloudmesh.top`（同一台 ECS `39.97.54.216`，共用宿主 nginx）。
本目录保存**加固后**的 nginx 配置副本，作为可复现产物。线上已生效并验证（PASS 10/FAIL 0）。

## 一、已实施的加固

### 1. demo 演示账号降为只读（最高危项）
- 官网公开的 `demo`（id=2, tenant=1）原角色 `it_admin`（可增删改），公网任何人可登录改数据。
- 已通过 DB 改为 `audit_admin`（全租户只读，40 个权限无任何 `:write`），并重设公开密码的 bcrypt hash 使旧 JWT 失效。
- 角色定义见 `internal/authz/roles.go` 的 `domainrole.AuditAdmin`。
- ⚠️ 副作用：demo 无 `ticket:write`，不能建单；官网文案应表述为"只读浏览系统全貌"。
- 回滚：ECS `/tmp/demo_backup_20261115.txt`（原 role=it_admin）。

### 2. nginx 全局（`nginx.conf.hardened`）
- `server_tokens off`：响应不再泄露 `nginx/1.20.1`，只显示 `nginx`。
- 两个限流 zone：`itsm_auth`（10r/m，登录防爆破）、`itsm_api`（60r/m，通用防 CC）。

### 3. 官网 deny 规则（`cloudmesh-common.conf` = HTTPS；`cloudmesh-http.conf` = HTTP）
- 新增危险扩展名封禁：`sh/bash/zsh/py/pyc/pyo/bat/cmd/exe/so/dll/bak/old/orig/save/swp/tmp/zip/tar/gz/tgz/bz2/7z/rar/sql/sqlite/db/pem/key/p12/pfx/crt/cer/der/log/env/example/mjs/ts/template`。
- 保留 `.md`（技能文档）、`.html`、`.json`（元数据）、`.css/.js/.png`（展示资源）。
- 🔴 **顺序要求**：deny 正则必须在静态资源 `location ~* \.(css|js|...)$` 之前（nginx 正则 location 首次匹配生效）。
- 🔴 **不能整目录 deny `/skills/`**：那是技能市场展示页；只封其中的脚本文件类型。

### 4. SaaS 限流（`nginx.itsm.cloudmesh.top.conf`）
- `/api/v1/auth/{login,register,refresh,password,forgot}`：10r/m + burst 5，超限 429。
- 其余 `/api/`：60r/m + burst 30，超限 429。
- `/health` 边缘直返不受影响；SPA 页面不受影响（只限 API）。

## 二、回滚点（ECS）
- nginx 配置：`/etc/nginx/*.bak-20261115-194343`、`conf.d/*.bak-20261115-194343`、`snippets/*.bak-20261115-194343`。
- 回滚命令示例：`cp /etc/nginx/nginx.conf.bak-20261115-194343 /etc/nginx/nginx.conf && nginx -t && nginx -s reload`。

## 三、验证结果
- 官网敏感路径（skills 脚本 / py 脚本 / .bak / deploy.sh）HTTP+HTTPS 均 403。
- 官网正常页（首页 / ITSM 产品页 / 技能市场 / 登录页）均 200。
- `server: nginx`（版本隐藏）；登录连击前 6 次 401、第 7 次起 429。
- demo 登录 `role=audit_admin`、`ticket:read=true`、`ticket:write=false`。
- SaaS health=ok，版本 v1.6.58；公网浏览器两域名渲染正常。

## 四、可观测性：如何查看「谁访问 / 谁登录 demo」
 ECS 上已部署只读查询脚本 `/opt/itsm-saas/saas-audit.sh`（仓库源 `deploy/saas-audit.sh`）：
 ```
 ./saas-audit.sh visits [N]       # 最近 N 条真实访问（nginx access.log：公网 IP/UA/URI/状态）
 ./saas-audit.sh demo-login [N]   # demo 登录审计（audit_logs：北京时间/结果/状态/IP）
 ./saas-audit.sh login-fails [N]  # 全站登录失败（暴力破解迹象）
 ./saas-audit.sh top-ips [N]      # 访问来源 IP TOP N
 ./saas-audit.sh blocked          # fail2ban 封禁概况
 ```
 两套日志分工：`audit_logs`（应用审计）记录**哪个账号**（user_id/action=LOGIN_SUCCESS|FAILED），公网登录能拿到真实来源 IP；
 nginx `access.log` 记录**所有请求的真实来源 IP/UA**，二者按时间对齐。容器直连测试会显示
 `127.0.0.1`/`172.29.0.1` 属正常。日志按天轮转 gzip（保留至 11-06），历史可 `zcat access.log-YYYYMMDD.gz | grep ...`。

## 五、未做 / 待办
- **SSH 保持原状**：用户决定仅依赖 fail2ban（bantime 3600s / maxretry 3 / 已封 219 IP / 开机自启）。未改 `PermitRootLogin`/`PasswordAuthentication`。
- **云安全组**：aliyun CLI 反查实例返回空（cred/区域问题），需在控制台人工复核仅开放 80/443/22。
- **官网 HSTS 仍注释**：证书自动续期未闭环，尊重原设计；续期自动化就绪后再开。
- **fail2ban jail 已修（2026-11-15 19:55）**：`nginx-404`/`nginx-scan` 之前 Total banned=0 的根因是 logpath 写成
  `cloudmesh-access.log`（中划线，空文件），实际日志是 `cloudmesh_access.log`（下划线）。已改 jail.local 两处并
  `fail2ban-client reload`，日志确认 `Added logfile` 且开始 `Found <IP>`；备份 `/etc/fail2ban/jail.local.bak-20261115-1956`。
  Total banned 暂为 0 是因 maxretry=20 门槛未到，filter 实测匹配 52 条，工作正常。
- **demo 定期重置**：建议建立轮换机制（如每月重设公开密码 hash），降低长期暴露风险。
