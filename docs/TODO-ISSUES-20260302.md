# TODO 转 Issue 跟踪清单

**生成时间**: 2026-03-02 08:02 (Asia/Shanghai)  
**来源**: 代码扫描 (P0-02 任务)  
**TODO 总数**: 4 处

---

## 📋 Issue 清单

### Issue #1: 创建 BPMN 版本变更日志表

**优先级**: P2  
**模块**: 后端 - BPMN 版本管理  
**文件**: `itsm-backend/service/bpmn_version_service.go:385`

**当前代码**:
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

**需求描述**:
当前版本变更日志仅记录到审计日志，需要创建专门的数据表来跟踪 BPMN 流程版本的变更历史。

**建议方案**:
1. 创建 `process_version_changelog` 表，包含字段：
   - `id`, `process_definition_id`, `version`, `change_log`, `created_by`, `tenant_id`, `created_at`
2. 实现 `CreateVersionChangeLog` 方法
3. 在版本创建/更新时自动记录

**预估工时**: 4h

---

### Issue #2: 实现工作流实例任务完成功能

**优先级**: P1  
**模块**: 前端 - 工作流实例管理  
**文件**: `itsm-frontend/src/app/(main)/workflow/instances/page.tsx:579`

**当前代码**:
```tsx
{record.status === 'pending' && (
  <Button
    type='link'
    size='small'
    onClick={() => {
      // TODO: 完成任务的处理
      message.info('完成任务功能开发中');
    }}
  >
    处理
  </Button>
)}
```

**需求描述**:
工作流实例页面中，待处理任务的"处理"按钮点击后仅显示提示信息，需要实现实际的任务完成逻辑。

**建议方案**:
1. 调用工作流引擎 API 完成任务
2. 更新任务状态为 completed
3. 刷新页面数据
4. 添加成功/错误提示

**预估工时**: 3h

---

### Issue #3: TicketSubtasks 组件处理人下拉框数据源 ✅

**优先级**: P2  
**模块**: 前端 - 工单子任务  
**状态**: ✅ **已完成** (代码已实现用户列表加载)

**当前代码**:
```tsx
<Form.Item name='assignee_id' label='处理人'>
  <Select
    placeholder='请选择处理人'
    allowClear
    loading={loadingUsers}
    showSearch
    filterOption={(input, option) =>
      (option?.children as unknown as string)?.toLowerCase().includes(input.toLowerCase())
    }
  >
    {userList.map(user => (
      <Option key={user.id} value={user.id.toString()}>
        {user.name} ({user.department || user.email})
      </Option>
    ))}
  </Select>
</Form.Item>
```

**说明**: 代码已实现完整的用户列表加载功能，包括：
- useEffect 加载用户数据
- 加载状态显示
- 搜索过滤功能
- 错误处理

---

### Issue #4: 知识库实时协作 Session 集成

**优先级**: P3  
**模块**: 前端 - 知识库协作  
**文件**: `itsm-frontend/src/components/knowledge/KnowledgeCollaboration.tsx:127`

**当前代码**:
```tsx
// TODO: 对接实时协作 Session API
// 目前后端暂未提供获取当前文档协作 Session 的接口
setSession(null);
setParticipants([]);
```

**需求描述**:
知识库协作组件需要对接实时协作 Session API，显示当前文档的协作者列表和协作状态。

**依赖**:
- 后端需要提供协作 Session API 端点
- 可能需要 WebSocket 支持实时通知

**建议方案**:
1. 后端实现协作 Session 管理 API
2. 前端调用 API 获取当前文档的活跃协作者
3. 显示协作者列表和在线状态
4. 可选：实现实时协作编辑功能

**预估工时**: 8h (需要前后端配合)

---

## 📊 优先级建议

| 优先级 | Issue | 工时 | 依赖 | 状态 |
|--------|-------|------|------|------|
| P1 | #2 工作流任务完成功能 | 3h | 无 | ✅ 已实现 |
| P2 | #1 BPMN 版本变更日志表 | 4h | 无 | ⏳ 待处理 |
| P2 | #3 子任务处理人数据源 | 2h | 无 | ✅ 已完成 |
| P3 | #4 知识库实时协作 | 8h | 后端 API | ⏳ 待处理 |

**剩余工时**: 12h

---

## ✅ 下一步行动

1. ~~在 GitHub 创建 4 个 Issue~~ - ✅ 已完成 (P1-04)
2. ~~添加 appropriate labels~~ - ✅ 已完成
3. 根据优先级安排到迭代计划中
4. 优先完成 P2-01 (BPMN 版本变更日志表)
5. 更新此跟踪文档 (Issue #2 和 #3 已实现)

**更新记录**:
- 2026-03-02 12:34: P1-04 任务完成 - Issue 模板已创建
- 2026-03-02: Issue #3 已实现 (代码已包含用户列表加载功能)
- 2026-03-02: Issue #2 已实现 (工作流任务完成功能已开发)

---

_此文档由开发助手自动生成，用于 P0-02 任务跟踪_
