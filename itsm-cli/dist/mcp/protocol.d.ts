/**
 * Minimal, dependency-free JSON-RPC 2.0 over stdio — the MCP wire format.
 *
 * Why not `@modelcontextprotocol/sdk`: `itsm-cli/dist` is committed and runs
 * from a fresh clone with no `npm install` (the agent surface deliberately has
 * zero runtime dependencies). Adding the SDK would make the MCP server the only
 * surface that needs a node_modules tree, and every protocol change would then
 * ride on a third-party release. The protocol itself is small and stable:
 * newline-delimited JSON-RPC with a handful of methods.
 *
 * Framing: the MCP stdio transport is newline-delimited JSON. Some hosts still
 * emit LSP-style `Content-Length: N\r\n\r\n{...}` headers, so the reader accepts
 * both instead of hanging on the first edit to the client.
 */
export type JsonRpcId = string | number | null;
export interface JsonRpcRequest {
    jsonrpc: '2.0';
    id?: JsonRpcId;
    method: string;
    params?: Record<string, unknown>;
}
export interface JsonRpcSuccess {
    jsonrpc: '2.0';
    id: JsonRpcId;
    result: unknown;
}
export interface JsonRpcFailure {
    jsonrpc: '2.0';
    id: JsonRpcId;
    error: {
        code: number;
        message: string;
        data?: unknown;
    };
}
export interface JsonRpcNotification {
    jsonrpc: '2.0';
    method: string;
    params?: Record<string, unknown>;
}
export type JsonRpcMessage = JsonRpcSuccess | JsonRpcFailure | JsonRpcNotification;
/** JSON-RPC 2.0 reserved codes (plus the MCP-recommended resource-not-found). */
export declare const RPC: {
    readonly ParseError: -32700;
    readonly InvalidRequest: -32600;
    readonly MethodNotFound: -32601;
    readonly InvalidParams: -32602;
    readonly InternalError: -32603;
    /** MCP: the requested resource URI does not exist. */
    readonly ResourceNotFound: -32002;
};
export declare class RpcError extends Error {
    readonly code: number;
    readonly data?: unknown;
    constructor(code: number, message: string, data?: unknown);
}
export interface TransportHandlers {
    onMessage: (msg: unknown) => void;
    /** Fatal reader problem (stdin closed or unparseable framing). */
    onClose?: (reason: string) => void;
}
/**
 * Reads one JSON-RPC message at a time from a byte stream and writes
 * serialised messages back. stdout belongs to the protocol: every diagnostic
 * goes to stderr, and nothing else may write to stdout.
 */
export declare class StdioTransport {
    private readonly input;
    private readonly output;
    private readonly handlers;
    private buffer;
    /** Input ended: stop parsing, but keep writing so in-flight calls still answer. */
    private ended;
    constructor(input: NodeJS.ReadableStream, output: NodeJS.WritableStream, handlers: TransportHandlers);
    start(): void;
    get isEnded(): boolean;
    /**
     * Writing is not gated on `ended`: when the client closes stdin right after
     * sending its batch, responses already in flight must still be delivered —
     * otherwise a short-lived session silently loses results it asked for.
     */
    send(message: JsonRpcMessage): void;
    close(reason: string): void;
    private pump;
    private isContentLengthFramed;
    private emit;
}
export declare function isRequest(msg: unknown): msg is JsonRpcRequest;
export declare function hasId(msg: JsonRpcRequest): boolean;
