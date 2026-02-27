# ITSM 自动化系统

## 📊 系统概览

已创建三个协同工作的 Cronjobs，实现 ITSM 项目的完整自动化流程。

```
┌─────────────────────────────┐
│  GitHub Actions Monitor     │
│  (每 15 分钟)                │
│  Agent: main                │
└──────────┬──────────────────┘
           │ 构建状态
           ↓
┌─────────────────────────────┐
│  ITSM Task Analyzer         │
│  (每 2 小时)                 │
│  Agent: dev (小开)          │
└──────────┬──────────────────┘
           │ 任务列表
           ↓
┌─────────────────────────────┐
│  ITSM Auto Dev              │
│  (每天 10:00)                │
│  Agent: dev (小开)          │
└──────────┬──────────────────┘
           │ 代码提交
           ↓
┌─────────────────────────────┐
│  GitHub 仓库                │
│  feature 分支 / PR          │
└─────────────────────────────┘
```

---

## 🕐 Cronjob 配置

### 1️⃣ GitHub Actions Monitor

| 配置项 | 值 |
|--------|-----|
| **ID** | `4f7b26d3-eb5b-4a33-a7c5-f834b88e16d4` |
| **名称** | `github-actions-monitor` |
| **频率** | 每 15 分钟 |
| **Agent** | `main` |
| **职责** | 监控 GitHub Actions 构建状态 |

**检查内容**:
- ✅ 最近的 workflow runs 状态
- ❌ 失败的构建及原因
- ⏱️ 构建耗时
- 📈 构建趋势

### 2️⃣ ITSM Task Analyzer

| 配置项 | 值 |
|--------|-----|
| **ID** | `e24d1189-ef9e-4dd6-b0fe-915bcdb2764a` |
| **名称** | `itsm-task-analyzer` |
| **频率** | 每 2 小时 |
| **Agent** | `dev` (小开) |
| **职责** | 需求分析与任务生成 |

**分析内容**:
1. GitHub Actions 构建状态
2. ITERATION_PLAN.md 迭代进度
3. GitHub Issues & PRs
4. 代码变更分析
5. 阻塞问题识别

**输出内容**:
- 构建状态总结
- 当前迭代进度
- 开发任务列表（按优先级）
- 注意事项

### 3️⃣ ITSM Auto Dev

| 配置项 | 值 |
|--------|-----|
| **ID** | `223b888f-1985-45d1-ad6f-5b12b4174674` |
| **名称** | `itsm-auto-dev` |
| **频率** | 每天 10:00 |
| **Agent** | `dev` (小开) |
| **职责** | 自动完成开发任务 |

**工作内容**:
1. 读取 ITSM-TASK-LIST.md
2. 选择 P1/P2 优先级小任务
3. 编写/修改代码
4. 本地验证（格式检查、编译）
5. 提交到 feature 分支或创建 PR

**安全限制**:
- ❌ 不自动 push 到 main 分支
- ❌ 单次最多修改 5 个文件
- ❌ 不删除代码/文件
- ⚠️ 复杂任务等待审查

---

## 📄 相关文件

| 文件 | 路径 | 说明 |
|------|------|------|
| 监控配置 | `/root/.openclaw/workspace/cron/github-actions-monitor.json` | 15 分钟监控配置 |
| 分析配置 | `/root/.openclaw/workspace/cron/itsm-task-analyzer.json` | 2 小时分析配置 |
| 开发配置 | `/root/.openclaw/workspace/cron/itsm-auto-dev.json` | 每天 10:00 自动开发 |
| 任务列表 | `/root/.openclaw/workspace/dev/ITSM-TASK-LIST.md` | 自动生成的任务文档 |
| 监控说明 | `/root/.openclaw/workspace/cron/GITHUB-ACTIONS-MONITOR.md` | 监控 cronjob 文档 |
| 分析说明 | `/root/.openclaw/workspace/cron/ITSM-TASK-ANALYZER.md` | 分析 cronjob 文档 |
| 开发说明 | `/root/.openclaw/workspace/cron/ITSM-AUTO-DEV.md` | 自动开发文档 |
| 检查脚本 | `/root/.openclaw/workspace/scripts/check-github-actions.sh` | 手动检查脚本 |

---

## 🎯 工作流程

### 正常流程
```
1. 每 15 分钟 → github-actions-monitor 检查构建
   ↓
2. 发现新状态 → 记录并通知
   ↓
3. 每 2 小时 → itsm-task-analyzer 综合分析
   ↓
4. 生成任务列表 → 更新 ITSM-TASK-LIST.md
   ↓
5. 推送报告 → 通知开发者
```

### 异常情况
- **构建失败**: 立即通知（15 分钟内）
- **连续失败**: 标记为 P0 优先级任务
- **阻塞问题**: 单独通知并建议解决方案

---

## 🔧 管理命令

### 查看状态
```bash
# 查看所有 cronjobs
openclaw cron list

# 查看详细信息
openclaw cron status
```

### 查看执行历史
```bash
# 监控历史
openclaw cron runs --name "github-actions-monitor"

# 分析历史
openclaw cron runs --name "itsm-task-analyzer"
```

### 手动触发
```bash
# 立即检查构建
openclaw cron run --name "github-actions-monitor"

# 立即分析任务
openclaw cron run --name "itsm-task-analyzer"
```

### 修改配置
```bash
# 修改频率（如改为 30 分钟监控）
openclaw cron edit github-actions-monitor --every "30m"

# 修改分析频率（如改为每天 9 点）
openclaw cron edit itsm-task-analyzer --cron "0 9 * * *"
```

### 禁用/启用
```bash
# 禁用
openclaw cron disable github-actions-monitor
openclaw cron disable itsm-task-analyzer

# 启用
openclaw cron enable github-actions-monitor
openclaw cron enable itsm-task-analyzer

# 删除
openclaw cron rm github-actions-monitor
openclaw cron rm itsm-task-analyzer
```

---

## 📊 当前状态

| Cronjob | 状态 | 下次执行 |
|---------|------|----------|
| `github-actions-monitor` | ✅ idle | 5 分钟后 |
| `itsm-task-analyzer` | ✅ idle | 2 小时后 |
| `itsm-auto-dev` | ✅ idle | 18 小时后 (明天 10:00) |

---

## 💡 使用建议

### 开发阶段
- **监控频率**: 15 分钟（当前）
- **分析频率**: 2 小时（当前）
- **开发频率**: 每天 10:00（当前）
- **优点**: 快速发现问题，自动完成简单任务

### 稳定阶段
- **监控频率**: 30 分钟 - 1 小时
- **分析频率**: 每天 9:00
- **开发频率**: 每周一 10:00
- **优点**: 减少打扰，专注开发

### 发布阶段
- **监控频率**: 5 分钟
- **分析频率**: 1 小时
- **开发频率**: 暂停自动开发
- **优点**: 紧密跟踪，快速响应

---

## 🎉 预期效果

1. **自动化监控**: 无需手动检查构建状态 ✅
2. **智能分析**: 自动生成开发任务列表 ✅
3. **优先级排序**: 清晰的任务优先级 ✅
4. **进度跟踪**: 实时了解迭代进度 ✅
5. **问题预警**: 早期发现阻塞问题 ✅
6. **自动开发**: 完成简单任务，生成代码 ✅

---

_创建时间：2026-02-27_
_系统状态：运行中 ✅_
