# SSL 证书配置指南

**日期**: 2026-03-02  
**状态**: ✅ 临时证书已配置

---

## 📋 当前状态

### 已配置
- ✅ 自签名证书（临时方案）
- ✅ HTTPS 访问正常
- ✅ 域名：deploy.cloudmesh.top

### 访问地址
- **管理后台**: https://deploy.cloudmesh.top/
- **登录页面**: https://deploy.cloudmesh.top/login.html

### 注意事项
⚠️ **自签名证书会导致浏览器警告**，需要手动信任证书。

---

## 🔧 方案一：手动申请 Let's Encrypt 证书（推荐）

### 前提条件
1. 域名已解析到服务器 IP
2. 服务器 80 端口可访问
3. 有 sudo 权限

### 步骤 1: 停止 Nginx
```bash
sudo systemctl stop nginx
# 或
sudo pkill nginx
```

### 步骤 2: 安装 Certbot
```bash
# CentOS/RHEL
sudo yum install -y certbot

# Ubuntu/Debian
sudo apt-get install -y certbot
```

### 步骤 3: 申请证书
```bash
sudo certbot certonly --standalone \
  -d cloudmesh.top \
  -d www.cloudmesh.top \
  -d deploy.cloudmesh.top \
  -d admin.cloudmesh.top \
  --email admin@cloudmesh.top \
  --agree-tos \
  --non-interactive
```

### 步骤 4: 验证证书
```bash
ls -la /etc/letsencrypt/live/cloudmesh.top/
# 应看到:
# - cert.pem
# - chain.pem
# - fullchain.pem
# - privkey.pem
```

### 步骤 5: 配置 Nginx
```bash
sudo cat > /etc/nginx/conf.d/openclaw-deploy.conf << 'NGINX'
server {
    listen 80;
    server_name deploy.cloudmesh.top admin.cloudmesh.top;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name deploy.cloudmesh.top admin.cloudmesh.top;
    
    ssl_certificate /etc/letsencrypt/live/cloudmesh.top/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/cloudmesh.top/privkey.pem;
    
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    
    root /usr/share/nginx/html;
    index index.html;
    
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
NGINX
```

### 步骤 6: 启动 Nginx
```bash
sudo systemctl start nginx
# 或
sudo nginx
```

### 步骤 7: 测试访问
```bash
curl -sIk https://deploy.cloudmesh.top/
# 应返回 HTTP/2 200
```

---

## 🔧 方案二：使用 DNS 验证（如果 80 端口被占用）

### 步骤 1: 使用 DNS 验证
```bash
sudo certbot certonly --manual \
  -d cloudmesh.top \
  -d www.cloudmesh.top \
  -d deploy.cloudmesh.top \
  -d admin.cloudmesh.top \
  --preferred-challenges dns \
  --email admin@cloudmesh.top
```

### 步骤 2: 添加 DNS TXT 记录
根据 Certbot 提示，在域名 DNS 添加 TXT 记录

### 步骤 3: 验证并完成
等待 DNS 生效后，按回车继续

---

## 🔧 方案三：使用现有证书（如果已有）

如果已有 cloudmesh.top 的证书，可以直接使用：

```bash
# 复制现有证书
sudo cp /path/to/your/cert.pem /etc/letsencrypt/live/cloudmesh.top/cert.pem
sudo cp /path/to/your/privkey.pem /etc/letsencrypt/live/cloudmesh.top/privkey.pem
sudo cp /path/to/your/fullchain.pem /etc/letsencrypt/live/cloudmesh.top/fullchain.pem
sudo cp /path/to/your/chain.pem /etc/letsencrypt/live/cloudmesh.top/chain.pem

# 重启 Nginx
sudo nginx -s reload
```

---

## 📝 证书自动续期

### 创建续期脚本
```bash
sudo cat > /usr/local/bin/renew-cert.sh << 'SCRIPT'
#!/bin/bash
certbot renew --quiet
nginx -s reload
SCRIPT

sudo chmod +x /usr/local/bin/renew-cert.sh
```

### 添加定时任务
```bash
sudo crontab -e
# 添加以下行（每天凌晨 2 点检查）
0 2 * * * /usr/local/bin/renew-cert.sh
```

---

## 🐛 常见问题

### Q1: 证书申请失败
**解决**:
```bash
# 检查网络连接
ping -c 4 letsencrypt.org

# 检查 80 端口
sudo netstat -tlnp | grep :80

# 查看日志
sudo tail -f /var/log/letsencrypt/letsencrypt.log
```

### Q2: 浏览器提示证书不受信任
**解决**: 这是自签名证书的正常现象
- Chrome: 点击"高级" → "继续访问"
- Firefox: 点击"高级" → "接受风险并继续"

### Q3: Nginx 无法启动
**解决**:
```bash
# 测试配置
sudo nginx -t

# 查看错误日志
sudo tail -f /var/log/nginx/error.log
```

---

## 📞 技术支持

如需帮助，请联系技术支持团队。

---

**最后更新**: 2026-03-02
