/**
 * MCP server assembly — protocol methods wired to the ITSM command registry.
 *
 * Method coverage is deliberately the full surface a client actually uses:
 * `initialize`, `tools/list`, `tools/call`, `resources/list`+`read`,
 * `prompts/list`+`get`, `ping`, plus tolerant no-ops for the notification and
 * completion methods clients probe. Anything else answers MethodNotFound, which
 * is what the spec requires and what a client can act on.
 */

import { RPC, RpcError, StdioTransport, isRequest, hasId } from './protocol.js';
import type { JsonRpcId, JsonRpcMessage, JsonRpcRequest } from './protocol.js';
import { buildTools, describeTool, discoverTools } from './catalog.js';
import type { McpTool } from './catalog.js';
import { callCommand } from './invoke.js';
import type { CallResult } from './invoke.js';
import { listResources, readResource } from './resources.js';
import { PROMPTS, getPrompt } from './prompts.js';

/** Newest first. A client asking for an older supported version gets that version echoed. */
export const SUPPORTED_PROTOCOL_VERSIONS = ['2025-06-18', '2025-03-26', '2024-11-05'] as const;

export const SERVER_INFO = {
  name: 'itsm-mcp',
  title: 'ITSM（AI-Native ITSM 平台）',
  version: '1.0.0',
} as const;

const INSTRUCTIONS = [
  '这是 ITSM 平台的 MCP 连接：可以查询与操作工单、事件、问题、变更、发布、服务请求、CMDB、知识库、SLA 与 BPMN 工作流。',
  '',
  '调用策略：',
  '· 常见对象有精选工具（itsm_ticket_list / itsm_ticket_get / itsm_cmdb_impact / itsm_change_approvals 等），直接调用即可。',
  '· 精选工具不覆盖的动作：先 itsm_discover 检索 → itsm_describe 取入参契约 → itsm_call 执行。不要猜参数名。',
  '· 读操作可自由调用。写操作前先说明将要发生什么；高风险写（删除/终止/回滚/批量）会返回 confirmation_required，必须先获得用户同意再用 confirm=true 重试，且不得自动重试。',
  '· 列表默认只回 50 条，meta 里有 total 与 truncated；需要更多用 limit/pageSize 分页取，不要一次性拉全量。',
  '· 需要理解产品边界（路线图、变更记录、API 契约、架构）时读 itsm://docs/* 资源，再决定改什么。',
].join('\n');

type Handler = (req: JsonRpcRequest) => Promise<unknown>;

export class McpServer {
  private readonly tools: McpTool[];
  private readonly toolIndex: Map<string, McpTool>;
  private readonly handlers: Map<string, Handler>;
  private initialized = false;

  constructor(private readonly log: (msg: string) => void = () => undefined) {
    this.tools = buildTools();
    this.toolIndex = new Map(this.tools.map((t) => [t.name, t]));
    this.handlers = this.buildHandlers();
  }

  toolNames(): string[] {
    return this.tools.map((t) => t.name);
  }

  /** Route one already-parsed message. Returns undefined for notifications. */
  async dispatch(msg: unknown): Promise<JsonRpcMessage | undefined> {
    if (!isRequest(msg)) {
      return errorResponse(null, RPC.InvalidRequest, 'Not a JSON-RPC request');
    }
    const req = msg as JsonRpcRequest;
    const id: JsonRpcId = hasId(req) ? (req.id as JsonRpcId) : null;
    const isNotification = !hasId(req);
    const handler = this.handlers.get(req.method);

    if (!handler) {
      if (isNotification) return undefined;
      return errorResponse(id, RPC.MethodNotFound, `Method not found: ${req.method}`);
    }

    try {
      const result = await handler(req);
      if (isNotification) return undefined;
      return { jsonrpc: '2.0', id, result: result ?? {} };
    } catch (err) {
      if (isNotification) {
        this.log(`notification ${req.method} failed: ${err instanceof Error ? err.message : String(err)}`);
        return undefined;
      }
      if (err instanceof RpcError) return errorResponse(id, err.code, err.message, err.data);
      return errorResponse(id, RPC.InternalError, err instanceof Error ? err.message : String(err));
    }
  }

  private buildHandlers(): Map<string, Handler> {
    const map = new Map<string, Handler>();

    map.set('initialize', async (req) => {
      const requested = String(req.params?.['protocolVersion'] ?? '');
      const protocolVersion = (SUPPORTED_PROTOCOL_VERSIONS as readonly string[]).includes(requested)
        ? requested
        : SUPPORTED_PROTOCOL_VERSIONS[0];
      this.initialized = true;
      this.log(`initialize: client protocol=${requested || '(none)'} → ${protocolVersion}, tools=${this.tools.length}`);
      return {
        protocolVersion,
        capabilities: {
          tools: { listChanged: false },
          resources: { subscribe: false, listChanged: false },
          prompts: { listChanged: false },
          logging: {},
        },
        serverInfo: SERVER_INFO,
        instructions: INSTRUCTIONS,
      };
    });

    // Notifications the client sends after initialize (and periodically).
    for (const method of ['notifications/initialized', 'notifications/cancelled', 'notifications/roots/list_changed', 'notifications/progress']) {
      map.set(method, async () => ({}));
    }

    map.set('ping', async () => ({}));
    map.set('logging/setLevel', async () => ({}));

    map.set('tools/list', async () => ({ tools: this.tools }));

    map.set('tools/call', async (req) => {
      const name = req.params?.['name'];
      if (typeof name !== 'string' || !name) throw new RpcError(RPC.InvalidParams, 'tools/call 缺少 name');
      const rawArgs = req.params?.['arguments'];
      const args = (rawArgs && typeof rawArgs === 'object' ? rawArgs : {}) as Record<string, unknown>;

      const tool = this.toolIndex.get(name);
      if (!tool) {
        throw new RpcError(
          RPC.InvalidParams,
          `未知工具 '${name}'。可用工具：${this.tools.map((t) => t.name).join(', ')}。任意命令请走 itsm_call。`
        );
      }

      const result = tool.command ? await this.runCurated(tool, args) : await this.runMeta(tool.name, args);
      return toToolResult(result);
    });

    map.set('resources/list', async () => ({ resources: listResources() }));
    map.set('resources/templates/list', async () => ({ resourceTemplates: [] }));

    map.set('resources/read', async (req) => {
      const uri = req.params?.['uri'];
      if (typeof uri !== 'string' || !uri) throw new RpcError(RPC.InvalidParams, 'resources/read 缺少 uri');
      try {
        const content = readResource(uri);
        return { contents: [{ uri: content.uri, mimeType: content.mimeType, text: content.text }] };
      } catch (err) {
        throw new RpcError(RPC.ResourceNotFound, err instanceof Error ? err.message : String(err), { uri });
      }
    });

    map.set('prompts/list', async () => ({ prompts: PROMPTS }));
    map.set('prompts/get', async (req) => {
      const name = req.params?.['name'];
      if (typeof name !== 'string' || !name) throw new RpcError(RPC.InvalidParams, 'prompts/get 缺少 name');
      const rawArgs = req.params?.['arguments'];
      const args = (rawArgs && typeof rawArgs === 'object' ? rawArgs : {}) as Record<string, string>;
      try {
        return getPrompt(name, args);
      } catch (err) {
        throw new RpcError(RPC.InvalidParams, err instanceof Error ? err.message : String(err));
      }
    });

    // We publish no completions; answering empty is friendlier than MethodNotFound.
    map.set('completion/complete', async () => ({ completion: { values: [], total: 0, hasMore: false } }));

    return map;
  }

  /** A curated tool: its arguments are the command's own parameters plus shaping knobs. */
  private async runCurated(tool: McpTool, args: Record<string, unknown>): Promise<CallResult> {
    const { limit, fields, ...params } = args as { limit?: number; fields?: string[] } & Record<string, unknown>;
    const shaping = {
      ...(typeof limit === 'number' ? { limit } : {}),
      ...(Array.isArray(fields) ? { fields: fields.map(String) } : {}),
    };
    // Tool-level defaults first, then the caller's own values: an explicit
    // argument must always win over what the tool assumed on the caller's behalf.
    return callCommand(tool.command as string, { ...(tool.defaults ?? {}), ...params }, shaping);
  }

  private async runMeta(name: string, args: Record<string, unknown>): Promise<CallResult> {
    switch (name) {
      case 'itsm_discover': {
        const query = typeof args['query'] === 'string' ? args['query'] : undefined;
        const domain = typeof args['domain'] === 'string' ? args['domain'] : undefined;
        const rawLimit = Number(args['limit']);
        const limit = Number.isFinite(rawLimit) && rawLimit > 0 ? Math.min(rawLimit, 100) : 25;
        const payload = discoverTools(query, domain, limit);
        return { isError: false, payload, structured: payload };
      }
      case 'itsm_describe': {
        const command = args['command'];
        if (typeof command !== 'string' || !command.trim()) {
          throw new RpcError(RPC.InvalidParams, 'itsm_describe 需要 command，例如 ticket.create');
        }
        try {
          const payload = describeTool(command);
          return { isError: false, payload, structured: payload };
        } catch (err) {
          const message = err instanceof Error ? err.message : String(err);
          const payload = { ok: false, error: { kind: 'usage', message, retryable: false } };
          return { isError: true, payload, structured: payload };
        }
      }
      case 'itsm_call': {
        const command = args['command'];
        if (typeof command !== 'string' || !command.trim()) {
          throw new RpcError(RPC.InvalidParams, 'itsm_call 需要 command，例如 ticket.tags-add');
        }
        const rawParams = args['params'];
        const params = (rawParams && typeof rawParams === 'object' ? rawParams : {}) as Record<string, unknown>;
        const rawLimit = Number(args['limit']);
        const fields = Array.isArray(args['fields']) ? (args['fields'] as unknown[]).map(String) : undefined;
        return callCommand(command, params, {
          confirm: args['confirm'] === true,
          preview: args['preview'] === true,
          ...(Number.isFinite(rawLimit) && rawLimit > 0 ? { limit: Math.floor(rawLimit) } : {}),
          ...(fields?.length ? { fields } : {}),
        });
      }
      default:
        throw new RpcError(RPC.MethodNotFound, `Unhandled meta tool ${name}`);
    }
  }
}

function toToolResult(result: CallResult): Record<string, unknown> {
  return {
    content: [{ type: 'text', text: stringify(result.payload) }],
    structuredContent: result.structured,
    isError: result.isError,
  };
}

/** Compact the JSON we hand to the model: no indentation, but stable and readable. */
function stringify(payload: unknown): string {
  try {
    return JSON.stringify(payload);
  } catch {
    return JSON.stringify({ ok: false, error: { kind: 'internal', message: 'payload not serialisable' } });
  }
}

function errorResponse(id: JsonRpcId, code: number, message: string, data?: unknown): JsonRpcMessage {
  return { jsonrpc: '2.0', id, error: { code, message, ...(data === undefined ? {} : { data }) } };
}

export interface ServeOptions {
  input?: NodeJS.ReadableStream;
  output?: NodeJS.WritableStream;
  onClose?: () => void;
}

/** Wire the server to stdio (or injected streams in tests) and start reading. */
export function serve(opts: ServeOptions = {}): { transport: StdioTransport; server: McpServer } {
  const debug = process.env['ITSM_MCP_DEBUG'] === '1';
  const log = (msg: string): void => {
    if (debug) process.stderr.write(`[itsm-mcp] ${msg}\n`);
  };

  // stdout belongs to the protocol: reroute every console channel to stderr so a
  // stray library log can never corrupt a JSON-RPC frame.
  const toStderr = (...args: unknown[]): void => {
    const line = args.map((a) => (typeof a === 'string' ? a : JSON.stringify(a))).join(' ');
    process.stderr.write(`${line}\n`);
  };
  console.log = toStderr;
  console.info = toStderr;
  console.debug = toStderr;

  const server = new McpServer(log);
  const inflight = new Set<Promise<void>>();

  /**
   * stdin closing is not the same as "work is done": a client may send a batch
   * and close the pipe immediately. Answer every accepted request first, then
   * let the process exit — otherwise the last calls are silently lost.
   */
  const drain = async (): Promise<void> => {
    const deadline = Date.now() + 5000;
    while (inflight.size && Date.now() < deadline) {
      await new Promise((resolve) => setTimeout(resolve, 20));
    }
    if (inflight.size) log(`shutdown with ${inflight.size} request(s) still running`);
    opts.onClose?.();
  };

  const transport = new StdioTransport(opts.input ?? process.stdin, opts.output ?? process.stdout, {
    onMessage: (msg) => {
      const task = server
        .dispatch(msg)
        .then((response) => {
          if (response) transport.send(response);
        })
        .catch((err: unknown) => {
          transport.send(errorResponse(null, RPC.InternalError, err instanceof Error ? err.message : String(err)));
        });
      inflight.add(task);
      void task.then(() => inflight.delete(task));
    },
    onClose: (reason) => {
      log(`transport closed: ${reason}`);
      void drain();
    },
  });

  transport.start();
  log(`ready: ${server.toolNames().length} tools, protocol ${SUPPORTED_PROTOCOL_VERSIONS[0]}`);
  return { transport, server };
}
