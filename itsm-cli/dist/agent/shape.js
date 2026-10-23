/**
 * Output shaping — one implementation shared by the headless runner and the
 * MCP server.
 *
 * Both surfaces face the same problem: a list endpoint can return thousands of
 * items, and every item is billed as context. `--fields` / `--limit` (CLI) and
 * `fields` / `limit` (MCP) must mean exactly the same thing, so the projection
 * lives here once instead of being re-implemented per transport.
 */
/** Recognise the backend list envelope ({items,total,page,pageSize,totalPages}) or a bare array. */
export function extractList(data) {
    if (Array.isArray(data))
        return { items: data, total: data.length, isEnvelope: false };
    if (data && typeof data === 'object') {
        const obj = data;
        if (Array.isArray(obj['items'])) {
            const total = typeof obj['total'] === 'number' ? obj['total'] : obj['items'].length;
            return { items: obj['items'], total, isEnvelope: true };
        }
    }
    return undefined;
}
/** `--fields id,title,assignee.name` — dotted paths nest, missing fields stay explicit nulls. */
export function projectFields(item, fields) {
    if (!item || typeof item !== 'object' || Array.isArray(item))
        return item;
    const src = item;
    const out = {};
    for (const path of fields) {
        const segments = path.split('.').filter(Boolean);
        let cursor = src;
        for (const seg of segments) {
            cursor = cursor && typeof cursor === 'object' ? cursor[seg] : undefined;
        }
        if (segments.length === 1) {
            out[path] = cursor ?? null;
            continue;
        }
        let target = out;
        for (const seg of segments.slice(0, -1)) {
            const next = target[seg] ?? {};
            target[seg] = next;
            target = next;
        }
        target[segments[segments.length - 1]] = cursor ?? null;
    }
    return out;
}
export function applyShaping(data, opts) {
    const list = extractList(data);
    const { fields, limit } = opts;
    if (!list) {
        // `--fields`/`--limit` on a non-list payload is a caller mistake, not a
        // silent no-op: say so instead of returning the untrimmed object as if the
        // request had been honoured.
        const shapingRequested = Boolean(fields?.length) || limit !== undefined;
        if (!shapingRequested)
            return { isList: false, data, items: [] };
        const reason = 'payload is not a standard list envelope ({items,total,page,pageSize,totalPages}); --fields/--limit do not apply';
        return { isList: false, data, items: [], skippedReason: reason, meta: { shapingSkipped: reason } };
    }
    const items = limit !== undefined ? list.items.slice(0, limit) : list.items;
    const shapedItems = fields?.length ? items.map((item) => projectFields(item, fields)) : items;
    const meta = {
        returned: shapedItems.length,
        total: list.total,
        ...(shapedItems.length < list.total ? { truncated: true } : {}),
        ...(fields?.length ? { projected: fields } : {}),
    };
    const shaped = list.isEnvelope ? { ...data, items: shapedItems } : shapedItems;
    return { isList: true, data: shaped, items: shapedItems, meta };
}
