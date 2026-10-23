import type { FlowScenario } from '../types.js';
/**
 * Problem management: investigate → root cause → workaround/solution → close.
 * This is the path that feeds the known-error database, so a silent state bug
 * here breaks knowledge reuse downstream.
 */
export declare const problemLifecycle: FlowScenario;
