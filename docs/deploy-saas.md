# ITSM SaaS 单实例多租户部署手册

> 适用版本：`v1.6.58`（本地 HEAD，非 GA 发布）。协议：Apache-2.0。
> 形态：单台云主机（如阿里云 ECS）+ Docker Compose，单实例服务多租户。
> 范围：本手册是**部署 Runbook**，给出可复现的上云步骤；实际远程部署/推 ACR/上架需
> 用户提供云账号与镜像仓库凭证（见文末「执行边界」）。

---

## 1. 架构与边界

```
                公网 (80/443)
                    │
            ┌───────▼───────┐
            │  Caddy (自动  │  Let's Encrypt 证书，HTTP→HTTPS 重定向
            │  TLS 终止)    │
            └───┬───────┬───┘
         /api/* │       │ 其它（SPA）
                ▼       ▼
        itsm-backend:8090   itsm-frontend:3000
                │               │
        ┌───────┴────┴──────────┴───────────┐
        │  postgres │ redis │ ai-service │ worker │ init（一次性）
        └─────────────────────────────────────┘
```

- 与 `docker-compose.prod.yml` 的区别：边缘用 **Caddy 自动 HTTPS** 替代「仅 80 + 外部 TLS 终止」的 nginx；`DEPLOYMENT_MODE=saas`；镜像支持 `REGISTRY` 前缀（阿里云 ACR）。
- 多租户由应用层（tenant 上下文 + RLS 灰度）实现，**不是** Kubernetes 多命名空间。当前 RLS 默认 `off`，租户隔离依赖应用层鉴权。
- 本形态**非高可用**：单 DB、单 Redis、单 backend。生产级 SaaS（多副本、读写分离、跨 AZ）需 K8s（见 `deploy/k8s/`）。

## 2. 主机与前置

| 项 | 最低 | 推荐（正式试用） |
|---|---|---|
| vCPU / 内存 | 2c / 4G | 4c / 8G |
| 系统盘 | 40G SSD | 100G 高效云盘 |
| OS | Ubuntu 22.04 / Alibaba Cloud Linux 3 | 同上 |
| 入站安全组 | 放通 80、443（TCP）；22 仅限运维 IP | 同左 |
| 出网 | 放通 443（拉镜像 / ACME / LLM） | 同左 |

- 公网域名 `SAAS_DOMAIN` 已做 A 记录指向本机 ECS 公网 IP。
- 80/443 入站必须放通：Caddy 用 HTTP-01 挑战签发证书，签发期间需临时经 80 访问。

## 3. 安装 Docker

```bash
# Ubuntu / Alibaba Cloud Linux
curl -fsSL https://get.docker.com | sudo sh
sudo systemctl enable --now docker
# 可选：将当前用户加入 docker 组，免 sudo
sudo usermod -aG docker $USER   # 重新登录后生效
docker --version && docker compose version
```

## 4. 准备部署目录与配置

```bash
mkdir -p ~/itsm-saas && cd ~/itsm-saas
# 取部署文件（仓库内已有）：
#   deploy/docker-compose.saas.yml
#   deploy/Caddyfile
#   deploy/.env.saas.example
# 放到本目录，并重命名 env 模板：
cp deploy/.env.saas.example .env.saas
```

编辑 `.env.saas`，**至少**替换以下占位符（用强随机值）：

```bash
# 生成示例（本地执行，仅看长度，不回显内容）
openssl rand -hex 32      # → JWT_SECRET
openssl rand -base64 24   # → DB_PASSWORD / REDIS_PASSWORD / CONNECTOR_CONFIG_ENCRYPTION_KEY
# 自定 ADMIN_PASSWORD（勿用 admin123）
```

必填：`SAAS_DOMAIN`、`ACME_EMAIL`、`VERSION`、`DB_PASSWORD`、`REDIS_PASSWORD`、`JWT_SECRET`、`ADMIN_PASSWORD`、`CONNECTOR_CONFIG_ENCRYPTION_KEY`。

> 口令纪律：`.env.saas` 含密钥，**绝不**提交进 git；备份时仅存于受控主机。

## 5. 获取镜像

**方式 A — 本地构建（ECS 上编译，适合先验证）：**

```bash
# 在仓库根目录执行，构建三镜像并打 VERSION tag
VERSION=v1.6.58 make build-prod   # 或等价：docker compose -f docker-compose.prod.yml build
# 确认镜像存在
docker images | grep itsm
```

**方式 B — 从阿里云 ACR 拉取（上架镜像市场后推荐）：**

```bash
# 登录 ACR（需用户提供命名空间与凭证）
docker login registry.cn-hangzhou.aliyuncs.com \
  -u <ACR_USER> -p <ACR_PASSWORD>
# 在 .env.saas 设 REGISTRY=registry.cn-hangzhou.aliyuncs.com/<namespace>/
# 启动命令会自动带前缀拉取
```

> 镜像 tag（`VERSION`）、`/api/v1/version` 返回、发布版本三者必须一致，避免漂移。

## 6. 启动

```bash
docker compose --env-file .env.saas -f deploy/docker-compose.saas.yml up -d
# 观察初始化（init 容器跑迁移+种子，建 admin；跑完即退出）
docker compose --env-file .env.saas -f deploy/docker-compose.saas.yml logs -f itsm-init
# 全部健康
docker compose --env-file .env.saas -f deploy/docker-compose.saas.yml ps
```

预期：`postgres`/`redis` healthy → `itsm-init` completed → `itsm-backend`/`itsm-frontend`/`caddy` healthy。

## 7. 验证

```bash
# 边缘健康检查（Caddy 直接 200）
curl -fsS https://$SAAS_DOMAIN/health

# 业务就绪（后端）
curl -fsS https://$SAAS_DOMAIN/api/v1/readyz

# 版本指纹（须与 VERSION 一致）
curl -fsS https://$SAAS_DOMAIN/api/v1/version

# 登录拿 cookie + CSRF，验证 admin 可用
curl -c cookies.txt -s -o /dev/null -w "%{http_code}\n" \
  -X POST https://$SAAS_DOMAIN/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"<ADMIN_PASSWORD>"}'
```

浏览器访问 `https://$SAAS_DOMAIN`，应出现 ITSM 登录页且地址栏为锁标（有效证书）。

## 8. （可选）邮件建单闭环试用

如需让 SaaS 租户试用 Email-to-Ticket 可收费闭环，复用已验证的种子脚本：

```bash
# 在仓库根：用 demo 覆盖文件向 backend/worker 注入演示模式，再 seed
docker compose -f docker-compose.prod.yml -f docker-compose.demo.yml \
  --env-file .env.prod up -d itsm-backend itsm-worker
bash scripts/seed-email-intake-demo.sh
# 经 API 触发 confirm → 建单（事件号形如 INC-YYYYMM-xxxxxx）
# 清理：bash scripts/unseed-email-intake-demo.sh
```

> 演示模式默认 `fail-closed`（观察模式）；生产 SaaS 不应长期开启自动建单，除非已配置真实 IMAP 连接器与租户凭据。

## 9. 备份与恢复

```bash
# 备份 Postgres（落到 ./data/backups，已挂载为只读源供容器内 pg_dump 不便；
# 这里用宿主直接 exec）
docker exec itsm-saas-postgres pg_dump -U itsm itsm_prod \
  | gzip > ~/itsm-saas-backup-$(date +%F).sql.gz

# 卷位置（命名卷，docker 管理）：
docker volume ls | grep itsm_saas
# 整机快照：ECS 控制台对系统盘/数据盘打快照更稳妥。
```

恢复：新建实例或清空卷后，`docker compose ... up -d itsm-init` 前先把 `.sql` 还原进 `itsm-saas-postgres`。

## 10. 升级

```bash
# 1) 拉取/构建新镜像（改 VERSION）
# 2) 仅后端/前端/ai/worker 滚动：compose 会按 depends_on 顺序重建
docker compose --env-file .env.saas -f deploy/docker-compose.saas.yml pull   # 若用 ACR
docker compose --env-file .env.saas -f deploy/docker-compose.saas.yml up -d
# 3) 验证 /api/v1/version 与 /api/v1/readyz
```

> DB schema 迁移由 `itsm-init` 在 `ITSM_AUTO_MIGRATE=true` 时执行；升级前务必先备份。

## 11. 监控（可选）

```bash
docker compose --env-file .env.saas -f deploy/docker-compose.saas.yml \
  --profile monitoring up -d
# 并把 .env.saas 的 METRICS_REQUIRE_AUTH 设为 false，供 Prometheus 抓取
# 访问：ssh 隧道到 127.0.0.1:3001（Grafana）/9090（Prometheus），勿直暴露公网
```

## 12. 红线与口径

- **非 GA**：当前为 `v1.6.58` 本地 HEAD，仅供试用/演示，不得宣称「正式商用」「已上线 GA」。
- **不写「替代 ServiceNow」**：对外物料仅表述为「开源 ITSM 平台，对标企业级 ITSM 能力」。
- **协议不变**：Apache-2.0，部署包与文档随仓库分发，须保留 LICENSE/NOTICE。
- **口令**：所有密钥经环境变量注入，不回显、不落盘日志、不进 git。
- **证书**：由 Caddy 自动续期；若域名解析或 80 入站异常，证书签发会失败，业务不可用——先排网络再查 `docker logs itsm-saas-caddy`。

## 13. 执行边界（本环境能做的 / 不能做的）

- ✅ 本环境已完成：SaaS 部署包（`deploy/docker-compose.saas.yml` + `Caddyfile` + `.env.saas.example`）、本手册、下方阿里云镜像市场上架材料。
- ⏸️ 需用户提供凭证后方可执行：① 实际 SSH 到 ECS 跑部署；② `docker push` 到阿里云 ACR；③ 在阿里云镜像市场提交上架。以上不在本会话自动执行范围内，待用户授权与凭证。

---

相关文件：
- `deploy/docker-compose.saas.yml` — SaaS compose（Caddy 边缘 + saas 模式）
- `deploy/Caddyfile` — 自动 TLS 反代
- `deploy/.env.saas.example` — 环境变量模板
- `docker-compose.demo.yml` / `scripts/seed-email-intake-demo.sh` — 邮件建单试用
- `docs/aliyun-marketplace.md` — 镜像市场上架材料
