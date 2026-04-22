# ITSM 前端创建功能检查计划

## 检查日期: 2026-04-22

---

## 一、检查范围

### 创建页面清单
| 模块 | 页面路径 | 文件位置 |
|------|----------|----------|
| 工单创建 | /tickets/create | itsm-frontend/src/app/(main)/tickets/create/page.tsx |
| 事件创建 | /incidents/new | itsm-frontend/src/app/(main)/incidents/new/page.tsx |
| 事件创建(v2) | /incidents/create | itsm-frontend/src/app/(main)/incidents/create/page.tsx |
| 问题创建 | /problems/new | itsm-frontend/src/app/(main)/problems/new/page.tsx |
| 变更创建 | /changes/new | itsm-frontend/src/app/(main)/changes/new/page.tsx |
| CMDB创建 | /cmdb/cis/create | itsm-frontend/src/app/(main)/cmdb/cis/create/page.tsx |
| 知识库创建 | /knowledge/articles/create | itsm-frontend/src/app/(main)/knowledge/articles/create/page.tsx |
| 资产创建 | /assets/new | itsm-frontend/src/app/(main)/assets/new/page.tsx |
| 许可证创建 | /licenses/new | itsm-frontend/src/app/(main)/licenses/new/page.tsx |
| 发布创建 | /releases/new | itsm-frontend/src/app/(main)/releases/new/page.tsx |

---

## 二、检查项目

### 2.1 基础功能检查
- [ ] 表单是否正常渲染
- [ ] 必填字段验证是否生效
- [ ] 表单提交是否正常
- [ ] 提交后是否正确跳转
- [ ] 错误提示是否友好

### 2.2 表单验证检查
- [ ] 必填字段标记
- [ ] 输入格式验证
- [ ] 长度限制验证
- [ ] 数值范围验证
- [ ] 日期格式验证

### 2.3 用户体验检查
- [ ] 加载状态显示
- [ ] 提交按钮禁用状态
- [ ] 返回/取消功能
- [ ] 表单重置功能
- [ ] 帮助提示信息

### 2.4 特色功能检查
- [ ] 动态表单字段
- [ ] 关联数据选择
- [ ] AI智能建议
- [ ] 附件上传
- [ ] 多选/标签功能

---

## 三、检查结果详情

### 3.1 工单创建 (/tickets/create) ✅

**功能完整度**: 100%

**已实现功能**:
1. **类型预设模板** - 12+种工单类型可选
2. **分类筛选** - 硬件/软件/账户/网络/安全等
3. **AI智能分类** - 调用 `/api/ai/triage` 获取建议
4. **动态表单** - 根据类型渲染不同字段
5. **工作流关联** - 自动关联审批流程
6. **优先级设置** - 类型关联默认优先级
7. **表单验证** - 标题/描述最小长度验证
8. **无障碍支持** - aria-label 属性

**表单字段**:
- 标题（必填，2字符以上）
- 描述（必填，10字符以上）
- 优先级（默认：中）
- 分类
- 自定义字段（根据类型动态显示）

**API调用**:
```typescript
TicketApi.createTicket({
  title, description, priority, category,
  formFields, workflow_definition_key
})
```

---

### 3.2 事件创建 (/incidents/new) ✅

**功能完整度**: 95%

**已实现功能**:
1. **CI配置项关联** - 搜索并关联配置项
2. **实时搜索** - 300ms防抖搜索CI
3. **多CI关联** - 支持选择多个配置项
4. **优先级选择** - 低/中/高/紧急
5. **事件类型** - 硬件/软件/网络/安全/性能/其他
6. **已选CI标签显示** - 可删除已选项

**表单字段**:
- 标题（必填）
- 描述（必填，6行）
- 受影响配置项（搜索选择）
- 优先级
- 类型

**API调用**:
```typescript
incidentService.createIncident({
  title, description, priority, type,
  source: 'manual', affected_cis
})
```

---

### 3.3 事件创建v2 (/incidents/create) ✅

**功能完整度**: 100%

**已实现功能**:
1. **分页表单** - 基本信息/影响分析/附件
2. **影响范围评估** - 全局/部门级/团队级/个人
3. **紧急程度选择** - 紧急/高/中/低
4. **受影响系统多选** - Web/API/数据库/网络/存储
5. **初步原因分析** - 文本输入
6. **附件上传** - Upload组件
7. **右侧提示面板** - 优先级说明和联系方式

**表单字段**:
- 标题（必填，200字符限制）
- 描述（必填，6行）
- 优先级（默认：中）
- 来源（默认：手动创建）
- 类型（默认：事件）
- 分类
- 指派人
- 影响范围
- 紧急程度
- 受影响系统
- 初步原因
- 附件

---

### 3.4 问题创建 (/problems/new) ✅

**功能完整度**: 100%

**已实现功能**:
1. **事件触发创建** - URL参数传递事件信息
2. **自动填充** - 从事件信息自动填充标题/描述
3. **根本原因分析** - RCA字段
4. **影响范围记录** - 必填
5. **优先级选择** - 低/中/高/紧急
6. **分类选择** - 7种分类
7. **来源提示** - Alert显示来源事件
8. **Suspense加载** - 骨架屏支持

**表单字段**:
- 标题（必填，2-200字符）
- 描述（必填，10-5000字符）
- 优先级（默认：中）
- 分类（默认：系统问题）
- 根本原因分析（必填，10-5000字符）
- 影响范围（必填，10-5000字符）

**优先级选项**:
```typescript
const priorityOptions = [
  { value: 'low', label: '低' },
  { value: 'medium', label: '中' },
  { value: 'high', label: '高' },
  { value: 'critical', label: '紧急' },
];
```

---

### 3.5 变更创建 (/changes/new) ✅

**功能完整度**: 100%

**已实现功能**:
1. **变更类型** - 普通/标准/紧急
2. **优先级选择** - 紧急/高/中/低
3. **风险等级** - 高/中/低
4. **影响范围** - 高(核心业务)/中(部分业务)/低(较小影响)
5. **计划时间窗口** - 开始/结束时间
6. **受影响CI** - 多个CI ID，逗号分隔
7. **实施计划** - 必填
8. **回滚计划** - 必填

**中文到API映射**:
```typescript
const typeMap = {
  '普通变更': 'normal',
  '标准变更': 'standard',
  '紧急变更': 'emergency',
};
```

**API调用**:
```typescript
ChangeApi.createChange({
  title, description, justification, type, priority,
  impactScope, riskLevel, plannedStartDate, plannedEndDate,
  implementationPlan, rollbackPlan, affectedCis
})
```

---

### 3.6 CMDB配置项创建 (/cmdb/cis/create) ✅

**功能完整度**: 100%

**已实现功能**:
1. **资产类型动态加载** - 从API获取
2. **云资源关联** - 选择云资源自动填充
3. **动态属性渲染** - 根据云服务schema渲染
4. **状态选择** - 运行中/维护中/已停止
5. **环境选择** - 生产/预发布/开发
6. **重要性选择** - 低/中/高/关键
7. **数据来源** - 手工录入/自动发现/批量导入

**表单分组**:
- 基本信息：名称、类型、状态、描述
- 硬件信息：序列号、型号、厂商、供应商
- 采购财务：采购日期、价格
- 位置归属：位置、分配人、拥有者
- 云资源信息：厂商、账号、Region、Zone
- 动态属性：基于schema渲染

**云厂商选项**:
- 阿里云 (aliyun)
- 华为云 (huawei)
- 腾讯云 (tencent)
- Azure
- 私有云 (onprem)

---

### 3.7 知识库文章创建 (/knowledge/articles/create) ✅

**功能完整度**: 95%

**已实现功能**:
1. **文章标题** - 必填
2. **分类选择** - 动态加载或默认分类
3. **标签系统** - 多标签支持
4. **Markdown内容** - 16行文本框
5. **占位符提示** - Markdown格式示例

**默认分类**:
- 故障排查
- 解决方案
- 操作流程
- 最佳实践
- 技术文档

**待优化**:
- [ ] Markdown实时预览
- [ ] 图片粘贴上传
- [ ] 草稿自动保存

---

### 3.8 资产创建 (/assets/new) ✅

**功能完整度**: 100%

**已实现功能**:
1. **资产编号验证** - 正则验证格式
2. **资产名称验证** - 2-200字符
3. **资产类型选择** - 硬件/软件/云资源/许可证
4. **分类/子分类**
5. **硬件信息** - 序列号/型号/制造商/供应商
6. **采购财务** - 日期/价格/保修期/支持期
7. **位置归属** - 物理位置/所属部门
8. **标签系统**
9. **字段帮助提示** - Tooltip说明

**表单验证规则**:
```typescript
const formRules = {
  assetNumber: [
    { required: true, message: '请输入资产编号' },
    { pattern: /^[A-Z0-9-]+$/, message: '只能包含大写字母、数字和连字符' },
    { min: 3, max: 50 },
  ],
  assetName: [
    { required: true, message: '请输入资产名称' },
    { min: 2, max: 200 },
  ],
  purchasePrice: [
    { type: 'number', min: 0, max: 999999999 },
  ],
};
```

---

### 3.9 许可证创建 (/licenses/new) ✅

**功能完整度**: 100%

**已实现功能**:
1. **许可证名称** - 必填
2. **供应商**
3. **许可证类型** - 永久/订阅/按用户/按席位/站点
4. **许可证密钥**
5. **总数量** - InputNumber
6. **采购信息** - 日期/价格
7. **到期日期**
8. **续费成本**
9. **支持信息** - 供应商/联系方式
10. **备注/标签**

**许可证类型选项**:
```typescript
const licenseTypes = [
  { value: 'perpetual', label: '永久 (Perpetual)' },
  { value: 'subscription', label: '订阅 (Subscription)' },
  { value: 'per-user', label: '按用户 (Per-User)' },
  { value: 'per-seat', label: '按席位 (Per-Seat)' },
  { value: 'site', label: '站点 (Site)' },
];
```

---

### 3.10 发布创建 (/releases/new) ✅

**功能完整度**: 100%

**已实现功能**:
1. **发布编号** - 必填
2. **发布标题** - 必填
3. **发布类型** - 主版本/次版本/补丁/紧急修复
4. **目标环境** - 开发/预发布/生产
5. **严重程度** - 低/中/高/严重
6. **计划时间** - 发布日期/开始时间/结束时间
7. **发布内容** - 说明/部署步骤/受影响系统/组件
8. **回滚验证** - 回滚程序/验证标准
9. **开关选项** - 紧急发布/需要审批

**发布类型选项**:
```typescript
const releaseTypes = [
  { value: 'major', label: '主版本 (Major)' },
  { value: 'minor', label: '次版本 (Minor)' },
  { value: 'patch', label: '补丁 (Patch)' },
  { value: 'hotfix', label: '紧急修复 (Hotfix)' },
];
```

---

## 四、检查总结

### 完成度统计
| 指标 | 结果 |
|------|------|
| 总创建页面数 | 10 |
| 功能完整 | 10 |
| 表单验证完整 | 10 |
| API集成完成 | 10 |

### 特色功能统计
| 功能 | 页面数 |
|------|--------|
| 动态表单字段 | 2 (工单、CMDB) |
| AI智能建议 | 1 (工单) |
| CI配置项关联 | 1 (事件) |
| 云资源关联 | 1 (CMDB) |
| 事件触发创建 | 1 (问题) |
| 分页表单 | 1 (事件v2) |

### 代码质量
- ✅ TypeScript类型定义完整
- ✅ 表单验证规则清晰
- ✅ 错误处理完善
- ✅ 加载状态管理
- ⚠️ 部分页面存在重复（/incidents/new 和 /incidents/create）

---

## 五、建议优化项

### P0 - 必须修复
无

### P1 - 建议优化
1. 统一事件创建页面（合并 /new 和 /create）
2. 知识库编辑器添加Markdown预览
3. 添加表单草稿自动保存

### P2 - 功能增强
1. 批量导入功能
2. 模板系统
3. 表单字段排序
4. 表单配置化

---

## 六、测试用例建议

### 工单创建测试用例
```gherkin
Feature: 工单创建

Scenario: 选择工单类型并创建
  Given 用户在工单创建页面
  When 用户选择"VPN访问申请"类型
  Then 显示VPN相关的动态表单字段
  And 工作流关联显示

Scenario: AI智能分类
  Given 用户已填写标题和描述
  When 用户点击"获取智能分类建议"
  Then 显示AI推荐的分类和优先级
  And 自动应用到表单
```

### CMDB创建测试用例
```gherkin
Feature: CMDB配置项创建

Scenario: 关联云资源
  Given 用户在CMDB创建页面
  When 用户选择一个云资源
  Then 自动填充Region、Zone、账号信息
  And 显示该资源类型的动态属性字段
```
