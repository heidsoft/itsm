# OpenClaw 安全白皮书

> 透明、可审计、安全可信

---

## 🔒 安全承诺

OpenClaw 将用户安全和隐私放在首位。我们承诺：

1. **代码完全开源** - 所有代码在 GitHub 公开，接受社区审查
2. **本地执行** - 所有操作在本地完成，不上传敏感信息
3. **权限最小化** - 只请求必要的系统权限
4. **可审计** - 所有操作都有日志记录

---

## 📦 安装脚本安全说明

### 安装脚本做了什么？

我们的安装脚本 (`install.sh` / `install.ps1`) 只执行以下操作：

```bash
# 1. 检查系统环境
✓ 检测操作系统（Linux/macOS/Windows）
✓ 检查 Node.js 是否安装
✓ 检查网络连接

# 2. 安装依赖（如需要）
✓ 安装 Node.js（如果未安装）
✓ 安装 pnpm（包管理器）

# 3. 安装 OpenClaw
✓ 从 npm 官方仓库下载 OpenClaw
✓ 全局安装到系统目录

# 4. 创建配置文件
✓ 创建 ~/.openclaw/ 目录
✓ 生成默认配置文件
✓ 设置文件权限（600）

# 5. 可选：设置开机自启
✓ 创建 systemd 服务（Linux）
✓ 创建启动项（Windows）
```

### 安装脚本**不**做什么？

```
✗ 不收集任何个人信息
✗ 不上传 API Key 或 Secret
✗ 不修改系统关键配置
✗ 不安装任何后门或恶意软件
✗ 不连接除 npm 和配置平台外的任何服务器
```

---

## 🔍 如何验证脚本安全？

### 方法 1: 查看脚本内容

在安装前，你可以先查看脚本内容：

```bash
# Linux/macOS
curl -fsSL "https://cloudmesh.top/openclaw-generator/install.sh" -o install.sh
cat install.sh  # 查看脚本内容

# Windows
Invoke-RestMethod 'https://cloudmesh.top/openclaw-generator/install.ps1' -OutFile install.ps1
Get-Content install.ps1  # 查看脚本内容
```

### 方法 2: 在沙箱环境运行

建议在虚拟机或容器中先测试：

```bash
# 使用 Docker 测试
docker run -it node:20 bash
# 然后在容器内运行安装脚本

# 或使用虚拟机
# VirtualBox / VMware 等
```

### 方法 3: 检查网络连接

使用网络监控工具检查脚本连接了哪些服务器：

```bash
# Linux
sudo tcpdump -i any -n port 80 or port 443

# macOS
sudo tcpdump -i en0 -n port 80 or port 443

# Windows
使用 Wireshark 或 Fiddler
```

**正常连接：**
- `registry.npmjs.org` - npm 官方仓库
- `open.dingtalk.com` / `open.feishu.cn` 等 - 配置的平台 API
- `cloudmesh.top` - 配置生成器网站

**异常连接：** 任何其他域名都应立即报告！

---

## 🛡️ 敏感信息保护

### API Key/Secret 存储

OpenClaw 将敏感信息存储在 `~/.openclaw/workspace/.env` 文件中：

```bash
# 文件权限设置为仅所有者可读写
chmod 600 ~/.openclaw/workspace/.env

# 文件内容示例
PLATFORM_PROVIDER=dingtalk
DINGTALK_APPKEY=dingxxxxxxxxxx
DINGTALK_APPSECRET=your_secret_here
```

### 最佳实践

1. **定期轮换密钥** - 每 3-6 个月更换一次 API Key/Secret
2. **使用环境变量** - 生产环境建议使用环境变量而非文件
3. **备份配置** - 定期备份配置文件到安全位置
4. **监控日志** - 定期检查 `~/.openclaw/logs/openclaw.log`

---

## 🔓 卸载说明

### 完全卸载 OpenClaw

#### Linux/macOS

```bash
# 1. 停止服务
openclaw stop

# 2. 卸载 OpenClaw
npm uninstall -g openclaw

# 3. 删除配置目录
rm -rf ~/.openclaw

# 4. 删除日志（可选）
rm -rf ~/Library/Logs/openclaw  # macOS
rm -rf ~/.local/share/openclaw  # Linux

# 5. 删除开机自启（如果设置了）
sudo systemctl disable openclaw  # Linux
launchctl unload ~/Library/LaunchAgents/openclaw.plist  # macOS
```

#### Windows

```powershell
# 1. 停止服务
openclaw stop

# 2. 卸载 OpenClaw
npm uninstall -g openclaw

# 3. 删除配置目录
Remove-Item -Recurse -Force $env:USERPROFILE\.openclaw

# 4. 删除开机自启（如果设置了）
Remove-Item "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Startup\OpenClaw.lnk"
```

### 验证卸载完成

```bash
# 检查命令是否还存在
openclaw --version  # 应该显示"command not found"

# 检查目录是否删除
ls -la ~/.openclaw  # 应该显示"No such file or directory"
```

---

## 📋 安全审计

### 代码审计

OpenClaw 核心代码已在 GitHub 开源，接受社区审计：

- **GitHub 仓库:** https://github.com/openclaw/openclaw
- **npm 包:** https://www.npmjs.com/package/openclaw

### 依赖审计

我们使用标准、广泛使用的依赖：

| 依赖 | 用途 | 下载量 |
|------|------|--------|
| Node.js | 运行时环境 | 亿级 |
| pnpm | 包管理器 | 千万级 |
| Express | Web 框架 | 亿级 |
| axios | HTTP 客户端 | 亿级 |

### 漏洞报告

如发现安全漏洞，请通过以下方式报告：

- **邮箱:** security@openclaw.ai
- **GitHub Issues:** https://github.com/openclaw/openclaw/issues
- **GPG Key:** （可选，用于加密敏感报告）

我们承诺：
- 24 小时内响应
- 72 小时内提供修复方案
- 公开致谢报告者（如同意）

---

## ✅ 安全清单

使用前请确认：

- [ ] 已从官方渠道下载（cloudmesh.top 或 GitHub）
- [ ] 已查看安装脚本内容
- [ ] 已理解脚本执行的操作
- [ ] 已设置合适的文件权限
- [ ] 已备份重要数据
- [ ] 已记录 API Key/Secret 存储位置

---

## 🆘 获取帮助

如有安全问题或疑虑：

- **文档:** https://docs.openclaw.ai/security
- **邮箱:** security@openclaw.ai
- **社区:** Discord / 微信群

---

**最后更新:** 2026-03-11  
**版本:** 1.0.0
