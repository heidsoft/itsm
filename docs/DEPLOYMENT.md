# ITSM 生产环境部署指南

**最后更新**: 2026-03-04
**版本**: v2.0

---

## 📋 目录

- [部署概览](#部署概览)
- [部署方式选择](#部署方式选择)
- [Docker Compose 部署](#docker-compose-部署-推荐)
- [Kubernetes 部署](#kubernetes-部署-可选)
- [环境配置](#环境配置)
- [Nginx 配置](#nginx-配置)
- [HTTPS 配置](#https-配置)
- [数据初始化](#数据初始化)
- [部署验证](#部署验证)
- [备份与恢复](#备份与恢复)
- [监控告警](#监控告警)
- [性能调优](#性能调优)
- [安全加固](#安全加固)
- [升级流程](#升级流程)

---

## 部署概览

```
┌─────────────────────────────────────────────────────────────────┐
│                        生产环境架构                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌──────────┐                                                  │
│   │   用户    │                                                 │
│   └────┬─────┘                                                 │
│        │                                                        │
│        ▼                                                        │
│   ┌──────────┐                                                  │
│   │   CDN    │  (可选)                                          │
│   └────┬─────┘                                                 │
│        │                                                        │
│        ▼                                                        │
│   ┌────────────────────────────────────────────────────────┐    │
│   │                      Nginx                              │    │
│   │              负载均衡 / SSL / 静态资源                   │    │
│   └────────────────────────┬───────────────────────────────┘    │
│                            │                                     │
│            ┌───────────────┴───────────────┐                     │
│            ▼                               ▼                     │
│   ┌───────────────┐             ┌───────────────┐               │
│   │   前端服务    │             │   后端服务    │               │
│   │  (Next.js)   │             │    (Go)       │               │
│   │   端口: 80   │             │   端口: 8090  │               │
│   └───────────────┘             └───────┬───────┘               │
│                                         │                        │
│            ┌────────────────────────────┼────────────────┐      │
│            ▼                            ▼                ▼      │
│   ┌───────────────┐  ┌──────────────┐  ┌────────────┐          │
│   │  PostgreSQL   │  │    Redis     │  │   MinIO   │          │
│   │  端口: 5432   │  │  端口: 6379  │  │ 端口:9000 │          │
│   └───────────────┘  └──────────────┘  └────────────┘          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 部署方式选择

| 方式 | 适用场景 | 复杂度 | 扩展性 | 维护成本 |
|------|---------|--------|--------|---------|
| Docker Compose | 小规模（<100 用户） | 低 | 中 | 低 |
| Kubernetes | 中大规模（100+ 用户） | 高 | 高 | 中 |
| 物理机/虚拟机 | 传统部署 | 中 | 低 | 高 |

**推荐**:
- 初创/小团队: Docker Compose
- 中大型企业: Kubernetes

---

## Docker Compose 部署（推荐）

### 1. 环境准备

```bash
# 1. 准备服务器（以 Ubuntu 22.04 为例）

# 安装 Docker
curl -fsSL https://get.docker.com | sh
sudo systemctl enable docker
sudo systemctl start docker

# 安装 Docker Compose v2
sudo apt-get update
sudo apt-get install docker-compose-plugin

# 验证安装
docker --version
docker compose version

# 配置 Docker 开机启动
sudo systemctl enable docker
```

### 2. 准备配置文件

```bash
# 克隆项目到服务器
git clone https://github.com/heidsoft/itsm.git
cd itsm

# 创建生产环境配置文件
cp .env.example .env.prod
cp docker-compose.yml docker-compose.prod.yml
```

### 3. 配置环境变量

编辑 `.env.prod`:

```env
# ========== 服务器配置 ==========
SERVER_ENV=production
LOG_LEVEL=error
GIN_MODE=release

# ========== 后端配置 ==========
SERVER_PORT=8090
JWT_SECRET=your-very-long-random-secret-key-min-32-characters-change-this
JWT_EXPIRE=3600

# ========== 数据库配置 ==========
DB_HOST=postgres
DB_PORT=5432
DB_USER=itsm
DB_PASSWORD=VeryStrongPostgresPassword!@#$%
DB_NAME=itsm_prod
DB_SSLMODE=disable

# ========== Redis 配置 ==========
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=VeryStrongRedisPassword!@#$%
REDIS_DB=0

# ========== 功能开关 ==========
ENABLE_SWAGGER=false
ENABLE_CORS=true
ENABLE_AI=true
```

### 4. 构建并启动

```bash
# 构建镜像（首次）
docker compose -f docker-compose.prod.yml build

# 启动所有服务
docker compose -f docker-compose.prod.yml up -d

# 查看状态
docker compose -f docker-compose.prod.yml ps

# 查看日志
docker compose -f docker-compose.prod.yml logs -f backend
```

### 5. 数据初始化

```bash
# 等待数据库就绪
sleep 10

# 初始化数据库（自动迁移）
docker compose -f docker-compose.prod.yml exec -T backend go run main.go

# 初始化种子数据
docker compose -f docker-compose.prod.yml exec -T backend go run main.go seed
```

---

## Kubernetes 部署（可选）

### 1. 前提条件

- Kubernetes 集群（v1.24+）
- kubectl 已配置
- Helm 3+（可选）

### 2. 创建 Namespace

```bash
kubectl create namespace itsm
```

### 3. 创建 Secret

```bash
# 创建密钥
kubectl create secret generic itsm-secrets \
  --from-literal=db-password='StrongPostgresPassword' \
  --from-literal=redis-password='StrongRedisPassword' \
  --from-literal=jwt-secret='VeryLongRandomSecretKey' \
  -n itsm
```

### 4. 部署应用

详细 K8s 配置请参考 [K8S_DEPLOYMENT.md](./K8S_DEPLOYMENT.md)

---

## 环境配置

### 生产环境检查清单

- [ ] 服务器系统版本确认
- [ ] 防火墙端口开放（80, 443, 22）
- [ ] 时区设置正确
- [ ] 域名解析配置
- [ ] SSL 证书准备
- [ ] 备份策略配置

### 目录结构

```
/opt/itsm/
├── docker-compose.prod.yml
├── .env.prod
├── data/
│   ├── postgres/      # 数据库数据
│   ├── redis/         # Redis 数据
│   └── uploads/       # 上传文件
├── logs/              # 日志目录
└── backups/          # 备份目录
```

---

## Nginx 配置

### 1. 安装 Nginx

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install nginx

# CentOS/RHEL
sudo yum install nginx
sudo systemctl start nginx
```

### 2. 配置反向代理

创建 `/etc/nginx/sites-available/itsm`:

```nginx
upstream itsm_backend {
    server 127.0.0.1:8090 max_fails=3 fail_timeout=30s;
}

upstream itsm_frontend {
    server 127.0.0.1:3000 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    server_name itsm.yourdomain.com;

    # 重定向到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name itsm.yourdomain.com;

    # SSL 配置
    ssl_certificate /etc/ssl/certs/itsm.crt;
    ssl_certificate_key /etc/ssl/private/itsm.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
    ssl_prefer_server_ciphers off;
    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:50m;

    # HSTS
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;

    # 前端代理
    location / {
        proxy_pass http://itsm_frontend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # API 代理
    location /api/ {
        proxy_pass http://itsm_backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 300s;
        proxy_connect_timeout 75s;
    }

    # 健康检查
    location /health {
        proxy_pass http://itsm_backend/health;
        access_log off;
    }

    # 静态资源缓存
    location /_next/static/ {
        proxy_pass http://itsm_frontend;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

启用配置:

```bash
sudo ln -s /etc/nginx/sites-available/itsm /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

## HTTPS 配置

### 使用 Let's Encrypt（免费证书）

```bash
# 安装 Certbot
sudo apt-get install certbot python3-certbot-nginx

# 获取证书
sudo certbot --nginx -d itsm.yourdomain.com --agree-tos --email your@email.com

# 自动续期
sudo certbot renew --dry-run
```

---

## 数据初始化

### 首次启动自动初始化

```bash
# 启动服务后自动执行迁移
docker compose -f docker-compose.prod.yml up -d backend
sleep 5

# 检查迁移日志
docker compose -f docker-compose.prod.yml logs backend | grep migrate
```

### 手动初始化

```bash
# 进入后端容器
docker compose -f docker-compose.prod.yml exec backend sh

# 执行迁移
go run main.go migrate

# 执行种子数据
go run main.go seed

exit
```

---

## 部署验证

### 健康检查

```bash
# 检查后端健康
curl -f http://localhost:8090/health

# 检查 API 端点
curl -f http://localhost:8090/api/v1/health

# 检查前端
curl -f http://localhost:3000

# 检查数据库
docker compose -f docker-compose.prod.yml exec -T postgres pg_isready -U itsm

# 检查 Redis
docker compose -f docker-compose.prod.yml exec -T redis redis-cli ping
```

### 功能验证

```bash
# 测试登录
curl -X POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# 预期返回
# {"code":0,"message":"success","data":{"token":"..."}}
```

---

## 备份与恢复

### 自动备份脚本

```bash
#!/bin/bash
# scripts/backup.sh

BACKUP_DIR="/opt/itsm/backups"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

mkdir -p $BACKUP_DIR

# 备份数据库
docker compose -f docker-compose.prod.yml exec -T postgres pg_dump -U itsm itsm_prod | gzip > $BACKUP_DIR/db_$DATE.sql.gz

# 清理旧备份
find $BACKUP_DIR -name "*.sql.gz" -mtime +$RETENTION_DAYS -delete

echo "Backup completed: db_$DATE.sql.gz"
```

添加到 crontab:

```bash
# 每天凌晨 2 点备份
0 2 * * * /opt/itsm/scripts/backup.sh >> /var/log/itsm-backup.log 2>&1
```

### 恢复数据

```bash
# 解压备份
gunzip -c db_20260304.sql.gz > db_restore.sql

# 恢复
docker compose -f docker-compose.prod.yml exec -T postgres psql -U itsm -d itsm_prod < db_restore.sql
```

---

## 监控告警

### 1. 应用监控

```bash
# Prometheus 格式指标
curl http://localhost:8090/metrics

# 关键指标
# - http_requests_total
# - http_request_duration_seconds
# - goroutine_count
# - db_query_duration_seconds
```

### 2. 日志监控

```bash
# Docker 日志
docker compose -f docker-compose.prod.yml logs -f --tail=100 backend

# 使用 Loki（推荐）
# 查询: {app="itsm-backend"}
```

---

## 性能调优

### 数据库优化

```sql
-- 创建索引
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_created_at ON tickets(created_at DESC);
CREATE INDEX idx_tickets_assignee ON tickets(assignee_id);

-- 分析查询计划
EXPLAIN ANALYZE SELECT * FROM tickets WHERE status = 'open';

-- Vacuum
VACUUM ANALYZE;
```

### PostgreSQL 配置

```conf
# postgresql.conf
shared_buffers = 2GB
effective_cache_size = 6GB
maintenance_work_mem = 512MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
```

### Redis 优化

```bash
# 启用持久化
redis-cli CONFIG SET appendonly yes

# 设置内存策略
redis-cli CONFIG SET maxmemory-policy allkeys-lru
```

---

## 安全加固

### 防火墙规则

```bash
# 仅开放必要端口
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22    # SSH
sudo ufw allow 80    # HTTP
sudo ufw allow 443   # HTTPS
sudo ufw enable
```

### 安全检查清单

- [ ] 修改默认密码
- [ ] 启用 HTTPS
- [ ] 配置防火墙
- [ ] 定期更新依赖
- [ ] 启用审计日志
- [ ] 配置入侵检测

---

## 升级流程

### 步骤

1. **备份**: 完整备份数据库和文件
2. **通知**: 通知用户升级时间
3. **停止服务**: 停止应用服务
4. **拉取代码**: `git pull` 更新代码
5. **更新镜像**: 重新构建镜像
6. **迁移**: 如有数据库变更，执行迁移
7. **启动**: 启动服务
8. **验证**: 检查健康状态
9. **回滚**: 如失败，恢复备份

### 升级命令

```bash
# 1. 备份
/opt/itsm/scripts/backup.sh

# 2. 拉取更新
cd /opt/itsm
git pull

# 3. 重新构建
docker compose -f docker-compose.prod.yml build

# 4. 执行迁移
docker compose -f docker-compose.prod.yml exec -T backend go run main.go migrate

# 5. 重启服务
docker compose -f docker-compose.prod.yml up -d

# 6. 验证
curl -f http://localhost:8090/health
```

---

## 常见问题

### 问题 1：服务启动失败

```bash
# 检查日志
docker compose -f docker-compose.prod.yml logs backend

# 常见原因
# - 端口冲突
# - 数据库未就绪
# - 环境变量配置错误
```

### 问题 2：数据库连接失败

```bash
# 检查数据库状态
docker compose -f docker-compose.prod.yml ps postgres

# 检查连接
docker compose -f docker-compose.prod.yml exec backend sh -c "nc -zv postgres 5432"
```

---

**文档维护**: ITSM 运维团队
**最后更新**: 2026-03-04
