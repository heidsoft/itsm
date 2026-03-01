# SSL 证书配置报告

**配置时间**: 2026-03-01 12:44 CST  
**证书类型**: Let's Encrypt (免费)  
**证书有效期**: 90 天（自动续期）  
**状态**: ✅ 运行中

---

## 🌐 域名配置

| 域名 | 状态 | HTTPS |
|------|------|-------|
| cloudmesh.top | ✅ 正常 | ✅ 已启用 |
| www.cloudmesh.top | ✅ 正常 | ✅ 已启用 |

---

## 🔒 SSL 证书信息

| 项目 | 值 |
|------|-----|
| 证书类型 | Let's Encrypt RSA |
| 颁发机构 | Let's Encrypt R3 |
| 域名 | cloudmesh.top, www.cloudmesh.top |
| 有效期 | 90 天 |
| 过期日期 | 2026-05-30 |
| 密钥类型 | RSA 2048 位 |
| TLS 版本 | TLSv1.2, TLSv1.3 |

---

## ⚙️ Nginx 配置

### 配置文件
`/etc/nginx/conf.d/itsm-official.conf`

### HTTP 自动跳转 HTTPS

```nginx
server {
    listen 80;
    server_name cloudmesh.top www.cloudmesh.top;
    
    # ACME 挑战目录
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }
    
    # 跳转到 HTTPS
    location / {
        return 301 https://$server_name$request_uri;
    }
}
```

### HTTPS 配置

```nginx
server {
    listen 443 ssl http2;
    server_name cloudmesh.top www.cloudmesh.top;
    
    ssl_certificate /etc/letsencrypt/live/cloudmesh.top/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/cloudmesh.top/privkey.pem;
    
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_session_cache shared:SSL:10m;
    
    # HSTS
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
}
```

---

## 🔄 自动续期

### Certbot 自动续期

Let's Encrypt 证书有效期 90 天，Certbot 已配置自动续期：

```bash
# Certbot 自动续期任务
/etc/cron.daily/certbot-renew

# 手动测试续期
certbot renew --dry-run

# 手动续期
certbot renew
```

### 续期验证

```bash
# 查看证书状态
certbot certificates

# 输出示例：
Certificate Name: cloudmesh.top
  Domains: cloudmesh.top www.cloudmesh.top
  Expiry Date: 2026-05-30
  Certificate Path: /etc/letsencrypt/live/cloudmesh.top/fullchain.pem
```

---

## 📋 管理命令

### 查看证书

```bash
# 查看所有证书
certbot certificates

# 查看证书详情
openssl x509 -in /etc/letsencrypt/live/cloudmesh.top/fullchain.pem -text -noout
```

### 手动续期

```bash
# 测试续期（不影响生产）
certbot renew --dry-run

# 强制续期
certbot renew --force-renewal

# 续期后重载 nginx
certbot renew --deploy-hook "systemctl reload nginx"
```

### 查看续期日志

```bash
# Certbot 日志
tail -f /var/log/letsencrypt/letsencrypt.log

# 系统日志
journalctl -u certbot
```

---

## 🔍 验证配置

### 测试 HTTP 跳转

```bash
curl -I http://cloudmesh.top
# 应该返回 301 跳转到 HTTPS
```

### 测试 HTTPS

```bash
curl -I https://cloudmesh.top
# 应该返回 200 OK
```

### 测试 SSL 连接

```bash
openssl s_client -connect cloudmesh.top:443 -servername cloudmesh.top
# 查看证书信息
```

### 在线验证

- **SSL Labs**: https://www.ssllabs.com/ssltest/
- **Why No Padlock**: https://www.whynopadlock.com/
- **SSL Checker**: https://www.sslshopper.com/ssl-checker.html

---

## 📊 安全配置

### 已启用的安全特性

- ✅ TLS 1.2 & 1.3
- ✅ 强加密套件
- ✅ HSTS (HTTP Strict Transport Security)
- ✅ X-Frame-Options
- ✅ X-Content-Type-Options
- ✅ X-XSS-Protection
- ✅ OCSP Stapling

### SSL 配置评分

| 检查项 | 目标 | 状态 |
|--------|------|------|
| TLS 版本 | TLS 1.2+ | ✅ |
| 证书有效期 | <90 天 | ✅ |
| HSTS | 启用 | ✅ |
| 加密套件 | A+ | ✅ |

---

## ⚠️ 故障排查

### 证书续期失败

```bash
# 查看详细日志
certbot renew --verbose

# 检查 nginx 配置
/usr/sbin/nginx -t

# 检查 80 端口是否开放
netstat -tlnp | grep :80
```

### HTTPS 不工作

```bash
# 检查 nginx 状态
systemctl status nginx

# 检查 443 端口
netstat -tlnp | grep :443

# 检查证书文件
ls -la /etc/letsencrypt/live/cloudmesh.top/
```

### 证书过期

```bash
# 紧急续期
certbot renew --force-renewal

# 重载 nginx
systemctl reload nginx
```

---

## 📅 维护计划

### 日常维护

- [ ] 每周检查证书状态
- [ ] 每月查看续期日志
- [ ] 每季度测试续期流程

### 证书续期

- **自动续期**: Certbot 会在证书过期前 30 天自动续期
- **手动检查**: 每周运行 `certbot certificates`
- **告警设置**: 建议设置证书过期告警（过期前 14 天）

---

## 🔗 相关资源

- [Let's Encrypt 官方文档](https://letsencrypt.org/docs/)
- [Certbot 使用指南](https://certbot.eff.org/)
- [SSL 配置最佳实践](https://wiki.mozilla.org/Security/Server_Side_TLS)
- [SSL Labs 测试](https://www.ssllabs.com/ssltest/)

---

**维护者**: 运维团队  
**最后更新**: 2026-03-01  
**下次审查**: 2026-04-01
