# ITSM 大文件重构计划

## 📋 项目概览
- **项目路径**: ~/Downloads/research/itsm
- **重构范围**: 9个前端文件 + 4个后端大文件
- **优先级**: P0 (ticket_service.go + TicketModal.tsx)

---

## 🎯 后端重构计划: ticket_service.go

### 当前状态分析
- **文件行数**: 1,861行
- **方法数量**: 49个方法
- **问题**: 职责混杂，违反单一职责原则
- **已有子服务**: 
  - TicketLifecycleService (已存在)
  - TicketAssignmentService (已存在)
  - TicketSLAService (已存在)
  - TicketNotificationService (已存在)
  - TicketAutomationRuleService (已存在)

### 拆分策略

#### Phase 1: 提取核心CRUD到新服务 (ticket_core_service.go)
**目标行数**: ~400-500行
**职责**: 基础CRUD操作，不包含业务逻辑

待提取方法:
- CreateTicket (简化，仅做基本创建，业务逻辑移交)
- GetTicket
- ListTickets
- UpdateTicket (简化)
- DeleteTicket
- BatchDeleteTickets
- generateTicketNumber (私有辅助)

#### Phase 2: 增强生命周期服务 (ticket_lifecycle_service.go)
**当前行数**: ~500行 → 目标: ~700-800行
**职责**: 状态流转、解决、关闭

待增强方法:
- UpdateTicketStatus → 完全委托给LifecycleService
- ResolveTicket → 委托给LifecycleService.Resolve
- CloseTicket → 委托给LifecycleService.Close
- CancelWorkflow → 委托给WorkflowService
- SyncTicketStatusWithWorkflow → 委托给WorkflowService

#### Phase 3: 增强分配服务 (ticket_assignment_service.go)
**当前行数**: ~500行 → 目标: ~700-800行
**职责**: 分配、升级、查询分配人

待提取/增强方法:
- AssignTicket → 完全委托给AssignmentService.AssignTicket
- EscalateTicket → 委托给AssignmentService.EscalateTicket
- AssignTickets → 委托给AssignmentService.BatchAssign
- GetTicketsByAssignee → 委托给AssignmentService.GetTicketsByAssignee
- getEscalatedPriority (私有) → AssignmentService私有方法
- getEscalationAssignee (私有) → AssignmentService私有方法
- sendAssignmentNotification (私有) → 移到NotificationService
- sendEscalationNotification (私有) → 移到NotificationService

#### Phase 4: 创建新服务 - TicketSearchService (ticket_search_service.go)
**目标行数**: ~300-400行
**职责**: 搜索、过滤、统计

待提取方法:
- SearchTickets
- GetOverdueTickets
- ListTickets (搜索部分逻辑)
- GetTicketStats
- GetTicketAnalytics

#### Phase 5: 创建新服务 - TicketValidationService (ticket_validation_service.go)
**目标行数**: ~200-300行
**职责**: 权限验证、业务规则验证

待提取方法:
- canUpdateTicket (私有)
- 其他验证逻辑

#### Phase 6: 创建新服务 - TicketExportImportService (ticket_export_import_service.go)
**目标行数**: ~400-500行
**职责**: 导入导出、模板管理

待提取方法:
- ExportTickets
- ImportTickets
- CreateTicketTemplate
- UpdateTicketTemplate
- DeleteTicketTemplate
- GetTicketTemplates
- generateCSV (私有)
- generateExcel (私有)
- parseCSV (私有)
- parseExcel (私有)

#### Phase 7: 创建新服务 - TicketWorkflowService (ticket_workflow_service.go)
**目标行数**: ~300-400行
**职责**: 工作流集成

待提取方法:
- triggerWorkflowForTicket
- GetWorkflowStatus
- CancelWorkflow
- SyncTicketStatusWithWorkflow
- mapProcessStatus (私有)

#### Phase 8: 重构主服务 (ticket_service.go - Facade)
**目标行数**: ~300-400行
**职责**: 协调各子服务，提供统一入口

保留方法:
- 所有依赖注入设置方法 (Set*Service)
- 对外API方法 (精简版，主要委托给子服务)
- GetTicketSLAInfo (委托给slaService)
- GetTicketActivity (可能需要保留或提取到AuditService)

#### Phase 9: 提取通知相关 (ticket_notification_service.go 增强)
**当前已存在**: TicketNotificationService
**增强**: 接收从ticket_service.go移入的私有通知方法

---

### 渐进式执行步骤

**Day 1-2:**
1. ✅ 分析现有子服务接口，确保一致性
2. ✅ 创建 TicketCoreService (Phase 1)
3. ✅ 编写单元测试验证功能
4. ✅ 更新ticket_service.go的CreateTicket等方法调用CoreService

**Day 3-4:**
5. ✅ 增强TicketLifecycleService (Phase 2)
6. ✅ 编写测试
7. ✅ 更新ticket_service.go相关方法

**Day 5-6:**
8. ✅ 增强TicketAssignmentService (Phase 3)
9. ✅ 测试
10. ✅ 更新ticket_service.go

**Day 7-8:**
11. ✅ 创建TicketSearchService (Phase 4)
12. ✅ 测试
13. ✅ 更新ticket_service.go

**Day 9-10:**
14. ✅ 创建TicketValidationService (Phase 5)
15. ✅ 测试
16. ✅ 更新ticket_service.go

**Day 11-12:**
17. ✅ 创建TicketExportImportService (Phase 6)
18. ✅ 测试
19. ✅ 更新ticket_service.go

**Day 13-14:**
20. ✅ 创建TicketWorkflowService (Phase 7)
21. ✅ 测试
22. ✅ 更新ticket_service.go

**Day 15:**
23. ✅ 清理ticket_service.go，确保所有方法正确委托 (Phase 8)
24. ✅ 运行集成测试
25. ✅ 构建验证

---

## 🎨 前端重构计划: TicketModal.tsx

### 当前状态分析
- **文件行数**: 720行
- **组件结构**: 单一大型组件，包含多步骤表单
- **问题**: 职责混杂，可读性差，难以测试

### 拆分结构

```
TicketModal/
├── index.tsx                    # 主入口，合并导出
├── TicketModalContainer.tsx     # 容器组件，状态管理 (150行)
├── TicketModalSteps.tsx         # 步骤导航组件 (80行)
├── steps/
│   ├── index.ts
│   ├── Step1BasicInfo.tsx      # 基本信息 (120行)
│   ├── Step2Category.tsx       # 分类选择 (100行)
│   ├── Step3Assignment.tsx     # 分配设置 (100行)
│   └── Step4Review.tsx         # 确认预览 (80行)
├── TicketModal.utils.ts         # 工具函数 (50行)
└── TicketModal.types.ts         # 类型定义 (40行)
```

### 拆分步骤

#### Phase 1: 提取类型和工具函数
1. 创建 `TicketModal.types.ts`
   - 提取所有接口和类型定义
   - 导出供其他组件使用

2. 创建 `TicketModal.utils.ts`
   - 提取工具函数 (getFieldsForStep, validateStep等)
   - 数据处理函数

#### Phase 2: 提取步骤组件
3. 创建 `steps/Step1BasicInfo.tsx`
   - 提取基本信息表单
   - 包含: Title, Description, etc.

4. 创建 `steps/Step2Category.tsx`
   - 提取分类选择逻辑
   - 包含: TicketType, Priority, Category

5. 创建 `steps/Step3Assignment.tsx`
   - 提取分配设置
   - 包含: Assignee, DueDate, etc.

6. 创建 `steps/Step4Review.tsx`
   - 提取确认预览
   - 只读展示所有信息

#### Phase 3: 提取步骤导航和管理
7. 创建 `TicketModalSteps.tsx`
   - 管理步骤导航
   - 步骤间切换逻辑
   - 接收steps配置

#### Phase 4: 重构容器组件
8. 创建 `TicketModalContainer.tsx`
   - 状态管理 (currentStep, formData)
   - 事件处理 (handleNext, handleSubmit, handleCancel)
   - 集成所有子组件

#### Phase 5: 创建主入口
9. 创建 `TicketModal/index.tsx`
   - 导出所有组件
   - 原TicketModal.tsx改为引用新组件

#### Phase 6: 更新导入
10. 更新所有使用TicketModal的文件导入路径

---

### 渐进式执行

**Day 1:**
1. ✅ 分析TicketModal.tsx结构
2. ✅ 创建类型定义文件
3. ✅ 更新原文件导入新类型
4. ✅ 运行测试确保无误

**Day 2:**
5. ✅ 创建工具函数文件
6. ✅ 提取并测试
7. ✅ 更新原文件

**Day 3:**
8. ✅ 创建steps目录结构
9. ✅ 提取Step1BasicInfo
10. ✅ 测试Step1

**Day 4:**
11. ✅ 提取Step2Category
12. ✅ 测试
13. ✅ 验证步骤间数据传递

**Day 5:**
14. ✅ 提取Step3Assignment
15. ✅ 测试
16. ✅ 完整流程验证

**Day 6:**
17. ✅ 提取Step4Review
18. ✅ 测试
19. ✅ 确保所有步骤工作正常

**Day 7:**
20. ✅ 创建TicketModalSteps组件
21. ✅ 集成所有步骤
22. ✅ 测试步骤导航

**Day 8:**
23. ✅ 创建TicketModalContainer
24. ✅ 编写状态管理逻辑
25. ✅ 测试完整交互

**Day 9:**
26. ✅ 创建index.tsx
27. ✅ 更新原TicketModal.tsx为兼容层
28. ✅ 测试所有导入

**Day 10:**
29. ✅ 全局搜索并更新导入路径
30. ✅ 运行前端测试套件
31. ✅ 构建验证

---

## 📊 其他前端文件 (优先级P1-P2)

### P1 (本周)
- SLAViolationMonitor.tsx (768行)
- TicketAssociation.tsx (596行)
- TicketSubtasks.tsx (487行)
- TicketCategoryExport.tsx (436行)

### P2 (本月)
- CIDetail.tsx (548行)
- IncidentDetail.tsx (236行)
- IncidentList.tsx (254行)
- WorkflowDesigner.tsx (711行)

*(注: Incident相关文件较小，可能包含特殊业务逻辑，需谨慎拆分)*

---

## 🧪 测试策略

### 后端测试
1. 每个新服务都需要单元测试
2. 确保现有测试套件通过
3. 集成测试验证端到端功能
4. 保持代码覆盖率 >70%

### 前端测试
1. 单元测试: 每个新组件
2. 集成测试: 步骤间交互
3. E2E测试: 完整表单提交流程
4. 组件快照测试 (如适用)

### 测试命令
```bash
# 后端
cd itsm-backend
go test ./service/... -v
go test ./service/... -cover

# 前端
cd itsm-frontend
npm test -- --coverage
npm run test:e2e
```

---

## ⚠️ 注意事项

1. **渐进式重构**: 每次只拆分一个文件，立即测试
2. **接口稳定**: 保持对外API不变，避免破坏性变更
3. **依赖管理**: 提取服务时注意依赖注入
4. **测试覆盖**: 确保拆分后测试通过率不降低
5. **文档更新**: 更新相关文档和注释
6. **提交规范**: 每个拆分步骤独立commit，信息清晰

---

## 📈 预期效果

### ticket_service.go
- **拆分前**: 1,861行，49个方法
- **拆分后**: ~350行 (Facade) + 6个专门服务 (~2,500行总)
- **复杂度降低**: 每个方法平均依赖数从15+降至5以下
- **可维护性**: 每个服务职责单一，易于理解和修改

### TicketModal.tsx
- **拆分前**: 720行单一组件
- **拆分后**: 主组件150行 + 6个子组件 (~600行总)
- **可读性**: 每个组件功能明确，易于理解
- **可测试性**: 可独立测试每个步骤组件

---

## 📝 重构报告模板

完成后提供:
```markdown
# ITSM 大文件重构完成报告

## 1. 拆分的文件和职责说明
[详细列表]

## 2. 代码行数变化
[对比表格]

## 3. 复杂度降低情况
- 函数数量变化
- 平均依赖数变化
- 圈复杂度变化

## 4. 测试通过率
- 单元测试通过率: XX%
- 集成测试通过率: XX%
- E2E测试通过率: XX%

## 5. 进一步优化建议
[后续建议]
```

---

**开始执行**: 2026-03-04
**预计完成**: 2026-03-15 (2周)
**负责人**: 重构助手
