# GitHub Actions 监控 Cronjob

## 📋 配置信息

| 配置项 | 值 |
|--------|-----|
| **名称** | `github-actions-monitor` |
| **ID** | `4f7b26d3-eb5b-4a33-a7c5-f834b88e16d4` |
| **描述** | 定期检查 GitHub Actions 构建状态 |
| **执行频率** | 每 15 分钟 |
| **时区** | Asia/Shanghai |
| **Agent** | `main` |
| **会话模式** | `isolated`（独立会话） |
| **通知方式** | `announce`（推送到最后聊天） |
| **超时** | 60 秒 |

## 🕐 Cron 表达式

```
*/15 * * * *  # 每 15 分钟执行一次
```

等价于 `--every "15m"`

## 📝 检查内容

每次执行会检查：
1. ✅ 最近的 workflow runs 状态
2. ❌ 是否有失败的构建
3. ⏱️ 构建耗时
4. 🔍 失败原因分析（如有）

## 🎯 监控的仓库

**https://github.com/heidsoft/itsm**

- ITSM 前端 CI
- ITSM 后端 CI
- Automated Tests
- Build & Release

## 📊 查看状态

### 查看 cronjob 状态
```bash
openclaw cron list
openclaw cron status
```

### 查看执行历史
```bash
openclaw cron runs --name "github-actions-monitor"
```

### 手动触发一次
```bash
openclaw cron run --name "github-actions-monitor"
```

### 禁用/启用
```bash
openclaw cron disable github-actions-monitor
openclaw cron enable github-actions-monitor
```

### 删除
```bash
openclaw cron rm github-actions-monitor
```

## 📄 相关文件

| 文件 | 路径 |
|------|------|
| Cron 配置 | `/root/.openclaw/workspace/cron/github-actions-monitor.json` |
| 检查脚本 | `/root/.openclaw/workspace/scripts/check-github-actions.sh` |
| 说明文档 | `/root/.openclaw/workspace/cron/GITHUB-ACTIONS-MONITOR.md` |

## 🔧 修改频率

如果需要调整检查频率：

```bash
# 编辑 cronjob
openclaw cron edit github-actions-monitor --every "30m"  # 改为 30 分钟

# 或修改后重新创建
openclaw cron rm github-actions-monitor
openclaw cron add --name "github-actions-monitor" --every "30m" ...
```

## 💡 建议

- **开发期间**: 15 分钟（当前配置）
- **稳定期**: 可改为 30 分钟或 1 小时
- **发布期间**: 可临时改为 5 分钟

---

_创建时间：2026-02-27_
_下次执行：创建后 15 分钟_
