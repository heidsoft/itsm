import type { FlowScenario } from '../types.js';
/**
 * Drive a service request's approval chain to the fulfillment gate.
 *
 * The product advances one level per approval call (manager → IT → security),
 * and a parallel (quorum) level needs its other approvers. `read`/`approve` are
 * injected so the loop is testable without a backend; the scenario wires them to
 * the real ApiClient.
 *
 * Returns the final request state. Throws `blocked()` when a vote stops moving
 * the status — one identity cannot satisfy a quorum level, which is an
 * environment gap (no login accounts for the chain's approvers), not a product
 * failure.
 */
export declare function approveThroughChain(id: number, read: () => Promise<{
    status?: string;
}>, approve: () => Promise<unknown>, maxVotes?: number): Promise<{
    status?: string;
}>;
/**
 * Service request fulfillment: pick a real catalog offering, submit, approve
 * the full chain, provision, then confirm provisioning tasks exist.
 *
 * The scenario deliberately reads the catalog instead of hard-coding an id: a
 * hard-coded id makes the flow pass on one developer's database and fail
 * everywhere else, which trains people to ignore it.
 *
 * Approval is driven level by level (manager → IT → security), because the
 * product's fulfillment gate is `security_approved`: a single approve call only
 * advances one level, and a parallel (quorum) level needs its other approvers.
 * When the flow's single identity cannot advance the chain further the verdict
 * is `blocked` with the stuck status named — an environment gap (no login
 * accounts for the chain's approvers), not a product failure. The backend's own
 * scenario test covers the full chain with real approver accounts.
 */
export declare const serviceRequestFulfillment: FlowScenario;
