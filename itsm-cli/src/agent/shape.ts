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
export function extractList(data: unknown): ExtractedList | undefined {
  if (Array.isArray(data)) return { items: data, total: data.length, isEnvelope: false };
  if (data && typeof data === 'object') {
    const obj = data as Record<string, unknown>;
    if (Array.isArray(obj['items'])) {
      const total = typeof obj['total'] === 'number' ? obj['total'] : obj['items'].length;
      return { items: obj['items'], total, isEnvelope: true };
    }
  }
  return undefined;
}

/** `--fields id,title,assignee.name` — dotted paths nest, missing fields stay explicit nulls. */
export function projectFields(item: unknown, fields: string[]): unknown {
  if (!item || typeof item !== 'object' || Array.isArray(item)) return item;
  const src = item as Record<string, unknown>;
  const out: Record<string, unknown> = {};
  for (const path of fields) {
    const segments = path.split('.').filter(Boolean);
    let cursor: unknown = src;
    for (const seg of segments) {
      cursor = cursor && typeof cursor === 'object' ? (cursor as Record<string, unknown>)[seg] : undefined;
    }
    if (segments.length === 1) {
      out[path] = cursor ?? null;
      continue;
    }
    let target = out;
    for (const seg of segments.slice(0, -1)) {
      const next = (target[seg] as Record<string, unknown> | undefined) ?? {};
      target[seg] = next;
      target = next;
    }
    target[segments[segments.length - 1]] = cursor ?? null;
  }
  return out;
}

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

export function applyShaping(data: unknown, opts: ShapingOptions): ShapingOutcome {
  const list = extractList(data);
  const { fields, limit } = opts;

  if (!list) {
    // `--fields`/`--limit` on a non-list payload is a caller mistake, not a
    // silent no-op: say so instead of returning the untrimmed object as if the
    // request had been honoured.
    const shapingRequested = Boolean(fields?.length) || limit !== undefined;
    if (!shapingRequested) return { isList: false, data, items: [] };
    const reason =
      'payload is not a standard list envelope ({items,total,page,pageSize,totalPages}); --fields/--limit do not apply';
    return { isList: false, data, items: [], skippedReason: reason, meta: { shapingSkipped: reason } };
  }

  const items = limit !== undefined ? list.items.slice(0, limit) : list.items;
  const shapedItems = fields?.length ? items.map((item) => projectFields(item, fields)) : items;

  const meta: Record<string, unknown> = {
    returned: shapedItems.length,
    total: list.total,
    ...(shapedItems.length < list.total ? { truncated: true } : {}),
    ...(fields?.length ? { projected: fields } : {}),
  };

  const shaped = list.isEnvelope ? { ...(data as Record<string, unknown>), items: shapedItems } : shapedItems;
  return { isList: true, data: shaped, items: shapedItems, meta };
}
