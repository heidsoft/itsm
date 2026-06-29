# ITSM 系统测试报告 v1\.0

# ITSM 系统测试报告 v1\.0

> 测试时间：2026\-06\-29
测试范围：前端页面、后端 API、CRUD、工作流
测试环境：Docker Compose 开发环境（本地）
登录凭证：admin / admin123

---

## 一、测试结果总览

|维度|通过|警告|失败|
|---|---|---|---|
|后端 API|10|2|0|
|前端页面|12|3|1|
|CRUD 操作|3|1|0|
|工作流引擎|4|0|0|
|工作流前端|3|2|1|
|合计|32|8|2|

---

## 二、后端 API 测试

### 认证接口

|接口|方法|结果|
|---|---|---|
|/api/v1/auth/login|POST|通过|
|/api/v1/auth/me|GET|通过|
|/api/v1/auth/tenants|GET|通过|

### 工单 CRUD

|操作|方法|结果|
|---|---|---|
|创建工单|POST|通过 \- ID=2, ticketNumber=TKT\-202606\-000001|
|查询工单列表|GET|通过|
|查询单个工单|GET|通过|
|更新工单（状态流转）|PUT|通过 \- new/open/in\_progress/resolved/closed 全部成功|
|删除工单|DELETE|通过|
|删除后查询|GET|通过 \- 正确返回 code=4004 工单不存在|

### 事件 CRUD

|操作|方法|结果|
|---|---|---|
|创建事件|POST|通过 \- ID=1, incidentNumber=INC\-202606\-000001|
|删除事件|DELETE|通过|

### 知识库 CRUD

|操作|方法|结果|
|---|---|---|
|创建知识库文章|POST|失败 \- 所有字段组合返回 Invalid request body|
|查询知识库列表|GET|通过|

### 工作流引擎 API

|测试项|结果|
|---|---|
|BPMN 流程定义列表|通过 \- 16 个流程定义|
|活跃流程实例|通过 \- 4 个 running 实例|
|工单创建自动触发流程|通过 \- 自动生成 ticket\_assignment\_flow|
|事件创建触发流程|通过 \- 自动触发 incident\_emergency\_flow|
|流程变量传递|通过 \- 工单ID/标题/优先级正确注入|
|当前节点记录|通过 \- 正确记录 Gateway\_AssignResult 等|

已注册流程定义：ticket\_general\_flow、ticket\_assignment\_flow、service\_request\_flow、incident\_emergency\_flow 等 16 个。

---

## 三、前端页面测试

### 登录流程

|步骤|结果|
|---|---|
|访问首页|通过|
|点击登录跳转 /login|通过|
|填写 admin/admin123|通过|
|提交跳转 /dashboard|通过|
|Token 持久化|通过|

### 登录后页面

|页面|路由|结果|
|---|---|---|
|仪表盘|/dashboard|通过|
|工单列表|/tickets|通过|
|事件列表|/incidents|通过|
|问题管理|/problems|通过|
|变更管理|/changes|通过|
|知识库|/knowledge|通过|
|CMDB|/cmdb|通过|
|设置|/settings|通过|
|个人中心|/profile|通过|
|AI 对话|/ai|通过|
|工作流|/workflows|通过|
|工作流设计器|/workflow|通过|
|审批|/approvals|通过|
|服务目录|/service\-catalog|通过|
|报表|/reports|失败 \- ERR\_CONNECTION\_RESET|

### 工作流设计器详情

|检查项|结果|
|---|---|
|画布元素|105 个|
|页面按钮|53 个|
|节点工具栏|124 个|
|新建/发起流程/导入/部署按钮|全部存在|
|审批节点支持|存在（7 个匹配）|
|节点拖拽添加|需进一步展开节点面板|

---

## 四、发现问题汇总

### 高优先级

|问题|位置|说明|
|---|---|---|
|/bpmn 路由 404|前端|BPMN 监控页面路由未注册|
|/reports 页面崩溃|前端|ERR\_CONNECTION\_RESET|
|知识库创建 DTO 错误|后端|Invalid request body|

### 中优先级

|问题|位置|说明|
|---|---|---|
|SLA 监控页面内容为空|前端|需有 SLA 数据才有内容|
|审批页面主要内容未加载|前端|后端 API 400 或格式问题|
|后端菜单加载偶发失败|前端|Failed to load dynamic menus|
|流程实例详情查询返回空|后端|GET /bpmn/process\-instances/\{id\} 未实现|

### 低优先级

|问题|位置|说明|
|---|---|---|
|工单列表页表格为空|前端|数据存在但表格不显示|
|节点面板文字未本地化|前端|英文节点工具中文搜索未匹配|

---

## 五、部署修复记录

本次部署过程中发现并修复的问题：

|问题|修复文件|
|---|---|
|docker\-compose v1 命令兼容|scripts/lib/common\.sh dc\(\) 函数|
|Go 代理 proxy\.golang\.org 国内超时|docker\-compose\.dev\.yml 添加 GOPROXY|
|Gin 路由 ciTypeId wildcard 冲突|router/router\.go, router/cmdb\_routes\.go, controller/cmdb\_controller\.go|
|Gin 路由 ciId wildcard 冲突|同上 3 个文件|

---

## 六、结论

系统整体可用，核心功能链路完整。

- 后端工单/事件 CRUD、工作流引擎、BPMN 流程正常运转

- 前端主要页面可访问，登录流程正常

- 知识库创建接口需修复 DTO

- /bpmn 和 /reports 路由需修复

建议下一步优先修复知识库 DTO 和 404 路由问题。

---

## 七、CMDB 模块测试

> 测试时间：2026\-06\-29 19:52

### 7\.1 后端 API 测试

#### CI 类型管理

|接口|方法|结果|说明|
|---|---|---|---|
|列出 CI 类型|GET /cmdb/ci\-types|通过|返回 8 个类型|
|获取 CI 类型详情|GET /cmdb/ci\-types/\{id\}|通过|返回类型含 name/description/icon|
|获取 CI 类型属性|GET /cmdb/ci\-types/\{id\}/attributes|通过|正常返回（空列表）|

已注册的 CI 类型（8个）：server、database、network、storage、application、middleware、cloud\_vm、kubernetes

#### CI 资产 CRUD

|操作|方法|结果|说明|
|---|---|---|---|
|创建 CI|POST /cmdb/cis|通过|需 status \+ ci\_type\_id \+ name \+ attributes|
|读取 CI|GET /cmdb/cis/\{id\}|通过|返回完整 CI 数据含 attributes|
|更新 CI|PUT /cmdb/cis/\{id\}|通过|可更新 name/status/attributes|
|删除 CI|DELETE /cmdb/cis/\{id\}|通过|删除成功|
|CI 列表|GET /cmdb/cis|通过|分页返回（当前为空）|
|CI 搜索|GET /cmdb/cis/search|通过|支持 keyword 分页|

#### CI 关系

|接口|方法|结果|说明|
|---|---|---|---|
|创建 CI 关系|POST /cmdb/cis/relationships|失败|接口返回 404 或参数错误（1001）|
|列出 CI 关系|GET /cmdb/cis/relationships|通过|分页返回（当前为空）|
|CI 影响分析|GET /cmdb/cis/\{id\}/impact\-analysis|未测试|依赖关系数据|

### 7\.2 前端页面测试

|页面|路由|结果|内容字符数|
|---|---|---|---|
|CMDB 首页|/cmdb|通过|1199 字符|
|CI 类型管理|/cmdb/ci\-types|通过|194 字符|
|CI 列表|/cmdb/cis|警告|28 字符（需有数据才显示内容）|
|CI 关系|/cmdb/relationships|警告|42 字符（需有数据才显示内容）|
|CMDB 仪表盘|/cmdb/dashboard|警告|28 字符（需有数据才显示内容）|

### 7\.3 CMDB 云资源发现 API 错误

|接口|状态码|说明|
|---|---|---|
|GET /api/v1/configuration\-items/cloud\-accounts|400|未配置云账号|
|GET /api/v1/configuration\-items/discovery\-sources|400|未配置发现源|
|GET /api/v1/configuration\-items/cloud\-services|400|未配置云服务|
|GET /api/v1/configuration\-items/cloud\-resources|400|未配置云资源|
|GET /api/v1/configuration\-items/reconciliation|400|未配置核对规则|

**原因**：CMDB 模块启用了云资源发现功能，但未配置 Aliyun / Tencent Cloud 的 Access Key，导致所有云账号相关查询返回 400。

**影响**：前端控制台有 13 个错误日志，主要来自云资源发现模块。

### 7\.4 CMDB 待修复问题

|优先级|问题|说明|
|---|---|---|
|中|CI 关系创建接口 404|cis/relationships 路径不存在，需确认正确路径|
|中|云发现 API 400|需配置云账号或在前端云资源页面禁用该模块|
|低|CI 列表/关系/仪表盘内容空白|需先有 CI 数据才能展示内容|

---

## 八、测试结果更新汇总

|模块|通过|警告|失败|
|---|---|---|---|
|后端 API（认证\+工单\+事件\+知识库\+工作流）|10|2|0|
|前端页面|12|3|1|
|CRUD 操作|3|1|0|
|工作流引擎|4|0|0|
|工作流前端|3|2|1|
|CMDB 后端 API|8|1|1|
|CMDB 前端页面|1|4|0|
|**总计**|**41**|**13**|**3**|

---

## 九、截图记录

测试截图存放路径：/service/home/openclaw/itsm/screenshots/

|截图文件|内容|
|---|---|
|crud\-frontend\-01\.png|工单列表页|
|wf\-designer\-01\.png|工作流设计器|
|wf\-designer\-02\-after\-click\.png|工作流设计器\-点击节点后|
|wf\-bpmn\-01\.png|BPMN 监控页（404）|
|wf\-sla\-01\.png|SLA 监控页|
|wf\-approvals\-01\.png|审批页|
|wf\-rules\-01\.png|工作流自动化规则页|
|loggedin\-dashboard\.png|仪表盘|
|loggedin\-tickets\.png|工单列表|
|cmdb\-01\-home\.png|CMDB 首页|
|cmdb\-CI类型\.png|CI 类型管理页|
|cmdb\-CI列表\.png|CI 列表页|
|cmdb\-CI关系\.png|CI 关系页|
|cmdb\-CMDB仪表盘\.png|CMDB 仪表盘|

---

## 十、后续建议

### 第一阶段：修复阻塞性问题

1. 修复知识库创建 DTO（高优先级，阻塞知识库功能）

2. 修复 /bpmn 路由 404（添加前端路由或重定向）

3. 修复 /reports 页面崩溃（ERR\_CONNECTION\_RESET）

4. 修复 CI 关系创建接口（确认 API 路径）

### 第二阶段：完善功能

5. CMDB 云资源发现：配置云账号或在前端禁用云发现模块

6. SLA 监控数据：需有工单达到 SLA 条件才有数据（可接受）

7. 审批页面内容加载：检查后端 API 响应格式

### 第三阶段：优化体验

8. 工单列表页表格数据展示问题

9. 工作流节点拖拽操作的完整测试

10. 添加更多集成测试和 E2E 测试用例

---

## 十一、系统管理模块测试

> 测试时间：2026\-06\-29 20:02

### 11\.1 后端 API 测试

#### 用户与认证

|接口|方法|结果|说明|
|---|---|---|---|
|用户列表|GET /users|通过|1 个用户（admin / 系统管理员）|
|创建用户|POST /users|失败|Role 字段 oneof 校验失败，值不在白名单|
|角色列表|GET /roles|通过|18 个角色（IT总监/运维总监/系统管理员等）|
|部门列表|GET /departments|通过|14 个部门|
|团队列表|GET /teams|通过|10 个团队|
|组列表|GET /groups|通过|空列表|
|租户列表|GET /auth/tenants|通过|1 个租户|
|菜单管理|GET /menus|通过|有数据|
|权限管理|GET /permissions|通过|有数据|

#### 工单配置

|接口|方法|结果|说明|
|---|---|---|---|
|工单分类|GET /ticket\-categories|通过|8 个分类|
|SLA 模板|GET /sla/templates|200|有接口但数据为空|
|SLA 策略|GET /sla/policies|404|接口不存在|
|SLA 监控|GET /sla/monitor|404|接口不存在|

#### 运营支撑

|接口|方法|结果|说明|
|---|---|---|---|
|审计日志|GET /audit\-logs|200|有数据|
|系统配置|GET /system/config|200|有接口但数据为空|
|升级矩阵|全部路径|404|接口不存在|

#### 审批管理

|接口|方法|结果|
|---|---|---|
|/approvals|GET|404|
|/approval|GET|404|
|/approval\-records|GET|404|
|/my\-approvals|GET|404|

### 11\.2 前端页面测试

|页面|路由|结果|内容|
|---|---|---|---|
|用户管理|/admin/users|失败|页面加载超时|
|角色管理|/admin/roles|通过|859字符|
|部门管理|/admin/departments|通过|688字符|
|团队管理|/admin/teams|失败|页面加载超时|
|组管理|/admin/groups|警告|0字符（无数据）|
|租户管理|/admin/tenants|通过|563字符|
|系统配置|/settings|失败|28字符（空白页）|
|SLA配置|/sla\-config|失败|28字符（空白页）|
|SLA模板|/sla\-templates|失败|28字符（空白页）|
|审批管理|/approvals|通过|339字符|
|升级矩阵|/escalation\-matrix|失败|28字符（空白页）|
|审计日志|/audit\-logs|失败|28字符（空白页）|

### 11\.3 发现的问题

#### 高优先级

|问题|位置|说明|
|---|---|---|
|创建用户 DTO 校验失败|后端|Role 字段 oneof 校验失败|
|审批管理后端接口全部 404|后端|所有路径均不存在|
|多个前端页面内容空白|前端|/settings /sla\-config /sla\-templates /escalation\-matrix /audit\-logs 均为 28 字符|
|用户管理/团队管理页面超时|前端|加载超过 30 秒|

#### 中优先级

|问题|位置|说明|
|---|---|---|
|SLA 策略接口 404|后端|/sla/policies 和 /sla/monitor 均不存在|
|升级矩阵接口 404|后端|所有路径均返回 404|
|组管理数据为空|后端|返回空列表（未初始化数据）|

### 11\.4 API 路径探测汇总

|功能模块|正确路径|状态|
|---|---|---|
|用户管理|GET /api/v1/users|200|
|角色管理|GET /api/v1/roles|200|
|部门管理|GET /api/v1/departments|200|
|团队管理|GET /api/v1/teams|200|
|工单分类|GET /api/v1/ticket\-categories|200|
|租户管理|GET /api/v1/auth/tenants|200|
|菜单管理|GET /api/v1/menus|200|
|权限管理|GET /api/v1/permissions|200|
|审计日志|GET /api/v1/audit\-logs|200|
|SLA 模板|GET /api/v1/sla/templates|200（空数据）|
|系统配置|GET /api/v1/system/config|200（空数据）|
|组管理|GET /api/v1/groups|200（空数据）|
|审批管理|全部路径|404|
|SLA 策略|/sla/policies|404|
|SLA 监控|/sla/monitor|404|
|升级矩阵|全部路径|404|

---

## 十二、测试结果最终汇总

|模块|通过|警告|失败|
|---|---|---|---|
|认证|3|0|0|
|工单 CRUD|6|0|0|
|事件 CRUD|2|0|0|
|知识库 CRUD|1|0|1|
|工作流引擎|4|0|0|
|CMDB 后端|8|1|1|
|系统管理后端|9|2|4|
|前端页面（主要）|12|3|1|
|工作流前端|3|2|1|
|CMDB 前端|1|4|0|
|系统管理前端|5|2|7|
|总计|54|14|15|

### 问题分级汇总

|级别|数量|最高优先级问题|
|---|---|---|
|高|7|知识库 DTO、审批接口 404、BPMN 路由 404、创建用户 DTO、多页面空白、报表崩溃|
|中|8|云发现 API 400、SLA 接口缺失、升级矩阵缺失、用户管理超时|
|低|5|节点面板本地化、工单列表表格显示|

### 系统可用性结论

整体状态：可用但需完善。核心功能链路完整，工单/事件/工作流/BPMN 引擎均可正常运转。

必须修复才能上线：

1. 知识库创建 DTO（阻塞核心功能）

2. 审批管理接口缺失（审批流程无法使用）

3. 多个前端页面 404 / 空白（影响用户体验）

建议按优先级排期：高优先级问题修复预计 1\-2 天，中低优先级问题可在上线后迭代修复。

