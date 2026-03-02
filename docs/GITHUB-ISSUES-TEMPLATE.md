# GitHub Issues 模板

**创建时间**: 2026-03-02 12:34 (Asia/Shanghai)  
**任务**: P1-04 - 将 TODO 转为 Issue 跟踪  
**状态**: 模板已生成，待手动/自动创建到 GitHub

---

## Issue #1: 创建 BPMN 版本变更日志表

**Labels**: `enhancement`, `backend`, `bpmn`, `P2`  
**Assignee**: 后端开发  
**Milestone**: 迭代四  
**Estimate**: 4h

### 描述

当前 BPMN 版本变更日志仅记录到审计日志，需要创建专门的数据表来跟踪 BPMN 流程版本的变更历史。

### TODO 位置

`itsm-backend/service/bpmn_version_service.go:385`

```go
// recordVersionChangeLog 记录版本变更日志
func (s *BPMNVersionService) recordVersionChangeLog(ctx context.Context, processDefID string, changeLog, createdBy string, tenantID int) error {
	// 记录版本变更到审计日志
	s.logger.Infow("Version change logged",
		"process_def_id", processDefID,
		"change_log", changeLog,
		"created_by", createdBy,
		"tenant_id", tenantID)
	// TODO: 创建专门的版本变更日志表（未来版本）
	return nil
}
```

### 需求

创建专门的数据表来跟踪 BPMN 流程版本的变更历史。

### 建议方案

1. **创建数据表** `process_version_changelog`：
   ```sql
   CREATE TABLE process_version_changelog (
       id BIGSERIAL PRIMARY KEY,
       process_definition_id VARCHAR(64) NOT NULL,
       version INT NOT NULL,
       change_log TEXT NOT NULL,
       created_by VARCHAR(64) NOT NULL,
       tenant_id INT NOT NULL,
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
       INDEX idx_process_def (process_definition_id),
       INDEX idx_version (version),
       INDEX idx_created_at (created_at)
   );
   ```

2. **实现 Service 方法**：
   - `CreateVersionChangeLog(ctx, processDefID, version, changeLog, createdBy, tenantID)`
   - `GetVersionHistory(ctx, processDefID, limit)`

3. **集成到版本管理流程**：
   - 在版本创建时自动记录
   - 在版本更新时记录变更内容

### 验收标准

- [ ] 数据库表创建完成
- [ ] Service 方法实现并测试
- [ ] 版本创建/更新时自动记录变更日志
- [ ] 提供查询版本历史的 API

---

## Issue #4: 知识库实时协作 Session 集成

**Labels**: `enhancement`, `frontend`, `backend`, `collaboration`, `P3`  
**Assignee**: 前后端配合  
**Milestone**: 迭代五  
**Estimate**: 8h

### 描述

知识库协作组件需要对接实时协作 Session API，显示当前文档的协作者列表和协作状态。

### TODO 位置

`itsm-frontend/src/components/knowledge/KnowledgeCollaboration.tsx:127`

```tsx
// TODO: 对接实时协作 Session API
// 目前后端暂未提供获取当前文档协作 Session 的接口
setSession(null);
setParticipants([]);
```

### 需求

实现知识库文档的实时协作功能，显示当前正在编辑文档的协作者列表。

### 依赖

- 后端需要提供协作 Session 管理 API
- 可能需要 WebSocket 支持实时通知

### 建议方案

#### 后端任务

1. **创建协作 Session 表**：
   ```sql
   CREATE TABLE knowledge_collaboration_sessions (
       id BIGSERIAL PRIMARY KEY,
       article_id BIGINT NOT NULL,
       user_id BIGINT NOT NULL,
       session_token VARCHAR(128) UNIQUE,
       last_active_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
       INDEX idx_article (article_id),
       INDEX idx_user (user_id)
   );
   ```

2. **实现 API 端点**：
   - `POST /api/knowledge/articles/{id}/collaborate` - 加入协作 Session
   - `GET /api/knowledge/articles/{id}/collaborators` - 获取当前协作者列表
   - `DELETE /api/knowledge/articles/{id}/collaborate` - 离开协作 Session

3. **可选：WebSocket 实时通知**
   - 协作者加入/离开时推送通知
   - 实时显示协作者在线状态

#### 前端任务

1. **调用协作 API**：
   - 打开文档时自动加入协作 Session
   - 定时刷新协作者列表

2. **显示协作者 UI**：
   - 显示当前协作者头像/名称
   - 显示协作者在线状态
   - 可选：显示协作者正在编辑的位置

3. **清理逻辑**：
   - 页面关闭时离开协作 Session
   - 超时自动清理不活跃 Session

### 验收标准

- [ ] 后端协作 Session API 实现
- [ ] 前端调用 API 获取协作者列表
- [ ] 显示协作者 UI 组件
- [ ] 可选：WebSocket 实时通知
- [ ] 超时清理机制

---

## 创建说明

### 手动创建

1. 访问 https://github.com/heidsoft/itsm/issues/new
2. 复制上方 Issue 内容
3. 添加对应 Labels
4. 设置 Milestone 和 Assignee

### 自动创建 (使用 gh CLI)

```bash
# Issue #1
gh issue create \
  --title "创建 BPMN 版本变更日志表" \
  --body-file /tmp/issue1.md \
  --label "enhancement,backend,bpmn,P2" \
  --milestone "迭代四"

# Issue #4
gh issue create \
  --title "知识库实时协作 Session 集成" \
  --body-file /tmp/issue4.md \
  --label "enhancement,frontend,backend,collaboration,P3" \
  --milestone "迭代五"
```

---

**生成者**: 开发助手 (小开) 💻  
**关联任务**: P1-04 - 创建 GitHub Issues
