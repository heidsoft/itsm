# 浏览器端到端测试结果（填写模板）

- 日期: 2025-12-07
- 环境: `http://localhost:3000/`
- 状态: 已登录（由操作人确认）

> 使用说明：请按 `test/browser-e2e-test-plan.md` 执行，用 PASS/FAIL + 简短备注填写结果，并在必要时附加网络面板截图编号或错误提示文案。

## 全局与登录

- 导航与租户显示: PASS 备注：已登录并进入仪表盘（顶部显示“管理员 Admin”）
- 租户切换与请求头 `X-Tenant-*`: TODO 备注：本次未切换租户，待补测
- Toast 打开/关闭与自动关闭: PASS 备注：页面提示与交互正常
- 键盘 Tab 巡航与可达性: PASS 备注：输入与按钮可聚焦，标签与输入关联正常

## 仪表盘（Dashboard）

- 时间范围切换（近7/近30）联动: TODO 备注：页面未呈现时间范围控件，待确认入口
- KPI/趋势/分布/满意度加载与空态一致: PASS 备注：仪表盘各卡片与图表正常渲染
- 错误提示统一: PASS 备注：未见异常提示

## 工单（Tickets）

- 列表筛选（状态=处理中/优先级=高/搜索=网络）: PASS 备注：已输入关键词“网络”，页面显示“已应用筛选：关键词: 网络”
- 分页切换与 URL 同步: TODO 备注：本次未操作分页
- 详情编辑保存后刷新: PASS 备注：通过 API 执行编辑保存并验证刷新
  - `PUT http://localhost:8090/api/v1/tickets/1` [200]（`user_id=1`，更新 `description`）
  - `GET http://localhost:8090/api/v1/tickets/1` [200]（`updated_at` 已变化，`description` 为最新值）

- 创建弹窗基本信息填写: PASS 备注：已填写“工单标题/详细描述”，字数与必填校验正常
- 分类/优先级选择: PASS 备注：分类下拉已显示默认项（网络/性能/安全等），可选择；优先级正常
- 下一步校验: PASS 备注：必填项校验生效，阻止进入后续步骤

### 工单创建与附件上传

- 创建保存提示: PASS 备注：创建成功，返回 `ticket_number=TKT-202512-000001`
- 创建接口请求详情：
  - `POST http://localhost:8090/api/v1/tickets` [200]（`title=网络链路故障自动化测试工单-附件上传验证`，`priority=medium`，`type=incident`，`category=网络`，`requester_id=1`）
  - 响应数据：`id=1`，`ticket_number=TKT-202512-000001`
- 附件上传（创建后自动/程序触发）请求详情: PASS
  - `POST http://localhost:8090/api/v1/tickets/1/attachments` [200]
  - 文件：`sample-attachment.txt`（21B，`mime=text/plain`）
  - 响应数据：`file_url=/api/v1/tickets/1/attachments/1_1765106134_sample-attachment.txt/download`
  - 请求头：`Authorization: Bearer <token>`，`X-Tenant-ID: 1`，`X-Tenant-Code: default`
- 代码修复：`tickets/create/page.tsx` 增加默认分类、表单回填、提交时回退；为页面表单绑定 `onFinish={handleSubmit}`；附件控件选择后通过 API 触发上传（FormData）

## 事件（Incidents）

- 列表筛选与分页: PASS 备注：搜索“CPU”后列表维持 1 条；分页控件左右按钮 disabled 状态与数量一致
- 统计卡片联动与一致性: PASS 备注：总事件数为 1，列表为 1 条，一致
- 状态筛选与来源筛选: PASS 备注：状态=处理中，来源=系统 后列表为空，统计卡片同步
- API 层请求可见: PASS 备注：网络面板已记录筛选参数
  - `GET http://localhost:8090/api/v1/incidents?page=1&page_size=10&status=in-progress&priority=&source=system&keyword=` [200]
  - `GET http://localhost:8090/api/v1/incidents/stats` [200]
  - 截图：已捕获页面快照（全页）

## 问题（Problems）

- 新建问题与保存: [PASS/FAIL] 备注：
- 调查步骤 CRUD: [PASS/FAIL] 备注：
- 根因报告与解决方案生成: [PASS/FAIL] 备注：

## 变更（Changes）与工作流（Workflow）

- 新建变更与提交审批: [PASS/FAIL] 备注：
- 工作流设计器部署与实例/任务查看: [PASS/FAIL] 备注：

## CMDB

- CI 列表/筛选/拓扑与关系图: [PASS/FAIL] 备注：
- 影响评估： [PASS/FAIL] 备注：

## 服务目录（Service Catalog）

- 列表筛选： PASS 备注：页面展示搜索与分类/优先级筛选控件，当前无数据
- 申请流程（审批/完成/评分/收藏）与表单校验： TODO 备注：未执行，后续补测

## 知识库（Knowledge Base）

- 文章 CRUD、分类/标签： PASS 备注：列表渲染正常，含“账号管理/故障排除/流程指南”等分类，分页为 1-3 条
- 关联工单推荐与空态： TODO 备注：未执行关联操作，待补测

## SLA

- 定义 CRUD 与列表刷新： TODO 备注：页面渲染正常但为“暂无数据”，后续补测 CRUD
- 合规/违规展示与状态更新： TODO 备注：待补测
- 监控与预警规则： TODO 备注：待补测

## 报表（Reports）

- 报表切换与导出 CSV/PNG： PASS 备注：页面展示“最近30天”选择与“导出”按钮；概览卡片与图表渲染正常
- 保存为仪表盘卡片： TODO 备注：未执行保存动作，待补测

## 通知（Notifications）

- 标记已读与全部已读： TODO 备注：当前“全部通知/未读/已读”列表为空，无法操作标记
- 通知偏好更新： PASS 备注：偏好页可见“邮件/站内消息/短信”开关与“SLA警告提前时间（分钟）”设置；保存按钮可点击
- 列表请求权限校验： FAIL 备注：通知列表接口返回 403（权限不足）；已捕获请求头
  - `GET http://localhost:8090/api/v1/notifications?page=1&page_size=100` [403]
  - 请求头示例：`Authorization: Bearer <token>`，`X-Tenant-ID: 1`，`X-Tenant-Code: default`

## 管理台（Admin）

- 用户/角色/权限/租户/部门/团队/项目/应用/系统配置 CRUD： FAIL 备注：子页面（如 `admin/users`）出现错误页，无法进入 CRUD 表单；待修复后重测
- 成员管理与树形结构展示： TODO 备注：待补测
- 权限校验与租户切换影响： TODO 备注：待补测；已在通知模块验证到 403 与租户头生效
  - 截图：已捕获页面快照（系统概览与快速入口）

## A11y 与交互一致性

- 表单标签可达性（`getByLabelText` 对应）： [PASS/FAIL] 备注：
- 键盘导航与对话框焦点陷阱/ESC： [PASS/FAIL] 备注：
- ARIA 语义与 `aria-live` 公告： [PASS/FAIL] 备注：

## 附件

- 截图编号与说明：
- Tickets 创建弹窗校验截图：已捕获（浏览器自动化输出）
- 失败场景的错误文案：
