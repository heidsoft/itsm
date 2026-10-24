import { registry, domains, findCommandByTokens } from './registry.js';
import { parseArgs } from './args.js';
import { describeCommand } from './spec.js';
import { paramToJsonSchema } from './params.js';
import { applyShaping } from './shape.js';
import { addFailureHint } from './hints.js';
import { ExitCode, classifyError, usageError, successEnvelope, failureEnvelope } from '../lib/errors.js';
import { resolveRuntimeConfig, hydrateAgentToken, getAutoLoginFailure } from '../lib/runtime.js';
import { configureTransport, configureBaseURL } from '../lib/api-client.js';
import { isClosedStreamError, writeStderrLine } from '../lib/streams.js';
export async function runAgent(argv) {
    installStreamErrorGuards();
    installStdoutGuard();
    let result;
    try {
        result = await dispatch(argv);
    }
    catch (err) {
        // `parseArgs` (bad flag value), runtime secret-file validation and the
        // transport policy all throw CliErrors before a command spec is resolved.
        // They must still produce the one-JSON-envelope contract, not a stack trace.
        const cliError = classifyError(err);
        result = { code: cliError.exitCode, envelope: failureEnvelope(cliError) };
    }
    if (result.ndjson) {
        for (const line of result.ndjson)
            process.stdout.write(JSON.stringify(line) + '\n');
        return result.code;
    }
    if (!argv.includes('--json'))
        return result.code;
    const payload = result.envelope ?? { ok: true, data: null };
    process.stdout.write(JSON.stringify(payload) + '\n');
    return result.code;
}
/**
 * An async EPIPE on stdout/stderr would otherwise surface as an unhandled
 * 'error' event and kill the process mid-run (skipping flow teardown).
 */
function installStreamErrorGuards() {
    for (const stream of [process.stdout, process.stderr]) {
        stream.on('error', (err) => {
            if (isClosedStreamError(err))
                return;
            // Keep unexpected I/O failures loud instead of turning them into silence.
            process.nextTick(() => {
                throw err;
            });
        });
    }
}
/**
 * stdout carries exactly one JSON document. Any stray console.log — ours, a
 * dependency's, or a future debug line — would corrupt that contract, so in
 * agent mode every console channel is rerouted to stderr.
 */
function installStdoutGuard() {
    if (process.env['ITSM_CLI_AGENT_NO_GUARD'] === '1')
        return;
    const toStderr = (...args) => {
        const line = args.map((a) => (typeof a === 'string' ? a : JSON.stringify(a))).join(' ');
        writeStderrLine(line);
    };
    console.log = toStderr;
    console.info = toStderr;
    console.debug = toStderr;
}
async function dispatch(argv) {
    const args = parseArgs(argv);
    const domain = args.positional[0];
    configureBaseURL(args.baseUrl);
    configureTransport({
        ...(args.timeoutMs !== undefined ? { timeoutMs: args.timeoutMs } : {}),
        ...(args.retries !== undefined ? { retries: args.retries } : {}),
    });
    // Env-provided auth (ITSM_TOKEN, or ITSM_USERNAME/ITSM_PASSWORD auto-login)
    // must be usable without a prior interactive `itsm auth login`.
    await hydrateAgentToken(resolveRuntimeConfig({ baseUrl: args.baseUrl })).catch(() => undefined);
    // ---- builtins (not part of the registry to avoid import cycles) ----
    if (!domain || domain === 'help' || domain === 'manifest') {
        return { code: ExitCode.Success, envelope: manifestEnvelope() };
    }
    if (domain === 'commands') {
        return { code: ExitCode.Success, envelope: successEnvelope(commandCatalog()) };
    }
    if (domain === 'schema') {
        return schemaBuiltin(args);
    }
    // `--help` must short-circuit BEFORE the exit-10 risk gate: an agent asking
    // "what does this command need" should not be handed a destructive-write
    // confirmation. Advertised in the README since the agent protocol landed, but
    // it never actually worked (`incident close --help` returned exit 10).
    if (args.flags['help'] === true || args.flags['h'] === true) {
        return helpBuiltin(args);
    }
    // `itsm doctor` is the natural first call for a cold agent; alias it to the
    // registered command so the catalog stays the single source of truth.
    if (domain === 'doctor') {
        const doctor = registry().find((s) => s.name === 'infra.doctor');
        if (doctor)
            return executeSpec(doctor, args, args.positional.slice(1));
    }
    if (args.positional.length < 2) {
        return { code: ExitCode.Usage, envelope: failureEnvelope(usageError('Missing verb. Run `itsm commands --json` for the command catalog.')) };
    }
    const spec = findCommandByTokens(args.positional);
    if (!spec) {
        const knownList = registry().length;
        const suggestion = suggestCommand(args.positional.slice(0, 2).join(' '));
        return {
            code: ExitCode.Usage,
            envelope: failureEnvelope(usageError(`Unknown command '${args.positional.slice(0, 2).join(' ')}'${suggestion ? ` — did you mean '${suggestion}'?` : ''}. ${knownList} commands available — run 'itsm commands --json'.`)),
        };
    }
    // Skip the tokens that composed the command name (e.g. `ticket comments add`), keep the rest.
    return executeSpec(spec, args, args.positional.slice(spec.name.split('.').length));
}
/** Shared execution path: flag contract → dry-run preview → exit-10 gate → run → classify failures. */
async function executeSpec(spec, args, rest) {
    validateFlags(spec, args);
    validatePositionals(spec, rest);
    if (args.dryRun) {
        return {
            code: ExitCode.Success,
            envelope: successEnvelope({
                dryRun: true,
                command: spec.name,
                risk: spec.risk,
                endpoints: spec.endpoints,
                requiresConfirmation: spec.risk === 'high-risk-write',
                input: spec.input,
                note: 'No request was sent. Remove --dry-run to execute; high-risk writes additionally need --yes.',
            }),
        };
    }
    if (spec.risk === 'high-risk-write' && !args.yes) {
        // confirmation_required protocol (exit 10). Do not proceed on retry without the user's consent.
        return {
            code: 10,
            envelope: {
                ok: false,
                error: {
                    kind: 'confirmation_required',
                    message: `${spec.name} is a high-risk write and requires explicit confirmation`,
                    retryable: false,
                    hint: 'Verify the action with the user; if they agree, retry the SAME command with --yes appended. Never retry automatically.',
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
        };
    }
    try {
        const data = await spec.run({ args: { positional: rest, flags: args.flags, yes: args.yes, dryRun: args.dryRun } });
        return shapeResult(data, args);
    }
    catch (err) {
        const cliError = classifyError(err);
        addFailureHint(cliError, spec);
        if (cliError.agentError.kind === 'not_authenticated') {
            const f = getAutoLoginFailure();
            if (f) {
                // Distinguish "auto-login never got a token" from "no token configured" —
                // the raw 401 alone sends agents chasing the wrong fix.
                const where = f.httpStatus ? `HTTP ${f.httpStatus}${f.businessCode ? ` / code ${f.businessCode}` : ''}` : 'network error';
                cliError.agentError.message = `${cliError.agentError.message} (auto-login failed: ${where} — ${f.message})`;
                cliError.agentError.hint =
                    f.businessCode === 2003
                        ? 'Login rate limit hit (10/min per IP). Do not retry in a loop: run `itsm auth login --username <u> --password <p>` once, then unset ITSM_USERNAME/ITSM_PASSWORD so later calls reuse ~/.itsm/credentials, or pass a token via ITSM_TOKEN.'
                        : 'ITSM_USERNAME/ITSM_PASSWORD auto-login failed. Verify the credentials, or run `itsm auth login` once and let subsequent calls reuse the stored token.';
            }
        }
        return { code: cliError.exitCode, envelope: failureEnvelope(cliError) };
    }
}
/**
 * Strict flag validation at dispatch.
 *
 * `parseArgs` is deliberately permissive (it cannot know the command yet), so
 * without this check an unknown flag lands in `flags`, is never read by the
 * command, and the call silently succeeds without the requested effect — a
 * wrong answer, not an error. The declared contract is `spec.input` (derived
 * from the usage string plus explicit params), which is the SAME source the
 * MCP/agent schemas advertise, so a flag accepted here is always visible in
 * `itsm schema` output and vice versa.
 *
 * Single-character names (`-p`) are allowed through: they are legacy aliases
 * read via `stringFlag(flags, 'password', 'p')` and never advertised.
 */
function validateFlags(spec, args) {
    const known = new Set(spec.input.params.filter((p) => p.kind !== 'positional').map((p) => p.name));
    // `body()`/`jsonFlag()` accept `--payload` wherever `--data` is declared.
    if (known.has('data'))
        known.add('payload');
    // Policy denials come first: a secret on argv should explain WHY it is
    // refused and point at the safe channel, not read as a typo.
    if (spec.denyFlags) {
        for (const flag of Object.keys(args.flags)) {
            const reason = spec.denyFlags[flag];
            if (!reason)
                continue;
            throw usageError(`Refusing '--${flag}' on the command line for '${spec.name}': ${reason}`);
        }
    }
    const unknown = Object.keys(args.flags).filter((f) => !known.has(f) && f.length > 1);
    if (!unknown.length)
        return;
    const first = unknown[0];
    const suggestion = closestName(first, [...known]);
    const more = unknown.length > 1 ? ` (and ${unknown.length - 1} more: ${unknown.slice(1).map((f) => `--${f}`).join(', ')})` : '';
    const cliName = spec.name.split('.').join(' ');
    throw usageError(`Unknown flag '--${first}' for '${spec.name}'${more}` +
        `${suggestion ? ` — did you mean '--${suggestion}'? ` : ' '}` +
        `Run 'itsm ${cliName} --help' for the declared contract.`);
}
/**
 * Strict positional validation at dispatch — the mirror of `validateFlags`.
 *
 * Only *extra* positionals are rejected (a declared-but-unused one is harmless).
 * Without this, `itsm connector configs garbage` resolves to `connector.configs`,
 * never reads the trailing token, and returns ok:true — the agent believes a
 * per-connector config was fetched when the argument was silently dropped.
 * Under-declared positionals are caught statically by `audit-positionals.mjs`.
 */
function validatePositionals(spec, rest) {
    const declared = spec.input.params.filter((p) => p.kind === 'positional');
    if (rest.length <= declared.length)
        return;
    const expected = declared.map((p) => `<${p.name}>`).join(' ');
    const cliName = spec.name.split('.').join(' ');
    throw usageError(`Unexpected positional argument '${rest[declared.length]}' for '${spec.name}'` +
        ` — this command takes ${declared.length ? expected : 'no positional arguments'} (usage: ${spec.usage}).`);
}
/** Smallest edit distance to a known flag, surfaced only when meaningfully close. */ function closestName(name, candidates) {
    let best;
    let bestDist = Infinity;
    for (const c of candidates) {
        const d = editDistance(name, c);
        if (d < bestDist) {
            bestDist = d;
            best = c;
        }
    }
    return bestDist <= Math.max(2, Math.floor(name.length / 3)) ? best : undefined;
}
function editDistance(a, b) {
    const prev = new Array(b.length + 1).fill(0).map((_, i) => i);
    for (let i = 1; i <= a.length; i++) {
        let diag = prev[0];
        prev[0] = i;
        for (let j = 1; j <= b.length; j++) {
            const tmp = prev[j];
            prev[j] = Math.min(prev[j] + 1, prev[j - 1] + 1, diag + (a[i - 1] === b[j - 1] ? 0 : 1));
            diag = tmp;
        }
    }
    return prev[b.length];
}
/** Deterministic, side-effect-free output shaping for agent context budgets. */
function shapeResult(data, args) {
    const outcome = applyShaping(data, { fields: args.fields, limit: args.limit });
    if (!outcome.isList) {
        if (!outcome.meta)
            return { code: ExitCode.Success, envelope: successEnvelope(data) };
        return { code: ExitCode.Success, envelope: { ok: true, data, meta: outcome.meta } };
    }
    if (args.format === 'ndjson') {
        return { code: ExitCode.Success, ndjson: [...outcome.items, { _meta: outcome.meta }] };
    }
    const hasShaping = args.fields?.length || args.limit !== undefined;
    return {
        code: ExitCode.Success,
        envelope: hasShaping ? { ok: true, data: outcome.data, meta: outcome.meta } : successEnvelope(outcome.data),
    };
}
function commandCatalog() {
    return {
        count: registry().length,
        domains: domains(),
        detail: 'itsm --json schema [<domain>|<command>] [--as summary|detail|tools]',
        note: 'Run `itsm --json schema <command>` for required params before first use.',
    };
}
function helpBuiltin(args) {
    const target = findCommandByTokens(args.positional);
    if (target) {
        return { code: ExitCode.Success, envelope: successEnvelope(describeCommand(target)) };
    }
    const domainName = args.positional[0];
    const specs = domainName ? registry().filter((s) => s.name.split('.')[0] === domainName) : [];
    if (specs.length) {
        return {
            code: ExitCode.Success,
            envelope: successEnvelope({
                domain: domainName,
                commands: specs.map((s) => ({
                    command: s.name,
                    summary: s.summary,
                    usage: s.usage,
                    risk: s.risk,
                    required: s.input.params.filter((p) => p.required).map((p) => p.name),
                })),
            }),
        };
    }
    return {
        code: ExitCode.Success,
        envelope: successEnvelope({ count: registry().length, domains: domains(), usage: 'itsm --json [<domain> [<verb>...]] --help' }),
    };
}
/**
 * `itsm schema` — the input-contract surface.
 *   summary (default): every command, name + summary + required params (cheap).
 *   detail: full params (types, enums, defaults) for one command or a domain.
 *   tools: function-calling definitions (name/description/JSON Schema + CLI mapping).
 */
function schemaBuiltin(args) {
    const as = typeof args.flags['as'] === 'string' ? args.flags['as'] : 'summary';
    const target = args.positional[1];
    if (as === 'tools') {
        const domainFilter = typeof args.flags['domain'] === 'string' ? String(args.flags['domain']) : undefined;
        const specs = registry().filter((s) => !domainFilter || s.name.split('.')[0] === domainFilter);
        return {
            code: ExitCode.Success,
            envelope: successEnvelope({
                format: 'tools',
                count: specs.length,
                note: 'x-itsm.cli maps each tool call back to a CLI invocation; positionals are passed in declared order.',
                tools: specs.map((spec) => ({
                    name: `itsm_${spec.name.replace(/[.-]/g, '_')}`,
                    description: `${spec.summary} (${spec.name})${spec.risk === 'high-risk-write' ? ' [HIGH RISK: requires --yes]' : ''}`,
                    input_schema: {
                        type: 'object',
                        properties: Object.fromEntries(spec.input.params
                            .filter((p) => p.name !== 'data')
                            .map((p) => [p.name, paramToJsonSchema(p)])),
                        required: spec.input.params.filter((p) => p.required && p.name !== 'data').map((p) => p.name),
                        additionalProperties: false,
                    },
                    'x-itsm': {
                        command: spec.name,
                        usage: spec.usage,
                        positional: spec.input.params.filter((p) => p.kind === 'positional').map((p) => p.name),
                        bodyFlag: spec.input.body ? '--data' : undefined,
                        risk: spec.risk,
                        endpoints: spec.endpoints,
                    },
                })),
            }),
        };
    }
    if (!target) {
        if (as === 'detail') {
            const commands = registry().map((s) => describeCommand(s));
            return {
                code: ExitCode.Success,
                envelope: successEnvelope({ count: commands.length, commands }),
            };
        }
        const commands = registry().map((s) => ({
            command: s.name,
            summary: s.summary,
            risk: s.risk,
            required: s.input.params.filter((p) => p.required).map((p) => p.name),
        }));
        return {
            code: ExitCode.Success,
            envelope: successEnvelope({
                count: commands.length,
                commands,
                next: 'itsm --json schema <domain|command> --as detail   |   --as tools --domain <domain>',
            }),
        };
    }
    const exact = registry().find((s) => s.name === target);
    if (exact) {
        return { code: ExitCode.Success, envelope: successEnvelope(describeCommand(exact)) };
    }
    const domainSpecs = registry().filter((s) => s.name.split('.')[0] === target);
    if (domainSpecs.length) {
        return {
            code: ExitCode.Success,
            envelope: successEnvelope({
                domain: target,
                count: domainSpecs.length,
                commands: domainSpecs.map((s) => describeCommand(s)),
            }),
        };
    }
    return {
        code: ExitCode.Usage,
        envelope: failureEnvelope(usageError(`Unknown command or domain '${target}'. Run 'itsm --json commands' to list domains.`)),
    };
}
/** Cheap Levenshtein-free suggestion: shared-prefix match over known command names. */
function suggestCommand(input) {
    const [d, v = ''] = input.split(/\s+/);
    if (!d)
        return undefined;
    const names = registry().map((s) => s.name);
    const domainMatch = names.filter((n) => n.split('.')[0] === d);
    if (domainMatch.length && v) {
        const close = domainMatch.find((n) => n.split('.')[1]?.startsWith(v.slice(0, 3)));
        if (close)
            return close.replace('.', ' ');
    }
    const anyDomain = [...new Set(names.map((n) => n.split('.')[0]))].find((x) => x.startsWith(d.slice(0, 3)));
    return anyDomain;
}
function manifestEnvelope() {
    return successEnvelope({
        service: 'itsm-cli',
        agentProtocol: true,
        commandCatalog: 'itsm commands --json',
        inputSchemas: 'itsm schema --json [<domain>|<command>] [--as summary|detail|tools]',
        commands: registry().length,
        domains: domains(),
        exitCodes: {
            0: 'success',
            1: 'network (backend unreachable or timed out; retryable)',
            2: 'usage (bad arguments or unknown command)',
            3: 'not authenticated (run auth login or set ITSM_TOKEN)',
            4: 'forbidden (missing permission)',
            5: 'request rejected (validation/not found/conflict; see error.kind)',
            6: 'internal CLI error',
            10: 'confirmation_required (high-risk write; show risk, get consent, then retry with --yes)',
        },
        outputControls: {
            '--fields id,title': 'project list items to these fields (dotted paths allowed)',
            '--limit N': 'cap list items; response meta reports total vs returned',
            '--format ndjson': 'stream one JSON object per line (a final {_meta} line carries counts)',
            '--dry-run': 'show endpoints/risk/required params without sending a request',
        },
        env: {
            ITSM_BASE_URL: 'override backend URL (default http://localhost:8090)',
            ITSM_TOKEN: 'agent identity token (headless, no login file needed)',
            ITSM_TOKEN_FILE: 'path to a 0600 file containing the identity token (preferred over ITSM_TOKEN)',
            ITSM_USERNAME: 'non-interactive login (paired with ITSM_PASSWORD or ITSM_PASSWORD_FILE)',
            ITSM_PASSWORD: 'password for non-interactive login (visible to same-user processes)',
            ITSM_PASSWORD_FILE: 'path to a 0600 file containing the password (preferred over ITSM_PASSWORD)',
            ITSM_RESET_TOKEN: 'password-reset token read from the environment (prefer ITSM_RESET_TOKEN_FILE)',
            ITSM_RESET_TOKEN_FILE: 'path to a 0600 file containing a password-reset token',
            ITSM_CREDENTIALS_FILE: 'override the credential store path (default ~/.itsm/credentials)',
            ITSM_ALLOW_INSECURE_HTTP: 'set to 1 to allow sending credentials over non-loopback plaintext HTTP (not recommended)',
            ITSM_CLI_AGENT_NO_GUARD: 'set to 1 to disable the stdout guard (debugging only)',
        },
        authSecurity: {
            argvSecrets: 'password/reset-token flags are refused with exit 2 (argv leaks via shell history and ps); use --password-stdin/--password-file, --token-stdin, --reset-token-stdin/--reset-token-file',
            credentialStore: '0700 dir / 0600 file, atomic writes, symlink writes refused',
            transport: 'credentials are refused over http:// unless the host is loopback or ITSM_ALLOW_INSECURE_HTTP=1',
            logout: 'revokes the refresh token server-side before deleting the local store',
            output: 'access/refresh tokens are never included in command output or error messages',
        },
    });
}
