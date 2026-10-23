import type { FlowScenario } from './types.js';
/**
 * Built-in business-flow scenarios.
 *
 * These are not examples: each one is an executable acceptance test that an
 * agent can run against a live backend to prove (or disprove) a product flow,
 * including the negative paths. Adding a scenario here automatically publishes
 * it through `itsm flow list`, and its endpoints join the route-drift check.
 */
export declare function flowScenarios(): FlowScenario[];
export declare function findScenario(name: string): FlowScenario | undefined;
/**
 * Every endpoint any scenario touches. `flow run` declares these as its
 * endpoints so the drift check fails when a scenario points at a route the
 * backend no longer registers.
 */
export declare function scenarioEndpoints(): string[];
/** Compact catalog entry (list output). */
export declare function summarizeScenario(scenario: FlowScenario): Record<string, unknown>;
/** Full declaration for `flow show` — enough for an agent to reason before running. */
export declare function describeScenario(scenario: FlowScenario): Record<string, unknown>;
