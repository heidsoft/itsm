# ITSM Bug 完整清单 v1\.1（字段校验专项）

# ITSM 测试发现 Bug 完整清单 v1\.1（字段校验专项）

> 整理时间：2026\-06\-29 20:51
测试方法：API 路径探测、并发测试、状态机边界测试、分页边界测试、重复数据测试、字段命名一致性测试、枚举边界值测试
相关文档：ITSM 系统测试报告 v1\.0（https://feishu\.cn/docx/KONpdmYeqoPlX1xaEFWcrhKxnYg）

---

## Bug 总览

共发现 30 个 Bug，分布在后端 DTO 校验缺失、前端路由/API 路径不匹配、枚举校验漏洞等多个层面。

|优先级|数量|Bug 编号|
|---|---|---|
|P0（阻塞）|6|B\-1, B\-2, B\-4, B\-5, B\-15, B\-23|
|P1（影响体验）|9|B\-3, B\-8, B\-9, B\-16, B\-17, B\-21, B\-22, B\-25, B\-26|
|P2（次要功能）|10|B\-6, B\-7, B\-10, B\-11, B\-12, B\-18, B\-20, B\-24, B\-27, B\-28|
|P3（后续迭代）|5|B\-13, B\-14, B\-19, B\-29, B\-30|

---

## P0：阻塞功能（必须修复）

### B\-1：知识库文章创建失败

现象：POST /api/v1/knowledge/articles 所有字段组合均返回 "Invalid request body"

根因：category 为字符串传入 ent 枚举类型，未做映射，错误被 handler 统一掩盖为 "Invalid request body"

代码位置：dto/knowledge\_dto\.go、service/knowledge\_service\.go:48、handlers/knowledge/handler\.go:47\-49

---

### B\-2：创建用户 Role 校验失败

现象：传入 "role":"user" 返回 oneof 校验失败

根因：oneof 白名单无 "user"，正确值应为 "end\_user" 或 "agent"

代码位置：dto/user\_dto\.go:24

---

### B\-4：审批管理接口路径全部 404

现象：GET /api/v1/approvals 等全部返回 404

根因：实际路径是 /approval\-workflows 和 /approval\-chains，前端调用了错误路径

代码位置：router/router\.go:381\-526、src/lib/api/ticket\-approval\-api\.ts

---

### B\-5：SLA 接口路径全部 404

现象：/sla/policies、/sla/monitor 均返回 404

根因：实际路径是 /sla/definitions 和 /sla/alert\-rules

代码位置：router/router\.go:877\-893、src/lib/api/sla\-api\.ts

---

### B\-15：Description min=1 导致空描述工单无法创建

现象：并发创建5个工单全部失败

根因：Description 要求 min=1，空字符串不满足。前端表单允许提交空 description，但后端拒绝。

代码位置：dto/ticket\_dto\.go CreateTicketRequest Description binding

---

### B\-23：Service Request catalog\_id 蛇形命名不匹配

根因：后端 DTO 期望 catalog\_id（蛇形），前端发送 catalogId（驼峰）

代码位置：dto/service\_dto\.go:21、src/lib/api/service\-request\-api\.ts

---

## P1：影响体验（尽快修复）

### B\-3：Goroutine 无法被取消

6处使用 context\.Background\(\) 替代请求 context

代码位置：service/ticket\_service\.go 多处

### B\-8/B\-9：前端页面空白（5个页面 \+ BPMN 404）

/settings、/sla\-config、/sla\-templates、/escalation\-matrix、/audit\-logs 均为空白页

### B\-16：分页 page 无边界校验

page=0、\-1、999999 均返回 code=0

### B\-17：工单标题无重复校验

可创建完全相同标题的多个工单

### B\-21：前端空白页面路由未对齐

5个页面的菜单 path 与 Next\.js 文件系统路由不一致

### B\-22：审批前端 API 路径全部缺少 /v1 前缀

ticket\-approval\-api\.ts 所有路径缺少 /v1 前缀段

代码位置：src/lib/api/ticket\-approval\-api\.ts:82/87/95/100/125/136

### B\-25：工单创建 priority 完全无枚举校验 ⭐新增

现象：priority='invalid'、priority='HIGH' 均返回 code=0（创建成功）

根因：CreateTicketRequest\.Priority 只有 binding:"required"，无 oneof 枚举校验

代码位置：dto/ticket\_dto\.go:18

修复建议：添加 binding:"required,oneof=low medium high critical urgent"

### B\-26：知识库 category 字段完全无枚举校验 ⭐新增

现象：任意字符串（"invalid\_xyz"、"123"、" "）都能创建知识库文章

根因：dto\.CreateKnowledgeArticleRequest\.Category 无 oneof 校验

代码位置：dto/knowledge\_dto\.go:7\-11

---

## P2：次要功能（计划修复）

### B\-6：升级矩阵接口完全缺失

所有路径均返回 404，需从零开发

### B\-7：系统配置路径错误

前端用 /configs，后端注册为 /config

### B\-10：报表页面崩溃

ERR\_CONNECTION\_RESET

### B\-11：工单列表表格数据为空

API 有数据但前端不显示

### B\-12：CI 关系创建接口路径错误

cis/relationships 返回 404 或参数错误

### B\-18：Service Request 创建字段名不匹配

已确认 catalog\_id（蛇形）问题（B\-23 的子问题）

### B\-20：重复状态更新无幂等性保护

closed 工单再次 PUT status=closed 返回 code=0 无提示

### B\-24：SLA monitor 接口路径不存在

前端调用 /sla/monitor，后端未注册

### B\-27：Service Request 必填字段未完整暴露 ⭐新增

catalog\_id 正确，但提示 "Expiration date required"（未告知 expireAt 必填）

代码位置：handlers/service\_request/service\.go Create 方法

### B\-28：工单 status 更新枚举校验错误信息不明确 ⭐新增

status='invalid' 返回 1001 但错误信息未列出有效状态值

---

## P3：后续迭代

### B\-13：云资源发现 API 全部 400

未配置云厂商 Access Key

### B\-14：Service Requests status=pending 返回 400

pending 不是有效状态值

### B\-19：SLA 告警未实时触发

预期行为（需超时才告警）

### B\-29：Service Request status 参数在创建时被忽略 ⭐新增

创建时传 status 参数无效，状态由后端控制

### B\-30：知识库 content 空字符串行为不一致 ⭐新增

有时返回 400 有时返回 0，待进一步确认

---

## 字段校验漏洞专项分析

|字段|创建时校验|更新时校验|问题|
|---|---|---|---|
|Ticket\.priority|❌ 无枚举|✅ oneof|B\-25|
|Ticket\.status|❌ 无枚举|✅ oneof|B\-28|
|Ticket\.description|✅ min=10|✅ min=10|前端表单 10 字符，后端 10 字符，一致|
|Ticket\.title|✅ min=2/max=200|✅ min=2/max=200|一致|
|Incident\.description|❌ 无校验|❌ 无校验|可为空，可任意长度|
|KnowledgeArticle\.category|❌ 无枚举|❌ 无枚举|B\-26|
|Change\.riskLevel|❌ 无枚举|❌ 无枚举|可接受任意值|
|ServiceRequest\.catalog\_id|✅ 蛇形必填|N/A|B\-23, B\-27|
|ServiceRequest\.status|N/A（创建时忽略）|❌ 无枚举|B\-29|

---

## 测试方法总结

|测试方法|发现的问题|
|---|---|
|API 路径探测|B\-4/B\-5/B\-6/B\-7/B\-22|
|并发创建测试|B\-15|
|状态机边界测试|B\-20/B\-28|
|分页边界测试|B\-16|
|重复数据测试|B\-17|
|错误消息分析|B\-1/B\-2/B\-18/B\-27|
|前端页面路径探测|B\-8/B\-9/B\-21|
|通知通道检查|B\-19|
|字段命名一致性测试|B\-23|
|枚举边界值测试|B\-25/B\-26/B\-29|

---

## P1 新增 Bug（字段校验缺失·第二轮）

### B\-31：知识库 content 前端必填但后端不禁

`content=''`、`content=' '` 均可创建知识库文章（code=0）

- 前端：required: true

- 后端：dto/knowledge\_dto\.go:8 — `Content string json:"content"` 无 binding 校验

对比：工单评论 content 有正确校验（required,min=1,max=5000）

### B\-32：用户创建 role 字段被前端忽略

前端用户管理页面有 role 下拉框，但创建时未发送 role 字段，用户总是被赋予默认值 end\_user。

代码位置：src/app/\(main\)/admin/users/page\.tsx:114\-119

### B\-33：事件 description 完全无长度限制

Incident description 长度 0\-10000 字符均返回 code=0（全部成功）

根因：dto/incident\_dto\.go:40 — 无 binding 校验

### B\-34：Problem Investigation 接口测试异常

article\_content 蛇形和驼峰均返回空响应，可能需要额外必填字段或接口路径错误。待进一步诊断。

---

## Bug 数量最终汇总

|优先级|数量|Bug 编号|
|---|---|---|
|P0（阻塞）|6|B\-1, B\-2, B\-4, B\-5, B\-15, B\-23|
|P1（影响体验）|11|B\-3, B\-8, B\-9, B\-16, B\-17, B\-21, B\-22, B\-25, B\-26, B\-31, B\-32|
|P2（次要功能）|11|B\-6, B\-7, B\-10, B\-11, B\-12, B\-18, B\-20, B\-24, B\-27, B\-28, B\-33|
|P3（后续迭代）|6|B\-13, B\-14, B\-19, B\-29, B\-30, B\-34|

**总计：34 个 Bug**

