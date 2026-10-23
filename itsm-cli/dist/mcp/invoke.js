/**
 * The single execution gateway for MCP tool calls.
 *
 * Every path — curated tool, `itsm_call`, future resources — funnels through
 * `callCommand`, so the invariants are enforced once:
 *
 *   · the command exists in the CLI registry (no second endpoint table),
 *   · arguments are validated against the declared contract before any request,
 *   · high-risk writes never execute without an explicit `confirm`,
 *   · failures keep their classification (`forbidden` ≠ `not_found`) and hint,
 *   · list payloads are bounded so one call cannot flood the model's context.
 */
import { applyShaping } from '../agent/shape.js';
import { addFailureHint } from '../agent/hints.js';
import { classifyError, usageError } from '../lib/errors.js';
import { getAutoLoginFailure, hydrateAgentToken, resolveRuntimeConfig } from '../lib/runtime.js';
import { configureBaseURL } from '../lib/api-client.js';
import { resolveCommand, specIndex, suggestCommand } from './catalog.js';
/** Default cap for list payloads. `total`/`truncated` in meta keep the whole picture visible. */
const DEFAULT_LIST_LIMIT = 50;
let authReady;
/**
 * Lazy auth: the MCP server must answer `initialize` instantly even when the
 * backend is down or credentials are wrong — hydration (which may perform one
 * login round-trip) happens on the first real call, once per process.
 */
function ensureAuth() {
    if (!authReady) {
        const baseUrl = (process.env['ITSM_BASE_URL'] || '').trim() || undefined;
        configureBaseURL(baseUrl);
        authReady = hydrateAgentToken(resolveRuntimeConfig(baseUrl ? { baseUrl } : {})).catch(() => undefined);
    }
    return authReady;
}
function asCsv(value) {
    if (Array.isArray(value))
        return value.map((v) => String(v)).join(',');
    return String(value);
}
/**
 * Query knobs every list endpoint understands but which not every command
 * declares in its usage string (`ticket list [--status s] [--page N]` omits
 * pageSize). Accepting them here keeps the MCP schema honest without inventing
 * a second parameter table.
 */
const TRANSPORT_FLAGS = new Set(['page', 'pageSize']);
/**
 * Map MCP arguments onto the CLI's (positional, flags) shape using the
 * command's own declared parameters — the only place that knows what an
 * argument is called and whether it takes a value.
 */
export function buildInvocation(spec, params) {
    const allowed = new Map(spec.input.params.map((p) => [p.name, p]));
    const unknown = Object.keys(params).filter((k) => !allowed.has(k) && !TRANSPORT_FLAGS.has(k));
    if (unknown.length) {
        const known = [...allowed.keys()].join(', ') || '（无参数）';
        throw usageError(`命令 ${spec.name} 不接受参数 ${unknown.join(', ')}。可用参数：${known}。用 itsm_describe 查看完整契约。`);
    }
    // `--data <json>` is an alternative input form: when it is supplied, the
    // positional/flag requirements of the short form no longer apply.
    const bodyProvided = Boolean(spec.input.body) && params['data'] !== undefined && params['data'] !== null;
    const positional = [];
    const flags = {};
    const missing = [];
    for (const p of spec.input.params) {
        const value = params[p.name];
        const absent = value === undefined || value === null || value === '';
        if (absent) {
            if (p.required && !bodyProvided)
                missing.push(p.name);
            continue;
        }
        if (p.kind === 'positional') {
            positional.push(asCsv(value));
            continue;
        }
        switch (p.type) {
            case 'json':
                flags[p.name] = typeof value === 'string' ? value : JSON.stringify(value);
                break;
            case 'boolean':
                flags[p.name] = value === true || value === 'true';
                break;
            case 'number': {
                const n = Number(value);
                if (Number.isNaN(n))
                    throw usageError(`参数 ${p.name} 需要数字，收到 ${JSON.stringify(value)}`);
                flags[p.name] = n;
                break;
            }
            default:
                flags[p.name] = asCsv(value);
        }
    }
    if (missing.length) {
        throw usageError(`缺少必填参数：${missing.join(', ')}。用法：${spec.usage}`);
    }
    // Declared page/pageSize went through the ordered loop above; the rest are
    // undeclared-but-supported query knobs.
    for (const key of TRANSPORT_FLAGS) {
        if (allowed.has(key))
            continue;
        const value = params[key];
        if (value === undefined || value === null || value === '')
            continue;
        const n = Number(value);
        if (Number.isNaN(n))
            throw usageError(`参数 ${key} 需要数字，收到 ${JSON.stringify(value)}`);
        flags[key] = n;
    }
    return { positional, flags };
}
export async function callCommand(command, params, opts = {}) {
    const spec = resolveCommand(command);
    if (!spec) {
        const suggestion = suggestCommand(command);
        const hint = suggestion ? `，是否要 '${suggestion}'？` : '。';
        return failure(command, usageError(`未知命令 '${command}'${hint}先用 itsm_discover 检索可用命令（当前 ${specIndex().size} 条）。`));
    }
    if (opts.preview) {
        return {
            isError: false,
            payload: {
                ok: true,
                preview: true,
                command: spec.name,
                risk: spec.risk,
                endpoints: spec.endpoints,
                requiresConfirmation: spec.risk === 'high-risk-write',
                usage: spec.usage,
                required: spec.input.params.filter((p) => p.required).map((p) => p.name),
                note: '未发送任何请求。去掉 preview 即执行；高风险写操作还需 confirm=true。',
            },
            structured: { command: spec.name, preview: true, risk: spec.risk },
        };
    }
    if (spec.risk === 'high-risk-write' && !opts.confirm) {
        return {
            isError: true,
            payload: {
                ok: false,
                command: spec.name,
                error: {
                    kind: 'confirmation_required',
                    message: `${spec.name} 属于高风险写操作，未执行`,
                    retryable: false,
                    hint: '先向用户说明将要发生什么；得到明确同意后，用相同参数再调用一次并把 confirm 置为 true。不得自动重试。',
                    risk: {
                        command: spec.name,
                        summary: spec.summary,
                        endpoints: spec.endpoints,
                        usage: spec.usage,
                        requiredParams: spec.input.params.filter((p) => p.required).map((p) => p.name),
                        irreversible: /delete|revoke|terminate|purge|rollback|replay/.test(spec.name),
                    },
                },
            },
            structured: { command: spec.name, kind: 'confirmation_required', risk: spec.risk },
        };
    }
    let invocation;
    try {
        invocation = buildInvocation(spec, params ?? {});
    }
    catch (err) {
        return failure(spec.name, err);
    }
    await ensureAuth();
    try {
        const data = await spec.run({
            args: { positional: invocation.positional, flags: invocation.flags, yes: opts.confirm === true, dryRun: false },
        });
        return success(spec, data, opts);
    }
    catch (err) {
        const cliError = classifyError(err);
        addFailureHint(cliError, spec);
        if (cliError.agentError.kind === 'not_authenticated') {
            const auto = getAutoLoginFailure();
            if (auto) {
                const where = auto.httpStatus ? `HTTP ${auto.httpStatus}${auto.businessCode ? ` / code ${auto.businessCode}` : ''}` : '网络错误';
                cliError.agentError.hint =
                    auto.businessCode === 2003
                        ? '登录触发限流（10 次/分钟/IP）。不要循环重试：改用一次 itsm auth login 或提供 ITSM_TOKEN。'
                        : `自动登录失败（${where}）。检查 ITSM_USERNAME/ITSM_PASSWORD，或先执行一次 itsm auth login 复用已存令牌。`;
            }
        }
        return {
            isError: true,
            payload: { ok: false, command: spec.name, exitCode: cliError.exitCode, error: cliError.agentError },
            structured: {
                command: spec.name,
                kind: cliError.agentError.kind,
                exitCode: cliError.exitCode,
                httpStatus: cliError.agentError.httpStatus,
                businessCode: cliError.agentError.businessCode,
            },
        };
    }
}
function success(spec, data, opts) {
    const outcome = applyShaping(data, {
        fields: opts.fields,
        // Bounded by default: an unbounded list is the fastest way to blow up a
        // context window, and `meta.total` preserves the unanswered part.
        limit: opts.limit ?? (isListPayload(data) ? DEFAULT_LIST_LIMIT : undefined),
    });
    const payload = { ok: true, command: spec.name, data: outcome.data };
    if (outcome.meta)
        payload['meta'] = outcome.meta;
    if (outcome.skippedReason)
        payload['note'] = outcome.skippedReason;
    return {
        isError: false,
        payload,
        structured: {
            command: spec.name,
            risk: spec.risk,
            ...(outcome.meta ? { meta: outcome.meta } : {}),
        },
    };
}
function isListPayload(data) {
    if (Array.isArray(data))
        return true;
    return Boolean(data) && typeof data === 'object' && Array.isArray(data['items']);
}
function failure(command, err) {
    const cliError = classifyError(err);
    const agentError = cliError.agentError;
    return {
        isError: true,
        payload: { ok: false, command, exitCode: cliError.exitCode, error: agentError },
        structured: { command, kind: agentError.kind, exitCode: cliError.exitCode },
    };
}
