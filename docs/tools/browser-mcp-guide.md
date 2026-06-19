# 浏览器 MCP 方案对比与配置指南

> 本文档整理当前主流浏览器 MCP（Model Context Protocol）实现方案，涵盖配置方法、实现原理、架构差异及选型建议。

---

## 一、方案总览

| 方案 | 实现方式 | Chrome 扩展 | STDIO | 已知问题 |
|------|----------|-------------|-------|----------|
| **chrome-devtools-mcp** | Puppeteer + CDP 直连 | ❌ 不需要 | ✅ | 需要 Chrome 已启动并开启远程调试 |
| **@agent-infra/mcp-server-browser** | CDP 直连（无 Puppeteer） | ❌ 不需要 | ✅ | 依赖 Chrome `--remote-debugging-port` |
| **playwright-mcp** | 扩展 Relay + WebSocket → CDP | ✅ 需要（注入 debugger） | ✅ | Chrome 策略阻止扩展访问 localhost |
| **browsermcp.io** | 扩展 + HTTP → MCP | ✅ 需要 | ✅ | 同上，localhost 阻塞 |
| **@agent360/browser-mcp** | 扩展注入 CDP | ✅ 需要（加载解压扩展） | ✅ | 需要手动加载 .crx |

---

## 二、实现原理详解

### 2.1 两类核心架构

#### A. Extension-Relay 架构（扩展桥接型）

```
┌─────────────┐      WebSocket      ┌──────────────┐      CDP       ┌─────────────┐
│ Claude Code │◄─────── stdio ─────►│  扩展 Relay   │◄── chrome.    │  Chrome     │
│   (MCP Client) │                  │  (node.js)    │    debugger  │  Browser    │
└─────────────┘                     └──────────────┘               └─────────────┘
                                        ▲
                                        │ 安装 Chrome 扩展
                                    ┌───┴────┐
                                    │ 扩展 ID │
                                    │ (mmlmf..) │
                                    └────────┘
```

**原理**：
1. Chrome 扩展注入 `chrome.debugger` API，获得 CDP 访问能力
2. 扩展通过 `chrome.runtime.connect` 与本地 Relay 服务建立 WebSocket 连接
3. Relay 将 CDP 命令转发给浏览器，将结果通过 WebSocket → MCP stdio 返回给 Claude Code

**致命缺陷**：Chrome 安全策略默认阻止扩展访问 `localhost`（`ERR_BLOCKED_BY_CLIENT`）

- 变通方案：启动 Chrome 时加 `--allow-insecure-localhost` 参数
- 但每次打开浏览器都要手动加参数，体验差
- 某些企业 Chrome 策略完全禁用此参数

#### B. CDP-Direct 架构（直连型）

```
┌─────────────┐      STDIO        ┌──────────────┐      CDP       ┌─────────────┐
│ Claude Code │◄─────── stdio ─────►│  MCP Server  │◄── Puppeteer  │  Chrome     │
│   (MCP Client) │                  │  (Node.js)   │    或直接 CDP │  Browser    │
└─────────────┘                     └──────────────┘               └─────────────┘
                                                                     ▲
                                                          启动时加 --remote-debugging-port
```

**原理**：
1. MCP Server 通过 Puppeteer 启动 Chrome（或连接已有 Chrome 实例）
2. Puppeteer 底层使用 CDP（Chrome DevTools Protocol）与浏览器通信
3. MCP Server 将工具调用转换为 CDP 命令，通过 stdio 与 Claude Code 交互

**优势**：
- 无需 Chrome 扩展，不受 localhost 策略限制
- 架构简单，调试路径短

---

## 三、配置详解

### 3.1 chrome-devtools-mcp（Google 官方）

**仓库**：`https://github.com/ChromeDevTools/chrome-devtools-mcp`

**安装**：
```bash
npm install -g chrome-devtools-mcp
```

**配置**（`~/.claude.json` 中的 MCP servers）：

```json
{
  "mcpServers": {
    "chrome-devtools": {
      "command": "npx",
      "args": ["-y", "chrome-devtools-mcp"],
      "env": {
        "CHROME_PATH": "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
      },
      "type": "stdio"
    }
  }
}
```

**依赖**：
- Chrome 已在运行（带 `--remote-debugging-port=9222`），或由 MCP 自动启动
- 推荐启动方式：`"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --remote-debugging-port=9222`

**工具集**（27+ 个）：
- `browser_navigate`：页面导航
- `browser_screenshot`：截图
- `browser_evaluate`：执行 JS
- `browser_snapshot`：获取无障碍快照
- `browser_click` / `browser_type` / `browser_select_option`：交互操作
- `browser_network_requests` / `browser_console_messages`：诊断
- 等

---

### 3.2 @agent-infra/mcp-server-browser

**安装**：
```bash
npx -y @agent-infra/mcp-server-browser@latest --help
```

**配置**：

```json
{
  "mcpServers": {
    "browser": {
      "command": "npx",
      "args": ["-y", "@agent-infra/mcp-server-browser@latest", "--headless"],
      "env": {},
      "type": "stdio"
    }
  }
}
```

**启动参数**：
- `--headless`：无头模式（不显示浏览器窗口）
- `--browser`：指定浏览器（`chrome`/`firefox`/`webkit`，默认 `chromium`）
- `--port`：远程调试端口（默认 9222）
- `--url`：初始导航 URL

**原理**：直接使用 CDP 协议连接 Chrome，不依赖 Puppeteer。需要在 `~/.claude.json` 中先配置 Chrome 的路径。

---

### 3.3 playwright-mcp（已弃用方向）

**问题根源**：
- 依赖 Chrome 扩展注入 `chrome.debugger`
- Chrome 扩展无法访问 `localhost` WebSocket
- 报错：`ERR_BLOCKED_BY_CLIENT`，扩展 ID `mmlmfjhmonkocbjadbfplnigmagldckm`

**配置（仅供参考，不推荐）**：
```json
{
  "mcpServers": {
    "playwright": {
      "command": "npx",
      "args": ["-y", "playwright-mcp", "--extension-id", "mmlmfjhmonkocbjadbfplnigmagldckm"],
      "type": "stdio"
    }
  }
}
```

**变通（不完美）**：在 `playwright.config.ts` 中添加 `--allow-insecure-localhost` 参数，但只对 Playwright 启动的浏览器生效，不影响用户日常 Chrome。

---

### 3.4 browsermcp.io

**仓库**：`https://github.com/browsermcp/mcp`

**问题**：同样依赖 Chrome 扩展，同样受 localhost 策略限制。

**安装**：
1. 从 Chrome Web Store 安装扩展（或加载解压的 .crx）
2. 配置 MCP server

```json
{
  "mcpServers": {
    "browsermcp": {
      "command": "npx",
      "args": ["-y", "@browsermcp/extension"],
      "type": "stdio"
    }
  }
}
```

---

## 四、架构对比图

```
                    Extension-Relay 架构
                    (playwright-mcp / browsermcp)

  Claude Code                      Chrome 扩展
  ┌──────────┐                    ┌──────────┐
  │ MCP Client│◄──stdio──►│ Relay (npx) │
  └──────────┘                    └────┬─────┘
                                        │ chrome.runtime.connect
                                        │ (WebSocket)
                                        ▼
                                   ┌──────────┐
                                   │ localhost│
                                   │ Relay    │◄── ERR_BLOCKED_BY_CLIENT
                                   └──────────┘
                                        │
                                   chrome.debugger API
                                        │
                                   ┌────▼────┐
                                   │ Chrome  │
                                   │ Browser │
                                   └─────────┘

                    CDP-Direct 架构
                    (chrome-devtools-mcp / @agent-infra/mcp-server-browser)

  Claude Code                      MCP Server
  ┌──────────┐                    ┌──────────┐
  │ MCP Client│◄──stdio──►│  Server    │
  └──────────┘                    └────┬────┘
                                        │ CDP (直接协议)
                                        ▼
                                   ┌──────────┐
                                   │ Chrome   │
                                   │ Browser  │
                                   └─────────┘
                                        ▲
                              --remote-debugging-port
                                        │
                              ┌────────┴────────┐
                              │ 已启动的 Chrome │
                              └────────────────┘
```

---

## 五、工具能力对比

| 工具能力 | chrome-devtools-mcp | agent-infra/mcp-server-browser | playwright-mcp |
|----------|---------------------|-------------------------------|---------------|
| 页面导航 | ✅ | ✅ | ✅ |
| 截图 | ✅ | ✅ | ✅ |
| 执行 JS | ✅ | ✅ | ✅ |
| 无障碍快照 | ✅ | ✅ | ✅ |
| 表单填写 | ✅ | ✅ | ✅ |
| 下拉选择 | ✅ | ✅ | ✅ |
| 文件上传 | ❌ | ✅ | ✅ |
| 多标签管理 | ✅ | ✅ | ✅ |
| 网络请求监控 | ✅ | ✅ | ✅ |
| 控制台日志 | ✅ | ✅ | ✅ |
| Dialog 处理 | ❌ | ✅ | ✅ |
| 拖拽 | ❌ | ✅ | ✅ |
| 浏览器标签操作 | ✅ | ✅ | ✅ |

---

## 六、选型建议

### 推荐方案

**首选：chrome-devtools-mcp**
- Google 官方维护，质量有保障
- 27+ 工具，能力最全面
- 无扩展依赖，不受 localhost 策略限制
- 适合有稳定 Chrome 环境的工作流

**备选：@agent-infra/mcp-server-browser**
- 轻量级实现，启动快
- 支持 `--headless` 无头模式
- 适合 CI/CD 场景或不需要完整工具集时

### 不推荐方案

**避免：playwright-mcp / browsermcp.io**
- 依赖 Chrome 扩展，受 localhost 安全策略限制
- 用户体验差，需要额外配置 Chrome 启动参数
- 架构上存在无法绕过的设计缺陷

---

## 七、常见问题

### Q1: ERR_BLOCKED_BY_CLIENT
**原因**：Chrome 扩展被安全策略阻止访问 localhost WebSocket。

**解决**：
1. **短期**：在 Chrome 快捷方式中添加 `--allow-insecure-localhost` 参数
2. **长期**：切换到 CDP-Direct 方案（chrome-devtools-mcp 或 agent-infra）
3. **企业 Chrome**：联系 IT 部门调整 Chrome 策略（`ExtensionSettings` 白名单）

### Q2: MCP Server 启动后工具不可用
**检查**：
1. `claude code` 会话中是否提示 MCP 服务器连接成功
2. Chrome 是否已在运行并开启 `--remote-debugging-port=9222`
3. MCP server 是否报错（`claude code -d` 查看调试日志）

### Q3: 多个浏览器 MCP 同时配置冲突
**解决**：同一时间只启用一个浏览器 MCP server，避免端口竞争。

---

## 八、快速上手命令

```bash
# 1. 安装 chrome-devtools-mcp
npm install -g chrome-devtools-mcp

# 2. 配置 ~/.claude.json（添加 mcpServers.browserDevtools）

# 3. 启动 Chrome（带远程调试端口）
open -a "Google Chrome" --args --remote-debugging-port=9222

# 4. 重启 Claude Code 或 /claude mcp add browserDevtools

# 5. 验证：/browse 工具是否可用
```

---

## 九、总结

| 维度 | CDP-Direct（推荐） | Extension-Relay（不推荐） |
|------|-------------------|--------------------------|
| 配置复杂度 | 中（需启动 Chrome 加参数） | 高（需安装扩展 + 解决策略限制） |
| 稳定性 | ✅ 高 | ❌ 受 Chrome 策略影响 |
| 工具丰富度 | ✅ 27+ | ✅ 较全 |
| 维护活跃度 | ✅ Google 官方 / 活跃 | ⚠️ 可能停止维护 |
| 企业环境兼容 | ✅ | ❌ 受策略限制 |

**最终建议**：所有新项目统一使用 **chrome-devtools-mcp**，已有的 playwright-mcp 配置可逐步迁移。