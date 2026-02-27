# ITSM 需求分析与任务生成 Cronjob

## 📋 配置信息

| 配置项 | 值 |
|--------|-----|
| **名称** | `itsm-task-analyzer` |
| **描述** | 基于 GitHub Actions 状态分析 ITSM 需求并生成开发任务列表 |
| **执行频率** | 每 2 小时 |
| **时区** | Asia/Shanghai |
| **Agent** | `dev` (小开 - 开发助手) |
| **会话模式** | `isolated`（独立会话） |
| **通知方式** | `announce`（推送到最后聊天） |
| **超时** | 120 秒 |

## 🕐 Cron 表达式

```
0 */2 * * *  # 每 2 小时的整点执行
```

等价于 `--every "2h"`

## 📊 分析内容

### 1. GitHub Actions 状态
- 最近构建是否成功
- 失败的构建及原因
- 构建耗时趋势

### 2. 迭代计划进度
- 读取 `ITERATION_PLAN.md`
- 当前迭代完成度
- 待完成任务清单

### 3. GitHub Issues & PRs
- 新创建的 issues
- 已关闭的 issues
- 待 review 的 PRs
- 合并的 PRs

### 4. 代码变更分析
- 最近的 commits
- 修改的文件和模块
- 潜在的破坏性变更

### 5. 阻塞问题识别
- 构建失败
- 测试失败
- 依赖问题
- 代码冲突

## 📝 输出格式

每次分析生成如下报告：

```markdown
## 📊 ITSM 项目状态报告

**分析时间**: 2026-02-27 16:00

### ✅ 构建状态
- Frontend CI: 成功 (最后运行：15 分钟前)
- Backend CI: 成功 (最后运行：15 分钟前)

### 📋 当前迭代
- **迭代**: 迭代四 - 自动化与通知增强
- **进度**: 0/6 完成
- **阻塞**: 无

### 🎯 下一步开发任务

#### P0 - 高优先级
1. [ ] 创建自动化规则服务框架
   - 文件：`itsm-backend/service/ticket_automation_rule_service.go`
   - 预估：2h

2. [ ] 实现规则匹配引擎
   - 文件：`itsm-backend/service/rule_engine.go`
   - 预估：3h

#### P1 - 中优先级
3. [ ] 初始化自动化规则数据
   - 文件：`itsm-backend/sql/init_automation_rules.sql`
   - 预估：0.5h

4. [ ] 集成到工单创建流程
   - 文件：`itsm-backend/service/ticket_service.go`
   - 预估：1h

### ⚠️ 注意事项
- 无

### 📚 相关文档
- [迭代计划](/root/.openclaw/workspace/itsm/ITERATION_PLAN.md)
- [开发指南](/root/.openclaw/workspace/dev/ITSM-DEV-GUIDE.md)
```

## 🎯 任务优先级定义

| 优先级 | 说明 | 响应时间 |
|--------|------|----------|
| **P0** | 阻塞性问题/核心功能 | 立即处理 |
| **P1** | 重要功能/改进 | 24 小时内 |
| **P2** | 优化/增强 | 本周内 |
| **P3** | 长期规划 | 后续迭代 |

## 📄 输出位置

分析报告会：
1. 推送到当前聊天（announce）
2. 更新 `/root/.openclaw/workspace/dev/ITSM-TASK-LIST.md`
3. 如有紧急问题，单独通知

## 🔧 管理命令

```bash
# 查看状态
openclaw cron list

# 查看执行历史
openclaw cron runs --name "itsm-task-analyzer"

# 手动触发一次
openclaw cron run --name "itsm-task-analyzer"

# 修改频率（如改为每天）
openclaw cron edit itsm-task-analyzer --cron "0 9 * * *"

# 禁用/启用
openclaw cron disable itsm-task-analyzer
openclaw cron enable itsm-task-analyzer
```

## 📋 关联的 Cronjobs

| Cronjob | 频率 | 说明 |
|---------|------|------|
| `github-actions-monitor` | 15 分钟 | 监控构建状态 |
| `itsm-task-analyzer` | 2 小时 | 需求分析与任务生成 |

## 💡 优化建议

根据项目阶段调整频率：
- **开发期**: 每 2 小时（当前）
- **稳定期**: 每天 9:00
- **发布期**: 每 30 分钟

---

_创建时间：2026-02-27_
_下次执行：创建后 2 小时内_
