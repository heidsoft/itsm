import { apiClient } from '../../../lib/api-client.js';
import type { FlowScenario } from '../types.js';
import { blocked, idOf, requireNumber } from '../vars.js';

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
export async function approveThroughChain(
  id: number,
  read: () => Promise<{ status?: string }>,
  approve: () => Promise<unknown>,
  maxVotes = 6
): Promise<{ status?: string }> {
  for (let vote = 0; vote < maxVotes; vote++) {
    const before = await read();
    const status = before.status;
    if (status === undefined || status === 'security_approved' || status === 'rejected' || status === 'cancelled') {
      return before;
    }
    await approve();
    const after = await read();
    if (after.status === status) {
      blocked(
        `approval chain cannot advance past '${status}' with the flow's single identity ` +
          '(a parallel level needs its other approvers); run this flow with accounts that satisfy the chain'
      );
    }
  }
  return read();
}

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
export const serviceRequestFulfillment: FlowScenario = {
  name: 'service-request-fulfillment',
  summary: 'Catalog → request → full approval chain → provisioning → tasks, against a real catalog offering',
  domain: 'service-catalog',
  tags: ['itil', 'fulfillment', 'write'],
  requires: ['at least one enabled service catalog offering'],
  proves: [
    'a request can be submitted from a catalog offering',
    'the approval chain advances level by level to the fulfillment gate',
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
      name: 'approve through the full chain',
      endpoint: 'POST /api/v1/service-requests/:id/approval',
      run: async (ctx) => {
        const id = requireNumber(ctx, 'serviceRequestId');
        // 审批链是逐级推进的（manager → IT → security），一次审批只走一级；
        // 会签层（parallel）还需要多人投票。用当前身份尽量推进整条链：
        // 状态不再变化说明会签层在等其他审批人——那是环境前置（缺可登录的
        // 审批账号），判 blocked，不判产品失败。单级/串行链则一轮即达
        // security_approved，直接进入交付。
        return approveThroughChain(
          id,
          () => apiClient.getServiceRequest(id) as Promise<{ status?: string }>,
          () => apiClient.serviceRequestApproval(id, { action: 'approve', comment: 'Flow approval' })
        );
      },
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
            check: 'approval chain reached the fulfillment gate (security_approved)',
            ok: status === 'security_approved',
            expected: 'security_approved',
            actual: status,
          },
        ];
      },
    },
    {
      name: 'start provisioning',
      endpoint: 'POST /api/v1/service-requests/:id/provision',
      run: (ctx) => apiClient.startServiceRequestProvisioning(requireNumber(ctx, 'serviceRequestId')),
      // 状态守卫：链未走完就启动交付必须被拒（409 + 缺哪级审批的文案）。
      // 走到这里说明上一步的链推进判定与守卫判断不一致——先按环境前置
      // 记录，不吞掉真正的产品回归（消息不含守卫文案时仍会 failed）。
      blockedWhen: {
        httpStatus: 409,
        messageIncludes: '启动交付',
        reason: 'approval chain did not reach security_approved; provisioning is correctly refused until it does',
      },
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
