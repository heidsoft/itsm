# ITSM 官网部署报告

**部署时间**: 2026-03-01 12:27 CST  
**部署环境**: 阿里云 ECS (Anolis OS)  
**Web 服务器**: nginx/1.20.1  
**状态**: ✅ 运行中

---

## 🌐 访问信息

### 访问地址

| 类型 | 地址 | 状态 |
|------|------|------|
| 本地访问 | http://localhost/ | ✅ 正常 |
| 健康检查 | http://localhost/health | ✅ healthy |
| 公网访问 | http://[服务器 IP]/ | ⏳ 待配置安全组 |

### 获取公网 IP

```bash
# 查看服务器公网 IP
curl ifconfig.me
```

---

## 📁 文件结构

```
/var/www/itsm-official/
├── index.html          # 官网首页
├── css/
│   └── style.css       # 样式文件
└── js/
    └── main.js         # JavaScript 文件
```

---

## ⚙️ Nginx 配置

### 配置文件位置
`/etc/nginx/conf.d/itsm-official.conf`

### 主要配置

```nginx
server {
    listen 80 default_server;
    server_name _;
    
    root /var/www/itsm-official;
    index index.html;
    
    # 静态文件缓存 1 年
    location ~* \.(jpg|jpeg|png|gif|ico|css|js|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
    
    # HTML 不缓存
    location ~* \.html$ {
        expires -1;
        add_header Cache-Control "no-cache, no-store, must-revalidate";
    }
    
    # 健康检查
    location /health {
        return 200 "healthy\n";
    }
}
```

---

## 🔧 管理命令

### Nginx 状态管理

```bash
# 查看状态
systemctl status nginx

# 启动
systemctl start nginx

# 停止
systemctl stop nginx

# 重启
systemctl restart nginx

# 重载配置
/usr/sbin/nginx -s reload

# 测试配置
/usr/sbin/nginx -t
```

### 日志查看

```bash
# 访问日志
tail -f /var/log/nginx/itsm-official-access.log

# 错误日志
tail -f /var/log/nginx/itsm-official-error.log
```

---

## 🔒 阿里云安全组配置

### 需要开放的端口

| 端口 | 协议 | 用途 | 授权对象 |
|------|------|------|---------|
| 80 | TCP | HTTP 访问 | 0.0.0.0/0 |
| 443 | TCP | HTTPS 访问（待配置） | 0.0.0.0/0 |
| 22 | TCP | SSH 管理 | 特定 IP |

### 配置步骤

1. 登录阿里云控制台
2. 进入 ECS 实例详情页
3. 点击"安全组"
4. 点击"配置规则"
5. 添加入方向规则：
   - 端口范围：80/80
   - 授权对象：0.0.0.0/0
   - 优先级：1

---

## 📊 性能优化

### 已启用优化

- ✅ Gzip 压缩
- ✅ 静态资源缓存（1 年）
- ✅ HTTP/2（待配置 HTTPS）
- ✅ 安全响应头

### 性能指标

| 指标 | 目标值 | 当前值 |
|------|--------|--------|
| 首屏加载 | <1 秒 | 待测试 |
| Lighthouse | >90 | 待测试 |
| TTFB | <100ms | 待测试 |

---

## 🎯 下一步行动

### 立即可做

1. **配置域名**
   ```bash
   # 在阿里云 DNS 解析添加 A 记录
   itsm.com -> [服务器 IP]
   www.itsm.com -> [服务器 IP]
   ```

2. **配置 HTTPS**
   ```bash
   # 申请免费 SSL 证书（阿里云/Let's Encrypt）
   # 配置 SSL 到 nginx
   ```

3. **开放安全组**
   - 在阿里云控制台开放 80 端口

### 短期计划（1 周）

- [ ] 配置域名和 HTTPS
- [ ] 添加更多页面（产品、客户、定价等）
- [ ] 添加联系表单
- [ ] 集成统计代码（Google Analytics/百度统计）

### 中期计划（1 月）

- [ ] 添加博客系统
- [ ] 添加文档中心
- [ ] 添加试用申请功能
- [ ] 添加演示预约功能

---

## 📈 监控告警

### 健康检查

```bash
# 手动检查
curl http://localhost/health

# 添加 crontab 监控
*/5 * * * * curl -sf http://localhost/health > /dev/null || systemctl restart nginx
```

### 监控指标

- 网站可用性
- 响应时间
- 错误率
- 访问量

---

## 🔗 相关文档

- [官网建设方案](./website/WEBSITE_PLAN.md)
- [Nginx 配置最佳实践](./nginx/NGINX_BEST_PRACTICES.md)
- [HTTPS 配置指南](./security/HTTPS_SETUP.md)

---

**维护者**: 运维团队  
**最后更新**: 2026-03-01  
**下次审查**: 2026-03-08
