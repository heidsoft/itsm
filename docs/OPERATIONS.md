# 运维手册

**最后更新**: 2026-03-04
**版本**: v1.0
**适用对象**: 运维工程师、系统管理员

---

## 📋 目录

- [系统监控](#系统监控)
- [日志查看](#日志查看)
- [性能调优](#性能调优)
- [升级流程](#升级流程)
- [容量规划](#容量规划)
- [备份与恢复](#备份与恢复)
- [故障应急](#故障应急)
- [安全加固](#安全加固)

---

## 系统监控

### 1. 监控指标

#### 应用层指标（Prometheus）

后端暴露 `/metrics` 端点:

```bash
curl http://localhost:8090/metrics
```

**关键指标**:

| 指标名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| `http_requests_total` | Counter | HTTP 请求总数 | `http_requests_total{method="GET",path="/api/v1/tickets",code="200"} 1024` |
| `http_request_duration_seconds` | Histogram | 请求耗时（秒） | `http_request_duration_seconds_bucket{le="0.5"} 890` |
| `goroutine_count` | Gauge | 当前 Goroutine 数量 | `goroutine_count 256` |
| `db_query_duration_seconds` | Histogram | 数据库查询耗时 | `db_query_duration_seconds_bucket{query="SELECT"} 1500` |
| `redis_hits_total` | Counter | Redis 命中次数 | `redis_hits_total 50000` |
| `queue_messages_total` | Counter | 消息队列处理数 | `queue_messages_total{queue="email"} 1200` |

---

#### 数据库指标

使用 `postgres_exporter`:

```bash
# 安装
docker run -d \
  -e DATA_SOURCE_NAME="postgresql://itsm:password@localhost:5432/itsm?sslmode=disable" \
  -p 9187:9187 \
  quay.io/prometheuscommunity/postgres-exporter

# 查看指标
curl http://localhost:9187/metrics
```

**关键指标**:

| 指标名 | 说明 |
|--------|------|
| `pg_stat_database_blks_hit` | 缓存命中率 |
| `pg_stat_database_xact_commit` | 事务提交数 |
| `pg_stat_database_conflicts` | 冲突数 |
| `pg_stat_user_tables_n_tup_ins` | 表插入行数 |
| `pg_stat_user_tables_n_dead_tup` | 死行数（需 vacuum） |

---

#### Redis 指标

使用 `redis_exporter`:

```bash
docker run -d \
  -e REDIS_ADDR=redis://localhost:6379 \
  -p 9121:9121 \
  oliver006/redis_exporter

curl http://localhost:9121/metrics
```

**关键指标**:

| 指标名 | 说明 |
|--------|------|
| `redis_up` | 服务状态 |
| `redis_connected_clients` | 连接数 |
| `redis_memory_used_bytes` | 内存使用 |
| `redis_keyspace_hits_total` | 缓存命中 |
| `redis_keyspace_misses_total` | 缓存未命中 |
| `redis_evicted_keys_total` | 驱逐 key 数 |

---

### 2. Grafana 仪表盘

导入以下仪表盘 ID:

- **Go Application Dashboard**: 11982
- **PostgreSQL Dashboard**: 9628
- **Redis Dashboard**: 763

或使用自定义仪表盘 `grafana/itsm-dashboard.json`。

---

### 3. 告警规则（Prometheus Alertmanager）

```yaml
# alerts.yml
groups:
  - name: itsm_alerts
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{code=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.05
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "HTTP 错误率超过 5%"
          description: "{{ $labels.instance }} 错误率 {{ $value }}"

      - alert: HighLatency
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "P95 延迟超过 1秒"

      - alert: DatabaseDown
        expr: up{job="postgres"} == 0
        for: 1m
        labels:
          severity: critical

      - alert: HighMemoryUsage
        expr: (node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes > 0.9
        for: 5m
        labels:
          severity: warning
```

---

## 日志查看

### 1. 结构化日志

后端使用 `slog` 输出 JSON 格式日志:

```json
{"time":"2026-03-04T10:30:00Z","level":"INFO","msg":"User login","user_id":1,"tenant_id":1,"duration_ms":15}
{"time":"2026-03-04T10:31:00Z","level":"ERROR","msg":"Database query failed","error":"deadline exceeded","stack_trace":"..."}
```

---

### 2. 查看日志（Docker Compose）

```bash
# 查看所有服务日志
docker compose -f docker-compose.prod.yml logs -f

# 只看后端
docker compose -f docker-compose.prod.yml logs -f backend

# 查看最近 100 行
docker compose -f docker-compose.prod.yml logs --tail=100 backend

# 按时间筛选
docker compose -f docker-compose.prod.yml logs --since 1h backend

# 导出到文件
docker compose -f docker-compose.prod.yml logs backend > backend_20260304.log
```

---

### 3. 查看日志（Kubernetes）

```bash
# 查看 Pod 日志
kubectl logs -f deployment/itsm-backend -n itsm

# 查看之前容器日志（已重启）
kubectl logs -f deployment/itsm-backend -n itsm --previous

# 多容器 Pod 查看指定容器
kubectl logs -f pod/itsm-backend-xxxx -c backend -n itsm

# 按标签筛选
kubectl logs -f -l app=itsm-backend -n itsm --all-containers=true

# 导出到文件
kubectl logs deployment/itsm-backend -n itsm > backend.log
```

---

### 4. 日志聚合（Loki + Grafana）

安装 Loki 后，使用 Grafana 查询:

```sql
# 查询错误日志
{app="itsm-backend"} |= "error"

# 查询某个用户
{app="itsm-backend"} |= "user_id=123"

# 查询最近 1 小时
{app="itsm-backend"} | json | time >= now() - 1h

# 统计错误级别
sum by (level) (count_over_time({app="itsm-backend"}[1h]))
```

---

## 性能调优

### 1. 数据库优化

#### 索引优化

```sql
-- 检查缺失索引（PostgreSQL）
SELECT
  schemaname AS schema,
  tablename AS table,
  seq_scan,
  idx_scan,
  seq_scan + idx_scan AS total_scans,
  ROUND(seq_scan::numeric / (seq_scan + idx_scan)::numeric * 100, 2) AS seq_percent,
  ROUND(idx_scan::numeric / (seq_scan + idx_scan)::numeric * 100, 2) AS idx_percent
FROM pg_stat_user_tables
WHERE seq_scan + idx_scan > 0
  AND seq_scan > 0
ORDER BY seq_percent DESC
LIMIT 10;

-- 创建常用索引
CREATE INDEX idx_tickets_tenant_id_status ON tickets(tenant_id, status);
CREATE INDEX idx_tickets_assignee_id ON tickets(assignee_id) WHERE assignee_id IS NOT NULL;
CREATE INDEX idx_tickets_created_at ON tickets(created_at DESC);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_incidents_impact_urgency ON incidents(impact, urgency);

-- 分析查询计划
EXPLAIN ANALYZE
SELECT * FROM tickets WHERE tenant_id = 1 AND status = 'open';
```

#### VACUUM & ANALYZE

```bash
# 手动执行
docker exec itsm-postgres psql -U itsm -d itsm_prod -c "VACUUM ANALYZE;"

# 设置自动 vacuum（postgresql.conf）
autovacuum = on
autovacuum_vacuum_scale_factor = 0.05
autovacuum_analyze_scale_factor = 0.02
```

#### 连接池（PgBouncer）

```yaml
# docker-compose.prod.yml 添加
pgbouncer:
  image: pgbouncer/pgbouncer:latest
  environment:
    - DATABASES__itsm__host=postgres
    - DATABASES__itsm__dbname=itsm_prod
    - DATABASES__itsm__username=itsm
    - DATABASES__itsm__password=${DB_PASSWORD}
    - PGBOUNCER_AUTH_TYPE=md5
    - PGBOUNCER_SERVER_RESET_QUERY=DISCARD ALL
    - PGBOUNCER_IGNORE_STARTUP_parameters=1
    - PGBOUNCER_POOL_MODE=transaction
  ports:
    - "6432:6432"
  depends_on:
    - postgres
```

后端连接字符串修改为:
```env
DB_HOST=pgbouncer
DB_PORT=6432
```

---

### 2. Redis 优化

```bash
# 查看内存使用
redis-cli info memory

# 查看键空间统计
redis-cli info stats

# 设置内存上限（redis.conf）
maxmemory 2gb
maxmemory-policy allkeys-lru

# 启用持久化
redis-cli config set appendonly yes
redis-cli config set save "900 1 300 10 60 10000"
```

---

### 3. 应用层优化

#### 缓存策略

```go
// 使用 redis 缓存热点数据
func (s *TicketService) GetTicket(id uint) (*Ticket, error) {
    cacheKey := fmt.Sprintf("ticket:%d", id)

    // 尝试从缓存获取
    if val, err := s.cache.Get(cacheKey).Result(); err == nil {
        var t Ticket
        json.Unmarshal([]byte(val), &t)
        return &t, nil
    }

    // 查询数据库
    t, err := s.repo.FindByID(id)
    if err != nil {
        return nil, err
    }

    // 写入缓存（5分钟）
    s.cache.SetEx(cacheKey, 300, json.Marshal(t))
    return t, nil
}

// 更新时删除缓存
func (s *TicketService) UpdateTicket(id uint, updates TicketUpdateDTO) error {
    err := s.repo.Update(id, updates)
    if err == nil {
        s.cache.Del(fmt.Sprintf("ticket:%d", id))
    }
    return err
}
```

#### 连接池配置

```go
// GORM 连接池
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
    ConnPool: pgxpool.NewConnConfig(pgxConfig), // 使用 pgxpool
})

// 调整连接池参数
pgxConfig.ConnConfig.RuntimeParams["pool_size"] = 20
pgxConfig.ConnConfig.RuntimeParams["max_connections"] = 50
```

---

## 升级流程

### 1. 预升级检查

```bash
# ✅ 完整备份（见备份章节）
# ✅ 检查当前版本
git log --oneline -1

# ✅ 查看变更日志
cat CHANGELOG.md | head -50

# ✅ 检查数据库占用
docker exec itsm-postgres psql -U itsm -d itsm_prod -c "\l+"

# ✅ 验证备份可恢复
pg_restore --list backup.dump | head
```

---

### 2. 升级步骤

#### Docker Compose 升级

```bash
cd /path/to/itsm

# 1. 拉取最新代码
git fetch origin
git checkout main
git pull origin main

# 2. 查看数据库迁移（如有）
ls migrations/new_*.sql

# 3. 停止服务（可选）
docker compose -f docker-compose.prod.yml down

# 4. 构建新镜像
docker compose -f docker-compose.prod.yml build --no-cache

# 5. 启动服务（会自动运行迁移）
docker compose -f docker-compose.prod.yml up -d

# 6. 等待启动
sleep 30

# 7. 验证健康
curl http://localhost:8090/health

# 8. 测试关键功能
curl -X POST http://localhost:8090/api/v1/auth/login ...
```

#### 数据库迁移

```bash
# 自动迁移（随应用启动）
# 或手动执行
docker exec itsm-backend go run cmd/migrate/main.go

# 迁移后验证
docker exec itsm-postgres psql -U itsm -d itsm_prod -c "\dt"
```

---

### 3. 回滚流程

```bash
# 1. 恢复数据库
dropdb itsm_prod
createdb itsm_prod
pg_restore -U itsm -d itsm_prod backup_20260304.dump

# 2. 回滚代码
git checkout <previous-commit-hash>

# 3. 重启服务
docker compose -f docker-compose.prod.yml up -d

# 4. 验证
curl http://localhost:8090/health
```

---

## 容量规划

### 1. 资源评估

| 用户规模 | CPU | 内存 | 数据库 | Redis | 磁盘 |
|---------|-----|------|--------|-------|------|
| < 100 人 | 2 核 | 4 GB | 1 核 2GB | 1 核 1GB | 50 GB |
| 100-500 人 | 4 核 | 8 GB | 2 核 4GB | 2 核 2GB | 200 GB |
| 500-2000 人 | 8 核 | 16 GB | 4 核 8GB | 4 核 4GB | 500 GB |
| > 2000 人 | 16+ 核 | 32+ GB | 8+ 核 16+GB | 8+ 核 8+GB | 1 TB+ |

**估算公式**:

- **QPS 估算**: 预估用户并发 × 10-20 次 API 调用/分钟
- **数据库磁盘**: 每月新增数据 ≈ (工单数 × 5KB) + (日志 × 1KB)
- **网络带宽**: 峰值 QPS × 平均响应大小 × 8

---

### 2. 扩展策略

**垂直扩展**:
- 升级服务器配置（CPU、内存）
- 增加磁盘容量

**水平扩展**:
- 增加应用副本数（K8s Deployment replicas）
- 读写分离（数据库从库）
- Redis 哨兵/集群模式

---

## 备份与恢复

### 自动化备份脚本

```bash
#!/bin/bash
# scripts/backup.sh

set -e

BACKUP_DIR="/backups/itsm"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

mkdir -p $BACKUP_DIR

echo "[$(date)] 开始备份..."

# 1. 数据库备份
docker exec itsm-postgres pg_dump -U itsm itsm_prod > $BACKUP_DIR/db_$DATE.sql
gzip $BACKUP_DIR/db_$DATE.sql
echo "  ✓ 数据库备份完成: db_$DATE.sql.gz"

# 2. 文件上传备份
tar -czf $BACKUP_DIR/uploads_$DATE.tar.gz /app/uploads 2>/dev/null || true
echo "  ✓ 上传文件备份完成: uploads_$DATE.tar.gz"

# 3. 清理旧备份
find $BACKUP_DIR -name "*.sql.gz" -mtime +$RETENTION_DAYS -delete
find $BACKUP_DIR -name "*.tar.gz" -mtime +$RETENTION_DAYS -delete

echo "[$(date)] 备份完成"
```

添加到 crontab:

```bash
# 每天凌晨 2:00 备份
0 2 * * * /path/to/backup.sh >> /var/log/itsm-backup.log 2>&1
```

---

### 恢复流程

```bash
# 1. 停止服务
docker compose -f docker-compose.prod.yml down

# 2. 恢复数据库
gunzip -c db_20260304.sql.gz | docker exec -i itsm-postgres psql -U itsm itsm_prod

# 3. 恢复上传文件
tar -xzf uploads_20260304.tar.gz -C /app/

# 4. 重启服务
docker compose -f docker-compose.prod.yml up -d

# 5. 验证数据
curl http://localhost:8090/api/v1/tickets?page=1&page_size=1
```

---

## 故障应急

### 常见故障处理

#### 1. 服务无法访问

```
症状: curl http://localhost:8090/health 超时
可能原因: 进程崩溃、端口占用、防火墙
```

**排查步骤**:

```bash
# 1. 查看容器状态
docker ps | grep itsm

# 2. 查看容器日志
docker logs itsm-backend --tail 100

# 3. 检查端口
lsof -i :8090
netstat -an | grep 8080

# 4. 重启容器
docker restart itsm-backend
```

---

#### 2. 数据库连接池耗尽

```
症状: 日志: "too many connections"
```

**解决**:

```bash
# 1. 查看连接数
docker exec itsm-postgres psql -U itsm -c "SELECT count(*) FROM pg_stat_activity;"

# 2. 杀死空闲连接
docker exec itsm-postgres psql -U itsm -c "
  SELECT pg_terminate_backend(pid)
  FROM pg_stat_activity
  WHERE state = 'idle' AND now() - state_change > interval '5 minutes';
"

# 3. 调整连接池大小（.env 中 DB_MAX_OPEN_CONNS、DB_MAX_IDLE_CONNS）
```

---

#### 3. Redis 内存不足

```
症状: Redis OOM 错误
```

**解决**:

```bash
# 1. 查看内存使用
redis-cli info memory

# 2. 设置淘汰策略
redis-cli config set maxmemory-policy allkeys-lru

# 3. 清理部分 key（谨慎）
redis-cli keys '*' | head -1000 | xargs redis-cli del

# 4. 增加内存（长期方案）
```

---

#### 4. 磁盘空间不足

```bash
# 1. 查看磁盘使用
df -h

# 2. 清理 Docker 无用资源
docker system prune -a --volumes

# 3. 清理日志
find /var/log -name "*.log" -size +100M -exec truncate -s 0 {} \;

# 4. 清理备份（保留最近 7 天）
find /backups -name "*.sql.gz" -mtime +7 -delete
```

---

### 灾难恢复流程

1. **确认故障级别**: RTO（恢复时间目标）与 RPO（恢复点目标）
2. **切换至备份环境** (如果有): 修改 DNS/LB 指向备用服务器
3. **恢复数据库**: 从最新备份恢复
4. **恢复文件**: 上传文件、配置文件
5. **启动服务**: 逐层启动（DB → Redis → App）
6. **功能验证**: 核心业务功能测试
7. **监控观察**: 持续监控 2-4 小时
8. **回切主环境** (如适用)

---

## 安全加固

### 1. 系统安全

```bash
# 更新系统
sudo apt-get update && sudo apt-get upgrade -y

# 安装 Fail2ban
sudo apt-get install fail2ban

# 配置防火墙
sudo ufw default deny incoming
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

# 禁用密码登录（SSH）
sudo sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
sudo systemctl restart sshd
```

---

### 2. 应用安全

- ✅ 启用 HTTPS（Let's Encrypt）
- ✅ 设置强密码策略（最小长度 12，包含大小写、数字、特殊符号）
- ✅ 启用 2FA（可选）
- ✅ 限制 API 速率（每 IP 100 次/分钟）
- ✅ 定期更新依赖（每周检查）

---

### 3. 数据库安全

```sql
-- 修改默认密码
ALTER USER itsm PASSWORD 'StrongRandomPassword123!@#';

-- 限制连接来源
ALTER SYSTEM SET listen_addresses = '127.0.0.1';

-- 启用 SSL
ALTER SYSTEM SET ssl = on;
ALTER SYSTEM SET ssl_cert_file = '/path/to/server.crt';
ALTER SYSTEM SET ssl_key_file = '/path/to/server.key';
```

---

### 4. 审计日志

开启 PostgreSQL 审计:

```sql
-- 记录所有 DDL 操作
ALTER SYSTEM SET log_statement = 'ddl';

-- 记录所有连接
ALTER SYSTEM SET log_connections = on;
ALTER SYSTEM SET log_disconnections = on;

-- 重启生效
SELECT pg_reload_conf();
```

---

## 常用运维命令速查

```bash
# 系统状态
top -u $(whoami)    # 查看进程
df -h               # 磁盘空间
free -h             # 内存使用

# Docker 管理
docker ps -a                          # 容器列表
docker stats                          # 资源监控
docker logs -f --tail 100 itsm-backend  # 查看日志
docker exec -it itsm-backend sh      # 进入容器

# 数据库
psql -U itsm -d itsm_prod            # 连接
\dt                                   # 列出表
\z                                    # 查看权限
SELECT pg_size_pretty(pg_database_size('itsm_prod'));  # 数据库大小

# 网络诊断
curl -I http://localhost:8090/health  # 检查健康
netstat -an | grep 8080             # 端口状态
traceroute google.com               # 路由跟踪
```

---

**文档维护**: ITSM 运维团队
**最后更新**: 2026-03-04
