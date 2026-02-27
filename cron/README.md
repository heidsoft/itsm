# ITSM 自动化系统 - Cronjobs 配置

## 📊 系统总览

三个 Cronjobs 协同工作，实现 ITSM 项目的完整自动化：

```
监控 → 分析 → 开发 → 提交
```

---

## 🕐 Cronjobs 列表

| # | 名称 | 频率 | Agent | 职责 | 下次执行 |
|---|------|------|-------|------|----------|
| 1️⃣ | `github-actions-monitor` | 每 15 分钟 | `main` | 监控 GitHub Actions 构建状态 | 5 分钟后 |
| 2️⃣ | `itsm-task-analyzer` | 每 2 小时 | `dev` | 需求分析并生成开发任务 | 2 小时后 |
| 3️⃣ | `itsm-auto-dev` | 每天 10:00 | `dev` | 自动完成开发任务 | 明天 10:00 |

---

## 📋 详细说明

### 1. GitHub Actions Monitor

**配置**: `/root/.openclaw/workspace/cron/github-actions-monitor.json`  
**文档**: `GITHUB-ACTIONS-MONITOR.md`

**职责**:
- 每 15 分钟检查一次 GitHub Actions
- 报告构建状态（成功/失败/耗时）
- 发现失败立即通知

### 2. ITSM Task Analyzer

**配置**: `/root/.openclaw/workspace/cron/itsm-task-analyzer.json`  
**文档**: `ITSM-TASK-ANALYZER.md`

**职责**:
- 每 2 小时分析一次项目状态
- 读取迭代计划、Issues、PRs
- 生成带优先级的开发任务列表
- 更新 `ITSM-TASK-LIST.md`

### 3. ITSM Auto Dev

**配置**: `/root/.openclaw/workspace/cron/itsm-auto-dev.json`  
**文档**: `ITSM-AUTO-DEV.md`

**职责**:
- 每天 10:00 执行开发任务
- 选择 P1/P2 优先级的小任务
- 编写代码、验证、提交
- 提交到 feature 分支或创建 PR

**安全限制**:
- ❌ 不自动 push 到 main 分支
- ❌ 单次最多修改 5 个文件
- ❌ 不删除代码/文件
- ⚠️ 复杂任务等待审查

---

## 🔄 工作流程

```
┌─────────────────────────────────────────────────────────────┐
│                    每 15 分钟                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ GitHub Actions Monitor                              │    │
│  │ - 检查构建状态                                       │    │
│  │ - 发现失败立即通知                                   │    │
│  └─────────────────────────────────────────────────────┘    │
│                          ↓                                   │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              每 2 小时                                  │    │
│  │  ITSM Task Analyzer                                 │    │
│  │  - 分析构建状态                                      │    │
│  │  - 读取迭代计划                                      │    │
│  │  - 生成任务列表                                      │    │
│  └─────────────────────────────────────────────────────┘    │
│                          ↓                                   │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              每天 10:00                                │    │
│  │  ITSM Auto Dev                                      │    │
│  │  - 选择任务                                          │    │
│  │  - 编写代码                                          │    │
│  │  - 验证提交                                          │    │
│  └─────────────────────────────────────────────────────┘    │
│                          ↓                                   │
│              GitHub 仓库 (feature 分支 / PR)                  │
└─────────────────────────────────────────────────────────────┘
```

---

## 🔧 管理命令

### 查看所有 Cronjobs
```bash
openclaw cron list
```

### 查看执行历史
```bash
openclaw cron runs
openclaw cron runs --name "github-actions-monitor"
openclaw cron runs --name "itsm-task-analyzer"
openclaw cron runs --name "itsm-auto-dev"
```

### 手动触发
```bash
# 立即检查构建
openclaw cron run --name "github-actions-monitor"

# 立即分析任务
openclaw cron run --name "itsm-task-analyzer"

# 立即执行开发
openclaw cron run --name "itsm-auto-dev"
```

### 修改配置
```bash
# 修改监控频率为 30 分钟
openclaw cron edit github-actions-monitor --every "30m"

# 修改分析频率为每天 9 点
openclaw cron edit itsm-task-analyzer --cron "0 9 * * *"

# 修改开发时间为每周一 10 点
openclaw cron edit itsm-auto-dev --cron "0 10 * * 1"
```

### 禁用/启用
```bash
# 禁用
openclaw cron disable github-actions-monitor
openclaw cron disable itsm-task-analyzer
openclaw cron disable itsm-auto-dev

# 启用
openclaw cron enable github-actions-monitor
openclaw cron enable itsm-task-analyzer
openclaw cron enable itsm-auto-dev

# 删除
openclaw cron rm github-actions-monitor
openclaw cron rm itsm-task-analyzer
openclaw cron rm itsm-auto-dev
```

---

## 📄 文件结构

```
/root/.openclaw/workspace/cron/
├── README.md                          # 本文件
├── github-actions-monitor.json        # 监控配置
├── GITHUB-ACTIONS-MONITOR.md          # 监控文档
├── itsm-task-analyzer.json            # 分析配置
├── ITSM-TASK-ANALYZER.md              # 分析文档
├── itsm-auto-dev.json                 # 开发配置
├── ITSM-AUTO-DEV.md                   # 开发文档
└── ITSM-AUTOMATION-SUMMARY.md         # 系统总览

/root/.openclaw/workspace/dev/
├── ITSM-TASK-LIST.md                  # 自动生成的任务列表
├── PROJECTS.md                        # 项目信息
├── ITSM-DEV-GUIDE.md                  # 开发指南
└── ...

/root/.openclaw/workspace/scripts/
└── check-github-actions.sh            # 手动检查脚本
```

---

## 💡 最佳实践

### 第一周：观察模式
1. 禁用自动开发：`openclaw cron disable itsm-auto-dev`
2. 每天查看监控和分析报告
3. 手动执行一次开发任务看效果

### 第二周：审查模式
1. 启用自动开发
2. 每天审查 feature 分支
3. 确认无误后手动合并

### 稳定后：自动模式
1. 信任自动化流程
2. 每周审查代码质量
3. 根据需求调整频率

---

## ⚠️ 注意事项

1. **自动开发有限制**: 不会自动 push 到 main 分支
2. **复杂任务需审查**: 业务逻辑修改会等待人工确认
3. **频率可调整**: 根据项目阶段调整执行频率
4. **随时可禁用**: 有需要可随时禁用任意 Cronjob

---

## 🎯 预期效果

| 功能 | 状态 | 说明 |
|------|------|------|
| 构建监控 | ✅ 运行中 | 15 分钟检查一次 |
| 任务分析 | ✅ 运行中 | 2 小时生成任务列表 |
| 自动开发 | ✅ 运行中 | 每天 10:00 执行 |
| 代码审查 | ⚠️ 需人工 | feature 分支等待合并 |
| 自动部署 | ❌ 未配置 | 需额外配置 |

---

_创建时间：2026-02-27_  
_系统状态：运行中 ✅_  
_下次执行：5 分钟后（监控）_
