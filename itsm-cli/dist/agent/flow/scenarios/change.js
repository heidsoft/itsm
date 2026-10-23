import { apiClient } from '../../../lib/api-client.js';
import { idOf, requireNumber } from '../vars.js';
/**
 * Change management: draft → submit → approve → schedule → start → complete →
 * close, plus the submit gate that blocks a change linked to CIs without an
 * implementation/rollback plan.
 *
 * The gate step is the reason this scenario exists: approval and rollback
 * planning are the parts of change management that quietly rot.
 */
export const changeLifecycle = {
    name: 'change-lifecycle',
    summary: 'Change: full approval chain through close, plus the CI submit gate and terminal-state guard',
    domain: 'change',
    tags: ['itil', 'cab', 'write'],
    proves: [
        'changes are created in draft',
        'submit hands the change to the approval engine (pending)',
        'approve → schedule → start → complete → close persists every state',
        'a change linked to CIs without plans is refused at submit (422)',
        'a closed change rejects rollback',
    ],
    steps: [
        {
            name: 'create change',
            endpoint: 'POST /api/v1/changes',
            run: (ctx) => apiClient.createChange({
                title: `[${ctx.tag}] change lifecycle`,
                description: 'Acceptance flow: standard change through CAB. Safe to delete.',
                justification: 'Verify the change state machine end to end',
                type: 'normal',
                priority: 'medium',
                riskLevel: 'low',
                impactScope: 'low',
                implementationPlan: 'Apply the prepared patch and restart the service',
                rollbackPlan: 'Revert the patch and restart the previous build',
            }),
            expect: [
                { path: 'status', eq: 'draft' },
                { path: 'changeNumber', contains: 'CHG-' },
            ],
            capture: (result) => ({ changeId: idOf(result, 'change create') }),
        },
        {
            name: 'submit change',
            endpoint: 'POST /api/v1/changes/:id/submit',
            run: (ctx) => apiClient.changeLifecycle(requireNumber(ctx, 'changeId'), 'submit'),
            // The approval engine takes ownership here; `pending` is the observable
            // state (the `submitted` constant exists but is not what the API returns).
            expect: [{ path: 'status', eq: 'pending' }],
        },
        {
            name: 'approve change',
            endpoint: 'POST /api/v1/changes/:id/approve',
            run: (ctx) => apiClient.changeLifecycle(requireNumber(ctx, 'changeId'), 'approve', { comment: 'Flow: CAB approval' }),
            expect: [{ path: 'status', eq: 'approved' }],
        },
        {
            name: 'schedule change',
            endpoint: 'POST /api/v1/changes/:id/schedule',
            run: (ctx) => apiClient.changeLifecycle(requireNumber(ctx, 'changeId'), 'schedule'),
            expect: [{ path: 'status', eq: 'scheduled' }],
        },
        {
            name: 'start change',
            endpoint: 'POST /api/v1/changes/:id/start',
            run: (ctx) => apiClient.changeLifecycle(requireNumber(ctx, 'changeId'), 'start'),
            expect: [{ path: 'status', eq: 'in_progress' }],
        },
        {
            name: 'complete change',
            endpoint: 'POST /api/v1/changes/:id/complete',
            run: (ctx) => apiClient.changeLifecycle(requireNumber(ctx, 'changeId'), 'complete', { comment: 'Flow: patch applied' }),
            expect: [{ path: 'status', eq: 'completed' }],
        },
        {
            name: 'close change',
            endpoint: 'POST /api/v1/changes/:id/close',
            run: (ctx) => apiClient.changeLifecycle(requireNumber(ctx, 'changeId'), 'close'),
            expect: [{ path: 'status', eq: 'closed' }],
        },
        {
            name: 'closed change rejects rollback (negative)',
            endpoint: 'POST /api/v1/changes/:id/rollback',
            run: (ctx) => apiClient.changeLifecycle(requireNumber(ctx, 'changeId'), 'rollback', { reason: 'should be rejected' }),
            expectError: { kind: 'conflict', httpStatus: 409 },
        },
        {
            name: 'create change linked to a CI without plans',
            endpoint: 'POST /api/v1/changes',
            run: (ctx) => apiClient.createChange({
                title: `[${ctx.tag}] change submit gate`,
                description: 'Acceptance flow: submit gate negative case. Safe to delete.',
                type: 'normal',
                priority: 'low',
                affectedCis: ['CI-FLOW-GATE'],
            }),
            capture: (result) => ({ gateChangeId: idOf(result, 'change create (gate)') }),
        },
        {
            name: 'submit gate blocks unplanned CI change (negative)',
            endpoint: 'POST /api/v1/changes/:id/submit',
            run: (ctx) => apiClient.changeLifecycle(requireNumber(ctx, 'gateChangeId'), 'submit'),
            expectError: { kind: 'conflict' },
        },
    ],
    teardown: [
        {
            name: 'delete gate change',
            endpoint: 'DELETE /api/v1/changes/:id',
            run: async (ctx) => {
                const id = ctx.vars['gateChangeId'];
                if (typeof id === 'number')
                    await apiClient.deleteChange(id);
            },
        },
        {
            name: 'delete flow change',
            endpoint: 'DELETE /api/v1/changes/:id',
            run: async (ctx) => {
                const id = ctx.vars['changeId'];
                if (typeof id === 'number')
                    await apiClient.deleteChange(id);
            },
        },
    ],
};
