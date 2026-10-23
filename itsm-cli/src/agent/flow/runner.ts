import { classifyError } from '../../lib/errors.js';
import type { AgentError } from '../../lib/errors.js';
import { evaluateExpectations } from './assertions.js';
import type { Assertion, FlowContext, FlowScenario, FlowStep, ScenarioReport, StepReport } from './types.js';

export interface RunScenarioOptions {
  /** Overrides the generated run tag (useful for correlating with a bug report). */
  tag?: string;
  /** Keep created artifacts: skips teardown. */
  keep?: boolean;
  /** Attach (truncated) response bodies to each step report. */
  verbose?: boolean;
  /** Extra variables available to the scenario before it starts. */
  vars?: Record<string, unknown>;
  /** Progress sink; must write to stderr. */
  log?: (message: string) => void;
}

/** Short, sortable, collision-resistant run tag embedded in created titles. */
export function makeRunTag(): string {
  const now = new Date();
  const stamp = [
    now.getFullYear(),
    String(now.getMonth() + 1).padStart(2, '0'),
    String(now.getDate()).padStart(2, '0'),
    String(now.getHours()).padStart(2, '0'),
    String(now.getMinutes()).padStart(2, '0'),
    String(now.getSeconds()).padStart(2, '0'),
  ].join('');
  const rand = Math.random().toString(36).slice(2, 6);
  return `agent-${stamp}-${rand}`;
}

const MAX_RESPONSE_CHARS = 4000;

function truncate(value: unknown): unknown {
  try {
    const text = JSON.stringify(value);
    if (text === undefined) return value;
    if (text.length <= MAX_RESPONSE_CHARS) return value;
    return { truncated: true, bytes: text.length, preview: `${text.slice(0, MAX_RESPONSE_CHARS)}…` };
  } catch {
    return '<unserializable>';
  }
}

function agentErrorOf(err: unknown): AgentError {
  return classifyError(err).agentError;
}

/**
 * A negative-path step passes only when the backend rejects it in the declared
 * shape. Getting 500 instead of 409 still fails: "the product refused" is not
 * the same claim as "the product refused for the right reason".
 */
function matchesExpectedError(expected: NonNullable<FlowStep['expectError']>, err: unknown): { ok: boolean; failures: Assertion[]; agentError: AgentError } {
  const agentError = agentErrorOf(err);
  const failures: Assertion[] = [];
  const check = (label: string, ok: boolean, expectedValue: unknown, actual: unknown) => {
    if (!ok) failures.push({ check: label, ok: false, expected: expectedValue, actual });
  };

  if (expected.kind && agentError.kind !== expected.kind) {
    check(`rejection kind == ${expected.kind}`, false, expected.kind, agentError.kind);
  }
  if (expected.httpStatus && agentError.httpStatus !== expected.httpStatus) {
    check(`httpStatus == ${expected.httpStatus}`, false, expected.httpStatus, agentError.httpStatus);
  }
  if (expected.businessCode && agentError.businessCode !== expected.businessCode) {
    check(`businessCode == ${expected.businessCode}`, false, expected.businessCode, agentError.businessCode);
  }
  if (expected.messageIncludes && !agentError.message.includes(expected.messageIncludes)) {
    check(`message contains ${JSON.stringify(expected.messageIncludes)}`, false, expected.messageIncludes, agentError.message);
  }
  return { ok: failures.length === 0, failures, agentError };
}

async function runStep(step: FlowStep, ctx: FlowContext, opts: RunScenarioOptions): Promise<StepReport> {
  const started = Date.now();
  ctx.log(`→ ${step.name}`);
  try {
    const result = await step.run(ctx);
    const assertions = [
      ...evaluateExpectations(step.expect ?? [], result),
      ...(step.verify ? step.verify(result, ctx) : []),
    ];
    const ok = assertions.every((a) => a.ok);

    let captured: Record<string, unknown> | undefined;
    if (ok && step.capture) {
      captured = step.capture(result, ctx);
      Object.assign(ctx.vars, captured);
    }

    return {
      name: step.name,
      ok,
      ms: Date.now() - started,
      ...(step.endpoint ? { endpoint: step.endpoint } : {}),
      assertions,
      ...(captured ? { captured } : {}),
      ...(opts.verbose ? { response: truncate(result) } : {}),
    };
  } catch (err) {
    const agentError = agentErrorOf(err);

    // A declared governance/environment restriction is not a product defect:
    // report the flow as blocked and stop pretending we verified it.
    if (step.blockedWhen) {
      const statusMatches = step.blockedWhen.httpStatus === undefined || agentError.httpStatus === step.blockedWhen.httpStatus;
      const messageMatches =
        step.blockedWhen.messageIncludes === undefined || agentError.message.includes(step.blockedWhen.messageIncludes);
      if (statusMatches && messageMatches) {
        return {
          name: step.name,
          ok: false,
          ms: Date.now() - started,
          ...(step.endpoint ? { endpoint: step.endpoint } : {}),
          assertions: [
            {
              check: `flow blocked by a declared precondition: ${step.blockedWhen.reason}`,
              ok: false,
              expected: 'the step to run',
              actual: { kind: agentError.kind, httpStatus: agentError.httpStatus, message: agentError.message },
            },
          ],
          error: {
            kind: 'unavailable',
            message: `Blocked: ${step.blockedWhen.reason} (backend said: ${agentError.message})`,
            ...(agentError.httpStatus ? { httpStatus: agentError.httpStatus } : {}),
            ...(agentError.businessCode ? { businessCode: agentError.businessCode } : {}),
          },
        };
      }
    }

    if (step.expectError) {
      const match = matchesExpectedError(step.expectError, err);
      return {
        name: step.name,
        ok: match.ok,
        ms: Date.now() - started,
        ...(step.endpoint ? { endpoint: step.endpoint } : {}),
        assertions: [
          {
            check: `rejected as expected (${agentError.kind}${agentError.httpStatus ? `/${agentError.httpStatus}` : ''})`,
            ok: match.ok,
            expected: step.expectError,
            actual: { kind: agentError.kind, httpStatus: agentError.httpStatus, businessCode: agentError.businessCode, message: agentError.message },
          },
          ...match.failures,
        ],
        error: {
          // A matched rejection keeps the backend's kind (useful evidence). A
          // MISMATCH is an assertion failure: a negative test that sees a 500
          // instead of a 404 is a product defect, and must not be softened into
          // verdict `blocked` just because the observed kind was `unavailable`.
          kind: match.ok ? agentError.kind : 'assertion_failed',
          message: match.ok
            ? agentError.message
            : `expected rejection ${JSON.stringify(step.expectError)} but got ${agentError.kind}${agentError.httpStatus ? `/${agentError.httpStatus}` : ''}: ${agentError.message}`,
          ...(agentError.httpStatus ? { httpStatus: agentError.httpStatus } : {}),
          ...(agentError.businessCode ? { businessCode: agentError.businessCode } : {}),
          ...(agentError.hint ? { hint: agentError.hint } : {}),
        },
      };
    }

    return {
      name: step.name,
      ok: false,
      ms: Date.now() - started,
      ...(step.endpoint ? { endpoint: step.endpoint } : {}),
      assertions: [],
      error: {
        kind: agentError.kind,
        message: agentError.message,
        ...(agentError.httpStatus ? { httpStatus: agentError.httpStatus } : {}),
        ...(agentError.businessCode ? { businessCode: agentError.businessCode } : {}),
        ...(agentError.hint ? { hint: agentError.hint } : {}),
      },
      ...(opts.verbose ? { response: truncate(err) } : {}),
    };
  }
}

/**
 * Execute a scenario and produce an evidence report.
 *
 * Failure policy: stop at the first failing step (later steps depend on captured
 * ids from earlier ones, so continuing would produce noise, not evidence), then
 * always attempt teardown. Teardown problems are warnings — they never mask the
 * real finding.
 */
export async function runScenario(scenario: FlowScenario, opts: RunScenarioOptions = {}): Promise<ScenarioReport> {
  const tag = opts.tag ?? makeRunTag();
  const startedAt = new Date();
  const ctx: FlowContext = {
    vars: { ...(opts.vars ?? {}) },
    tag,
    log: opts.log ?? ((message) => process.stderr.write(`${message}\n`)),
  };

  const steps: StepReport[] = [];
  for (const step of scenario.steps) {
    const report = await runStep(step, ctx, opts);
    steps.push(report);
    if (!report.ok) break;
  }

  const teardown: StepReport[] = [];
  const warnings: string[] = [];
  if (!opts.keep && scenario.teardown?.length) {
    for (const step of [...scenario.teardown].reverse()) {
      const report = await runStep(step, ctx, opts);
      teardown.push(report);
      if (!report.ok) warnings.push(`teardown step '${step.name}' failed: ${report.error?.message ?? 'assertion failed'}`);
    }
  } else if (opts.keep) {
    warnings.push(`artifacts retained (--keep); run tag '${tag}' identifies them`);
  }

  const failed = steps.find((s) => !s.ok);
  const verdict = !failed ? 'passed' : failed.error?.kind === 'unavailable' || failed.error?.kind === 'not_authenticated' ? 'blocked' : 'failed';

  const report: ScenarioReport = {
    scenario: scenario.name,
    summary: scenario.summary,
    verdict,
    runTag: tag,
    startedAt: startedAt.toISOString(),
    durationMs: Date.now() - startedAt.getTime(),
    steps,
    teardown,
    context: ctx.vars,
    warnings,
  };

  if (failed) {
    const failedStep = scenario.steps.find((s) => s.name === failed.name);
    report.failure = {
      step: failed.name,
      kind: failed.error?.kind ?? 'assertion_failed',
      message: failed.error?.message ?? describeFailedAssertions(failed),
      ...(failed.error?.hint ? { hint: failed.error.hint } : {}),
    };
    if (failedStep?.expectError) {
      report.failure.message = `negative-path step did not fail as expected: ${report.failure.message}`;
    }
  }

  report.next = nextStepFor(report, scenario);
  return report;
}

function describeFailedAssertions(step: StepReport): string {
  const failed = step.assertions.filter((a) => !a.ok);
  if (!failed.length) return 'step failed without a diagnostic';
  return failed
    .map((a) => `${a.check} (expected ${JSON.stringify(a.expected)}, got ${JSON.stringify(a.actual)})`)
    .join('; ');
}

function nextStepFor(report: ScenarioReport, scenario: FlowScenario): string {
  if (report.verdict === 'passed') {
    return `Flow '${scenario.name}' passed. Re-run with --keep to inspect the created artifacts (run tag ${report.runTag}).`;
  }
  if (report.verdict === 'blocked') {
    return `Environment is not ready for '${scenario.name}'${scenario.requires?.length ? ` (requires: ${scenario.requires.join(', ')})` : ''}. Fix the dependency, then re-run; this is not a product-flow failure.`;
  }
  const step = report.failure?.step;
  return `Inspect step '${step}' (${report.failure?.kind}). Re-run with --report <file> to capture full evidence, and compare the endpoint contract with the backend handler before filing a defect.`;
}
