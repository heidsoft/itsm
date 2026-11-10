/**
 * Defensive readers for `unknown` payloads.
 *
 * The HTTP client returns `unknown` for domains whose DTOs the CLI does not
 * model in full (problem/release/service-request). The TUI must still render
 * them, but "cast to any and hope" would turn a backend field rename into a
 * blank screen. These helpers narrow once, at the edge, so the components
 * below can stay typed without `as any`.
 */
export type JsonRecord = Record<string, unknown>;
export declare function isRecord(value: unknown): value is JsonRecord;
export declare function asRecord(value: unknown): JsonRecord;
/** Standard list envelope (`{items,total,...}`) or a bare array. */
export declare function asList(value: unknown): {
    items: JsonRecord[];
    total: number;
};
/** First string-valued key among `keys`; numbers are stringified (IDs are commonly numeric). */
export declare function pickString(row: JsonRecord, ...keys: string[]): string | undefined;
export declare function pickNumber(row: JsonRecord, ...keys: string[]): number | undefined;
export declare function pickValue(row: JsonRecord, ...keys: string[]): unknown;
/** Nested object field, e.g. `catalog.name` on a service request. */
export declare function pickNested(row: JsonRecord, key: string, ...innerKeys: string[]): unknown;
