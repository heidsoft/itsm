import { cmd } from '../spec.js';
import { requirePositional, withPagination, body, stringFlag } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
import { usageError } from '../../lib/errors.js';

/** Service catalog (browse) & service requests (fulfillment lifecycle). */

export function serviceCatalogCommands() {
  return [
    cmd('catalog.list', 'Browse service catalog offerings', 'itsm catalog list [--page N]', ['GET /api/v1/service-catalogs'], async (ctx) => apiClient.listServiceCatalog(withPagination(ctx.args))),
    cmd('catalog.get', 'Catalog item detail', 'itsm catalog get <id>', ['GET /api/v1/service-catalogs/:id'], async (ctx) => apiClient.getServiceCatalogItem(requirePositional(ctx.args, 0, 'catalogId'))),
    cmd('catalog.search', 'Search catalog', 'itsm catalog search <q>', ['GET /api/v1/service-catalogs/search'], async (ctx) => apiClient.searchServiceCatalog(requirePositional(ctx.args, 0, 'query'))),
    cmd('request.list', 'List service requests', 'itsm request list [--page N]', ['GET /api/v1/service-requests'], async (ctx) => apiClient.listServiceRequests(withPagination(ctx.args))),
    cmd('request.get', 'Service request detail', 'itsm request get <id>', ['GET /api/v1/service-requests/:id'], async (ctx) => apiClient.getServiceRequest(requirePositional(ctx.args, 0, 'requestId'))),
    cmd('request.create', 'Submit a service request (from a catalog offering)', 'itsm request create --data <json>', ['POST /api/v1/service-requests'], async (ctx) => apiClient.createServiceRequest(body(ctx.args) ?? {})),
    cmd('request.update', 'Update a service request', 'itsm request update <id> [--data <json>]', ['PUT /api/v1/service-requests/:id'], async (ctx) => apiClient.updateServiceRequest(requirePositional(ctx.args, 0, 'requestId'), body(ctx.args) ?? {})),
    cmd('request.status', 'Update fulfillment status', 'itsm request status <id> --status <value>', ['PUT /api/v1/service-requests/:id/status'], async (ctx) => {
      const requestId = requirePositional(ctx.args, 0, 'requestId');
      const status = body(ctx.args)?.status;
      if (typeof status !== 'string' || !status) throw usageError('Missing required --data status field, e.g. --data \'{"status":"approved"}\'');
      return apiClient.updateServiceRequestStatus(requestId, status);
    }),
    cmd('request.approval', 'Approve/reject a service request (action required)', 'itsm request approval <id> --action approve|reject [--comment text]', ['POST /api/v1/service-requests/:id/approval', 'POST /api/v1/service-requests/:id/approvals'], async (ctx) => {
      const requestId = requirePositional(ctx.args, 0, 'requestId');
      const action = stringFlag(ctx.args.flags, 'action');
      if (action !== 'approve' && action !== 'reject') throw usageError('Missing required flag --action (approve|reject)');
      const comment = stringFlag(ctx.args.flags, 'comment');
      return apiClient.serviceRequestApproval(requestId, comment ? { action, comment } : { action });
    }),
    cmd('request.provision', 'Kick off provisioning', 'itsm request provision <id> [--data <json>]', ['POST /api/v1/service-requests/:id/provision'], async (ctx) => apiClient.startServiceRequestProvisioning(requirePositional(ctx.args, 0, 'requestId'), body(ctx.args))),
    cmd('request.provisioning-tasks', 'Provisioning task list for a request', 'itsm request provisioning-tasks <id>', ['GET /api/v1/service-requests/:id/provisioning-tasks'], async (ctx) => apiClient.listProvisioningTasks(requirePositional(ctx.args, 0, 'requestId'))),
    cmd('request.sla', 'Service request SLA state (unbound vs active/warning/breached)', 'itsm request sla <id>', ['GET /api/v1/service-requests/:id/sla'], async (ctx) => apiClient.getServiceRequestSLA(requirePositional(ctx.args, 0, 'requestId'))),
    cmd('request.approvals', 'Approval steps for a service request', 'itsm request approvals <id>', ['GET /api/v1/service-requests/:id/approvals'], async (ctx) => apiClient.listServiceRequestApprovals(requirePositional(ctx.args, 0, 'requestId'))),
    cmd('request.delete', 'Delete a service request (submitted/rejected/cancelled only)', 'itsm request delete <id>', ['DELETE /api/v1/service-requests/:id'], async (ctx) => apiClient.deleteServiceRequest(requirePositional(ctx.args, 0, 'requestId'))),
    cmd('provisioning.execute', 'Execute (or re-execute) a provisioning task', 'itsm provisioning execute <taskId> [--data <json>]', ['POST /api/v1/provisioning-tasks/:id/execute'], async (ctx) => apiClient.provisioningTaskAction(requirePositional(ctx.args, 0, 'taskId'), 'execute', body(ctx.args))),
    cmd('provisioning.retry', 'Requeue a failed provisioning task', 'itsm provisioning retry <taskId> [--data <json>]', ['POST /api/v1/provisioning-tasks/:id/retry'], async (ctx) => apiClient.provisioningTaskAction(requirePositional(ctx.args, 0, 'taskId'), 'retry', body(ctx.args))),
    cmd('provisioning.resolve', 'Manually resolve a permanently failed provisioning task', 'itsm provisioning resolve <taskId> --data <json>', ['POST /api/v1/provisioning-tasks/:id/resolve'], async (ctx) => apiClient.provisioningTaskAction(requirePositional(ctx.args, 0, 'taskId'), 'resolve', body(ctx.args))),
  ];
}
