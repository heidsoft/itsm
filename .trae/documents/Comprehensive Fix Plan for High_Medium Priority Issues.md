# 1. 修复 Ent Panic 问题
- **原因**: 代码中使用了 Ent 框架生成的 `Must*` 方法（如 `MustCreate`, `MustUpdate`），这些方法在出错时会直接 panic，导致服务崩溃。
- **方案**:
    - 扫描并替换所有 `Must*` 方法调用为常规的错误处理调用（如 `Create`, `Update`），并正确处理返回的 `error`。
    - 重点检查迁移脚本和服务层代码。

# 2. 修复错误处理不当
- **文件**: `service/ai/embedder_openai.go`
- **问题**: `json.Marshal` 和 `http.NewRequest` 的错误被忽略 (`_`)。
- **方案**: 添加显式的错误检查和处理。

# 3. 解决 React 19 兼容性问题
- **文件**: `package.json`
- **问题**: 项目依赖 React 19，但部分组件库可能尚未完全适配，且存在 peer dependency 警告风险。
- **方案**: 暂时保持 React 19（因已引入补丁），但建议检查并更新相关依赖配置，或在 `next.config.ts` 中添加必要的 transpile 配置（如果需要）。针对当前任务，主要是确认依赖状态。

# 4. 修复硬编码敏感信息
- **文件**: `config.yaml`, `migrate_simple.go`
- **问题**: 配置文件和迁移脚本中包含硬编码的密码和哈希。
- **方案**:
    - `config.yaml`: 使用 `${ADMIN_PASSWORD}` 环境变量替代硬编码密码。
    - `migrate_simple.go`: 使用 `bcrypt` 动态生成密码哈希，或从环境变量读取预计算的哈希。

# 5. 改进告警服务 (Incident Alerting)
- **文件**: `service/incident_alerting_service.go`
- **问题**:
    - 告警发送仅模拟 (`time.Sleep`)。
    - 存在硬编码的邮箱、用户ID (1) 和统计数据 (15.5)。
- **方案**:
    - 实现基础的 SMTP 邮件发送逻辑（使用 `net/smtp`）。
    - 将硬编码的邮箱地址改为从配置/环境变量读取。
    - 将硬编码的 UserID 改为参数传递或从 Context 获取。
    - 修复统计逻辑，改为从数据库查询实际数据。

# 6. 优化 LLM Provider 配置
- **文件**: `service/ai/llm_providers.go`
- **问题**: HTTP 客户端超时时间过长 (5分钟)。
- **方案**: 将超时时间缩短为可配置值（如 60秒），或使用环境变量控制。

# 7. 修复前端 Effect 依赖
- **文件**: `CIList.tsx`
- **问题**: `useEffect` 依赖数组不完整或错误。
- **方案**: 使用 `useCallback` 包装函数，并修正 `useEffect` 的依赖数组，消除 ESLint 警告。

# 8. 清理重复迁移脚本
- **文件**: `migrate_simple.go`, `migrate_fresh.go`
- **问题**: 功能重叠。
- **方案**: 保留功能更全的 `migrate_fresh.go`，将 `migrate_simple.go` 中的独特逻辑（如果有）合并过去，然后删除 `migrate_simple.go`。
