import { stringFlag, numberFlag, booleanFlag, jsonFlag, requirePositional, } from './args.js';
import { notAuthenticatedError, usageError } from '../lib/errors.js';
import { loadCredentials } from '../lib/credentials.js';
/** Shared helpers used by every domain command module. */
export { stringFlag, numberFlag, booleanFlag, jsonFlag, jsonFlag as bodyFlag, requirePositional };
function asArgContext(args) {
    return 'flags' in args ? args : { positional: [], flags: args };
}
export function withPagination(argsLike) {
    const args = asArgContext(argsLike);
    const out = {};
    const page = numberFlag(args.flags, 'page');
    const pageSize = numberFlag(args.flags, 'pageSize');
    if (page !== undefined)
        out['page'] = page;
    if (pageSize !== undefined)
        out['pageSize'] = pageSize;
    return out;
}
export function optionalString(argsLike, ...names) {
    return stringFlag(asArgContext(argsLike).flags, ...names);
}
export function requiredString(argsLike, name, usage) {
    const value = stringFlag(asArgContext(argsLike).flags, name);
    if (!value) {
        throw usageError(`Missing required flag --${name}. Usage: ${usage}`);
    }
    return value;
}
export function optionalBool(argsLike, ...names) {
    return booleanFlag(asArgContext(argsLike).flags, ...names);
}
export function optionalNumber(argsLike, ...names) {
    return numberFlag(asArgContext(argsLike).flags, ...names);
}
/** Inline JSON body (--data '{"key":"value"}'); undefined when absent. */
export function body(argsLike, names = ['data', 'payload']) {
    return jsonFlag(asArgContext(argsLike).flags, names);
}
/** Standard pagination query from a list envelope response — pass through untouched. */
export function mergeQuery(...parts) {
    const out = {};
    for (const part of parts) {
        for (const [k, v] of Object.entries(part)) {
            if (v !== undefined)
                out[k] = v;
        }
    }
    return out;
}
export function assertLoggedIn() {
    const token = (process.env['ITSM_TOKEN'] || '').trim() || loadCredentials()?.token;
    if (!token)
        throw notAuthenticatedError();
}
