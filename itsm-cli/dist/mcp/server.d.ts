/**
 * MCP server assembly — protocol methods wired to the ITSM command registry.
 *
 * Method coverage is deliberately the full surface a client actually uses:
 * `initialize`, `tools/list`, `tools/call`, `resources/list`+`read`,
 * `prompts/list`+`get`, `ping`, plus tolerant no-ops for the notification and
 * completion methods clients probe. Anything else answers MethodNotFound, which
 * is what the spec requires and what a client can act on.
 */
import { StdioTransport } from './protocol.js';
import type { JsonRpcMessage } from './protocol.js';
/** Newest first. A client asking for an older supported version gets that version echoed. */
export declare const SUPPORTED_PROTOCOL_VERSIONS: readonly ["2025-06-18", "2025-03-26", "2024-11-05"];
export declare const SERVER_INFO: {
    readonly name: "itsm-mcp";
    readonly title: "ITSM（AI-Native ITSM 平台）";
    readonly version: "1.0.0";
};
export declare class McpServer {
    private readonly log;
    private readonly tools;
    private readonly toolIndex;
    private readonly handlers;
    private initialized;
    constructor(log?: (msg: string) => void);
    toolNames(): string[];
    /** Route one already-parsed message. Returns undefined for notifications. */
    dispatch(msg: unknown): Promise<JsonRpcMessage | undefined>;
    private buildHandlers;
    /** A curated tool: its arguments are the command's own parameters plus shaping knobs. */
    private runCurated;
    private runMeta;
}
export interface ServeOptions {
    input?: NodeJS.ReadableStream;
    output?: NodeJS.WritableStream;
    onClose?: () => void;
}
/** Wire the server to stdio (or injected streams in tests) and start reading. */
export declare function serve(opts?: ServeOptions): {
    transport: StdioTransport;
    server: McpServer;
};
