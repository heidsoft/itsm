/**
 * Agent command surface specification.
 *
 * A command is identified by `domain verb` (e.g. `ticket.create`, `bpmn.task-complete`).
 * Every spec carries its usage string and endpoint list so `itsm commands --json`
 * (capability discovery) and the route-drift check can derive everything from
 * the registry — command docs and endpoint snapshots must not be hand-maintained
 * in two places.
 */
import { deriveParams, mergeParams, extractEnums, applyEnums } from './params.js';
const RISK_SKIP = new Set(['read', 'list', 'get', 'view', 'search', 'stats', 'health']);
/** Read-only HTTP methods can never be gated, whatever the verb is named. */
const READ_METHODS = new Set(['GET', 'HEAD', 'OPTIONS']);
/**
 * Derive risk from the verb when a spec doesn't declare it.
 *
 * `methods` is the set of HTTP methods the command's endpoints declare. The verb
 * regex alone mis-classified read-only commands whose names happen to contain a
 * mutating word — `change.rollback-steps` (GET /changes/:id/rollback-steps) was
 * flagged high-risk and demanded --yes just to list a checklist. Method wins.
 */
export function deriveRisk(verb, methods) {
    const onlyReads = methods && methods.size > 0 && [...methods].every((m) => READ_METHODS.has(m));
    if (onlyReads)
        return 'normal';
    if (RISK_SKIP.has(verb))
        return 'normal';
    if (/delete|revoke|terminate|cancel|close|bulk|batch|rollback|replay|archive|install/.test(verb))
        return 'high-risk-write';
    if (/create|update|patch|assign|resolve|approve|reject|submit|complete|claim|start|schedule|publish|send|provision|enable|disable|restore|reopen|escalate|withdraw|pause|resume|unpublish|status|accept|forward|convert/.test(verb))
        return 'write';
    return 'normal';
}
function methodsOf(endpoints) {
    const methods = new Set();
    for (const e of endpoints) {
        const m = e.match(/^([A-Z]+)\s/);
        if (m)
            methods.add(m[1]);
    }
    return methods.size > 0 ? methods : undefined;
}
/** Compact builder for command specs — keeps domain modules declarative. */
export function cmd(name, summary, usage, endpoints, run, risk, meta) {
    const verb = name.split('.').pop() ?? name;
    const derived = deriveParams(usage);
    const enums = extractEnums(`${summary} ${usage}`);
    const params = applyEnums(mergeParams(derived.params, meta?.params), enums);
    return {
        name,
        summary,
        usage,
        endpoints,
        risk: risk ?? deriveRisk(verb, methodsOf(endpoints)),
        examples: meta?.examples,
        input: { params, ...(derived.body ? { body: derived.body } : {}) },
        bodyFields: meta?.bodyFields,
        ...(meta?.denyFlags ? { denyFlags: meta.denyFlags } : {}),
        run,
    };
}
/** One command rendered for agents: what it does, what it needs, what it costs. */
export function describeCommand(spec) {
    const required = spec.input.params.filter((p) => p.required);
    return {
        command: spec.name,
        summary: spec.summary,
        usage: spec.usage,
        risk: spec.risk,
        endpoints: spec.endpoints,
        requiresConfirmation: spec.risk === 'high-risk-write',
        input: {
            params: spec.input.params,
            ...(spec.input.body ? { body: spec.input.body } : {}),
            ...(spec.bodyFields?.length ? { bodyFields: spec.bodyFields } : {}),
            ...(spec.denyFlags ? { deniedFlags: spec.denyFlags } : {}),
            required: required.map((p) => p.name),
        },
        examples: spec.examples ?? [],
    };
}
