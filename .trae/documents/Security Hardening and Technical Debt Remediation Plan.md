# 1. 修复硬编码密码

* **文件**: `docker-compose.yml`, `quick-start.sh`

* **问题**: 密码以明文形式存储。

* **方案**:

  * 在 `docker-compose.yml` 中使用环境变量 `${POSTGRES_PASSWORD:-default_secure_password}`。

  * 在 `quick-start.sh` 中提示用户输入密码或生成随机密码。

  * 创建 `.env.example` 模板文件。

# 2. 修复 XSS 风险

* **文件**: `ArticleVersionControl.tsx`, `app/layout.tsx`

* **问题**: 使用 `dangerouslySetInnerHTML` 但未进行消毒。

* **方案**:

  * 在 `ArticleVersionControl.tsx` 中引入 `dompurify` 库对 HTML 内容进行清洗。

  * 在 `app/layout.tsx` 中优化内联脚本，或确保其内容是静态安全的。

# 3. 处理静默忽略错误

* **文件**: `middleware/audit.go`, `service/escalation_service.go`, `service/tool_queue.go`

* **问题**: 使用 `_` 忽略了关键操作的错误返回。

* **方案**:

  * 将 `_` 替换为显式的错误检查，并使用 `logger.Error` 记录错误。

# 4. 完成未实现代码 (TODOs)

* **文件**: `teams/page.tsx`, `departments/page.tsx`, `KnowledgeCollaboration.tsx`

* **问题**: 存在未实现的 API 调用和占位符。

* **方案**:

  * 在 `teams/page.tsx` 和 `departments/page.tsx` 中完善用户加载逻辑（如果尚未完全修复）。

  * 在 `KnowledgeCollaboration.tsx` 中完善 Session 和 评论 API 的对接逻辑（如果尚未完全修复）。

  * 针对 `bpmn_version_service.go` 中的 XML 解析 TODO，添加基础的 XML 解析比较逻辑或标记为后续迭代。

# 5. 修复低优先级问题

* **问题**: 控制台日志过多，Any 类型滥用。

* **方案**:

  * 清理前端不必要的 `console.log`。

  * 尽可能将 `any` 替换为具体类型。

