/**
 * Business-flow scenario framework.
 *
 * A scenario is an executable acceptance test for one ITIL flow, written in
 * TypeScript so it reuses the same ApiClient (and therefore the same endpoint
 * contracts) that the CLI itself uses. An agent runs `itsm flow run <name>` and
 * gets a structured verdict: which step broke, what was expected, what the
 * backend actually returned, and how long each call took.
 *
 * Two deliberate design choices:
 *
 *   · Assertions are declarative (`expect: [{path:'status', eq:'resolved'}]`),
 *     not ad-hoc booleans. That is what makes evidence comparable across runs
 *     and lets the report say "expected X, got Y" without re-reading source.
 *   · Negative paths are first-class (`expectError`). A product test that only
 *     proves happy paths cannot detect the failures this repo keeps fixing —
 *     illegal transitions that silently succeed, RBAC holes, tenant leaks.
 */
import type { ErrorKind } from '../../lib/errors.js';
export type ScenarioVerdict = 'passed' | 'failed' | 'blocked';
/** Declarative assertion against one response body. */
export interface Expectation {
    /** Dotted path into the response, e.g. `status`, `items.0.id`, `sla.bound`. */
    path: string;
    eq?: unknown;
    ne?: unknown;
    in?: unknown[];
    exists?: boolean;
    truthy?: boolean;
    falsy?: boolean;
    contains?: string;
    gte?: number;
    lte?: number;
}
/** Result of evaluating one expectation (or one custom verify). */
export interface Assertion {
    check: string;
    ok: boolean;
    expected?: unknown;
    actual?: unknown;
}
export interface FlowContext {
    /** Values captured by earlier steps (`ticketId`, `changeId`, …). */
    vars: Record<string, unknown>;
    /** Unique per run; embedded in created titles so artifacts are traceable. */
    tag: string;
    /** Emit progress to stderr (never stdout — that carries the JSON envelope). */
    log: (message: string) => void;
}
export interface FlowStep {
    name: string;
    /** `METHOD /api/v1/...` this step exercises; used for evidence and drift checks. */
    endpoint?: string;
    run: (ctx: FlowContext) => Promise<unknown>;
    expect?: Expectation[];
    /**
     * Negative-path step: passes only when the call FAILS with the declared shape.
     * `kind` comes from the CLI's stable error taxonomy (conflict, forbidden, …).
     */
    expectError?: {
        kind?: ErrorKind;
        httpStatus?: number;
        businessCode?: number;
        messageIncludes?: string;
    };
    /**
     * Treat a specific rejection as "this flow cannot run here" rather than a
     * product failure (verdict `blocked`). Used for governance rules that need a
     * second identity, e.g. "an author may not approve their own article".
     */
    blockedWhen?: {
        httpStatus?: number;
        messageIncludes?: string;
        reason: string;
    };
    /** Pull ids/values out of a successful response for later steps. */
    capture?: (result: unknown, ctx: FlowContext) => Record<string, unknown>;
    /** Checks that need more than one field or a computed invariant. */
    verify?: (result: unknown, ctx: FlowContext) => Assertion[];
}
export interface FlowScenario {
    name: string;
    summary: string;
    domain: string;
    /** What this scenario proves, in the language of the product contract. */
    proves: string[];
    tags: string[];
    /** Warns the runner that the scenario needs a configured environment (SLA, approval template, …). */
    requires?: string[];
    steps: FlowStep[];
    /** Always executed (reverse order) unless `--keep`; failures here are warnings, not verdict changes. */
    teardown?: FlowStep[];
}
export interface StepReport {
    name: string;
    ok: boolean;
    ms: number;
    endpoint?: string;
    assertions: Assertion[];
    captured?: Record<string, unknown>;
    error?: {
        kind: ErrorKind;
        message: string;
        httpStatus?: number;
        businessCode?: number;
        hint?: string;
    };
    /** Present only with `--verbose`; truncated to keep reports small. */
    response?: unknown;
}
export interface ScenarioReport {
    scenario: string;
    summary: string;
    verdict: ScenarioVerdict;
    runTag: string;
    startedAt: string;
    durationMs: number;
    steps: StepReport[];
    teardown: StepReport[];
    /** Captured ids, so an agent can inspect artifacts left behind with `--keep`. */
    context: Record<string, unknown>;
    failure?: {
        step: string;
        kind: string;
        message: string;
        hint?: string;
    };
    warnings: string[];
    /** Human/agent next step derived from the verdict. */
    next?: string;
}
