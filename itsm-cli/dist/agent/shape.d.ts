/**
 * Output shaping — one implementation shared by the headless runner and the
 * MCP server.
 *
 * Both surfaces face the same problem: a list endpoint can return thousands of
 * items, and every item is billed as context. `--fields` / `--limit` (CLI) and
 * `fields` / `limit` (MCP) must mean exactly the same thing, so the projection
 * lives here once instead of being re-implemented per transport.
 */
export interface ExtractedList {
    items: unknown[];
    total: number;
    isEnvelope: boolean;
}
/** Recognise the backend list envelope ({items,total,page,pageSize,totalPages}) or a bare array. */
export declare function extractList(data: unknown): ExtractedList | undefined;
/** `--fields id,title,assignee.name` — dotted paths nest, missing fields stay explicit nulls. */
export declare function projectFields(item: unknown, fields: string[]): unknown;
export interface ShapingOptions {
    fields?: string[];
    limit?: number;
}
export interface ShapingOutcome {
    /** True when the payload was a recognised list envelope / array. */
    isList: boolean;
    /** Payload after shaping (envelope preserved when it was one). */
    data: unknown;
    /** Shaped items; empty for non-list payloads. */
    items: unknown[];
    /** Counts for list payloads, `shapingSkipped` for non-list payloads when shaping was requested. */
    meta?: Record<string, unknown>;
    skippedReason?: string;
}
export declare function applyShaping(data: unknown, opts: ShapingOptions): ShapingOutcome;
