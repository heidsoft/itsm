import type { FlowScenario, ScenarioReport } from './types.js';
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
export declare function makeRunTag(): string;
/**
 * Execute a scenario and produce an evidence report.
 *
 * Failure policy: stop at the first failing step (later steps depend on captured
 * ids from earlier ones, so continuing would produce noise, not evidence), then
 * always attempt teardown. Teardown problems are warnings — they never mask the
 * real finding.
 */
export declare function runScenario(scenario: FlowScenario, opts?: RunScenarioOptions): Promise<ScenarioReport>;
