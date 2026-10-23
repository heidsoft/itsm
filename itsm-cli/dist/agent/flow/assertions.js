/**
 * Read a dotted path out of an unknown JSON value.
 * Supports array indices (`items.0.id`) and is deliberately unforgiving:
 * a missing key returns `undefined` so assertions can say "missing" instead of
 * silently passing on a wrongly-typed payload.
 */
export function readPath(value, path) {
    let cursor = value;
    for (const segment of path.split('.')) {
        if (cursor === null || cursor === undefined)
            return undefined;
        if (Array.isArray(cursor)) {
            const index = Number(segment);
            if (!Number.isInteger(index))
                return undefined;
            cursor = cursor[index];
            continue;
        }
        if (typeof cursor !== 'object')
            return undefined;
        cursor = cursor[segment];
    }
    return cursor;
}
function describe(value) {
    if (value === undefined)
        return 'undefined';
    try {
        const text = JSON.stringify(value);
        return text.length > 120 ? `${text.slice(0, 117)}...` : text;
    }
    catch {
        return String(value);
    }
}
export function assertion(check, ok, expected, actual) {
    return { check, ok, ...(expected !== undefined ? { expected } : {}), ...(actual !== undefined ? { actual } : {}) };
}
/** Turn declarative expectations into named, evidence-bearing assertions. */
export function evaluateExpectations(expectations, response) {
    return expectations.map((exp) => {
        const actual = readPath(response, exp.path);
        const label = exp.path;
        if ('exists' in exp) {
            const ok = exp.exists ? actual !== undefined && actual !== null : actual === undefined || actual === null;
            return assertion(`${label} ${exp.exists ? 'exists' : 'is absent'}`, ok, exp.exists ? 'present' : 'absent', actual);
        }
        if ('eq' in exp && exp.eq !== undefined) {
            return assertion(`${label} == ${describe(exp.eq)}`, Object.is(actual, exp.eq) || JSON.stringify(actual) === JSON.stringify(exp.eq), exp.eq, actual);
        }
        if ('ne' in exp && exp.ne !== undefined) {
            return assertion(`${label} != ${describe(exp.ne)}`, JSON.stringify(actual) !== JSON.stringify(exp.ne), `not ${describe(exp.ne)}`, actual);
        }
        if (exp.in) {
            return assertion(`${label} in ${describe(exp.in)}`, exp.in.some((candidate) => JSON.stringify(candidate) === JSON.stringify(actual)), exp.in, actual);
        }
        if (exp.truthy !== undefined) {
            return assertion(`${label} is ${exp.truthy ? 'truthy' : 'falsy'}`, exp.truthy ? Boolean(actual) : !actual, exp.truthy, actual);
        }
        if (exp.falsy !== undefined) {
            return assertion(`${label} is ${exp.falsy ? 'falsy' : 'truthy'}`, exp.falsy ? !actual : Boolean(actual), !exp.falsy, actual);
        }
        if (exp.contains !== undefined) {
            const haystack = typeof actual === 'string' ? actual : JSON.stringify(actual ?? '');
            return assertion(`${label} contains ${describe(exp.contains)}`, haystack.includes(exp.contains), exp.contains, actual);
        }
        if (exp.gte !== undefined) {
            return assertion(`${label} >= ${exp.gte}`, typeof actual === 'number' && actual >= exp.gte, exp.gte, actual);
        }
        if (exp.lte !== undefined) {
            return assertion(`${label} <= ${exp.lte}`, typeof actual === 'number' && actual <= exp.lte, exp.lte, actual);
        }
        return assertion(`${label} (no operator declared)`, false, 'an operator', actual);
    });
}
/** Collection helper used by verify() hooks: does any item match `field === value`? */
export function listContains(response, field, value) {
    const items = Array.isArray(response)
        ? response
        : Array.isArray(response?.items)
            ? response.items
            : [];
    return items.some((item) => JSON.stringify(readPath(item, field)) === JSON.stringify(value));
}
/** Collection helper: how many items match? */
export function countMatching(response, predicate) {
    const items = Array.isArray(response)
        ? response
        : Array.isArray(response?.items)
            ? response.items
            : [];
    return items.filter(predicate).length;
}
