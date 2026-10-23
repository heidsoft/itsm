import { ticketLifecycle } from './scenarios/ticket.js';
import { incidentLifecycle } from './scenarios/incident.js';
import { problemLifecycle } from './scenarios/problem.js';
import { changeLifecycle } from './scenarios/change.js';
import { knowledgeLifecycle } from './scenarios/knowledge.js';
import { cmdbTopology } from './scenarios/cmdb.js';
import { serviceRequestFulfillment } from './scenarios/service-request.js';
/**
 * Built-in business-flow scenarios.
 *
 * These are not examples: each one is an executable acceptance test that an
 * agent can run against a live backend to prove (or disprove) a product flow,
 * including the negative paths. Adding a scenario here automatically publishes
 * it through `itsm flow list`, and its endpoints join the route-drift check.
 */
export function flowScenarios() {
    return [
        ticketLifecycle,
        incidentLifecycle,
        problemLifecycle,
        changeLifecycle,
        knowledgeLifecycle,
        cmdbTopology,
        serviceRequestFulfillment,
    ];
}
export function findScenario(name) {
    return flowScenarios().find((scenario) => scenario.name === name);
}
/**
 * Every endpoint any scenario touches. `flow run` declares these as its
 * endpoints so the drift check fails when a scenario points at a route the
 * backend no longer registers.
 */
export function scenarioEndpoints() {
    const endpoints = new Set();
    for (const scenario of flowScenarios()) {
        for (const step of [...scenario.steps, ...(scenario.teardown ?? [])]) {
            if (step.endpoint)
                endpoints.add(step.endpoint);
        }
    }
    return [...endpoints].sort();
}
/** Compact catalog entry (list output). */
export function summarizeScenario(scenario) {
    return {
        name: scenario.name,
        domain: scenario.domain,
        summary: scenario.summary,
        steps: scenario.steps.length,
        teardown: scenario.teardown?.length ? 'runs (delete artifacts)' : 'none',
        tags: scenario.tags,
        requires: scenario.requires ?? [],
    };
}
/** Full declaration for `flow show` — enough for an agent to reason before running. */
export function describeScenario(scenario) {
    return {
        scenario: scenario.name,
        domain: scenario.domain,
        summary: scenario.summary,
        tags: scenario.tags,
        requires: scenario.requires ?? [],
        proves: scenario.proves,
        steps: scenario.steps.map((step) => ({
            name: step.name,
            endpoint: step.endpoint ?? null,
            expectations: (step.expect ?? []).map((exp) => ({ ...exp })),
            negative: Boolean(step.expectError),
            ...(step.expectError ? { expectError: step.expectError } : {}),
        })),
        teardown: (scenario.teardown ?? []).map((step) => ({ name: step.name, endpoint: step.endpoint ?? null })),
    };
}
