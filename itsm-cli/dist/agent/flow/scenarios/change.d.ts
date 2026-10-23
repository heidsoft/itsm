import type { FlowScenario } from '../types.js';
/**
 * Change management: draft → submit → approve → schedule → start → complete →
 * close, plus the submit gate that blocks a change linked to CIs without an
 * implementation/rollback plan.
 *
 * The gate step is the reason this scenario exists: approval and rollback
 * planning are the parts of change management that quietly rot.
 */
export declare const changeLifecycle: FlowScenario;
