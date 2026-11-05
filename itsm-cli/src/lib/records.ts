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

export function isRecord(value: unknown): value is JsonRecord {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export function asRecord(value: unknown): JsonRecord {
  return isRecord(value) ? value : {};
}

/** Standard list envelope (`{items,total,...}`) or a bare array. */
export function asList(value: unknown): { items: JsonRecord[]; total: number } {
  if (Array.isArray(value)) {
    const items = value.filter(isRecord);
    return { items, total: items.length };
  }
  if (isRecord(value) && Array.isArray(value['items'])) {
    const items = value['items'].filter(isRecord);
    const total = typeof value['total'] === 'number' ? value['total'] : items.length;
    return { items, total };
  }
  return { items: [], total: 0 };
}

/** First string-valued key among `keys`; numbers are stringified (IDs are commonly numeric). */
export function pickString(row: JsonRecord, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = row[key];
    if (typeof value === 'string' && value !== '') return value;
    if (typeof value === 'number' && Number.isFinite(value)) return String(value);
  }
  return undefined;
}

export function pickNumber(row: JsonRecord, ...keys: string[]): number | undefined {
  for (const key of keys) {
    const value = row[key];
    if (typeof value === 'number' && Number.isFinite(value)) return value;
    if (typeof value === 'string' && value.trim() !== '' && Number.isFinite(Number(value))) return Number(value);
  }
  return undefined;
}

export function pickValue(row: JsonRecord, ...keys: string[]): unknown {
  for (const key of keys) {
    if (row[key] !== undefined && row[key] !== null && row[key] !== '') return row[key];
  }
  return undefined;
}

/** Nested object field, e.g. `catalog.name` on a service request. */
export function pickNested(row: JsonRecord, key: string, ...innerKeys: string[]): unknown {
  const child = row[key];
  return isRecord(child) ? pickValue(child, ...innerKeys) : undefined;
}
