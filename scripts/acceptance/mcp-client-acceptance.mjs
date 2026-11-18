#!/usr/bin/env node
// MCP 客户端验收脚本 —— 用官方 @modelcontextprotocol/sdk 验证 ITSM MCP 通道。
//
// 为什么需要这个脚本：
//   MCP 的服务端自测（go test）覆盖的是协议形状，而**真实客户端**还有额外要求：
//   官方 SDK 的 zod schema 会校验 tools/list 响应（例如 outputSchema.type
//   必须是字面量 "object"），不合规时整个工具面加载失败 —— 这类问题服务端
//   的契约测试发现不了（2026-11-18 就是这样抓到 outputSchema 缺陷的）。
//   本脚本用 Claude Desktop / Claude Code 背后的同一套官方 SDK 跑完整交互。
//
// 用法：
//   node scripts/acceptance/mcp-client-acceptance.mjs <endpoint> <token>
//   例：node scripts/acceptance/mcp-client-acceptance.mjs \
//         http://127.0.0.1:8090/api/v1/mcp "$TOKEN"
//
// 前置：
//   npm i @modelcontextprotocol/sdk@1.21.0   （本目录或上级 node_modules）
//   令牌来源：
//     - human：POST /api/v1/auth/login 拿 access token
//     - service：go run ./cmd/service-token-gen -tenant-id N -username svc-probe
//
// 退出码：0 全部通过；1 有未通过项；2 参数/环境错误。

import { createRequire } from 'node:module';
import { existsSync } from 'node:fs';
import path from 'node:path';

const require = createRequire(import.meta.url);

// SDK 可能装在本目录、上级目录或全局解析路径 —— 逐个尝试。
function loadSDK() {
  const candidates = [
    '@modelcontextprotocol/sdk/client/index.js',
    '@modelcontextprotocol/sdk/client/streamableHttp.js',
  ];
  for (const spec of candidates) {
    try {
      return require.resolve(spec);
    } catch (e) {
      // 继续尝试下一个
    }
  }
  return null;
}

const endpoint = process.argv[2];
const token = process.argv[3];

if (!endpoint || !token) {
  console.error('用法: node scripts/acceptance/mcp-client-acceptance.mjs <endpoint> <token>');
  console.error('依赖: npm i @modelcontextprotocol/sdk@1.21.0');
  process.exit(2);
}

const sdkEntry = loadSDK();
if (!sdkEntry) {
  console.error('未找到 @modelcontextprotocol/sdk —— 请先执行：npm i @modelcontextprotocol/sdk@1.21.0');
  process.exit(2);
}

const { Client } = await import('@modelcontextprotocol/sdk/client/index.js');
const { StreamableHTTPClientTransport } = await import('@modelcontextprotocol/sdk/client/streamableHttp.js');

const transport = new StreamableHTTPClientTransport(new URL(endpoint), {
  requestInit: { headers: { Authorization: `Bearer ${token}` } },
});

const client = new Client(
  { name: 'itsm-mcp-client-acceptance', version: '1.0.0' },
  { capabilities: {} },
);

let failures = 0;
function check(label, ok, detail) {
  if (ok) {
    console.log(`  PASS  ${label}${detail ? ' — ' + detail : ''}`);
  } else {
    failures += 1;
    console.log(`  FAIL  ${label}${detail ? ' — ' + detail : ''}`);
  }
}

try {
  console.log('\n[1] connect（initialize 握手与协议版本协商）');
  await client.connect(transport);
  const serverInfo = client.getServerVersion();
  const caps = client.getServerCapabilities();
  check('连接成功并完成 initialize', true);
  check('服务端声明 tools 能力', Boolean(caps && caps.tools),
    caps && caps.tools ? JSON.stringify(caps.tools) : '未声明');
  check('服务端返回 serverInfo', Boolean(serverInfo && serverInfo.name),
    serverInfo ? `${serverInfo.name}@${serverInfo.version}` : '无');

  console.log('\n[2] listTools（工具面须通过 SDK 的 zod schema 校验）');
  const listed = await client.listTools();
  check('tools/list 返回成功', true, `${listed.tools.length} 个工具`);
  check('工具面非空', listed.tools.length > 0);

  const badInput = listed.tools.filter((t) => !t.inputSchema || t.inputSchema.type !== 'object');
  check('每个工具都有 object 型 inputSchema', badInput.length === 0,
    badInput.length ? `不合规: ${badInput.map((t) => t.name).join(', ')}` : `${listed.tools.length}/${listed.tools.length}`);

  // outputSchema.type 必须是字面量 "object"（SDK zod 要求），非 object 型会让
  // 整个 tools/list 响应被客户端拒绝 —— 不是单个工具失效，是通道不可用。
  const withOutput = listed.tools.filter((t) => t.outputSchema !== undefined);
  const badOutput = withOutput.filter((t) => t.outputSchema.type !== 'object');
  check('每个 outputSchema.type 均为 object', badOutput.length === 0,
    badOutput.length ? `不合规: ${badOutput.map((t) => t.name).join(', ')}` : `${withOutput.length} 个带 outputSchema`);

  console.log('\n[3] callTool（真实执行一个只读工具）');
  const readOnly = listed.tools.find((t) => t.annotations && t.annotations.readOnlyHint === true);
  if (!readOnly) {
    check('存在只读工具可供调用', false, '工具面里没有 readOnlyHint=true 的工具');
  } else {
    const result = await client.callTool({ name: readOnly.name, arguments: {} });
    check('tools/call 协议层成功', true, `工具 ${readOnly.name}`);
    check('结果含 content 数组', Array.isArray(result.content),
      Array.isArray(result.content) ? `${result.content.length} 项` : '缺失');
    const text = Array.isArray(result.content) && result.content[0] ? (result.content[0].text || '') : '';
    if (result.isError === true) {
      // 主体未被运行期策略放行该工具码时，权限拒绝是**正确行为**：
      // service/agent 主体默认零权限（allowedPrincipals 恒为 ["human"]）。
      check('未放行时以 isError 表达权限拒绝', /权限|permission|denied/i.test(text),
        text.slice(0, 100));
      console.log('  提示: 当前主体未被运行期策略放行该工具码（符合 AI/服务主体零权限设计）');
    } else {
      check('放行后调用成功', true, text.slice(0, 100).replace(/\n/g, ' '));
      check('结构化内容为对象', result.structuredContent !== undefined
        && typeof result.structuredContent === 'object'
        && !Array.isArray(result.structuredContent),
        result.structuredContent === undefined ? '无 structuredContent'
          : JSON.stringify(result.structuredContent).slice(0, 120));
    }
  }

  console.log('\n[4] 未知工具（错误语义）');
  try {
    const r = await client.callTool({ name: '__definitely_not_a_real_tool__', arguments: {} });
    const text = Array.isArray(r.content) && r.content[0] ? (r.content[0].text || '') : '';
    check('未知工具未被当作成功', r.isError === true || text.length > 0,
      r.isError === true ? '以 isError 表达' : text.slice(0, 80));
  } catch (err) {
    check('未知工具按协议报错', true, String(err.message).slice(0, 120));
  }

  console.log('\n[5] ping（存活探测）');
  try {
    await client.ping();
    check('ping 成功', true);
  } catch (err) {
    check('ping 失败', false, String(err.message).slice(0, 120));
  }

  await client.close();
  console.log(`\n结论：${failures === 0 ? '全部通过 —— 真实 MCP 客户端可用' : failures + ' 项未通过'}`);
  process.exit(failures === 0 ? 0 : 1);
} catch (err) {
  console.error('\n致命错误:', err && err.message ? err.message : err);
  if (err && err.stack) console.error(err.stack.split('\n').slice(0, 6).join('\n'));
  process.exit(1);
}
