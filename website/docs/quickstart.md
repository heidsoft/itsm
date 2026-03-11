# OpenClaw 快速入门指南

> 5 分钟完成 OpenClaw 配置，立即开始使用

---

## 📋 前提条件

- Linux/macOS 或 Windows 系统
- 已注册钉钉/飞书/企业微信等平台账号
- 能够访问互联网

---

## 🚀 快速开始（5 分钟）

### 步骤 1: 访问配置生成器（1 分钟）

打开浏览器访问：
```
https://cloudmesh.top/openclaw-generator/
```

### 步骤 2: 选择平台（1 分钟）

选择你要使用的通讯平台：

| 平台 | 适用场景 | 创建指南 |
|------|----------|----------|
| 💬 钉钉 | 企业 IM | [open.dingtalk.com](https://open.dingtalk.com) |
| 🚀 飞书 | 企业协作 | [open.feishu.cn](https://open.feishu.cn) |
| 💚 企业微信 | 企业通讯 | [work.weixin.qq.com](https://work.weixin.qq.com) |
| ✈️ Telegram | 即时通讯 | @BotFather |
| 🎮 Discord | 社群聊天 | [discord.com/developers](https://discord.com/developers) |
| 💬 Slack | 团队协作 | [api.slack.com](https://api.slack.com) |

### 步骤 3: 填写凭证（1 分钟）

根据选择的平台，填写对应的凭证：

**钉钉示例:**
- AppKey: `dingxxxxxxxxxx`
- AppSecret: `你的应用密钥`

**飞书示例:**
- App ID: `cli_xxxxxxxxxx`
- App Secret: `你的应用密钥`

**企业微信示例:**
- BotID: `机器人 ID`
- Secret: `应用密钥`

> ⚠️ 敏感信息只会保存在本地配置文件中，不会上传到服务器

### 步骤 4: 配置 Agent（1 分钟）

选择你的 Agent 团队规模：

- **👤 1 个** - 单兵作战（个人使用）
- **👥 2 个** - 双人协作（小团队）
- **👨‍💼 3 个** - 专业小队（中型团队）
- **🏢 4 个** - 完整团队（大型团队）

选择协作模式：
- 🔄 顺序执行（流水线处理）
- ⚡ 并行执行（同时处理）
- 📊 层级执行（主管分配）
- 🤝 协作执行（自由协作）

### 步骤 5: 生成并安装（1 分钟）

1. 点击 **"🚀 生成配置"**
2. 复制安装命令
3. 打开终端，粘贴并执行

**Linux/macOS:**
```bash
curl -fsSL "https://cloudmesh.top/openclaw-generator/install.sh" -o install.sh && bash install.sh
```

**Windows (PowerShell):**
```powershell
powershell -c "Invoke-RestMethod 'https://cloudmesh.top/openclaw-generator/install.ps1' -OutFile install.ps1; .\install.ps1"
```

---

## 📁 生成的文件

安装完成后，会在 `~/.openclaw/` 目录生成以下文件：

```
~/.openclaw/
├── workspace/
│   ├── openclaw.json    # 主配置文件
│   └── .env             # 环境变量（敏感信息）
├── logs/
│   └── openclaw.log     # 运行日志
└── install.sh           # 安装脚本
```

---

## ⚙️ 配置文件说明

### openclaw.json

```json
{
  "meta": {
    "lastTouchedVersion": "2026.2.24",
    "generatedBy": "OpenClaw Config Generator"
  },
  "platform": {
    "provider": "dingtalk"
  },
  "agents": [
    {
      "name": "Coordinator",
      "role": "协调员",
      "skills": ["task_management", "communication", "planning"],
      "model": "bailian/qwen3.5-plus"
    }
  ],
  "orchestration": {
    "mode": "sequential",
    "timeout": 3600
  },
  "runtime": {
    "workdir": "~/.openclaw/workspace",
    "default_model": "bailian/qwen3.5-plus",
    "thinking": "auto"
  }
}
```

### .env

```bash
# 平台选择
PLATFORM_PROVIDER=dingtalk

# 平台凭证
DINGTALK_APPKEY=dingxxxxxxxxxx
DINGTALK_APPSECRET=your_appsecret_here

# 运行时配置
DEFAULT_MODEL=bailian/qwen3.5-plus
THINKING_MODE=auto
OPENCLAW_WORKDIR=~/.openclaw/workspace
```

---

## 🎯 启动服务

安装完成后，运行以下命令启动 OpenClaw：

```bash
openclaw start
```

查看状态：
```bash
openclaw status
```

查看日志：
```bash
openclaw logs
```

停止服务：
```bash
openclaw stop
```

---

## ✅ 验证安装

### 1. 检查服务状态

```bash
openclaw status
```

应该看到：
```
✅ OpenClaw 运行中
版本：2026.2.24
平台：dingtalk
```

### 2. 测试消息

在钉钉/飞书等平台发送测试消息，OpenClaw 应该能够响应。

---

## ❓ 常见问题

### Q: 安装失败怎么办？

**A:** 检查以下几点：
1. 确保已安装 Node.js（v20+）
2. 确保网络连接正常
3. 查看安装日志：`cat ~/.openclaw/logs/install.log`

### Q: 如何修改配置？

**A:** 编辑配置文件后重启服务：
```bash
# 编辑配置
vim ~/.openclaw/workspace/openclaw.json

# 重启服务
openclaw restart
```

### Q: 如何升级版本？

**A:** 重新运行安装脚本：
```bash
curl -fsSL "https://cloudmesh.top/openclaw-generator/install.sh" | bash
```

### Q: 敏感信息是否安全？

**A:** 是的！
- 敏感信息只保存在本地 `.env` 文件
- 不会上传到任何服务器
- 建议设置文件权限：`chmod 600 ~/.openclaw/workspace/.env`

### Q: 企业版和个人版有什么区别？

**A:** 主要区别：
- **个人版** - 免费，本地文件配置，适合个人/小团队
- **企业版** - ¥999/年，支持配置中心、密钥管理、多环境等

查看完整对比：[定价页面](/pricing.html)

---

## 📚 下一步

完成快速入门后，你可以：

1. **查看完整文档** - [docs.openclaw.ai](https://docs.openclaw.ai)
2. **了解 Skill 市场** - [Skill 市场](/skills/)
3. **加入社区** - 获取帮助和分享经验
4. **升级企业版** - [联系销售](/pricing.html#contact)

---

## 🆘 获取帮助

- **文档:** https://docs.openclaw.ai
- **GitHub:** https://github.com/openclaw/openclaw
- **邮件:** support@openclaw.ai

---

**祝你使用愉快！** 🎉
