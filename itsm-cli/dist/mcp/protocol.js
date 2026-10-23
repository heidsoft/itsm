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
/** JSON-RPC 2.0 reserved codes (plus the MCP-recommended resource-not-found). */
export const RPC = {
    ParseError: -32700,
    InvalidRequest: -32600,
    MethodNotFound: -32601,
    InvalidParams: -32602,
    InternalError: -32603,
    /** MCP: the requested resource URI does not exist. */
    ResourceNotFound: -32002,
};
export class RpcError extends Error {
    constructor(code, message, data) {
        super(message);
        this.name = 'RpcError';
        this.code = code;
        this.data = data;
    }
}
/**
 * Reads one JSON-RPC message at a time from a byte stream and writes
 * serialised messages back. stdout belongs to the protocol: every diagnostic
 * goes to stderr, and nothing else may write to stdout.
 */
export class StdioTransport {
    constructor(input, output, handlers) {
        this.input = input;
        this.output = output;
        this.handlers = handlers;
        this.buffer = Buffer.alloc(0);
        /** Input ended: stop parsing, but keep writing so in-flight calls still answer. */
        this.ended = false;
    }
    start() {
        this.input.on('data', (chunk) => {
            const buf = typeof chunk === 'string' ? Buffer.from(chunk, 'utf8') : chunk;
            this.buffer = this.buffer.length ? Buffer.concat([this.buffer, buf]) : buf;
            this.pump();
        });
        this.input.on('end', () => this.close('stdin closed'));
        this.input.on('error', (err) => this.close(`stdin error: ${err.message}`));
    }
    get isEnded() {
        return this.ended;
    }
    /**
     * Writing is not gated on `ended`: when the client closes stdin right after
     * sending its batch, responses already in flight must still be delivered —
     * otherwise a short-lived session silently loses results it asked for.
     */
    send(message) {
        try {
            this.output.write(`${JSON.stringify(message)}\n`);
        }
        catch {
            // The peer hung up mid-write; nothing left to deliver and no one to tell.
        }
    }
    close(reason) {
        if (this.ended)
            return;
        this.ended = true;
        this.handlers.onClose?.(reason);
    }
    pump() {
        if (this.ended)
            return;
        for (;;) {
            // Tolerate leading blank lines / CRLF between messages.
            let skip = 0;
            while (skip < this.buffer.length && (this.buffer[skip] === 0x0a || this.buffer[skip] === 0x0d || this.buffer[skip] === 0x20))
                skip++;
            if (skip)
                this.buffer = this.buffer.subarray(skip);
            if (!this.buffer.length)
                return;
            if (this.isContentLengthFramed()) {
                const headerEnd = this.buffer.indexOf('\r\n\r\n');
                if (headerEnd < 0)
                    return;
                const header = this.buffer.subarray(0, headerEnd).toString('latin1');
                const length = Number(/content-length:\s*(\d+)/i.exec(header)?.[1]);
                const bodyStart = headerEnd + 4;
                if (!Number.isFinite(length) || length < 0) {
                    this.close('malformed Content-Length header');
                    return;
                }
                if (this.buffer.length < bodyStart + length)
                    return;
                const body = this.buffer.subarray(bodyStart, bodyStart + length).toString('utf8');
                this.buffer = this.buffer.subarray(bodyStart + length);
                this.emit(body);
                continue;
            }
            const newline = this.buffer.indexOf('\n');
            if (newline < 0)
                return;
            const line = this.buffer.subarray(0, newline).toString('utf8');
            this.buffer = this.buffer.subarray(newline + 1);
            if (line.trim())
                this.emit(line);
        }
    }
    isContentLengthFramed() {
        const head = this.buffer.subarray(0, Math.min(14, this.buffer.length)).toString('latin1').toLowerCase();
        return head.startsWith('content-length');
    }
    emit(raw) {
        let parsed;
        try {
            parsed = JSON.parse(raw);
        }
        catch (err) {
            // A single malformed frame must not take the server down: report the
            // parse error with a null id (per JSON-RPC) and keep reading.
            this.send({
                jsonrpc: '2.0',
                id: null,
                error: { code: RPC.ParseError, message: `Invalid JSON: ${err instanceof Error ? err.message : String(err)}` },
            });
            return;
        }
        this.handlers.onMessage(parsed);
    }
}
export function isRequest(msg) {
    return Boolean(msg) && typeof msg === 'object' && typeof msg.method === 'string';
}
export function hasId(msg) {
    return Object.prototype.hasOwnProperty.call(msg, 'id') && msg.id !== undefined;
}
