import type { FlowScenario } from '../types.js';
/**
 * Service request fulfillment: pick a real catalog offering, submit, approve,
 * provision, then confirm provisioning tasks exist.
 *
 * The scenario deliberately reads the catalog instead of hard-coding an id: a
 * hard-coded id makes the flow pass on one developer's database and fail
 * everywhere else, which trains people to ignore it.
 */
export declare const serviceRequestFulfillment: FlowScenario;
