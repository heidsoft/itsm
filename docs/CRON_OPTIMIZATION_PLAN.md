# 定时任务优化方案

**版本**: 1.0  
**创建时间**: 2026-02-28  
**状态**: 执行中

---

## 📊 当前定时任务状态

### ✅ 正常运行的任务 (3/8)

| 任务 ID | 名称 | 频率 | 状态 | 说明 |
|--------|------|------|------|------|
| 4f7b26d3 | github-actions-monitor | 每 15m | ✅ OK | GitHub Actions 监控 |
| 571cfcf3 | dependency-security-audit | 每天 09:00 | ⏳ Idle | 依赖安全审计 |
| 6fdeeb3b | project-health-report | 每周一 09:00 | ⏳ Idle | 项目健康报告 |
| ce29b735 | docs-generator | 每周五 15:00 | ⏳ Idle | 文档生成 |

### ❌ 失败的任务 (3/8)

| 任务 ID | 名称 | 频率 | 最后执行 | 错误 |
|--------|------|------|---------|------|
| 223b888f | itsm-auto-dev | 每小时 | 39m 前 | ERROR |
| e24d1189 | itsm-task-analyzer | 每 2h | 26m 前 | ERROR |
| 189439ea | github-community-manager | 每 4h | 34m 前 | ERROR |

---

## 🔍 失败原因分析

### 1. itsm-auto-dev (每小时执行)

**失败原因：**
- ITSM 项目当前 Type Check 失败（15 个类型错误）
- 前端构建失败，无法执行自动化开发任务
- 依赖缺失（@ant-design/charts 类型声明）

**影响：**
- 自动化代码生成中断
- 任务分析无法执行

**解决方案：**
```bash
# 1. 修复 Type Check 错误
cd itsm-frontend
npx tsc --noEmit

# 2. 添加缺失的类型声明
# src/types/charts.d.ts

# 3. 重新安装依赖
npm install --legacy-peer-deps
```

**状态**: 🔄 修复中

---

### 2. itsm-task-analyzer (每 2 小时执行)

**失败原因：**
- 依赖 itsm-auto-dev 任务成功执行
- ITSM 项目构建失败导致连锁反应
- 任务队列阻塞

**影响：**
- 任务分析延迟
- 无法生成优化建议

**解决方案：**
```bash
# 1. 等待 itsm-auto-dev 修复
# 2. 清除失败的任务队列
# 3. 重新触发任务
```

**状态**: ⏳ 等待依赖任务修复

---

### 3. github-community-manager (每 4 小时执行)

**失败原因：**
- **缺少 GITHUB_TOKEN 环境变量**
- GitHub API 访问受限（未认证）
- 无法获取实时社区数据

**影响：**
- 社区报告使用缓存数据
- 无法创建/更新 Issue
- 无法自动管理 PR

**解决方案：**
```bash
# 1. 配置 GITHUB_TOKEN
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxx

# 2. 在 OpenClaw 配置中添加
openclaw config set GITHUB_TOKEN ghp_xxxxxxxxxxxxx

# 3. 验证配置
echo $GITHUB_TOKEN
```

**状态**: 🔧 需要配置

---

## 🎯 优化方案

### Phase 1: 紧急修复（今天完成）

#### 1.1 配置 GITHUB_TOKEN
**优先级**: P0  
**预计时间**: 10 分钟

```bash
# 获取 Token: https://github.com/settings/tokens
# 权限：repo, workflow, admin:org

# 配置到 OpenClaw
openclaw configure --env GITHUB_TOKEN=ghp_xxx

# 验证
openclaw cron test github-community-manager
```

#### 1.2 修复 ITSM Type Check
**优先级**: P0  
**预计时间**: 2 小时

- [x] 添加 @ant-design/charts 类型声明
- [ ] 修复 dates 参数类型（14 处）
- [ ] 修复测试文件类型错误
- [ ] 验证构建成功

#### 1.3 清除失败队列
**优先级**: P1  
**预计时间**: 5 分钟

```bash
# 重置失败的任务
openclaw cron reset itsm-auto-dev
openclaw cron reset itsm-task-analyzer
```

---

### Phase 2: 稳定性优化（本周完成）

#### 2.1 添加失败告警
**优先级**: P1  
**预计时间**: 1 小时

```yaml
# 配置失败通知
notifications:
  on_failure:
    - type: dingtalk
      channel: AI 创新
    - type: email
      recipient: admin@example.com
```

#### 2.2 优化任务调度
**优先级**: P2  
**预计时间**: 2 小时

- 避免任务并发冲突
- 添加任务依赖关系
- 设置超时时间

#### 2.3 添加重试机制
**优先级**: P1  
**预计时间**: 1 小时

```yaml
retry:
  max_attempts: 3
  delay: 5m
  backoff: exponential
```

---

### Phase 3: 性能优化（下周完成）

#### 3.1 任务并行化
**优先级**: P2  
**预计时间**: 4 小时

- 分析任务依赖图
- 并行执行独立任务
- 减少总执行时间

#### 3.2 缓存优化
**优先级**: P2  
**预计时间**: 3 小时

- 缓存 GitHub API 响应
- 缓存构建产物
- 缓存依赖安装

#### 3.3 资源限制
**优先级**: P3  
**预计时间**: 2 小时

- 设置 CPU/内存限制
- 避免资源竞争
- 优化任务队列

---

## 📈 监控指标

### 任务健康度

| 指标 | 当前值 | 目标值 | 状态 |
|------|--------|--------|------|
| 成功率 | 62.5% (5/8) | >95% | ❌ |
| 平均执行时间 | - | <5m | ⚠️ |
| 失败恢复时间 | - | <30m | ⚠️ |
| 告警响应时间 | - | <10m | ⚠️ |

### 改进目标

- **本周**: 成功率提升至 87.5% (7/8)
- **下周**: 成功率提升至 100% (8/8)
- **长期**: 保持 99% 以上成功率

---

## 🔧 执行清单

### 立即执行（今天）
- [ ] 配置 GITHUB_TOKEN
- [ ] 修复 ITSM Type Check
- [ ] 重置失败任务
- [ ] 验证所有任务正常

### 本周执行
- [ ] 添加失败告警
- [ ] 优化任务调度
- [ ] 添加重试机制
- [ ] 更新文档

### 下周执行
- [ ] 任务并行化
- [ ] 缓存优化
- [ ] 资源限制
- [ ] 性能监控

---

## 📝 执行日志

### 2026-02-28
- [x] 分析失败原因
- [x] 创建优化方案
- [ ] 配置 GITHUB_TOKEN ⏳
- [ ] 修复 ITSM Type Check 🔄
- [ ] 重置失败任务 ⏳

### 2026-02-27
- [x] 创建定时任务监控
- [x] 添加项目健康报告
- [ ] 配置告警通知

---

**最后更新**: 2026-02-28  
**维护者**: ITSM Team  
**审核者**: @刘彬
