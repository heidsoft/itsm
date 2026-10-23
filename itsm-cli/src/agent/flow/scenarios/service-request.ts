import { apiClient } from '../../../lib/api-client.js';
import type { FlowScenario } from '../types.js';
import { blocked, idOf, requireNumber } from '../vars.js';

/**
 * Service request fulfillment: pick a real catalog offering, submit, approve,
 * provision, then confirm provisioning tasks exist.
 *
 * The scenario deliberately reads the catalog instead of hard-coding an id: a
 * hard-coded id makes the flow pass on one developer's database and fail
 * everywhere else, which trains people to ignore it.
 */
export const serviceRequestFulfillment: FlowScenario = {
  name: 'service-request-fulfillment',
  summary: 'Catalog → request → approval → provisioning → tasks, against a real catalog offering',
  domain: 'service-catalog',
  tags: ['itil', 'fulfillment', 'write'],
  requires: ['at least one enabled service catalog offering'],
  proves: [
    'a request can be submitted from a catalog offering',
    'approval moves the request out of submitted',
    'provisioning starts and exposes ordered tasks',
  ],
  steps: [
    {
      name: 'pick a catalog offering',
      endpoint: 'GET /api/v1/service-catalogs',
      run: () => apiClient.listServiceCatalog({ pageSize: 50 }),
      verify: (result) => {
        const items = ((result as { items?: unknown[] }).items ?? []);
        return [{ check: 'catalog has at least one offering', ok: items.length > 0, expected: '>=1 offering', actual: items.length }];
      },
      capture: (result) => {
        const items = ((result as { items?: Array<{ id?: unknown; status?: unknown; isActive?: unknown }> }).items ?? []);
        const usable = items.find(
          (item) => typeof item.id === 'number' && (item.status === undefined || item.status === 'enabled' || item.isActive !== false)
        );
        if (!usable || typeof usable.id !== 'number') {
          blocked('No enabled service catalog offering found; seed the catalog before running this flow.');
        }
        return { catalogId: usable.id };
      },
    },
    {
      name: 'submit service request',
      endpoint: 'POST /api/v1/service-requests',
      run: (ctx) =>
        apiClient.createServiceRequest({
          catalogId: requireNumber(ctx, 'catalogId'),
          title: `[${ctx.tag}] flow service request`,
          reason: 'Acceptance flow: verify fulfillment path',
          // Create is a compliance gate, not a plain CRUD insert: the request is
          // rejected without an acknowledgement, a data classification, and a
          // future expiry. Omitting these is what a naive client does.
          complianceAck: true,
          dataClassification: 'internal',
          expireAt: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
        }),
      expect: [{ path: 'id', exists: true }],
      capture: (result) => ({ serviceRequestId: idOf(result, 'service request create') }),
    },
    {
      name: 'read submitted request',
      endpoint: 'GET /api/v1/service-requests/:id',
      run: (ctx) => apiClient.getServiceRequest(requireNumber(ctx, 'serviceRequestId')),
      expect: [{ path: 'status', exists: true }],
    },
    {
      name: 'approve service request',
      endpoint: 'POST /api/v1/service-requests/:id/approval',
      run: (ctx) => apiClient.serviceRequestApproval(requireNumber(ctx, 'serviceRequestId'), { action: 'approve', comment: 'Flow approval' }),
      // Approval is BPMN-backed and fail-closed: if the catalog offering has no
      // process binding, the backend refuses rather than self-approving. That is
      // correct product behavior, so classify it as blocked, not failed.
      blockedWhen: {
        messageIncludes: 'BPMN 审批任务',
        reason: 'the selected catalog offering has no BPMN approval process bound',
      },
    },
    {
      name: 'read approved request',
      endpoint: 'GET /api/v1/service-requests/:id',
      run: (ctx) => apiClient.getServiceRequest(requireNumber(ctx, 'serviceRequestId')),
      verify: (result) => {
        const status = (result as { status?: unknown }).status;
        return [
          {
            check: 'approval moved the request past submitted',
            ok: typeof status === 'string' && status !== 'submitted',
            expected: 'status != submitted',
            actual: status,
          },
        ];
      },
    },
    {
      name: 'start provisioning',
      endpoint: 'POST /api/v1/service-requests/:id/provision',
      run: (ctx) => apiClient.startServiceRequestProvisioning(requireNumber(ctx, 'serviceRequestId')),
    },
    {
      name: 'provisioning tasks are exposed',
      endpoint: 'GET /api/v1/service-requests/:id/provisioning-tasks',
      run: (ctx) => apiClient.listProvisioningTasks(requireNumber(ctx, 'serviceRequestId')),
      verify: (result) => {
        const items = Array.isArray(result) ? result : ((result as { items?: unknown[] }).items ?? []);
        return [{ check: 'provisioning exposes at least one task', ok: items.length >= 1, expected: '>=1 task', actual: items.length }];
      },
    },
  ],
  teardown: [
    {
      name: 'cancel + delete flow request',
      endpoint: 'DELETE /api/v1/service-requests/:id',
      run: async (ctx) => {
        const id = ctx.vars['serviceRequestId'];
        if (typeof id !== 'number') return;
        try {
          await apiClient.deleteServiceRequest(id);
        } catch {
          // Request progressed past `submitted`, where direct delete is refused.
          // Cancel first, then delete, so the run leaves no live request behind.
          await apiClient.updateServiceRequestStatus(id, 'cancelled');
          await apiClient.deleteServiceRequest(id);
        }
      },
    },
  ],
};
