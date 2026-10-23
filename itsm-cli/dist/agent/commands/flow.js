import { writeFileSync } from 'node:fs';
import { cmd } from '../spec.js';
import { requirePositional, stringFlag, booleanFlag } from '../helpers.js';
import { CliError, usageError } from '../../lib/errors.js';
import { runScenario } from '../flow/runner.js';
import { flowScenarios, findScenario, scenarioEndpoints, summarizeScenario, describeScenario } from '../flow/registry.js';
/**
 * `itsm flow` — run business-flow acceptance tests through the same API client
 * agents use for ordinary commands.
 *
 * The output contract is what makes this useful to an agent rather than a human
 * watching a TUI: one JSON envelope, a verdict, per-step evidence, a captured
 * context (ids of created artifacts), and a stable exit code. A failing flow is
 * `ok:false` with `kind: assertion_failed` and the full report under
 * `error.details` — never a silent success with a warning buried in text.
 */
function parseVars(raw) {
    if (!raw)
        return {};
    const vars = {};
    for (const pair of raw.split(',')) {
        const trimmed = pair.trim();
        if (!trimmed)
            continue;
        const eq = trimmed.indexOf('=');
        if (eq <= 0)
            throw usageError(`Invalid --var entry '${trimmed}'. Expected key=value[,key=value].`);
        const key = trimmed.slice(0, eq).trim();
        const value = trimmed.slice(eq + 1).trim();
        if (!key)
            throw usageError(`Invalid --var entry '${trimmed}': empty key.`);
        try {
            vars[key] = JSON.parse(value);
        }
        catch {
            vars[key] = value;
        }
    }
    return vars;
}
/** Keep stdout small on success; failures carry the full report in `error.details`. */
function compactReport(report, reportPath) {
    return {
        scenario: report.scenario,
        verdict: report.verdict,
        runTag: report.runTag,
        durationMs: report.durationMs,
        steps: report.steps.map((step) => ({ name: step.name, ok: step.ok, ms: step.ms, ...(step.endpoint ? { endpoint: step.endpoint } : {}) })),
        ...(report.teardown.length ? { teardown: report.teardown.map((step) => ({ name: step.name, ok: step.ok, ms: step.ms })) } : {}),
        context: report.context,
        warnings: report.warnings,
        ...(reportPath ? { reportPath } : {}),
        next: report.next,
    };
}
export function flowCommands() {
    return [
        cmd('flow.list', 'List built-in business-flow acceptance scenarios', 'itsm flow list', [], async () => {
            const items = flowScenarios().map(summarizeScenario);
            return { items, total: items.length, page: 1, pageSize: items.length, totalPages: items.length ? 1 : 0 };
        }, 'normal', {
            examples: ['itsm --json flow list', 'itsm --json flow show ticket-lifecycle'],
        }),
        cmd('flow.show', 'Show one scenario: steps, endpoints, assertions, negative paths', 'itsm flow show <scenario>', [], async (ctx) => {
            const name = requirePositional(ctx.args, 0, 'scenario');
            const scenario = findScenario(name);
            if (!scenario)
                throw usageError(`Unknown scenario '${name}'. Run \`itsm --json flow list\`.`);
            return describeScenario(scenario);
        }, 'normal'),
        cmd('flow.run', 'Execute a business-flow acceptance test against the live backend (writes test data)', 'itsm flow run <scenario> [--tag t] [--keep] [--report <path>] [--var k=v,k2=v2] [--verbose]', scenarioEndpoints(), async (ctx) => {
            const name = requirePositional(ctx.args, 0, 'scenario');
            const scenario = findScenario(name);
            if (!scenario)
                throw usageError(`Unknown scenario '${name}'. Run \`itsm --json flow list\`.`);
            const tag = stringFlag(ctx.args.flags, 'tag');
            const keep = booleanFlag(ctx.args.flags, 'keep') === true;
            const verbose = booleanFlag(ctx.args.flags, 'verbose') === true;
            const reportPath = stringFlag(ctx.args.flags, 'report');
            const vars = parseVars(stringFlag(ctx.args.flags, 'var'));
            const report = await runScenario(scenario, {
                ...(tag ? { tag } : {}),
                keep,
                verbose,
                vars,
            });
            if (reportPath) {
                writeFileSync(reportPath, `${JSON.stringify(report, null, 2)}\n`, 'utf8');
            }
            if (report.verdict !== 'passed') {
                const failedStep = report.failure?.step ?? 'unknown';
                const hint = report.failure?.hint ?? report.next;
                throw new CliError({
                    kind: report.verdict === 'blocked' ? 'unavailable' : 'assertion_failed',
                    message: report.verdict === 'blocked'
                        ? `Flow '${scenario.name}' could not run: ${report.failure?.message ?? 'environment not ready'}`
                        : `Flow '${scenario.name}' failed at step '${failedStep}': ${report.failure?.message ?? 'assertion failed'}`,
                    retryable: false,
                    ...(hint ? { hint } : {}),
                    details: {
                        ...report,
                        ...(reportPath ? { reportPath } : {}),
                    },
                });
            }
            return compactReport(report, reportPath);
        }, 'write', {
            examples: [
                'itsm --json flow list',
                'itsm --json flow run ticket-lifecycle',
                'itsm --json flow run ticket-lifecycle --keep --verbose --report /tmp/ticket-flow.json',
                'itsm --json flow run service-request-fulfillment --var catalogId=17',
            ],
        }),
    ];
}
